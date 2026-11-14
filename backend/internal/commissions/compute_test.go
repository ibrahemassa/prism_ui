package commissions

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func cfgWith(a Agent) Config { return Config{Agents: []Agent{a}, Payouts: map[string]Payout{}} }

func one(t *testing.T, cfg Config, deal Deal, now string) Line {
	t.Helper()
	lines := Compute(cfg, []Deal{deal}, d(now))
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(lines))
	}
	return lines[0]
}

func agent() Agent {
	a := NewAgent("8", "Ayşe Yılmaz")
	a.Rate = 1.5
	return a
}

// baseDeal: 10M sale, 3M planned peşinat paid over two receipts, contract on Sep 10.
func baseDeal() Deal {
	return Deal{
		ID: "100", AgentID: "8", Customer: "Test Müşteri", Unit: "B2-13", Stage: "sale", Currency: "TRY",
		SalePrice: 10_000_000, ListPrice: 11_000_000, DownPlan: 3_000_000,
		DealDate: d("2026-08-20"), Contracted: true, ContractAt: d("2026-09-10"),
		Receipts: []Receipt{{d("2026-08-20"), 500_000}, {d("2026-09-25"), 2_500_000}, {d("2026-10-15"), 200_000}},
	}
}

func TestEarnedWhenLaterConditionMet(t *testing.T) {
	l := one(t, cfgWith(agent()), baseDeal(), "2026-10-02")
	if l.DownPaidAt != "2026-09-25" {
		t.Errorf("down paid at = %q, want 2026-09-25", l.DownPaidAt)
	}
	if l.EligibleAt != "2026-09-25" { // peşinat finished after the contract
		t.Errorf("eligible = %q, want 2026-09-25", l.EligibleAt)
	}
	if l.BillMonth != "2026-10" {
		t.Errorf("bill month = %q, want 2026-10", l.BillMonth)
	}
	if l.Status != StatusDue {
		t.Errorf("status = %q, want due", l.Status)
	}
	if l.Amount != 150_000 {
		t.Errorf("amount = %v, want 150000", l.Amount)
	}
	if l.DownPaid != 3_000_000 {
		t.Errorf("down paid = %v, want capped at 3000000", l.DownPaid)
	}
}

func TestContractLaterThanDownPayment(t *testing.T) {
	deal := baseDeal()
	deal.ContractAt = d("2026-10-31")
	l := one(t, cfgWith(agent()), deal, "2026-10-02")
	if l.EligibleAt != "2026-10-31" || l.BillMonth != "2026-11" || l.Status != StatusScheduled {
		t.Errorf("got eligible %q bill %q status %q", l.EligibleAt, l.BillMonth, l.Status)
	}
}

func TestBillMonthEndOfMonthAndYearWrap(t *testing.T) {
	deal := baseDeal()
	deal.ContractAt = d("2026-01-31")
	deal.Receipts = []Receipt{{d("2026-01-31"), 3_000_000}}
	if l := one(t, cfgWith(agent()), deal, "2026-03-01"); l.BillMonth != "2026-02" {
		t.Errorf("Jan 31 + 1 month bills %q, want 2026-02", l.BillMonth)
	}
	deal.ContractAt = d("2026-12-05")
	deal.Receipts = []Receipt{{d("2026-12-01"), 3_000_000}}
	if l := one(t, cfgWith(agent()), deal, "2026-10-01"); l.BillMonth != "2027-01" {
		t.Errorf("Dec bills %q, want 2027-01", l.BillMonth)
	}
	a := agent()
	a.DelayMonths = 0
	if l := one(t, cfgWith(a), deal, "2026-10-01"); l.BillMonth != "2026-12" {
		t.Errorf("no delay bills %q, want 2026-12", l.BillMonth)
	}
}

func TestWaiting(t *testing.T) {
	deal := baseDeal()
	deal.Contracted, deal.ContractAt = false, time.Time{}
	deal.Receipts = deal.Receipts[:1]
	l := one(t, cfgWith(agent()), deal, "2026-10-02")
	if l.Status != StatusWaiting || l.BillMonth != "" || l.EligibleAt != "" {
		t.Errorf("got status %q bill %q eligible %q", l.Status, l.BillMonth, l.EligibleAt)
	}
	if !slices.Equal(l.Missing, []string{MissingContract, MissingDownPayment}) {
		t.Errorf("missing = %v", l.Missing)
	}
	if l.DownPaid != 500_000 {
		t.Errorf("down paid = %v, want 500000", l.DownPaid)
	}
}

