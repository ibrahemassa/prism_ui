package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"prismui/internal/commissions"
	"prismui/internal/xlsx"
)

// Commissions: the tracked agents, their rules, and every deal's commission.
// The two POSTs write the local settings file (commissions.json) — never Prism.

type commissionsPayload struct {
	Month  string               `json:"month"` // this month, YYYY-MM
	Agents []commissions.Agent  `json:"agents"`
	Lines  []commissions.Line   `json:"lines"`
	Staff  []commissionsStaffer `json:"staff"` // Prism personnel, for the add-agent picker
}

type commissionsStaffer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var errNoCommissions = errors.New("komisyon ayarları dosyası açılamadı — sunucu günlüğüne bakın (PRISM_COMMISSIONS_FILE)")

func (s *Server) commissionLines() (commissions.Config, []commissions.Line, error) {
	if s.comm == nil {
		return commissions.Config{}, nil, errNoCommissions
	}
	cfg := s.comm.Get()
	deals, err := s.store.CommissionDeals(cfg.AgentIDs())
	if err != nil {
		return cfg, nil, err
	}
	return cfg, commissions.Compute(cfg, deals, time.Now()), nil
}

func (s *Server) commissionsGet(w http.ResponseWriter, r *http.Request) {
	cfg, lines, err := s.commissionLines()
	if err != nil {
		fail(w, err)
		return
	}
	staff := []commissionsStaffer{}
	for id, name := range s.store.Personnel() {
		staff = append(staff, commissionsStaffer{id, strings.TrimSpace(name)})
	}
	writeJSON(w, commissionsPayload{Month: time.Now().Format("2006-01"), Agents: cfg.Agents, Lines: lines, Staff: staff})
}

