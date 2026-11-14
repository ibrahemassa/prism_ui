package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"prismui/internal/commissions"
	"prismui/internal/db"
	"prismui/internal/sync"
)

type Server struct {
	store  *db.Store
	pusher *sync.Pusher
	// static is the built frontend (frontend/dist), served on / so the whole
	// thing is one process on one port for every machine on the office LAN.
	// Empty = API only (the Vite dev server proxies to us instead).
	static string
	comm   *commissions.Store // agent commission rules + payouts (a local file)
}

func New(store *db.Store, pusher *sync.Pusher, comm *commissions.Store, static string) *Server {
	return &Server{store: store, pusher: pusher, comm: comm, static: static}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /api/stats", s.stats)
	mux.HandleFunc("GET /api/leads", s.leads)
	mux.HandleFunc("GET /api/leads/{id}", s.leadDetail)
	mux.HandleFunc("GET /api/units", s.units)
	mux.HandleFunc("GET /api/deals", s.deals)
	mux.HandleFunc("GET /api/deals/{id}", s.dealDetail)
	mux.HandleFunc("GET /api/agents", s.agents)
	mux.HandleFunc("GET /api/agencies", s.agencies)
	mux.HandleFunc("GET /api/reports", s.reports)
	mux.HandleFunc("GET /api/reports/full", s.fullReport)
	// CRM sync: status, a look at what would be sent, and a manual push.
	// The push is the one POST this service accepts — it writes nothing to
	// Prism, it sends Prism's data out — and readOnly lets exactly it through.
	mux.HandleFunc("GET /api/sync/status", s.syncStatus)
	mux.HandleFunc("GET /api/sync/preview", s.syncPreview)
	mux.HandleFunc("POST /api/sync/run", s.syncRun)
	// Commissions. The two POSTs write the commission settings file next to
	// this service (agents, rates, what was paid out) — never Prism.
	mux.HandleFunc("GET /api/commissions", s.commissionsGet)
	mux.HandleFunc("GET /api/commissions/export.xlsx", s.commissionsExport)
	mux.HandleFunc("POST /api/commissions/agents", s.commissionsSetAgents)
	mux.HandleFunc("POST /api/commissions/payouts", s.commissionsSetPaid)
	if s.static != "" {
		mux.Handle("/", spa(s.static))
	}
	return cors(readOnly(mux))
}

// spa serves the built frontend with the usual single-page fallback: a path
// that is not a file gets index.html so client-side routes deep-link.
func spa(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	index := filepath.Join(dir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean("/" + r.URL.Path)
		if clean != "/" {
			if st, err := os.Stat(filepath.Join(dir, clean)); err == nil && !st.IsDir() {
				files.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, index)
	})
}

func (s *Server) syncStatus(w http.ResponseWriter, r *http.Request) {
	if s.pusher == nil {
		writeJSON(w, sync.Status{})
		return
	}
	writeJSON(w, s.pusher.Status())
}

func (s *Server) syncPreview(w http.ResponseWriter, r *http.Request) {
	if s.pusher == nil {
		http.Error(w, "sync not configured", http.StatusNotFound)
		return
	}
	snap, err := s.pusher.Build()
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, snap)
}

func (s *Server) syncRun(w http.ResponseWriter, r *http.Request) {
	if s.pusher == nil {
		http.Error(w, "sync not configured", http.StatusNotFound)
		return
	}
	result, err := s.pusher.RunOnce(r.Context())
	if err != nil {
		if errors.Is(err, sync.ErrBusy) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]any{"error": err.Error(), "status": s.pusher.Status()})
		return
	}
	writeJSON(w, map[string]any{"result": result, "status": s.pusher.Status()})
}

// readOnly rejects any non-GET method at the edge — belt-and-braces on top of
// the fact that the data layer only ever issues SELECTs. The POSTs let through
// push Prism's data out (/api/sync/) or save the local commission settings
// (/api/commissions/); none of them writes to Prism.
func readOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodOptions &&
			!(r.Method == http.MethodPost && (strings.HasPrefix(r.URL.Path, "/api/sync/") || strings.HasPrefix(r.URL.Path, "/api/commissions/"))) {
			http.Error(w, "read-only service", http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	st, err := s.store.Stats()
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, st)
}

func (s *Server) leads(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	rows, total, err := s.store.Leads(q.Get("q"), limit, offset)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, map[string]any{"rows": rows, "total": total})
}

func (s *Server) leadDetail(w http.ResponseWriter, r *http.Request) {
	d, err := s.store.LeadDetail(r.PathValue("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		fail(w, err)
		return
	}
	writeJSON(w, d)
}

func (s *Server) dealDetail(w http.ResponseWriter, r *http.Request) {
	d, err := s.store.DealDetail(r.PathValue("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		fail(w, err)
		return
	}
	writeJSON(w, d)
}

func (s *Server) units(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.Units()
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, map[string]any{"rows": rows, "total": len(rows)})
}

func (s *Server) deals(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.Deals()
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, map[string]any{"rows": rows, "total": len(rows)})
}

func (s *Server) agents(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.Agents()
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, map[string]any{"rows": rows, "total": len(rows)})
}

func (s *Server) agencies(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.Agencies()
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, map[string]any{"rows": rows, "total": len(rows)})
}

func (s *Server) fullReport(w http.ResponseWriter, r *http.Request) {
	rep, err := s.store.FullReport()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, rep)
}

func (s *Server) reports(w http.ResponseWriter, r *http.Request) {
	rep, err := s.store.Reports()
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, rep)
}
