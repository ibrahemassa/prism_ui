package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"prismui/internal/commissions"
	"prismui/internal/xlsx"
)

func TestCommissionSheets(t *testing.T) {
	a := commissions.NewAgent("8", "Ayşe Yılmaz")
	a.Rate = 1.5
	cfg := commissions.Config{Agents: []commissions.Agent{a}, Payouts: map[string]commissions.Payout{}}
	day := func(s string) time.Time { t, _ := time.Parse(time.DateOnly, s); return t }
	due := commissions.Deal{ID: "1", AgentID: "8", Customer: "Ali <Veli>", Unit: "B2-13", Stage: "sale", Currency: "TRY",
		SalePrice: 10_000_000, DownPlan: 3_000_000, DealDate: day("2026-08-01"), Contracted: true, ContractAt: day("2026-09-10"),
		Receipts: []commissions.Receipt{{Date: day("2026-09-20"), Amount: 3_000_000}}}
	wait := due
	wait.ID, wait.Receipts = "2", nil
	overdue := due
	overdue.ID, overdue.ContractAt = "3", day("2026-07-01")
	overdue.Receipts = []commissions.Receipt{{Date: day("2026-07-01"), Amount: 3_000_000}}
	lines := commissions.Compute(cfg, []commissions.Deal{due, wait, overdue}, day("2026-10-02"))

	sheets := commissionSheets(cfg, lines, "2026-10", day("2026-10-02"))
	names := []string{}
	for _, s := range sheets {
		names = append(names, s.Name)
	}
	if got := strings.Join(names, ","); got != "Özet,Ekim 2026,Bekleyenler,Gecikmiş" {
		t.Errorf("sheets = %s", got)
	}
	if n := len(sheets[1].Rows); n != 3 { // header, the due line, total
		t.Errorf("month sheet has %d rows", n)
	}
	if n := len(sheets[2].Rows); n != 2 {
		t.Errorf("waiting sheet has %d rows", n)
	}
	var buf bytes.Buffer
	if err := xlsx.Write(&buf, sheets); err != nil {
		t.Fatal(err)
	}
}

func TestCommissionSheetsAllTime(t *testing.T) {
	a := commissions.NewAgent("8", "Ayşe Yılmaz")
	a.Rate = 1.5
	cfg := commissions.Config{Agents: []commissions.Agent{a}, Payouts: map[string]commissions.Payout{
		"4": {AgentID: "8", PaidAt: "2026-10-01", Amount: 150_000, Currency: "TRY", Month: "2026-10"},
	}}
	day := func(s string) time.Time { t, _ := time.Parse(time.DateOnly, s); return t }
	sep := commissions.Deal{ID: "1", AgentID: "8", Customer: "Eylül", Stage: "sale", Currency: "TRY",
		SalePrice: 10_000_000, DownPlan: 3_000_000, Contracted: true, ContractAt: day("2026-08-10"),
		Receipts: []commissions.Receipt{{Date: day("2026-08-20"), Amount: 3_000_000}}}
	oct := sep
	oct.ID, oct.ContractAt = "2", day("2026-09-10")
	oct.Receipts = []commissions.Receipt{{Date: day("2026-09-20"), Amount: 3_000_000}}
	paidThenCancelled := oct
	paidThenCancelled.ID, paidThenCancelled.Stage = "4", "cancelled"
	lines := commissions.Compute(cfg, []commissions.Deal{sep, oct, paidThenCancelled}, day("2026-10-02"))

	sheets := commissionSheets(cfg, lines, commissions.AllTime, day("2026-10-02"))
	names := []string{}
	for _, s := range sheets {
		names = append(names, s.Name)
	}
	if got := strings.Join(names, ","); got != "Özet,Tüm zamanlar,Bekleyenler,Aylara göre,Gecikmiş,Geri alınacak" {
		t.Errorf("sheets = %s", got)
	}
	// both months' lines, not the clawback, plus the total row
	if n := len(sheets[1].Rows); n != 4 {
		t.Errorf("all-time sheet has %d rows, want 4", n)
	}
	byMonth := sheets[3].Rows
	if len(byMonth) != 3 || byMonth[1][0].V != "Ekim 2026" || byMonth[2][0].V != "Eylül 2026" {
		t.Errorf("by-month rows %v", byMonth)
	}
	if s := commissions.Summarise(cfg, lines, commissions.AllTime)[0]; s.Deals != 2 || s.Amount["TRY"] != 300_000 || s.Clawback["TRY"] != 150_000 {
		t.Errorf("all-time summary %+v", s)
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{"Ayşe Yılmaz": "ayse-yilmaz", "  Mehmet  Kaya ": "mehmet-kaya", "İÇÖ": "ico"} {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReadOnlyLetsOnlyKnownPostsThrough(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := readOnly(ok)
	for _, c := range []struct {
		method, path string
		want         int
	}{
		{"GET", "/api/commissions", 200},
		{"POST", "/api/commissions/agents", 200},
		{"POST", "/api/commissions/payouts", 200},
		{"POST", "/api/sync/run", 200},
		{"POST", "/api/leads", 405},
		{"PUT", "/api/commissions/agents", 405},
		{"DELETE", "/api/commissions/payouts", 405},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
		if rec.Code != c.want {
			t.Errorf("%s %s = %d, want %d", c.method, c.path, rec.Code, c.want)
		}
	}
}

func TestCommissionsOffWithoutSettingsFile(t *testing.T) {
	s := &Server{}
	rec := httptest.NewRecorder()
	s.commissionsSetPaid(rec, httptest.NewRequest("POST", "/api/commissions/payouts", strings.NewReader(`{"deal_ids":["1"],"paid":true}`)))
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "komisyon ayarları") {
		t.Errorf("got %d %s", rec.Code, rec.Body.String())
	}
}
