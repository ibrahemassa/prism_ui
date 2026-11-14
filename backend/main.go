package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"prismui/internal/api"
	"prismui/internal/commissions"
	"prismui/internal/db"
	"prismui/internal/sync"
)

// dsnFromEnv builds a read-only SQL Server DSN. Preference order:
//  1. PRISM_DSN                     — a full sqlserver:// URL
//  2. PRISM_DB_* individual vars    — server/user/password/database
//  3. PRISM_CONFIG_DIR              — parse Prism's own Model.dll.config +
//     DBList.txt (so no secret lives in this repo; point it at the install
//     folder or the dump copy).
func dsnFromEnv() string {
	if v := os.Getenv("PRISM_DSN"); v != "" {
		return v
	}
	server := os.Getenv("PRISM_DB_SERVER")
	user := os.Getenv("PRISM_DB_USER")
	pass := os.Getenv("PRISM_DB_PASSWORD")
	name := os.Getenv("PRISM_DB_DATABASE")

	if dir := os.Getenv("PRISM_CONFIG_DIR"); dir != "" && (server == "" || user == "") {
		if b, err := os.ReadFile(filepath.Join(dir, "Model.dll.config")); err == nil {
			cfg := string(b)
			if m := regexp.MustCompile(`user id=([^;]+);`).FindStringSubmatch(cfg); m != nil {
				user = m[1]
			}
			if m := regexp.MustCompile(`password=([^;]+);`).FindStringSubmatch(cfg); m != nil {
				pass = m[1]
			}
		}
		if b, err := os.ReadFile(filepath.Join(dir, "DBList.txt")); err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				p := strings.Split(strings.TrimSpace(line), ";")
				if len(p) >= 3 && p[0] == "PRISM_PLUS" {
					server, name = p[1], p[2]
				}
			}
		}
	}
	if server == "" || user == "" {
		return ""
	}
	if name == "" {
		name = "PRISM_PLUS"
	}
	u := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(user, pass),
		Host:   server,
	}
	q := u.Query()
	q.Set("database", name)
	q.Set("app name", "prism_ui (read-only)")
	q.Set("connection timeout", "15")
	u.RawQuery = q.Encode()
	return u.String()
}

func main() {
	dsn := dsnFromEnv()
	if dsn == "" {
		log.Fatal("no DB config: set PRISM_DSN, or PRISM_DB_SERVER/USER/PASSWORD/DATABASE, or PRISM_CONFIG_DIR")
	}
	store, err := db.Open(dsn, os.Getenv("PRISM_CRM_DSN"))
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	if store.CRMLinked() {
		log.Print("live CRM enrichment: ON (read-only)")
	} else {
		log.Print("live CRM enrichment: off (set PRISM_CRM_DSN to enable)")
	}
	addr := os.Getenv("PRISM_UI_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	// Outbound push to the CRM. Off unless both are set.
	cfg := sync.Config{
		URL:      strings.TrimSpace(os.Getenv("CRM_INGEST_URL")),
		Key:      strings.TrimSpace(os.Getenv("CRM_INGEST_KEY")),
		Interval: 10 * time.Minute,
		DryRun:   os.Getenv("CRM_INGEST_DRY_RUN") == "1",
	}
	if v := os.Getenv("SYNC_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("SYNC_INTERVAL %q: %v", v, err)
		}
		cfg.Interval = d
	}
	pusher := sync.New(cfg, store)
	if cfg.Enabled() {
		log.Printf("CRM sync: ON → %s every %s (dry_run=%v)", cfg.URL, cfg.Interval, cfg.DryRun)
		pusher.Start(context.Background())
	} else {
		log.Print("CRM sync: off (set CRM_INGEST_URL and CRM_INGEST_KEY to enable)")
	}

	// Serve the built frontend when it is there, so one port is the whole app.
	static := os.Getenv("PRISM_UI_STATIC")
	if static == "" {
		if _, err := os.Stat(filepath.Join("..", "frontend", "dist", "index.html")); err == nil {
			static = filepath.Join("..", "frontend", "dist")
		}
	}
	if static != "" {
		log.Printf("serving frontend from %s", static)
	}

	// Commission rules and payouts: a JSON file beside the service, seeded
	// the first time with the agents named in PRISM_COMMISSIONS_SEED
	// (comma-separated first names).
	commFile := os.Getenv("PRISM_COMMISSIONS_FILE")
	if commFile == "" {
		commFile = "commissions.json"
	}
	// A file that cannot be opened turns the commissions page off, not the
	// whole viewer.
	seed := commissions.DefaultAgents(store.Personnel(), strings.Split(os.Getenv("PRISM_COMMISSIONS_SEED"), ","))
	comm, err := commissions.Open(commFile, seed)
	if err != nil {
		log.Printf("commissions: OFF — %v (set PRISM_COMMISSIONS_FILE to a writable path)", err)
	} else {
		log.Printf("commission settings: %s", commFile)
	}

	srv := api.New(store, pusher, comm, static)
	log.Printf("prism_ui backend (read-only) listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv.Routes()))
}
