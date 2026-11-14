// Package commissions works out what each sales agent is owed and when.
//
// The office rule: once a deal's peşinat (down payment) is fully paid AND its
// contract is signed, the agent's commission on it is billed the month after.
// The per-agent settings (who is tracked, their rate, the rule note) and the
// record of which commissions were paid out live in a small JSON file next to
// the service. That file is the only thing this package writes — Prism is
// never touched.
package commissions

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	BasisSalePrice = "sale_price" // a.SatisFiyati — what the buyer agreed to pay
	BasisListPrice = "list_price" // the list price the sale was written against
)

// Agent is one tracked agent and the rule their commission follows.
type Agent struct {
	PersonnelID     string  `json:"personnel_id"` // T_Personel.ID — the deal's Satış Personeli 1
	Name            string  `json:"name"`
	Rate            float64 `json:"rate"`              // percent: 1.5 means 1.5 %
	Basis           string  `json:"basis"`             // BasisSalePrice | BasisListPrice
	NeedDownPayment bool    `json:"need_down_payment"` // peşinat must be fully collected
	NeedContract    bool    `json:"need_contract"`     // the contract must be signed
	DelayMonths     int     `json:"delay_months"`      // billed this many months after it is earned
	AgencyDeals     bool    `json:"agency_deals"`      // count deals an agency (acenta) brought in
	Since           string  `json:"since"`             // YYYY-MM-DD: only deals made on/after; "" = all
	Rule            string  `json:"rule"`              // free-text rule, printed on the reports
}

// Payout records that a commission was paid, with what was paid at the time —
// a later rate change does not rewrite history.
type Payout struct {
	AgentID  string  `json:"agent_id"`
	PaidAt   string  `json:"paid_at"` // YYYY-MM-DD
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Month    string  `json:"month"` // the billing month it was paid against
}

type Config struct {
	Agents  []Agent           `json:"agents"`
	Payouts map[string]Payout `json:"payouts"` // by deal (T_Aktivite.ID)
}

const defaultRule = "Peşinat tamamen ödendi ve sözleşme imzalandı → komisyon ertesi ay faturalanır."

// NewAgent is a tracked agent on the office's standard rule, rate not yet set.
func NewAgent(id, name string) Agent {
	return Agent{
		PersonnelID: id, Name: name, Basis: BasisSalePrice,
		NeedDownPayment: true, NeedContract: true, DelayMonths: 1, AgencyDeals: true,
		Rule: defaultRule,
	}
}

// DefaultAgents picks the seed agents out of Prism's personnel list, by
// first name (case and Turkish letters folded, so "ayse" matches "Ayşe").
func DefaultAgents(personnel map[string]string, firstNames []string) []Agent {
	out := []Agent{}
	for _, want := range firstNames {
		want = fold(want)
		if want == "" {
			continue
		}
		for id, name := range personnel {
			first, _, _ := strings.Cut(fold(name), " ")
			if first == want {
				out = append(out, NewAgent(id, strings.TrimSpace(name)))
				break
			}
		}
	}
	return out
}

func fold(s string) string {
	return strings.NewReplacer("ı", "i", "ş", "s", "ğ", "g", "ü", "u", "ö", "o", "ç", "c", "i̇", "i").
		Replace(strings.ToLower(strings.TrimSpace(s)))
}

// Store is the settings file. Reads hand out copies; every write lands
// atomically (temp file + rename) so a crash never leaves half a file.
type Store struct {
	path string
	mu   sync.Mutex
	cfg  Config
}

// Open loads the settings file, creating it from seed when it does not exist.
func Open(path string, seed []Agent) (*Store, error) {
	s := &Store{path: path}
	b, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		s.cfg = Config{Agents: seed, Payouts: map[string]Payout{}}
		if err := s.save(); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(b, &s.cfg); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	if s.cfg.Agents == nil {
		s.cfg.Agents = []Agent{}
	}
	if s.cfg.Payouts == nil {
		s.cfg.Payouts = map[string]Payout{}
	}
	return s, nil
}

func (s *Store) Get() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.clone()
}

// SetAgents replaces the tracked agents and their rules.
func (s *Store) SetAgents(agents []Agent) (Config, error) {
	seen := map[string]bool{}
	for i := range agents {
		a := &agents[i]
		a.PersonnelID = strings.TrimSpace(a.PersonnelID)
		a.Name = strings.TrimSpace(a.Name)
		a.Rule = strings.TrimSpace(a.Rule)
		a.Since = strings.TrimSpace(a.Since)
		switch {
		case a.PersonnelID == "":
			return Config{}, errors.New("personel seçilmedi")
		case seen[a.PersonnelID]:
			return Config{}, fmt.Errorf("%s iki kez eklenmiş", a.Name)
		case a.Rate < 0 || a.Rate > 100:
			return Config{}, fmt.Errorf("%s: oran 0 ile 100 arasında olmalı", a.Name)
		case a.Basis != BasisSalePrice && a.Basis != BasisListPrice:
			return Config{}, fmt.Errorf("%s: geçersiz komisyon bazı %q", a.Name, a.Basis)
		case a.DelayMonths < 0 || a.DelayMonths > 24:
			return Config{}, fmt.Errorf("%s: fatura gecikmesi 0–24 ay olmalı", a.Name)
		}
		if a.Since != "" {
			if _, err := time.Parse(time.DateOnly, a.Since); err != nil {
				return Config{}, fmt.Errorf("%s: başlangıç tarihi YYYY-AA-GG olmalı", a.Name)
			}
		}
		seen[a.PersonnelID] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.cfg.Agents
	s.cfg.Agents = agents
	if err := s.save(); err != nil {
		s.cfg.Agents = prev
		return Config{}, err
	}
	return s.cfg.clone(), nil
}

// SetPayouts records (p != nil) or clears (p == nil) the payout on each deal.
func (s *Store) SetPayouts(set map[string]*Payout) (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.cfg.clone().Payouts
	for id, p := range set {
		if p == nil {
			delete(s.cfg.Payouts, id)
		} else {
			s.cfg.Payouts[id] = *p
		}
	}
	if err := s.save(); err != nil {
		s.cfg.Payouts = prev
		return Config{}, err
	}
	return s.cfg.clone(), nil
}

func (s *Store) save() error {
	b, err := json.MarshalIndent(s.cfg, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".commissions-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.path)
}

func (c Config) clone() Config {
	out := Config{Agents: append([]Agent{}, c.Agents...), Payouts: make(map[string]Payout, len(c.Payouts))}
	maps.Copy(out.Payouts, c.Payouts)
	return out
}

// AgentIDs is the tracked personnel IDs, sorted.
func (c Config) AgentIDs() []string {
	ids := make([]string, 0, len(c.Agents))
	for _, a := range c.Agents {
		ids = append(ids, a.PersonnelID)
	}
	sort.Strings(ids)
	return ids
}
