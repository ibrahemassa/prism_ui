package commissions

import (
	"math"
	"sort"
	"time"
)

// Deal is what Prism knows about one deal, as far as commission is concerned.
type Deal struct {
	ID         string
	AgentID    string // a.SatisPersonelID1
	Customer   string
	Unit       string // "B2-13"
	UnitType   string
	Agency     string
	Stage      string // sale | reserved | negotiation | cancelled
	Currency   string
	SalePrice  float64
	ListPrice  float64
	DownPlan   float64 // a.PlanPesinatToplam
	DownPrism  float64 // T_AktiviteBakiye.TahsilatPesinat — Prism's own peşinat tally
	DealDate   time.Time
	Contracted bool // a.SozlesmeYapildiMi
	ContractAt time.Time
	Receipts   []Receipt
}

type Receipt struct {
	Date   time.Time
	Amount float64
}

// Line is one deal's commission, worked out under its agent's rule.
type Line struct {
	DealID     string   `json:"deal_id"`
	AgentID    string   `json:"agent_id"`
	Agent      string   `json:"agent"`
	Customer   string   `json:"customer"`
	Unit       string   `json:"unit"`
	UnitType   string   `json:"unit_type"`
	Agency     string   `json:"agency"`
	Stage      string   `json:"stage"`
	Currency   string   `json:"currency"`
	DealDate   string   `json:"deal_date"`
	SalePrice  float64  `json:"sale_price"`
	ListPrice  float64  `json:"list_price"`
	Base       float64  `json:"base"`
	Rate       float64  `json:"rate"`
	Amount     float64  `json:"amount"`
	Contracted bool     `json:"contracted"`
	ContractAt string   `json:"contract_at"`
	DownNeed   float64  `json:"down_required"`
	DownPaid   float64  `json:"down_paid"`    // collected toward it, capped at DownNeed
	DownPaidAt string   `json:"down_paid_at"` // the day the last of it came in
	EligibleAt string   `json:"eligible_at"`  // both conditions met
	BillMonth  string   `json:"bill_month"`   // YYYY-MM it is billed in
	Status     string   `json:"status"`
	Missing    []string `json:"missing"` // conditions not yet met
	Notes      []string `json:"notes"`   // assumptions the figure rests on
	Payout     *Payout  `json:"payout,omitempty"`
}

const (
	StatusWaiting   = "waiting"   // a condition is not met yet
	StatusScheduled = "scheduled" // earned, billed in a later month
	StatusDue       = "due"       // billed this month
	StatusOverdue   = "overdue"   // billing month has passed, not marked paid
	StatusPaid      = "paid"
	StatusCancelled = "cancelled" // the deal was called off before anything was paid
	StatusClawback  = "clawback"  // paid, then the deal was cancelled
)

const (
	MissingContract    = "contract"
	MissingDownPayment = "down_payment"
	MissingDate        = "date" // conditions met but no date to bill from

	NoteNoDownPlan     = "no_down_plan"     // no planned peşinat: the full price counts as the down payment
	NoteContractNoDate = "contract_no_date" // flagged signed in Prism without a contract date
	NoteNoListPrice    = "no_list_price"    // list-price basis but no list price; sale price used
	NoteNoReceiptDate  = "no_receipt_date"  // Prism says peşinat collected but the receipts do not add up
	NotePayoutDiffers  = "payout_differs"   // what was paid differs from today's figure
)

// Compute applies each tracked agent's rule to their deals. now decides which
// billing month is "this month".
func Compute(cfg Config, deals []Deal, now time.Time) []Line {
	agents := map[string]Agent{}
	for _, a := range cfg.Agents {
		agents[a.PersonnelID] = a
	}
	thisMonth := now.Format("2006-01")
	out := []Line{}
	for _, d := range deals {
		a, ok := agents[d.AgentID]
		if !ok || d.Stage == "negotiation" {
			continue
		}
		if !a.AgencyDeals && d.Agency != "" {
			continue
		}
		if a.Since != "" && !d.DealDate.IsZero() && d.DealDate.Format(time.DateOnly) < a.Since {
			continue
		}
		out = append(out, line(a, d, cfg.Payouts, thisMonth))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].BillMonth != out[j].BillMonth {
			return out[i].BillMonth > out[j].BillMonth
		}
		return out[i].DealDate > out[j].DealDate
	})
	return out
}