func (s *Server) commissionsSetAgents(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Agents []commissions.Agent `json:"agents"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	// Prism's spelling of the name wins over whatever the page sent.
	per := s.store.Personnel()
	for i := range body.Agents {
		if n := strings.TrimSpace(per[strings.TrimSpace(body.Agents[i].PersonnelID)]); n != "" {
			body.Agents[i].Name = n
		}
	}
	if body.Agents == nil {
		body.Agents = []commissions.Agent{}
	}
	if s.comm == nil {
		fail(w, errNoCommissions)
		return
	}
	if _, err := s.comm.SetAgents(body.Agents); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.commissionsGet(w, r)
}

// commissionsSetPaid marks deals' commissions paid (recording today's figure)
// or takes the mark off again.
func (s *Server) commissionsSetPaid(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DealIDs []string `json:"deal_ids"`
		Paid    bool     `json:"paid"`
		PaidAt  string   `json:"paid_at"` // YYYY-MM-DD, default today
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.PaidAt == "" {
		body.PaidAt = time.Now().Format(time.DateOnly)
	} else if _, err := time.Parse(time.DateOnly, body.PaidAt); err != nil {
		http.Error(w, "ödeme tarihi YYYY-AA-GG olmalı", http.StatusBadRequest)
		return
	}
	_, lines, err := s.commissionLines()
	if err != nil {
		fail(w, err)
		return
	}
	byDeal := map[string]commissions.Line{}
	for _, l := range lines {
		byDeal[l.DealID] = l
	}
	set := map[string]*commissions.Payout{}
	for _, id := range body.DealIDs {
		if !body.Paid {
			set[id] = nil
			continue
		}
		l, ok := byDeal[id]
		switch {
		case !ok:
			http.Error(w, fmt.Sprintf("satış %s takip edilen bir danışmana ait değil", id), http.StatusBadRequest)
			return
		case l.Status == commissions.StatusCancelled:
			http.Error(w, fmt.Sprintf("%s (%s) iptal edilmiş — komisyon ödenmez", l.Customer, l.Unit), http.StatusBadRequest)
			return
		case l.Status == commissions.StatusWaiting:
			http.Error(w, fmt.Sprintf("%s (%s): komisyon şartları henüz sağlanmadı", l.Customer, l.Unit), http.StatusBadRequest)
			return
		case l.Status == commissions.StatusPaid || l.Status == commissions.StatusClawback:
			continue // keep the original record
		}
		set[id] = &commissions.Payout{AgentID: l.AgentID, PaidAt: body.PaidAt, Amount: l.Amount, Currency: l.Currency, Month: l.BillMonth}
	}
	if _, err := s.comm.SetPayouts(set); err != nil {
		fail(w, err)
		return
	}
	s.commissionsGet(w, r)
}

var monthRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

var trMonths = [...]string{"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran", "Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"}

func monthLabel(m string) string {
	if m == commissions.AllTime {
		return "Tüm zamanlar"
	}
	t, err := time.Parse("2006-01", m)
	if err != nil {
		return m
	}
	return fmt.Sprintf("%s %d", trMonths[t.Month()-1], t.Year())
}

var statusTR = map[string]string{
	commissions.StatusWaiting: "Bekliyor", commissions.StatusScheduled: "Planlandı", commissions.StatusDue: "Faturalanacak",
	commissions.StatusOverdue: "Gecikmiş", commissions.StatusPaid: "Ödendi", commissions.StatusCancelled: "İptal",
	commissions.StatusClawback: "Geri alınacak",
}

var stageTR = map[string]string{"sale": "Satıldı", "reserved": "Rezerve", "cancelled": "İptal", "negotiation": "Görüşme"}

var basisTR = map[string]string{commissions.BasisSalePrice: "Satış fiyatı", commissions.BasisListPrice: "Liste fiyatı"}

var missingTR = map[string]string{
	commissions.MissingContract: "Sözleşme imzalanmadı", commissions.MissingDownPayment: "Peşinat tamamlanmadı",
	commissions.MissingDate: "Tarih yok",
}

var noteTR = map[string]string{
	commissions.NoteNoDownPlan:     "Peşinat planı yok — tüm fiyat peşinat sayıldı",
	commissions.NoteContractNoDate: "Sözleşme işaretli ama tarihi yok",
	commissions.NoteNoListPrice:    "Liste fiyatı yok — satış fiyatı kullanıldı",
	commissions.NoteNoReceiptDate:  "Peşinat Prism'de tamam, tahsilat tarihleri tutmuyor",
	commissions.NotePayoutDiffers:  "Ödenen tutar bugünkü hesaptan farklı",
}

func tr(m map[string]string, codes []string) string {
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		if t, ok := m[c]; ok {
			out = append(out, t)
		} else {
			out = append(out, c)
		}
	}
	return strings.Join(out, "; ")
}

// commissionsExport is the month's commission report as an Excel workbook:
// a summary per agent, the month's lines, and what is still waiting.
// month=all reports every billing month, with a month-by-month sheet.
func (s *Server) commissionsExport(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	if month != commissions.AllTime && !monthRe.MatchString(month) {
		http.Error(w, "month YYYY-MM veya all olmalı", http.StatusBadRequest)
		return
	}
	cfg, lines, err := s.commissionLines()
	if err != nil {
		fail(w, err)
		return
	}
	file := "komisyon-" + month
	if month == commissions.AllTime {
		file = "komisyon-tum-zamanlar"
	}
	if agent := r.URL.Query().Get("agent"); agent != "" {
		kept := cfg.Agents[:0:0]
		for _, a := range cfg.Agents {
			if a.PersonnelID == agent {
				kept = append(kept, a)
				file += "-" + slug(a.Name)
			}
		}
		cfg.Agents = kept
		ls := lines[:0:0]
		for _, l := range lines {
			if l.AgentID == agent {
				ls = append(ls, l)
			}
		}
		lines = ls
	}
	sheets := commissionSheets(cfg, lines, month, time.Now())
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"; filename*=UTF-8''%s.xlsx`, file, url.PathEscape(file)))
	w.Header().Set("Cache-Control", "no-store")
	if err := xlsx.Write(w, sheets); err != nil {
		// headers are gone by now; the download will simply be broken
		return
	}
}

