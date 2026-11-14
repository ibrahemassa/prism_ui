// Package sync pushes a full Prism snapshot to an external CRM on a timer.
//
// Direction is deliberate: the Prism database is only reachable from the
// office network, so nothing outside can pull from here. This process makes
// one outbound HTTPS call per pass and the CRM does the matching. The payload
// is a whole snapshot every time — a few hundred units and people —
// which is small enough that watermarks and cursors would be more code than
// they save, and it means a run is always safe to repeat.
package sync

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"prismui/internal/db"
)

// Config comes from the environment (see main.go).
type Config struct {
	URL      string        // CRM ingest endpoint, e.g. https://crm.example/api/integrations/prism/snapshot
	Key      string        // bearer token the CRM was configured with
	Interval time.Duration // time between passes; 0 = never on a timer
	DryRun   bool          // ask the CRM to diff without writing
}

// Enabled reports whether a push can happen at all.
func (c Config) Enabled() bool { return c.URL != "" && c.Key != "" }

// Status is what the UI and /api/sync/status show.
type Status struct {
	Enabled     bool           `json:"enabled"`
	URL         string         `json:"url"`
	Interval    string         `json:"interval"`
	DryRun      bool           `json:"dry_run"`
	Running     bool           `json:"running"`
	LastStarted *time.Time     `json:"last_started,omitempty"`
	LastOK      *time.Time     `json:"last_ok,omitempty"`
	LastError   string         `json:"last_error,omitempty"`
	LastErrorAt *time.Time     `json:"last_error_at,omitempty"`
	LastResult  map[string]any `json:"last_result,omitempty"`
	Counts      map[string]int `json:"counts,omitempty"`
	DurationMS  int64          `json:"duration_ms,omitempty"`
	Runs        int            `json:"runs"`
}

// Pusher builds and sends snapshots.
type Pusher struct {
	cfg    Config
	store  *db.Store
	client *http.Client

	mu     sync.Mutex
	status Status
}

func New(cfg Config, store *db.Store) *Pusher {
	return &Pusher{
		cfg:    cfg,
		store:  store,
		client: &http.Client{Timeout: 120 * time.Second},
		status: Status{Enabled: cfg.Enabled(), URL: cfg.URL, Interval: cfg.Interval.String(), DryRun: cfg.DryRun},
	}
}

func (p *Pusher) Status() Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status
}

// Start runs passes on the interval until ctx ends. The first pass happens
// shortly after start so a freshly booted box reports within a minute.
func (p *Pusher) Start(ctx context.Context) {
	if !p.cfg.Enabled() || p.cfg.Interval <= 0 {
		return
	}
	go func() {
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
			if _, err := p.RunOnce(ctx); err != nil {
				log.Printf("sync: %v", err)
			}
			timer.Reset(p.cfg.Interval)
		}
	}()
}

// Snapshot is the payload the CRM ingests. Row shapes are documented in
// internal/db/sync.go.
type Snapshot struct {
	Version       int              `json:"version"`
	Source        string           `json:"source"`
	GeneratedAt   string           `json:"generated_at"`
	DryRun        bool             `json:"dry_run"`
	Agents        []db.Agent       `json:"agents"`
	Agencies      []db.Agency      `json:"agencies"`
	Leads         []db.SyncLead    `json:"leads"`
	Stock         []map[string]any `json:"stock"`
	Cancellations []map[string]any `json:"cancellations"`
	Ledgers       []db.DealLedger  `json:"ledgers"`
}

// Build reads everything from Prism. Exposed so a pass can be inspected
// without sending it (GET /api/sync/preview).
func (p *Pusher) Build() (*Snapshot, error) {
	agents, err := p.store.Agents()
	if err != nil {
		return nil, fmt.Errorf("agents: %w", err)
	}
	agencies, err := p.store.Agencies()
	if err != nil {
		return nil, fmt.Errorf("agencies: %w", err)
	}
	leads, err := p.store.LeadRows()
	if err != nil {
		return nil, fmt.Errorf("leads: %w", err)
	}
	stock, err := p.store.StockRows()
	if err != nil {
		return nil, fmt.Errorf("stock: %w", err)
	}
	cancels, err := p.store.CancellationRows()
	if err != nil {
		return nil, fmt.Errorf("cancellations: %w", err)
	}
	ledgers, err := p.store.Ledgers()
	if err != nil {
		return nil, fmt.Errorf("ledgers: %w", err)
	}
	return &Snapshot{
		Version:       1,
		Source:        "prism_ui",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		DryRun:        p.cfg.DryRun,
		Agents:        agents,
		Agencies:      agencies,
		Leads:         leads,
		Stock:         stock,
		Cancellations: cancels,
		Ledgers:       ledgers,
	}, nil
}

// RunOnce builds a snapshot and posts it. Concurrent calls collapse: a second
// caller while one is in flight gets ErrBusy.
func (p *Pusher) RunOnce(ctx context.Context) (map[string]any, error) {
	if !p.cfg.Enabled() {
		return nil, errors.New("sync is not configured (CRM_INGEST_URL / CRM_INGEST_KEY)")
	}
	p.mu.Lock()
	if p.status.Running {
		p.mu.Unlock()
		return nil, ErrBusy
	}
	now := time.Now()
	p.status.Running = true
	p.status.LastStarted = &now
	p.status.Runs++
	p.mu.Unlock()

	result, counts, err := p.push(ctx)

	p.mu.Lock()
	defer p.mu.Unlock()
	done := time.Now()
	p.status.Running = false
	p.status.DurationMS = done.Sub(now).Milliseconds()
	p.status.Counts = counts
	if err != nil {
		p.status.LastError = err.Error()
		p.status.LastErrorAt = &done
		return nil, err
	}
	p.status.LastOK = &done
	p.status.LastError = ""
	p.status.LastErrorAt = nil
	p.status.LastResult = result
	return result, nil
}

var ErrBusy = errors.New("a sync pass is already running")

func (p *Pusher) push(ctx context.Context) (map[string]any, map[string]int, error) {
	snap, err := p.Build()
	if err != nil {
		return nil, nil, err
	}
	counts := map[string]int{
		"agents": len(snap.Agents), "agencies": len(snap.Agencies), "leads": len(snap.Leads),
		"stock": len(snap.Stock), "cancellations": len(snap.Cancellations), "ledgers": len(snap.Ledgers),
	}

	var body bytes.Buffer
	gz := gzip.NewWriter(&body)
	if err := json.NewEncoder(gz).Encode(snap); err != nil {
		return nil, counts, err
	}
	if err := gz.Close(); err != nil {
		return nil, counts, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.URL, &body)
	if err != nil {
		return nil, counts, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Authorization", "Bearer "+p.cfg.Key)
	req.Header.Set("User-Agent", "prism_ui-sync/1")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, counts, fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode/100 != 2 {
		msg := string(raw)
		if len(msg) > 500 {
			msg = msg[:500]
		}
		return nil, counts, fmt.Errorf("crm answered %d: %s", resp.StatusCode, msg)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, counts, fmt.Errorf("crm answer is not JSON: %w", err)
	}
	log.Printf("sync: pushed %v → %v", counts, summary(result))
	return result, counts, nil
}

func summary(result map[string]any) string {
	if s, ok := result["summary"]; ok {
		b, _ := json.Marshal(s)
		return string(b)
	}
	b, _ := json.Marshal(result)
	if len(b) > 300 {
		return string(b[:300]) + "…"
	}
	return string(b)
}