func line(a Agent, d Deal, payouts map[string]Payout, thisMonth string) Line {
	l := Line{
		DealID: d.ID, AgentID: a.PersonnelID, Agent: a.Name, Customer: d.Customer, Unit: d.Unit,
		UnitType: d.UnitType, Agency: d.Agency, Stage: d.Stage, Currency: d.Currency,
		DealDate: day(d.DealDate), SalePrice: d.SalePrice, ListPrice: d.ListPrice, Rate: a.Rate,
		Contracted: d.Contracted || !d.ContractAt.IsZero(), ContractAt: day(d.ContractAt),
		Missing: []string{}, Notes: []string{},
	}

	l.Base = d.SalePrice
	if a.Basis == BasisListPrice {
		if d.ListPrice > 0 {
			l.Base = d.ListPrice
		} else {
			l.Notes = append(l.Notes, NoteNoListPrice)
		}
	}
	l.Amount = round2(l.Base * a.Rate / 100)

	// Peşinat: a deal with no planned down payment is a cash sale, so the
	// whole price is what has to come in first.
	l.DownNeed = d.DownPlan
	if l.DownNeed <= 0 {
		l.DownNeed = d.SalePrice
		l.Notes = append(l.Notes, NoteNoDownPlan)
	}
	paidAt, collected := downPaidAt(d.Receipts, l.DownNeed)
	if paidAt.IsZero() && d.DownPrism >= l.DownNeed-tolerance(l.DownNeed) && l.DownNeed > 0 {
		// Prism's tally says it is in, but the receipts do not reach it (a
		// receipt in another currency, say): take the last receipt's day.
		collected = math.Max(collected, d.DownPrism)
		if n := len(d.Receipts); n > 0 {
			paidAt = d.Receipts[n-1].Date
		}
		l.Notes = append(l.Notes, NoteNoReceiptDate)
	}
	l.DownPaid = math.Min(math.Max(collected, d.DownPrism), l.DownNeed)
	l.DownPaidAt = day(paidAt)
	downDone := !paidAt.IsZero() || (l.DownNeed > 0 && l.DownPaid >= l.DownNeed-tolerance(l.DownNeed))

	if d.Contracted && d.ContractAt.IsZero() {
		l.Notes = append(l.Notes, NoteContractNoDate)
	}

	// Earned the day the later of the required conditions was met.
	var earned time.Time
	if a.NeedContract {
		if !l.Contracted {
			l.Missing = append(l.Missing, MissingContract)
		}
		earned = later(earned, d.ContractAt)
	}
	if a.NeedDownPayment {
		if !downDone {
			l.Missing = append(l.Missing, MissingDownPayment)
		}
		earned = later(earned, paidAt)
	}
	if !a.NeedContract && !a.NeedDownPayment {
		earned = d.DealDate
	}
	if len(l.Missing) == 0 && earned.IsZero() {
		l.Missing = append(l.Missing, MissingDate)
	}
	if len(l.Missing) == 0 {
		l.EligibleAt = day(earned)
		l.BillMonth = earned.AddDate(0, a.DelayMonths, 1-earned.Day()).Format("2006-01")
	}

	p, paid := payouts[d.ID]
	if paid {
		l.Payout = &p
		if p.Amount != l.Amount || p.Currency != l.Currency {
			l.Notes = append(l.Notes, NotePayoutDiffers)
		}
	}
	switch {
	case d.Stage == "cancelled" && paid:
		l.Status = StatusClawback
	case d.Stage == "cancelled":
		l.Status = StatusCancelled
	case paid:
		l.Status = StatusPaid
	case len(l.Missing) > 0:
		l.Status = StatusWaiting
	case l.BillMonth > thisMonth:
		l.Status = StatusScheduled
	case l.BillMonth == thisMonth:
		l.Status = StatusDue
	default:
		l.Status = StatusOverdue
	}
	return l
}

// downPaidAt walks the receipts in date order and returns the day the running
// total first covered need, plus what had come in by the end.
func downPaidAt(receipts []Receipt, need float64) (time.Time, float64) {
	sum := 0.0
	var at time.Time
	for _, r := range receipts {
		sum += r.Amount
		if at.IsZero() && need > 0 && sum >= need-tolerance(need) {
			at = r.Date
		}
	}
	return at, sum
}

// tolerance absorbs kuruş rounding and small FX differences on the peşinat.
func tolerance(need float64) float64 { return math.Max(1, need*0.001) }

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func day(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.DateOnly)
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// Summary is one agent's figures for one billing month.
type Summary struct {
	AgentID  string             `json:"agent_id"`
	Agent    string             `json:"agent"`
	Rate     float64            `json:"rate"`
	Basis    string             `json:"basis"`
	Rule     string             `json:"rule"`
	Deals    int                `json:"deals"`    // deals billed in the month, paid or not
	Base     map[string]float64 `json:"base"`     // by currency
	Amount   map[string]float64 `json:"amount"`   // by currency
	Paid     map[string]float64 `json:"paid"`     // by currency
	Unpaid   map[string]float64 `json:"unpaid"`   // by currency
	Waiting  int                `json:"waiting"`  // deals still missing a condition
	Clawback map[string]float64 `json:"clawback"` // paid on deals since cancelled, any month
}

// AllTime in place of a YYYY-MM month means every billing month at once.
const AllTime = "all"

// Summarise totals each agent's lines billed in month (or in any month, for
// AllTime). Cancelled deals do not count; clawbacks are reported separately.
func Summarise(cfg Config, lines []Line, month string) []Summary {
	out := make([]Summary, 0, len(cfg.Agents))
	idx := map[string]int{}
	for _, a := range cfg.Agents {
		idx[a.PersonnelID] = len(out)
		out = append(out, Summary{
			AgentID: a.PersonnelID, Agent: a.Name, Rate: a.Rate, Basis: a.Basis, Rule: a.Rule,
			Base: map[string]float64{}, Amount: map[string]float64{}, Paid: map[string]float64{},
			Unpaid: map[string]float64{}, Clawback: map[string]float64{},
		})
	}
	for _, l := range lines {
		i, ok := idx[l.AgentID]
		if !ok {
			continue
		}
		s := &out[i]
		switch {
		case l.Status == StatusWaiting:
			s.Waiting++
		case l.Status == StatusClawback:
			s.Clawback[l.Payout.Currency] += l.Payout.Amount
		case l.Status == StatusCancelled || (month != AllTime && l.BillMonth != month):
			// not this month's bill
		default:
			s.Deals++
			s.Base[l.Currency] += l.Base
			if l.Status == StatusPaid {
				s.Amount[l.Payout.Currency] += l.Payout.Amount
				s.Paid[l.Payout.Currency] += l.Payout.Amount
			} else {
				s.Amount[l.Currency] += l.Amount
				s.Unpaid[l.Currency] += l.Amount
			}
		}
	}
	return out
}