func TestConditionsCanBeSwitchedOff(t *testing.T) {
	deal := baseDeal()
	deal.Contracted, deal.ContractAt = false, time.Time{}
	a := agent()
	a.NeedContract = false
	l := one(t, cfgWith(a), deal, "2026-10-02")
	if l.Status != StatusDue || l.EligibleAt != "2026-09-25" {
		t.Errorf("without contract rule: status %q eligible %q", l.Status, l.EligibleAt)
	}
	a.NeedDownPayment = false
	l = one(t, cfgWith(a), deal, "2026-10-02")
	if l.EligibleAt != "2026-08-20" || l.BillMonth != "2026-09" || l.Status != StatusOverdue {
		t.Errorf("no conditions: eligible %q bill %q status %q", l.EligibleAt, l.BillMonth, l.Status)
	}
}

func TestContractFlagWithoutDate(t *testing.T) {
	deal := baseDeal()
	deal.ContractAt = time.Time{} // flagged signed, no date
	l := one(t, cfgWith(agent()), deal, "2026-10-02")
	if l.Status != StatusDue || l.EligibleAt != "2026-09-25" {
		t.Errorf("status %q eligible %q", l.Status, l.EligibleAt)
	}
	if !slices.Contains(l.Notes, NoteContractNoDate) {
		t.Errorf("notes = %v, want contract_no_date", l.Notes)
	}
}

func TestNoDownPlanMeansFullPrice(t *testing.T) {
	deal := baseDeal()
	deal.DownPlan = 0
	l := one(t, cfgWith(agent()), deal, "2026-10-02")
	if l.DownNeed != 10_000_000 || l.Status != StatusWaiting || !slices.Contains(l.Notes, NoteNoDownPlan) {
		t.Errorf("need %v status %q notes %v", l.DownNeed, l.Status, l.Notes)
	}
}

func TestPrismTallyWithoutMatchingReceipts(t *testing.T) {
	deal := baseDeal()
	deal.Receipts = []Receipt{{d("2026-09-01"), 1_000}} // e.g. receipts in another currency
	deal.DownPrism = 3_000_000
	l := one(t, cfgWith(agent()), deal, "2026-10-02")
	if l.Status != StatusDue || l.DownPaidAt != "2026-09-01" || !slices.Contains(l.Notes, NoteNoReceiptDate) {
		t.Errorf("status %q paid at %q notes %v", l.Status, l.DownPaidAt, l.Notes)
	}
}

func TestToleranceAbsorbsRounding(t *testing.T) {
	deal := baseDeal()
	deal.Receipts = []Receipt{{d("2026-09-25"), 2_999_999.4}}
	if l := one(t, cfgWith(agent()), deal, "2026-10-02"); l.DownPaidAt != "2026-09-25" {
		t.Errorf("down paid at %q, want rounding tolerated", l.DownPaidAt)
	}
	deal.Receipts = []Receipt{{d("2026-09-25"), 2_900_000}}
	if l := one(t, cfgWith(agent()), deal, "2026-10-02"); l.Status != StatusWaiting {
		t.Errorf("100k short should still wait, got %q", l.Status)
	}
}

func TestListPriceBasis(t *testing.T) {
	a := agent()
	a.Basis = BasisListPrice
	if l := one(t, cfgWith(a), baseDeal(), "2026-10-02"); l.Base != 11_000_000 || l.Amount != 165_000 {
		t.Errorf("base %v amount %v", l.Base, l.Amount)
	}
	deal := baseDeal()
	deal.ListPrice = 0
	if l := one(t, cfgWith(a), deal, "2026-10-02"); l.Base != 10_000_000 || !slices.Contains(l.Notes, NoteNoListPrice) {
		t.Errorf("fallback base %v notes %v", l.Base, l.Notes)
	}
}

func TestPayoutsAndCancellations(t *testing.T) {
	cfg := cfgWith(agent())
	cfg.Payouts["100"] = Payout{AgentID: "8", PaidAt: "2026-10-05", Amount: 150_000, Currency: "TRY", Month: "2026-10"}
	l := one(t, cfg, baseDeal(), "2026-10-02")
	if l.Status != StatusPaid || l.Payout == nil || slices.Contains(l.Notes, NotePayoutDiffers) {
		t.Errorf("status %q payout %v notes %v", l.Status, l.Payout, l.Notes)
	}

	cfg.Agents[0].Rate = 2 // rate changed after paying
	if l := one(t, cfg, baseDeal(), "2026-10-02"); !slices.Contains(l.Notes, NotePayoutDiffers) {
		t.Errorf("notes %v, want payout_differs", l.Notes)
	}

	deal := baseDeal()
	deal.Stage = "cancelled"
	if l := one(t, cfg, deal, "2026-10-02"); l.Status != StatusClawback {
		t.Errorf("paid then cancelled = %q, want clawback", l.Status)
	}
	delete(cfg.Payouts, "100")
	if l := one(t, cfg, deal, "2026-10-02"); l.Status != StatusCancelled {
		t.Errorf("cancelled = %q", l.Status)
	}
}

