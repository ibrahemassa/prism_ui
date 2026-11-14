# Prism UI — read-only sales viewer for Prism

A modern front end over an existing **Prism** (PRISM_PLUS) real-estate sales
database. It only **reads** Prism — there is no write path anywhere, so it
cannot change Prism data.

- **backend/** — Go API (`net/http` + `microsoft/go-mssqldb`). Every query is a
  `SELECT`; there is no `Exec`. Decodes Prism's lookup codes (status, source,
  personnel, office, currency, category) into readable text server-side.
- **frontend/** — React + TypeScript + Vite + Tailwind. Turkish UI: Panel,
  Müşteriler, Daireler, Satışlar, Personel, Acentalar, Raporlar, Komisyonlar.

## Why it's safe
- No `INSERT/UPDATE/DELETE/Exec` against Prism — only `db.Query`.
- The only writes are a local commission-settings JSON file and an optional
  outbound push to a CRM (both below).
- Credentials are never committed; they're read at runtime from the Prism
  install/dump folder (`Model.dll.config` + `DBList.txt`) or from env vars.

## Run

**1. Backend** (needs Go 1.26+):
```bash
cd backend
cp .env.example .env        # then point it at your Prism DB
./run.sh                    # serves http://localhost:8080
```
Point it at the DB one of three ways (see `.env.example`): `PRISM_CONFIG_DIR`
(reads the Prism creds), a full `PRISM_DSN`, or `PRISM_DB_*` vars.

**2. Frontend** (needs Node 20+):
```bash
cd frontend
npm install
npm run dev                 # http://localhost:5180  (proxies /api -> :8080)
```
Open **http://localhost:5180**.

**One port for a whole LAN.** `npm run build` in `frontend/` and the Go
backend serves `frontend/dist/` on `/` automatically (or set
`PRISM_UI_STATIC`). Everyone opens `http://<this-box>:8080`. To keep it up
across reboots use `deploy/prism-ui.service` (instructions in the file).

## Branding

Project name, company name, logos and favicon are local settings, read at
build time. Copy `frontend/.env.example` to `frontend/.env.local` and put the
image files in `frontend/public/brand/` — both are gitignored. With nothing
set the UI shows generic names and no logos.

## Push to a CRM (optional)

When Prism's database is only reachable from the office network, a CRM cannot
pull from it. This service pushes instead: every `SYNC_INTERVAL` (default 10m)
it reads a full snapshot — units with their live sale, cancellations, every
customer, agents, agencies — and POSTs it gzip'd to `CRM_INGEST_URL` with a
bearer key. The receiving CRM does the matching.

Set `CRM_INGEST_URL` + `CRM_INGEST_KEY` (see `backend/.env.example`);
`CRM_INGEST_DRY_RUN=1` asks the CRM to answer with the diff without writing.
The top bar shows when the last push landed and has a "send now" button;
`GET /api/sync/status`, `GET /api/sync/preview` (the payload) and
`POST /api/sync/run` are the same thing over HTTP.

`PRISM_CRM_DSN` (optional, read-only Postgres) shows on a lead's page whether
the person also exists in the CRM, with its source and stage history.

## Commissions (Komisyonlar)

The **Komisyonlar** page tracks what each sales agent is owed. The rule: once a
deal's peşinat is fully collected **and** its contract is signed, the agent's
commission is billed the month after. A deal belongs to its *Satış Personeli 1*.

- Agents, their rate, basis (sale or list price) and rule note are edited on the
  page and saved to `backend/commissions.json` (or `PRISM_COMMISSIONS_FILE`).
  On first start the file is seeded with the agents named in
  `PRISM_COMMISSIONS_SEED`. "Paid" marks live in the same file. **Nothing here
  writes to Prism.** Back the file up; it is gitignored.
- Peşinat is "paid" the day the deal's receipts add up to the planned peşinat
  (`PlanPesinatToplam`; a deal with none is treated as cash — the whole price).
  The contract counts as signed from `SozlesmeTarihi`, or the
  `SozlesmeYapildiMi` flag when the date was left empty.
- Excel: `GET /api/commissions/export.xlsx?month=YYYY-MM[&agent=ID]`.
  PDF: the page's PDF button opens `/komisyonlar/pdf` and prints.
  "Tüm zamanlar" on the page (`month=all`, `?ay=all`) covers every billing
  month and adds a month-by-month breakdown to both.

## Dates

A sale's date is Prism's **RezerveTarihi**, falling back to `AktiviteTarihi`
only where the reservation date is unset. `AktiviteTarihi` is re-stamped
whenever something happens to a deal, so it reads as "last touched".

## Endpoints
All read-only against Prism. `POST /api/sync/run` sends Prism data *out*;
`POST /api/commissions/agents` and `/api/commissions/payouts` save the local
commission settings file.

`/api/stats` · `/api/leads?q=&limit=&offset=` · `/api/units` · `/api/deals` ·
`/api/agents` · `/api/agencies` · `/api/reports` · `/api/reports/full` ·
`/api/commissions` · `/api/commissions/export.xlsx` ·
`/api/sync/status` · `/api/sync/preview` · `/api/health`

## Notes
- `run.sh` points `GOCACHE`/`GOMODCACHE`/`GOPATH` at `$HOME` in case the default
  Go cache location isn't writable.
- Point it at a production Prism database only with a read-only login if you
  can; the code never writes, but least privilege is cheap.