func commissionSheets(cfg commissions.Config, lines []commissions.Line, month string, now time.Time) []xlsx.Sheet {
	S, N := xlsx.S, xlsx.N
	H := func(labels ...string) []xlsx.Cell {
		row := make([]xlsx.Cell, len(labels))
		for i, l := range labels {
			row[i] = xlsx.Cell{V: l, Style: xlsx.Header}
		}
		return row
	}
	stamp := "Oluşturma: " + now.Format("02.01.2006 15:04") + " · Prism canlı verisi"

	// ---- Özet: one row per agent and currency
	summary := xlsx.Sheet{
		Name:   "Özet",
		Widths: []float64{24, 8, 14, 9, 18, 16, 16, 16, 10, 12, 16, 60},
		Rows: [][]xlsx.Cell{
			{{V: "Komisyon Raporu — " + monthLabel(month), Style: xlsx.Title}},
			{S("Kural: peşinat tamamen ödendi + sözleşme imzalandı → komisyon ertesi ay faturalanır (danışmana göre değişebilir).")},
			{S(stamp)},
			{},
			H("Danışman", "Oran", "Baz", "Döviz", "Baz toplamı", "Komisyon", "Ödenen", "Ödenecek", "Satış", "Bekleyen", "Geri alınacak", "Danışman kuralı"),
		},
		Freeze: 5,
	}
	totals := map[string]*[4]float64{} // currency -> base, amount, paid, unpaid
	for _, sm := range commissions.Summarise(cfg, lines, month) {
		curs := currencies(sm.Amount, sm.Base, sm.Clawback)
		for i, cur := range curs {
			row := []xlsx.Cell{S(sm.Agent), N(sm.Rate, xlsx.Percent), S(basisTR[sm.Basis]), S(cur),
				N(sm.Base[cur], xlsx.Money), N(sm.Amount[cur], xlsx.Money), N(sm.Paid[cur], xlsx.Money), N(sm.Unpaid[cur], xlsx.Money)}
			if i == 0 {
				row = append(row, N(float64(sm.Deals), xlsx.Plain), N(float64(sm.Waiting), xlsx.Plain))
			} else {
				row = append(row, xlsx.Cell{}, xlsx.Cell{})
			}
			row = append(row, N(sm.Clawback[cur], xlsx.Money))
			if i == 0 {
				row = append(row, S(sm.Rule))
			}
			summary.Rows = append(summary.Rows, row)
			t := totals[cur]
			if t == nil {
				t = &[4]float64{}
				totals[cur] = t
			}
			t[0] += sm.Base[cur]
			t[1] += sm.Amount[cur]
			t[2] += sm.Paid[cur]
			t[3] += sm.Unpaid[cur]
		}
	}
	for _, cur := range sortedKeys(totals) {
		t := totals[cur]
		summary.Rows = append(summary.Rows, []xlsx.Cell{{V: "Toplam", Style: xlsx.Header}, {Style: xlsx.Header}, {Style: xlsx.Header},
			{V: cur, Style: xlsx.Header}, N(t[0], xlsx.Total), N(t[1], xlsx.Total), N(t[2], xlsx.Total), N(t[3], xlsx.Total)})
	}

	lineHead := H("Danışman", "Müşteri", "Daire", "Tip", "Acenta", "Satış tarihi", "Satış durumu", "Satış fiyatı", "Komisyon bazı",
		"Oran", "Komisyon", "Döviz", "Gereken peşinat", "Peşinat tamamlandı", "Sözleşme tarihi", "Hak ediş", "Fatura ayı", "Durum", "Ödeme tarihi", "Not")
	lineWidths := []float64{20, 26, 9, 10, 18, 12, 11, 16, 16, 8, 14, 7, 16, 14, 13, 12, 13, 13, 12, 40}
	lineRow := func(l commissions.Line) []xlsx.Cell {
		amount, paidAt := l.Amount, ""
		if l.Payout != nil {
			amount, paidAt = l.Payout.Amount, l.Payout.PaidAt
		}
		contract := l.ContractAt
		if contract == "" && l.Contracted {
			contract = "imzalı (tarihsiz)"
		}
		return []xlsx.Cell{S(l.Agent), S(l.Customer), S(l.Unit), S(l.UnitType), S(l.Agency), S(l.DealDate), S(stageTR[l.Stage]),
			N(l.SalePrice, xlsx.Money), N(l.Base, xlsx.Money), N(l.Rate, xlsx.Percent), N(amount, xlsx.Money), S(l.Currency),
			N(l.DownNeed, xlsx.Money), S(l.DownPaidAt), S(contract), S(l.EligibleAt), S(monthLabel(l.BillMonth)), S(statusTR[l.Status]),
			S(paidAt), S(tr(noteTR, l.Notes))}
	}

	// ---- the month's bill (every month's, for all time)
	bill := xlsx.Sheet{Name: monthLabel(month), Widths: lineWidths, Freeze: 1, Rows: [][]xlsx.Cell{lineHead}}
	billTotal := map[string]float64{}
	for _, l := range lines {
		if billed(l) && (month == commissions.AllTime || l.BillMonth == month) {
			bill.Rows = append(bill.Rows, lineRow(l))
			if l.Payout != nil {
				billTotal[l.Payout.Currency] += l.Payout.Amount
			} else {
				billTotal[l.Currency] += l.Amount
			}
		}
	}
	for _, cur := range sortedKeys(billTotal) {
		row := make([]xlsx.Cell, 12)
		for i := range row {
			row[i] = xlsx.Cell{Style: xlsx.Header}
		}
		row[0] = xlsx.Cell{V: "Toplam", Style: xlsx.Header}
		row[10] = N(billTotal[cur], xlsx.Total)
		row[11] = xlsx.Cell{V: cur, Style: xlsx.Header}
		bill.Rows = append(bill.Rows, row)
	}

	// ---- still waiting on peşinat or contract
	waiting := xlsx.Sheet{
		Name:   "Bekleyenler",
		Widths: []float64{20, 26, 9, 12, 11, 16, 14, 7, 16, 16, 13, 40},
		Freeze: 1,
		Rows:   [][]xlsx.Cell{H("Danışman", "Müşteri", "Daire", "Satış tarihi", "Satış durumu", "Satış fiyatı", "Olası komisyon", "Döviz", "Gereken peşinat", "Tahsil edilen", "Sözleşme", "Eksik")},
	}
	for _, l := range lines {
		if l.Status != commissions.StatusWaiting {
			continue
		}
		contract := l.ContractAt
		if contract == "" {
			contract = map[bool]string{true: "imzalı", false: "yok"}[l.Contracted]
		}
		waiting.Rows = append(waiting.Rows, []xlsx.Cell{S(l.Agent), S(l.Customer), S(l.Unit), S(l.DealDate), S(stageTR[l.Stage]),
			N(l.SalePrice, xlsx.Money), N(l.Amount, xlsx.Money), S(l.Currency), N(l.DownNeed, xlsx.Money), N(l.DownPaid, xlsx.Money),
			S(contract), S(tr(missingTR, l.Missing))})
	}

	sheets := []xlsx.Sheet{summary, bill, waiting}
	if month == commissions.AllTime {
		sheets = append(sheets, byMonthSheet(cfg, lines))
	}

	// ---- unpaid from earlier months, and paid-then-cancelled
	for _, extra := range []struct {
		name   string
		status string
	}{{"Gecikmiş", commissions.StatusOverdue}, {"Geri alınacak", commissions.StatusClawback}} {
		sh := xlsx.Sheet{Name: extra.name, Widths: lineWidths, Freeze: 1, Rows: [][]xlsx.Cell{lineHead}}
		for _, l := range lines {
			if l.Status == extra.status {
				sh.Rows = append(sh.Rows, lineRow(l))
			}
		}
		if len(sh.Rows) > 1 {
			sheets = append(sheets, sh)
		}
	}
	return sheets
}

