package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

// Store holds the read-only DB handle plus in-memory decode maps for Prism's
// lookup tables (loaded once at startup — they are tiny and change rarely).
type Store struct {
	db   *sql.DB
	per  map[string]string // T_Personel   id -> name
	ofi  map[string]string // T_Ofis       id -> name
	ace  map[string]string // T_Acenta     id -> name
	dur  map[string]string // Tn_DurumSec  id -> label
	kis  map[string]string // Tn_Kis       id -> label
	dov  map[string]string // Tn_Doviz     id -> code
	proj map[string]string // T_Proje      id -> name
	son  map[string]string // Tn_MusteriSonuc          id -> label
	pua  map[string]string // Tn_MusteriSonucPuanlama  id -> label
	ulke map[string]string // Tn_AdresUlke id -> country name
	kay  map[string]string // Tn_MusteriKaynagi id -> lead source
	crm  *crmStore         // optional live CRM (read-only)
}

func Open(dsn, crmDSN string) (*Store, error) {
	d, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(8)
	d.SetConnMaxIdleTime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := d.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	s := &Store{db: d}
	if err := s.loadMaps(); err != nil {
		return nil, fmt.Errorf("load lookups: %w", err)
	}
	s.crm = openCRM(crmDSN) // nil if unset/unreachable — enrichment just stays off
	return s, nil
}

// CRMLinked reports whether the live CRM enrichment is active.
func (s *Store) CRMLinked() bool { return s.crm != nil }

// query is the ONLY execution path in this service. Everything is a SELECT;
// there is no Exec anywhere in the codebase, so the UI cannot mutate Prism.
// The context must outlive the returned *sql.Rows (scanning happens after this
// returns), so we do not cancel here — the pool's ConnMaxIdleTime and the
// driver's own timeouts bound the work.
func (s *Store) query(q string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(context.Background(), q, args...)
}

func loadMap(s *Store, table, idCol, labelCol string) (map[string]string, error) {
	m := map[string]string{}
	rows, err := s.query(fmt.Sprintf("SELECT [%s],[%s] FROM [dbo].[%s]", idCol, labelCol, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id sql.NullString
		var lab sql.NullString
		if err := rows.Scan(&id, &lab); err != nil {
			return nil, err
		}
		m[id.String] = lab.String
	}
	return m, rows.Err()
}

func (s *Store) loadMaps() error {
	var err error
	if s.per, err = loadMap(s, "T_Personel", "ID", "PersonelAdi"); err != nil {
		return err
	}
	if s.ofi, err = loadMap(s, "T_Ofis", "ID", "OfisAdi"); err != nil {
		return err
	}
	if s.ace, err = loadMap(s, "T_Acenta", "ID", "AcentaAdi"); err != nil {
		return err
	}
	if s.dur, err = loadMap(s, "Tn_DurumSec", "ID", "DurumSecAciklama"); err != nil {
		return err
	}
	if s.kis, err = loadMap(s, "Tn_Kis", "ID", "Aciklama"); err != nil {
		return err
	}
	if s.dov, err = loadMap(s, "Tn_Doviz", "ID", "DovizKodu"); err != nil {
		return err
	}
	s.proj, _ = loadMap(s, "T_Proje", "ID", "ProjeAdi")
	s.son, _ = loadMap(s, "Tn_MusteriSonuc", "ID", "SonSonucAdi")
	s.pua, _ = loadMap(s, "Tn_MusteriSonucPuanlama", "ID", "Puanlama")
	s.ulke, _ = loadMap(s, "Tn_AdresUlke", "ID", "UlkeAdi")
	s.kay, _ = loadMap(s, "Tn_MusteriKaynagi", "ID", "MusteriKaynagi")
	return nil
}

func dec(m map[string]string, v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	k := strings.TrimSpace(v.String)
	if k == "" || k == "0" || k == "-1" {
		return ""
	}
	if lab, ok := m[k]; ok {
		return lab
	}
	return ""
}

func (s *Store) cur(v sql.NullString) string {
	c := dec(s.dov, v)
	if c == "TL" || c == "₺" {
		return "TRY"
	}
	return c
}
