package db

import (
	"context"
	"database/sql"
	"regexp"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// crmStore is an optional, read-only connection to a live CRM database (Postgres).
// It lets the Prism viewer show whether a Prism person also exists in the CRM,
// with the CRM's much richer source + stage history. The DSN pins
// default_transaction_read_only=on, and only SELECTs are issued here.
type crmStore struct {
	db *sql.DB
}

func openCRM(dsn string) *crmStore {
	if dsn == "" {
		return nil
	}
	d, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil
	}
	d.SetMaxOpenConns(4)
	d.SetConnMaxIdleTime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.PingContext(ctx); err != nil {
		return nil
	}
	return &crmStore{db: d}
}

var digitsRe = regexp.MustCompile(`\D`)

// normPhone turns a Prism phone into E.164 the CRM stores (+90XXXXXXXXXX).
func normPhone(raw string) string {
	d := digitsRe.ReplaceAllString(raw, "")
	switch {
	case len(d) == 10 && d[0] == '5':
		return "+90" + d
	case len(d) == 11 && d[0] == '0' && d[1] == '5':
		return "+90" + d[1:]
	case len(d) == 12 && d[0] == '9' && d[1] == '0':
		return "+" + d
	default:
		return ""
	}
}

type CRMStage struct {
	From string `json:"from"`
	To   string `json:"to"`
	At   string `json:"at"`
}

type CRMInfo struct {
	InCRM        bool       `json:"in_crm"`
	Phone        string     `json:"phone"`
	Name         string     `json:"name"`
	Source       string     `json:"source"`
	Status       string     `json:"status"`
	CreatedAt    string     `json:"created_at"`
	FirstContact string     `json:"first_contact_at"`
	Summary      string     `json:"summary"`
	Stages       []CRMStage `json:"stages"`
}

func (c *crmStore) enrich(prismPhone string) *CRMInfo {
	if c == nil {
		return nil
	}
	phone := normPhone(prismPhone)
	if phone == "" {
		return &CRMInfo{InCRM: false}
	}
	info := &CRMInfo{Phone: phone}
	var id string
	var name, source, status, created, first, summary sql.NullString
	err := c.db.QueryRow(
		`SELECT id, display_name, source, status,
		        to_char(created_at,'YYYY-MM-DD'), to_char(first_contact_at,'YYYY-MM-DD'), summary
		 FROM leads WHERE phone_number=$1 LIMIT 1`, phone,
	).Scan(&id, &name, &source, &status, &created, &first, &summary)
	if err != nil {
		return &CRMInfo{InCRM: false, Phone: phone}
	}
	info.InCRM = true
	info.Name = name.String
	info.Source = source.String
	info.Status = status.String
	info.CreatedAt = created.String
	info.FirstContact = first.String
	info.Summary = summary.String

	rows, err := c.db.Query(
		`SELECT COALESCE(from_status,''), COALESCE(to_status,''), to_char(created_at,'YYYY-MM-DD')
		 FROM lead_stage_events WHERE contact_id=$1 ORDER BY created_at`, id)
	if err == nil {
		for rows.Next() {
			var f, t, at sql.NullString
			rows.Scan(&f, &t, &at)
			info.Stages = append(info.Stages, CRMStage{From: f.String, To: t.String, At: at.String})
		}
		rows.Close()
	}
	if info.Stages == nil {
		info.Stages = []CRMStage{}
	}
	return info
}