// billed: earned and not called off — counts toward its billing month.
func billed(l commissions.Line) bool {
	return l.Status != commissions.StatusWaiting && l.Status != commissions.StatusCancelled && l.Status != commissions.StatusClawback
}

// byMonthSheet is the all-time history: one row per billing month, agent and
// currency, newest month first.
func byMonthSheet(cfg commissions.Config, lines []commissions.Line) xlsx.Sheet {
	type agg struct {
		deals               int
		amount, paid, owing float64
	}
	// month -> agent -> currency
	sums := map[string]map[string]map[string]*agg{}
	for _, l := range lines {
		if !billed(l) {
			continue
		}
		amount, cur := l.Amount, l.Currency
		if l.Payout != nil {
			amount, cur = l.Payout.Amount, l.Payout.Currency
		}
		if sums[l.BillMonth] == nil {
			sums[l.BillMonth] = map[string]map[string]*agg{}
		}
		if sums[l.BillMonth][l.AgentID] == nil {
			sums[l.BillMonth][l.AgentID] = map[string]*agg{}
		}
		a := sums[l.BillMonth][l.AgentID][cur]
		if a == nil {
			a = &agg{}
			sums[l.BillMonth][l.AgentID][cur] = a
		}
		a.deals++
		a.amount += amount
		if l.Payout != nil {
			a.paid += amount
		} else {
			a.owing += amount
		}
	}
	H := func(v string) xlsx.Cell { return xlsx.Cell{V: v, Style: xlsx.Header} }
	sh := xlsx.Sheet{Name: "Aylara göre", Widths: []float64{16, 24, 8, 8, 16, 16, 16}, Freeze: 1,
		Rows: [][]xlsx.Cell{{H("Fatura ayı"), H("Danışman"), H("Döviz"), H("Satış"), H("Komisyon"), H("Ödenen"), H("Ödenecek")}}}
	months := slices.Collect(maps.Keys(sums))
	slices.Sort(months)
	slices.Reverse(months)
	for _, m := range months {
		for _, ag := range cfg.Agents {
			byCur := sums[m][ag.PersonnelID]
			for _, cur := range sortedKeys(byCur) {
				a := byCur[cur]
				sh.Rows = append(sh.Rows, []xlsx.Cell{xlsx.S(monthLabel(m)), xlsx.S(ag.Name), xlsx.S(cur), xlsx.N(float64(a.deals), xlsx.Plain),
					xlsx.N(a.amount, xlsx.Money), xlsx.N(a.paid, xlsx.Money), xlsx.N(a.owing, xlsx.Money)})
			}
		}
	}
	return sh
}

// currencies lists the currencies any of the maps has money in; TRY when none.
func currencies(ms ...map[string]float64) []string {
	set := map[string]float64{}
	for _, m := range ms {
		for k, v := range m {
			if v != 0 {
				set[k] = 1
			}
		}
	}
	if len(set) == 0 {
		return []string{"TRY"}
	}
	return sortedKeys(set)
}

func sortedKeys[V any](m map[string]V) []string {
	out := slices.Collect(maps.Keys(m))
	// TRY first, the rest alphabetical
	slices.SortFunc(out, func(a, b string) int {
		if (a == "TRY") != (b == "TRY") {
			if a == "TRY" {
				return -1
			}
			return 1
		}
		return strings.Compare(a, b)
	})
	return out
}

// slug is an ASCII file-name piece: "Ayşe Yılmaz" -> "ayse-yilmaz".
func slug(s string) string {
	s = strings.NewReplacer("ı", "i", "İ", "i", "ş", "s", "Ş", "s", "ğ", "g", "Ğ", "g", "ü", "u", "Ü", "u", "ö", "o", "Ö", "o", "ç", "c", "Ç", "c").Replace(s)
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