func TestFilters(t *testing.T) {
	other := baseDeal()
	other.AgentID = "99" // not tracked
	neg := baseDeal()
	neg.ID, neg.Stage = "101", "negotiation"
	agency := baseDeal()
	agency.ID, agency.Agency = "102", "Acenta X"
	old := baseDeal()
	old.ID, old.DealDate = "103", d("2025-01-01")
	deals := []Deal{baseDeal(), other, neg, agency, old}

	if n := len(Compute(cfgWith(agent()), deals, d("2026-10-02"))); n != 3 {
		t.Errorf("default rule: %d lines, want 3", n)
	}
	a := agent()
	a.AgencyDeals = false
	a.Since = "2026-01-01"
	lines := Compute(cfgWith(a), deals, d("2026-10-02"))
	if len(lines) != 1 || lines[0].DealID != "100" {
		t.Errorf("filtered: %+v", lines)
	}
}

func TestSummarise(t *testing.T) {
	cfg := cfgWith(agent())
	paid := baseDeal()
	paid.ID = "200"
	cfg.Payouts["200"] = Payout{AgentID: "8", PaidAt: "2026-10-05", Amount: 140_000, Currency: "TRY", Month: "2026-10"}
	waiting := baseDeal()
	waiting.ID, waiting.Receipts = "201", nil
	nextMonth := baseDeal()
	nextMonth.ID, nextMonth.ContractAt = "202", d("2026-10-20")
	usd := baseDeal()
	usd.ID, usd.Currency, usd.SalePrice = "203", "USD", 200_000
	lines := Compute(cfg, []Deal{baseDeal(), paid, waiting, nextMonth, usd}, d("2026-10-02"))
	s := Summarise(cfg, lines, "2026-10")[0]
	if s.Deals != 3 || s.Waiting != 1 {
		t.Errorf("deals %d waiting %d", s.Deals, s.Waiting)
	}
	if s.Amount["TRY"] != 290_000 || s.Paid["TRY"] != 140_000 || s.Unpaid["TRY"] != 150_000 || s.Unpaid["USD"] != 3_000 {
		t.Errorf("amount %v paid %v unpaid %v", s.Amount, s.Paid, s.Unpaid)
	}
}

func TestDefaultAgentsMatchFirstNames(t *testing.T) {
	got := DefaultAgents(map[string]string{
		"8": "Ayşe Yılmaz", "17": "Zeynep Demir", "19": "Mehmet Kaya", "10": "Elif Şahin", "6": "Can Öztürk",
	}, []string{"Ayse", "zeynep", " mehmet ", "elif", "", "nobody"})
	ids := []string{}
	for _, a := range got {
		ids = append(ids, a.PersonnelID)
		if !a.NeedContract || !a.NeedDownPayment || a.DelayMonths != 1 || a.Basis != BasisSalePrice {
			t.Errorf("%s not on the standard rule: %+v", a.Name, a)
		}
	}
	if !slices.Equal(ids, []string{"8", "17", "19", "10"}) {
		t.Errorf("seeded %v", ids)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "commissions.json")
	s, err := Open(path, []Agent{agent()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("seed not written: %v", err)
	}
	bad := agent()
	bad.Rate = 150
	if _, err := s.SetAgents([]Agent{bad}); err == nil {
		t.Error("rate 150 accepted")
	}
	if _, err := s.SetAgents([]Agent{agent(), agent()}); err == nil {
		t.Error("duplicate agent accepted")
	}
	b := agent()
	b.Rate = 2.25
	if _, err := s.SetAgents([]Agent{b}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetPayouts(map[string]*Payout{"100": {AgentID: "8", PaidAt: "2026-10-01", Amount: 1, Currency: "TRY"}}); err != nil {
		t.Fatal(err)
	}

	again, err := Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := again.Get()
	if len(c.Agents) != 1 || c.Agents[0].Rate != 2.25 || c.Payouts["100"].PaidAt != "2026-10-01" {
		t.Errorf("reloaded %+v", c)
	}
	if _, err := again.SetPayouts(map[string]*Payout{"100": nil}); err != nil {
		t.Fatal(err)
	}
	if len(again.Get().Payouts) != 0 {
		t.Error("payout not cleared")
	}
	// Get hands out a copy
	c = again.Get()
	c.Agents[0].Rate = 99
	if again.Get().Agents[0].Rate != 2.25 {
		t.Error("Get leaked internal state")
	}
}
