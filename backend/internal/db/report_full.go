package db

import (
	"database/sql"
	"time"
)

// FullReport is the long-form printed report: every figure the office asks
// for, computed once server-side so the PDF page only lays it out. Same rules
// as Reports(): state is the hareket tipi, cancelled deals are out of every
// money figure, the contract flag is never consulted.
type FullReport struct {
	GeneratedAt   string          `json:"generated_at"`
	Summary       Summary         `json:"summary"`
	Daily         []DayPoint      `json:"daily"`
	Receipts      ReceiptsReport  `json:"receipts"`
	Schedule      []ScheduleMonth `json:"schedule"`
	DownPayment   DownPayment     `json:"down_payment"`
	Agents        []AgentFull     `json:"agents"`
	Agencies      []AgencyPerf    `json:"agencies"`
	Sources       []SourcePerf    `json:"sources"`
	AdSales       AdSales         `json:"ad_sales"`
	Blocks        []BlockFull     `json:"blocks"`
	Types         []TypeFull      `json:"types"`
	M2            M2Report        `json:"m2"`
	Customers     CustomerReport  `json:"customers"`
	Cancellations CancelReport    `json:"cancellations"`
	Plans         PlanReport      `json:"plans"`
}

type Summary struct {
	Deals          int     `json:"deals"`
	Live           int     `json:"live"`
	Sold           int     `json:"sold"`
	Reserved       int     `json:"reserved"`
	Cancelled      int     `json:"cancelled"`
	Customers      int     `json:"customers"`
	Buyers         int     `json:"buyers"` // distinct customers on live deals
	Units          int     `json:"units"`
	UnitsSold      int     `json:"units_sold"`
	UnitsReserved  int     `json:"units_reserved"`
	UnitsAvailable int     `json:"units_available"`
	UnitsClosed    int     `json:"units_closed"`
	M2Total        float64 `json:"m2_total"` // brüt
	M2Sold         float64 `json:"m2_sold"`
	M2Reserved     float64 `json:"m2_reserved"`
	M2Available    float64 `json:"m2_available"`
	PipelineValue  float64 `json:"pipeline_value"`
	ListValue      float64 `json:"list_value"` // Σ list price of live deals
	Discount       float64 `json:"discount"`   // ListValue − PipelineValue
	Collected      float64 `json:"collected"`
	ToCollect      float64 `json:"to_collect"`
	KaporaPaid     float64 `json:"kapora_paid"`
	RefundsOwed    float64 `json:"refunds_owed"`
	AvgPrice       float64 `json:"avg_price"`
	PricePerM2     float64 `json:"price_per_m2"`    // pipeline / m² of live deals
	StockValue     float64 `json:"stock_value"`     // list price of available units
	AgenciesActive int     `json:"agencies_active"` // agencies holding a live deal
	ViaAgency      int     `json:"via_agency"`      // live deals with an agency
}

type DayPoint struct {
	Date      string  `json:"date"`
	Reserved  int     `json:"reserved"`
	Sold      int     `json:"sold"`
	Cancelled int     `json:"cancelled"`
	Revenue   float64 `json:"revenue"`
	Receipts  float64 `json:"receipts"`
}

type ReceiptsReport struct {
	ByMonth   []LabeledMoney `json:"by_month"`
	ByMethod  []LabeledMoney `json:"by_method"`
	ByAccount []LabeledMoney `json:"by_account"`
	Count     int            `json:"count"`
	Total     float64        `json:"total"`
	Largest   float64        `json:"largest"`
	Average   float64        `json:"average"`
}

type ScheduleMonth struct {
	Month     string  `json:"month"`
	Due       float64 `json:"due"`
	Paid      float64 `json:"paid"`
	Remaining float64 `json:"remaining"`
	Count     int     `json:"count"`
}

type DownPayment struct {
	Planned      float64 `json:"planned"`      // Σ PlanPesinatToplam, live deals
	Paid         float64 `json:"paid"`         // matched against peşinat rows
	Installments float64 `json:"installments"` // Σ PlanVadeli, live deals
	BankLoan     float64 `json:"bank_loan"`
	AvgDownPct   float64 `json:"avg_down_pct"` // planned / pipeline
}

type AgentFull struct {
	AgentPerf
	M2        float64 `json:"m2"`
	ListValue float64 `json:"list_value"`
	Customers int     `json:"customers"`
}

type AgencyPerf struct {
	Agency        string         `json:"agency"`
	Reserved      int            `json:"reserved"`
	Sold          int            `json:"sold"`
	Cancelled     int            `json:"cancelled"`
	Revenue       float64        `json:"revenue"`
	Collected     float64        `json:"collected"`
	M2            float64        `json:"m2"`
	CommissionPct float64        `json:"commission_pct"` // average on live deals
	Commission    float64        `json:"commission"`     // Σ AcentaKomisyonu, live deals
	Types         []LabeledCount `json:"types"`
}

type SourcePerf struct {
	Source    string  `json:"source"`
	Reserved  int     `json:"reserved"`
	Sold      int     `json:"sold"`
	Cancelled int     `json:"cancelled"`
	Revenue   float64 `json:"revenue"`
	Collected float64 `json:"collected"`
}

type AdDeal struct {
	ID        string  `json:"id"`
	Date      string  `json:"date"`
	Customer  string  `json:"customer"`
	Unit      string  `json:"unit"`
	UnitType  string  `json:"unit_type"`
	Stage     string  `json:"stage"`
	Price     float64 `json:"price"`
	Collected float64 `json:"collected"`
	Source    string  `json:"source"`
	Agent     string  `json:"agent"`
	Agency    string  `json:"agency"` // set when an agency is on the deal despite the tag
}

// AdSales is the office's own category: deals opened on/after the campaign
// date with no agency on them — what the ads brought.
type AdSales struct {
	Since     string         `json:"since"`
	Deals     []AdDeal       `json:"deals"`
	Count     int            `json:"count"`
	Reserved  int            `json:"reserved"`
	Sold      int            `json:"sold"`
	Cancelled int            `json:"cancelled"`
	Revenue   float64        `json:"revenue"`
	Collected float64        `json:"collected"`
	AllSince  int            `json:"all_since"` // every deal since the date, for the share
	BySource  []LabeledMoney `json:"by_source"`
	ByAgent   []LabeledMoney `json:"by_agent"`
	ByType    []LabeledCount `json:"by_type"`
}

type BlockFull struct {
	Label     string  `json:"label"`
	Units     int     `json:"units"`
	Sold      int     `json:"sold"`
	Reserved  int     `json:"reserved"`
	Available int     `json:"available"`
	Closed    int     `json:"closed"`
	M2Total   float64 `json:"m2_total"`
	M2Live    float64 `json:"m2_live"` // sold + reserved
	M2Sold    float64 `json:"m2_sold"`
	M2Res     float64 `json:"m2_reserved"`
	M2Avail   float64 `json:"m2_available"`
	Revenue   float64 `json:"revenue"`
	Collected float64 `json:"collected"`
	StockList float64 `json:"stock_list"` // list value of available units
}

type TypeFull struct {
	Label      string  `json:"label"`
	Total      int     `json:"total"`
	Sold       int     `json:"sold"`
	Reserved   int     `json:"reserved"`
	Available  int     `json:"available"`
	Closed     int     `json:"closed"`
	AvgM2      float64 `json:"avg_m2"`
	M2Total    float64 `json:"m2_total"`
	M2Live     float64 `json:"m2_live"` // sold + reserved
	M2Sold     float64 `json:"m2_sold"`
	M2Res      float64 `json:"m2_reserved"`
	M2Avail    float64 `json:"m2_available"`
	Revenue    float64 `json:"revenue"`
	AvgPrice   float64 `json:"avg_price"`
	PricePerM2 float64 `json:"price_per_m2"`
	StockList  float64 `json:"stock_list"`
	Collected  float64 `json:"collected"`
}

// M2Report is the square-metre chapter: brüt and net by state, achieved
// ₺/m² on sold vs reserved deals, asking ₺/m² on what is still in stock, and
// who is carrying how many m².
type M2Report struct {
	NetTotal     float64   `json:"net_total"`
	NetSold      float64   `json:"net_sold"`
	NetReserved  float64   `json:"net_reserved"`
	NetAvailable float64   `json:"net_available"`
	NetClosed    float64   `json:"net_closed"`
	BrutClosed   float64   `json:"brut_closed"`
	CancelledM2  float64   `json:"cancelled_m2"` // brüt m² on cancelled deals (freed back to stock)
	AskPerM2     float64   `json:"ask_per_m2"`   // stock list value ÷ available brüt m²
	SoldPerM2    float64   `json:"sold_per_m2"`  // Σ price ÷ Σ brüt m² on Satıldı deals
	ResPerM2     float64   `json:"res_per_m2"`   // same on Rezerve deals
	ListPerM2    float64   `json:"list_per_m2"`  // Σ list price ÷ Σ brüt m², live deals
	ByMonth      []M2Month `json:"by_month"`
	ByType       []M2Group `json:"by_type"`
	ByBlock      []M2Group `json:"by_block"`
	ByAgent      []M2Group `json:"by_agent"`
	ByAgency     []M2Group `json:"by_agency"`
	Largest      []M2Deal  `json:"largest"` // biggest live deals by m²
}

type M2Month struct {
	Month    string  `json:"month"`
	Sold     float64 `json:"m2_sold"`
	Reserved float64 `json:"m2_reserved"`
	Count    int     `json:"count"`
	Revenue  float64 `json:"revenue"`
}

type M2Group struct {
	Label     string  `json:"label"`
	M2Sold    float64 `json:"m2_sold"`
	M2Res     float64 `json:"m2_reserved"`
	M2Avail   float64 `json:"m2_available"`
	M2Total   float64 `json:"m2_total"`
	NetSold   float64 `json:"net_sold"`
	NetRes    float64 `json:"net_reserved"`
	NetAvail  float64 `json:"net_available"`
	Revenue   float64 `json:"revenue"`
	RevSold   float64 `json:"revenue_sold"`
	RevRes    float64 `json:"revenue_reserved"`
	ListValue float64 `json:"list_value"`
	StockList float64 `json:"stock_list"`
	Collected float64 `json:"collected"`
	Deals     int     `json:"deals"`
}

type M2Deal struct {
	Unit     string  `json:"unit"`
	UnitType string  `json:"unit_type"`
	Customer string  `json:"customer"`
	Stage    string  `json:"stage"`
	BrutM2   float64 `json:"brut_m2"`
	NetM2    float64 `json:"net_m2"`
	Price    float64 `json:"price"`
}

type CustomerReport struct {
	Total      int            `json:"total"`
	Companies  int            `json:"companies"`
	Buyers     int            `json:"buyers"`
	ByCategory []LabeledCount `json:"by_category"`
	BySource   []LabeledCount `json:"by_source"`
	ByCity     []LabeledCount `json:"by_city"`
	ByWeek     []LabeledCount `json:"by_week"` // new customers per ISO week
	ByAgent    []LabeledCount `json:"by_agent"`
}

type CancelReport struct {
	Total         int            `json:"total"`
	Value         float64        `json:"value"`
	Refunds       float64        `json:"refunds"`
	AvgDays       float64        `json:"avg_days"` // AktiviteTarihi → IptalTarihi
	ByMonth       []LabeledMoney `json:"by_month"`
	ByReason      []LabeledCount `json:"by_reason"`
	ByKind        []LabeledCount `json:"by_kind"` // Sözleşmesiz İptal / Sistem
	ByAgent       []LabeledCount `json:"by_agent"`
	ByType        []LabeledCount `json:"by_type"`
	WithDeposit   int            `json:"with_deposit"` // cancelled deals that had paid something
	DepositAmount float64        `json:"deposit_amount"`
}

type PlanReport struct {
	ByType       []LabeledMoney `json:"by_type"` // Peşin / Peşin + Vadeli / —
	NPVTotal     float64        `json:"npv_total"`
	Discount     float64        `json:"discount"`
	AvgDiscount  float64        `json:"avg_discount_pct"`
	AvgDownPct   float64        `json:"avg_down_pct"`
	AvgTermMonth float64        `json:"avg_term_months"` // from installment rows
	Installments int            `json:"installments"`    // pending installment rows
}

func (s *Store) FullReport() (FullReport, error) {
	var r FullReport
	r.GeneratedAt = time.Now().Format(time.RFC3339)
	const live = liveDeal
	sold := "a.HareketTipiKisID_300=" + hareketSatildi
	rez := "a.HareketTipiKisID_300=" + hareketRezerve

	// ---------------- summary ----------------
	sm := &r.Summary
	sm.Deals = s.scanInt("SELECT COUNT(*) FROM T_Aktivite")
	sm.Cancelled = s.scanInt("SELECT COUNT(DISTINCT AktiviteID) FROM T_AktiviteIptal")
	sm.Sold = s.scanInt("SELECT COUNT(*) FROM T_Aktivite a WHERE " + sold + " AND " + live)
	sm.Reserved = s.scanInt("SELECT COUNT(*) FROM T_Aktivite a WHERE " + rez + " AND " + live)
	sm.Live = s.scanInt("SELECT COUNT(*) FROM T_Aktivite a WHERE " + live)
	sm.Customers = s.scanInt("SELECT COUNT(*) FROM T_Musteri")
	sm.Buyers = s.scanInt("SELECT COUNT(DISTINCT a.AnaMusteriID) FROM T_Aktivite a WHERE " + live)
	sm.Units = s.scanInt("SELECT COUNT(*) FROM T_Gayrimenkul")
	sm.UnitsSold = s.scanInt("SELECT COUNT(*) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketSatildi)
	sm.UnitsReserved = s.scanInt("SELECT COUNT(*) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketRezerve)
	sm.UnitsAvailable = s.scanInt("SELECT COUNT(*) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketSatilik)
	sm.UnitsClosed = s.scanInt("SELECT COUNT(*) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketKapali)
	sm.M2Total = s.scanF("SELECT SUM(BrutM2) FROM T_Gayrimenkul")
	sm.M2Sold = s.scanF("SELECT SUM(BrutM2) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketSatildi)
	sm.M2Reserved = s.scanF("SELECT SUM(BrutM2) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketRezerve)
	sm.M2Available = s.scanF("SELECT SUM(BrutM2) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketSatilik)
	sm.PipelineValue = s.scanF("SELECT SUM(a.SatisFiyati) FROM T_Aktivite a WHERE " + live)
	sm.ListValue = s.scanF("SELECT SUM(ISNULL(ag.ListeFiyati,0)) FROM T_Aktivite a JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID WHERE " + live)
	sm.Discount = sm.ListValue - sm.PipelineValue
	sm.Collected = s.scanF(`SELECT SUM(c.AlacakTutari) FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551 AND NOT EXISTS (SELECT 1 FROM T_AktiviteIptal ip WHERE ip.AktiviteID=c.KayitID)`)
	sm.RefundsOwed = s.scanF(`SELECT SUM(c.AlacakTutari) FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551 AND EXISTS (SELECT 1 FROM T_AktiviteIptal ip WHERE ip.AktiviteID=c.KayitID)`)
	sm.KaporaPaid = s.scanF(`SELECT SUM(c.AlacakTutari) FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551 AND c.KaporaMi=1 AND NOT EXISTS (SELECT 1 FROM T_AktiviteIptal ip WHERE ip.AktiviteID=c.KayitID)`)
	sm.ToCollect = sm.PipelineValue - sm.Collected
	if sm.Live > 0 {
		sm.AvgPrice = sm.PipelineValue / float64(sm.Live)
	}
	if m2 := s.scanF("SELECT SUM(g.BrutM2) FROM T_Aktivite a JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID WHERE " + live); m2 > 0 {
		sm.PricePerM2 = sm.PipelineValue / m2
	}
	sm.StockValue = s.scanF("SELECT SUM(ISNULL(f.SatisFiyati01,0)) FROM T_Gayrimenkul g LEFT JOIN T_GayrimenkulFiyatListesi1 f ON f.GayrimenkulID=g.ID WHERE g.SonHareketTipiKisID_300=" + hareketSatilik)
	sm.AgenciesActive = s.scanInt("SELECT COUNT(DISTINCT a.Araci1AcentaID) FROM T_Aktivite a WHERE a.Araci1AcentaID>0 AND " + live)
	sm.ViaAgency = s.scanInt("SELECT COUNT(*) FROM T_Aktivite a WHERE a.Araci1AcentaID>0 AND " + live)

	// ---------------- daily ----------------
	// One row per calendar day that saw a deal, a cancellation or a receipt.
	if rows, err := s.query(`
		WITH d AS (
			SELECT CONVERT(date, ` + dealOpened + `) dt,
				SUM(CASE WHEN ` + rez + ` THEN 1 ELSE 0 END) rez,
				SUM(CASE WHEN ` + sold + ` THEN 1 ELSE 0 END) sld,
				SUM(a.SatisFiyati) rev, 0 canc, CAST(0 AS money) rcpt
			FROM T_Aktivite a WHERE ` + live + ` AND YEAR(` + dealOpened + `)>1901
			GROUP BY CONVERT(date, ` + dealOpened + `)
			UNION ALL
			SELECT CONVERT(date, ip.IptalTarihi), 0, 0, 0, COUNT(*), 0
			FROM T_AktiviteIptal ip WHERE YEAR(ip.IptalTarihi)>1901 GROUP BY CONVERT(date, ip.IptalTarihi)
			UNION ALL
			SELECT CONVERT(date, c.Tarih), 0, 0, 0, 0, SUM(c.AlacakTutari)
			FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551
			  AND NOT EXISTS (SELECT 1 FROM T_AktiviteIptal ip WHERE ip.AktiviteID=c.KayitID)
			GROUP BY CONVERT(date, c.Tarih)
		)
		SELECT CONVERT(varchar(10), dt, 23), SUM(rez), SUM(sld), SUM(canc), SUM(rev), SUM(rcpt) FROM d GROUP BY dt ORDER BY dt`); err == nil {
		for rows.Next() {
			var p DayPoint
			var dt sql.NullString
			var rev, rc sql.NullFloat64
			rows.Scan(&dt, &p.Reserved, &p.Sold, &p.Cancelled, &rev, &rc)
			p.Date, p.Revenue, p.Receipts = dt.String, rev.Float64, rc.Float64
			r.Daily = append(r.Daily, p)
		}
		rows.Close()
	}

	// ---------------- receipts ----------------
	liveReceipt := `NOT EXISTS (SELECT 1 FROM T_AktiviteIptal ip WHERE ip.AktiviteID=c.KayitID)`
	r.Receipts.ByMonth = s.labeledMoney(`SELECT FORMAT(c.Tarih,'yyyy-MM') m, COUNT(*), SUM(c.AlacakTutari)
		FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551 AND `+liveReceipt+` GROUP BY FORMAT(c.Tarih,'yyyy-MM') ORDER BY m`, nil)
	r.Receipts.ByMethod = s.labeledMoney(`SELECT c.OdemeTipiKisID_302, COUNT(*), SUM(c.AlacakTutari)
		FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551 AND `+liveReceipt+` GROUP BY c.OdemeTipiKisID_302 ORDER BY SUM(c.AlacakTutari) DESC`, s.kis)
	acc, _ := loadMap(s, "T_HesapKartlari", "ID", "HesapAciklama")
	r.Receipts.ByAccount = s.labeledMoney(`SELECT c.HesapKartlariID, COUNT(*), SUM(c.AlacakTutari)
		FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551 AND `+liveReceipt+` GROUP BY c.HesapKartlariID ORDER BY SUM(c.AlacakTutari) DESC`, acc)
	r.Receipts.Count = s.scanInt(`SELECT COUNT(*) FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551 AND ` + liveReceipt)
	r.Receipts.Total = sm.Collected
	r.Receipts.Largest = s.scanF(`SELECT MAX(c.AlacakTutari) FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551 AND ` + liveReceipt)
	if r.Receipts.Count > 0 {
		r.Receipts.Average = r.Receipts.Total / float64(r.Receipts.Count)
	}

	// ---------------- schedule (T_CariBorc on live deals) ----------------
	if rows, err := s.query(`SELECT FORMAT(b.Tarih,'yyyy-MM') m, COUNT(*), SUM(b.BorcTutari), SUM(ISNULL(k.paid,0))
		FROM T_CariBorc b JOIN T_Aktivite a ON a.ID=b.KayitID
		LEFT JOIN (SELECT CariBorcID, SUM(KapatilanTutar) paid FROM T_CariAlacakBorcKapatma GROUP BY CariBorcID) k ON k.CariBorcID=b.ID
		WHERE b.CariIslemYeriKisID_310=550 AND b.BorcTutari>0 AND b.Tarih IS NOT NULL AND YEAR(b.Tarih)>1901 AND ` + live + `
		GROUP BY FORMAT(b.Tarih,'yyyy-MM') ORDER BY m`); err == nil {
		for rows.Next() {
			var m sql.NullString
			var c int
			var due, paid sql.NullFloat64
			rows.Scan(&m, &c, &due, &paid)
			r.Schedule = append(r.Schedule, ScheduleMonth{m.String, due.Float64, paid.Float64, due.Float64 - paid.Float64, c})
		}
		rows.Close()
	}

	// ---------------- down payment ----------------
	r.DownPayment.Planned = s.scanF("SELECT SUM(a.PlanPesinatToplam) FROM T_Aktivite a WHERE " + live)
	r.DownPayment.Installments = s.scanF("SELECT SUM(a.PlanVadeli) FROM T_Aktivite a WHERE " + live)
	r.DownPayment.BankLoan = s.scanF("SELECT SUM(a.PlanBanka) FROM T_Aktivite a WHERE " + live)
	r.DownPayment.Paid = s.scanF(`SELECT SUM(k.KapatilanTutar) FROM T_CariAlacakBorcKapatma k
		JOIN T_CariBorc b ON b.ID=k.CariBorcID JOIN T_Aktivite a ON a.ID=b.KayitID
		WHERE b.TipKisID_304=381 AND ` + live)
	if sm.PipelineValue > 0 {
		r.DownPayment.AvgDownPct = r.DownPayment.Planned / sm.PipelineValue * 100
	}

	// ---------------- agents ----------------
	if rows, err := s.query(`SELECT a.SatisPersonelID1,
			SUM(CASE WHEN ip.AktiviteID IS NULL AND ` + sold + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL AND ` + rez + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NOT NULL THEN 1 ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL THEN a.SatisFiyati ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL THEN ISNULL(bk.Odedigi,0) ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL THEN ISNULL(g.BrutM2,0) ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL THEN ISNULL(ag.ListeFiyati,0) ELSE 0 END),
			COUNT(DISTINCT CASE WHEN ip.AktiviteID IS NULL THEN a.AnaMusteriID END)
		FROM T_Aktivite a
		LEFT JOIN T_AktiviteIptal ip ON ip.AktiviteID=a.ID
		LEFT JOIN T_AktiviteBakiye bk ON bk.AktiviteID=a.ID
		LEFT JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID
		LEFT JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		GROUP BY a.SatisPersonelID1
		HAVING SUM(CASE WHEN ip.AktiviteID IS NULL THEN 1 ELSE 0 END) > 0
		ORDER BY 5 DESC`); err == nil {
		for rows.Next() {
			var id sql.NullString
			var a AgentFull
			var rev, col, m2, lst sql.NullFloat64
			rows.Scan(&id, &a.Sales, &a.Reserved, &a.Cancelled, &rev, &col, &m2, &lst, &a.Customers)
			a.Agent = dec(s.per, id)
			if a.Agent == "" {
				a.Agent = "—"
			}
			a.Revenue, a.Collected, a.M2, a.ListValue = rev.Float64, col.Float64, m2.Float64, lst.Float64
			r.Agents = append(r.Agents, a)
		}
		rows.Close()
	}

	// ---------------- agencies ----------------
	if rows, err := s.query(`SELECT a.Araci1AcentaID,
			SUM(CASE WHEN ip.AktiviteID IS NULL AND ` + sold + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL AND ` + rez + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NOT NULL THEN 1 ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL THEN a.SatisFiyati ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL THEN ISNULL(bk.Odedigi,0) ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL THEN ISNULL(g.BrutM2,0) ELSE 0 END),
			AVG(CASE WHEN ip.AktiviteID IS NULL THEN NULLIF(pr.AcentaninKomisyonYuzdesi,0) END),
			SUM(CASE WHEN ip.AktiviteID IS NULL THEN ISNULL(pr.AcentaKomisyonu,0) ELSE 0 END)
		FROM T_Aktivite a
		LEFT JOIN T_AktiviteIptal ip ON ip.AktiviteID=a.ID
		LEFT JOIN T_AktiviteBakiye bk ON bk.AktiviteID=a.ID
		LEFT JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID
		LEFT JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		LEFT JOIN T_AktivitePrim pr ON pr.AktiviteID=a.ID
		GROUP BY a.Araci1AcentaID
		ORDER BY 5 DESC`); err == nil {
		for rows.Next() {
			var id sql.NullString
			var p AgencyPerf
			var rev, col, m2, pct, com sql.NullFloat64
			rows.Scan(&id, &p.Sold, &p.Reserved, &p.Cancelled, &rev, &col, &m2, &pct, &com)
			p.Agency = dec(s.ace, id)
			if p.Agency == "" {
				p.Agency = "Doğrudan (acentasız)"
			}
			p.Revenue, p.Collected, p.M2, p.CommissionPct, p.Commission = rev.Float64, col.Float64, m2.Float64, pct.Float64, com.Float64
			p.Types = s.labeledCount(`SELECT g.GayrimenkulTipDurumSecID_206, COUNT(*) FROM T_Aktivite a
				JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
				WHERE a.Araci1AcentaID=@p1 AND `+live+` GROUP BY g.GayrimenkulTipDurumSecID_206 ORDER BY COUNT(*) DESC`, s.dur, id.String)
			if p.Sold+p.Reserved+p.Cancelled > 0 {
				r.Agencies = append(r.Agencies, p)
			}
		}
		rows.Close()
	}

	// ---------------- sources ----------------
	if rows, err := s.query(`SELECT a.MusteriKaynagiID_1,
			SUM(CASE WHEN ip.AktiviteID IS NULL AND ` + sold + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL AND ` + rez + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NOT NULL THEN 1 ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL THEN a.SatisFiyati ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL THEN ISNULL(bk.Odedigi,0) ELSE 0 END)
		FROM T_Aktivite a
		LEFT JOIN T_AktiviteIptal ip ON ip.AktiviteID=a.ID
		LEFT JOIN T_AktiviteBakiye bk ON bk.AktiviteID=a.ID
		GROUP BY a.MusteriKaynagiID_1 ORDER BY 5 DESC`); err == nil {
		for rows.Next() {
			var id sql.NullString
			var p SourcePerf
			var rev, col sql.NullFloat64
			rows.Scan(&id, &p.Sold, &p.Reserved, &p.Cancelled, &rev, &col)
			p.Source = dec(s.kay, id)
			if p.Source == "" {
				p.Source = "Belirtilmemiş"
			}
			p.Revenue, p.Collected = rev.Float64, col.Float64
			r.Sources = append(r.Sources, p)
		}
		rows.Close()
	}

	// ---------------- ad sales ----------------
	// The office's rule, and nothing else: opened on/after the campaign date
	// with no agency on the deal. The Müşteri Kaynağı tag is deliberately
	// ignored — it is filled by hand and does not decide anything here.
	//
	// "Opened" is RezerveTarihi, not AktiviteTarihi (see dealOpened): Prism
	// re-stamps AktiviteTarihi when a reservation turns into a sale, which
	// pulled older deals into the campaign. RezerveTarihi stays put, and it is
	// the date the CRM feed opens the deal on, so both reports count the same
	// deals.
	r.AdSales.Since = "2026-09-04"
	opened := dealOpened
	adWhere := opened + ` >= '2026-09-04' AND ISNULL(a.Araci1AcentaID,0)=0`
	r.AdSales.AllSince = s.scanInt(`SELECT COUNT(*) FROM T_Aktivite a WHERE ` + opened + ` >= '2026-09-04'`)
	if rows, err := s.query(`SELECT a.ID, CONVERT(varchar(10), ` + opened + `, 23), m.MusteriAdiSoyadi,
			COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo), g.DaireNo, g.GayrimenkulTipDurumSecID_206,
			a.HareketTipiKisID_300, ip.AktiviteID, a.SatisFiyati, ISNULL(bk.Odedigi,0), a.MusteriKaynagiID_1, a.SatisPersonelID1, a.Araci1AcentaID
		FROM T_Aktivite a
		LEFT JOIN T_Musteri m ON m.ID=a.AnaMusteriID
		LEFT JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID
		LEFT JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		LEFT JOIN T_AktiviteIptal ip ON ip.AktiviteID=a.ID
		LEFT JOIN T_AktiviteBakiye bk ON bk.AktiviteID=a.ID
		WHERE ` + adWhere + ` ORDER BY ` + opened + `, a.ID`); err == nil {
		for rows.Next() {
			var id, dt, cust, blk, num, typ, hareket, iptal, src, per, agy sql.NullString
			var price, col sql.NullFloat64
			rows.Scan(&id, &dt, &cust, &blk, &num, &typ, &hareket, &iptal, &price, &col, &src, &per, &agy)
			d := AdDeal{ID: id.String, Date: dt.String, Customer: cust.String, Unit: blk.String + "-" + num.String,
				UnitType: dec(s.dur, typ), Price: price.Float64, Collected: col.Float64, Source: dec(s.kay, src), Agent: dec(s.per, per), Agency: dec(s.ace, agy)}
			if d.Source == "" {
				d.Source = "Belirtilmemiş"
			}
			switch {
			case iptal.Valid && iptal.String != "":
				d.Stage = "cancelled"
				r.AdSales.Cancelled++
			case hareket.String == hareketSatildi:
				d.Stage = "sale"
				r.AdSales.Sold++
				r.AdSales.Revenue += d.Price
				r.AdSales.Collected += d.Collected
			case hareket.String == hareketRezerve:
				d.Stage = "reserved"
				r.AdSales.Reserved++
				r.AdSales.Revenue += d.Price
				r.AdSales.Collected += d.Collected
			default:
				d.Stage = "negotiation"
			}
			r.AdSales.Deals = append(r.AdSales.Deals, d)
		}
		rows.Close()
	}
	r.AdSales.Count = len(r.AdSales.Deals)
	r.AdSales.BySource = s.labeledMoney(`SELECT a.MusteriKaynagiID_1, COUNT(*), SUM(a.SatisFiyati) FROM T_Aktivite a WHERE `+adWhere+` AND `+live+` GROUP BY a.MusteriKaynagiID_1 ORDER BY COUNT(*) DESC`, s.kay)
	r.AdSales.ByAgent = s.labeledMoney(`SELECT a.SatisPersonelID1, COUNT(*), SUM(a.SatisFiyati) FROM T_Aktivite a WHERE `+adWhere+` AND `+live+` GROUP BY a.SatisPersonelID1 ORDER BY COUNT(*) DESC`, s.per)
	r.AdSales.ByType = s.labeledCount(`SELECT g.GayrimenkulTipDurumSecID_206, COUNT(*) FROM T_Aktivite a
		JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		WHERE `+adWhere+` AND `+live+` GROUP BY g.GayrimenkulTipDurumSecID_206 ORDER BY COUNT(*) DESC`, s.dur)

	// ---------------- blocks ----------------
	if rows, err := s.query(`SELECT COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo) blk, COUNT(*),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatildi + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketRezerve + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatilik + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketKapali + ` THEN 1 ELSE 0 END),
			SUM(ISNULL(g.BrutM2,0)),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatildi + ` THEN ISNULL(g.BrutM2,0) ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketRezerve + ` THEN ISNULL(g.BrutM2,0) ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatilik + ` THEN ISNULL(g.BrutM2,0) ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatilik + ` THEN ISNULL(f.SatisFiyati01,0) ELSE 0 END)
		FROM T_Gayrimenkul g LEFT JOIN T_GayrimenkulFiyatListesi1 f ON f.GayrimenkulID=g.ID
		GROUP BY COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo) ORDER BY blk`); err == nil {
		for rows.Next() {
			var b BlockFull
			var lab sql.NullString
			var m2t, m2s, m2r, m2a, stock sql.NullFloat64
			rows.Scan(&lab, &b.Units, &b.Sold, &b.Reserved, &b.Available, &b.Closed, &m2t, &m2s, &m2r, &m2a, &stock)
			b.Label, b.M2Total, b.M2Sold, b.M2Res, b.M2Avail, b.StockList = lab.String, m2t.Float64, m2s.Float64, m2r.Float64, m2a.Float64, stock.Float64
			b.M2Live = b.M2Sold + b.M2Res
			r.Blocks = append(r.Blocks, b)
		}
		rows.Close()
	}
	// money per block from live deals
	if rows, err := s.query(`SELECT COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo) blk, SUM(a.SatisFiyati), SUM(ISNULL(bk.Odedigi,0))
		FROM T_Aktivite a JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		LEFT JOIN T_AktiviteBakiye bk ON bk.AktiviteID=a.ID
		WHERE ` + live + ` GROUP BY COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo)`); err == nil {
		money := map[string][2]float64{}
		for rows.Next() {
			var lab sql.NullString
			var rev, col sql.NullFloat64
			rows.Scan(&lab, &rev, &col)
			money[lab.String] = [2]float64{rev.Float64, col.Float64}
		}
		rows.Close()
		for i := range r.Blocks {
			m := money[r.Blocks[i].Label]
			r.Blocks[i].Revenue, r.Blocks[i].Collected = m[0], m[1]
		}
	}

	// ---------------- types ----------------
	if rows, err := s.query(`SELECT g.GayrimenkulTipDurumSecID_206, COUNT(*),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatildi + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketRezerve + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatilik + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketKapali + ` THEN 1 ELSE 0 END),
			AVG(NULLIF(g.BrutM2,0)),
			SUM(ISNULL(g.BrutM2,0)),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatildi + ` THEN ISNULL(g.BrutM2,0) ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketRezerve + ` THEN ISNULL(g.BrutM2,0) ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatilik + ` THEN ISNULL(g.BrutM2,0) ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatilik + ` THEN ISNULL(f.SatisFiyati01,0) ELSE 0 END)
		FROM T_Gayrimenkul g LEFT JOIN T_GayrimenkulFiyatListesi1 f ON f.GayrimenkulID=g.ID
		GROUP BY g.GayrimenkulTipDurumSecID_206 ORDER BY COUNT(*) DESC`); err == nil {
		for rows.Next() {
			var t TypeFull
			var id sql.NullString
			var avg, m2t, m2s, m2r, m2a, stock sql.NullFloat64
			rows.Scan(&id, &t.Total, &t.Sold, &t.Reserved, &t.Available, &t.Closed, &avg, &m2t, &m2s, &m2r, &m2a, &stock)
			t.Label = dec(s.dur, id)
			if t.Label == "" {
				t.Label = "—"
			}
			t.AvgM2, t.M2Total, t.M2Sold, t.M2Res, t.M2Avail, t.StockList = avg.Float64, m2t.Float64, m2s.Float64, m2r.Float64, m2a.Float64, stock.Float64
			t.M2Live = t.M2Sold + t.M2Res
			r.Types = append(r.Types, t)
		}
		rows.Close()
	}
	if rows, err := s.query(`SELECT g.GayrimenkulTipDurumSecID_206, COUNT(*), SUM(a.SatisFiyati), SUM(ISNULL(bk.Odedigi,0)), SUM(ISNULL(g.BrutM2,0))
		FROM T_Aktivite a JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		LEFT JOIN T_AktiviteBakiye bk ON bk.AktiviteID=a.ID
		WHERE ` + live + ` GROUP BY g.GayrimenkulTipDurumSecID_206`); err == nil {
		type agg struct {
			n            int
			rev, col, m2 float64
		}
		by := map[string]agg{}
		for rows.Next() {
			var id sql.NullString
			var n int
			var rev, col, m2 sql.NullFloat64
			rows.Scan(&id, &n, &rev, &col, &m2)
			lab := dec(s.dur, id)
			if lab == "" {
				lab = "—"
			}
			by[lab] = agg{n, rev.Float64, col.Float64, m2.Float64}
		}
		rows.Close()
		for i := range r.Types {
			a := by[r.Types[i].Label]
			r.Types[i].Revenue, r.Types[i].Collected = a.rev, a.col
			if a.n > 0 {
				r.Types[i].AvgPrice = a.rev / float64(a.n)
			}
			if a.m2 > 0 {
				r.Types[i].PricePerM2 = a.rev / a.m2
			}
		}
	}

	// ---------------- m² ----------------
	m := &r.M2
	m.NetTotal = s.scanF("SELECT SUM(NetM2) FROM T_Gayrimenkul")
	m.NetSold = s.scanF("SELECT SUM(NetM2) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketSatildi)
	m.NetReserved = s.scanF("SELECT SUM(NetM2) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketRezerve)
	m.NetAvailable = s.scanF("SELECT SUM(NetM2) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketSatilik)
	m.NetClosed = s.scanF("SELECT SUM(NetM2) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketKapali)
	m.BrutClosed = s.scanF("SELECT SUM(BrutM2) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketKapali)
	m.CancelledM2 = s.scanF(`SELECT SUM(ISNULL(g.BrutM2,0)) FROM T_Aktivite a JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		WHERE EXISTS (SELECT 1 FROM T_AktiviteIptal ip WHERE ip.AktiviteID=a.ID)`)
	if sm.M2Available > 0 {
		m.AskPerM2 = sm.StockValue / sm.M2Available
	}
	if v := s.scanF(`SELECT SUM(ISNULL(g.BrutM2,0)) FROM T_Aktivite a JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID WHERE ` + sold + ` AND ` + live); v > 0 {
		m.SoldPerM2 = s.scanF(`SELECT SUM(a.SatisFiyati) FROM T_Aktivite a WHERE `+sold+` AND `+live) / v
	}
	if v := s.scanF(`SELECT SUM(ISNULL(g.BrutM2,0)) FROM T_Aktivite a JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID WHERE ` + rez + ` AND ` + live); v > 0 {
		m.ResPerM2 = s.scanF(`SELECT SUM(a.SatisFiyati) FROM T_Aktivite a WHERE `+rez+` AND `+live) / v
	}
	if v := s.scanF(`SELECT SUM(ISNULL(g.BrutM2,0)) FROM T_Aktivite a JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID WHERE ` + live); v > 0 {
		m.ListPerM2 = sm.ListValue / v
	}
	if rows, err := s.query(`SELECT FORMAT(` + dealOpened + `,'yyyy-MM') mo,
			SUM(CASE WHEN ` + sold + ` THEN ISNULL(g.BrutM2,0) ELSE 0 END),
			SUM(CASE WHEN ` + rez + ` THEN ISNULL(g.BrutM2,0) ELSE 0 END),
			COUNT(*), SUM(a.SatisFiyati)
		FROM T_Aktivite a JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		WHERE ` + live + ` AND YEAR(` + dealOpened + `)>1901
		GROUP BY FORMAT(` + dealOpened + `,'yyyy-MM') ORDER BY mo`); err == nil {
		for rows.Next() {
			var x M2Month
			var mo sql.NullString
			var ms, mr, rv sql.NullFloat64
			rows.Scan(&mo, &ms, &mr, &x.Count, &rv)
			x.Month, x.Sold, x.Reserved, x.Revenue = mo.String, ms.Float64, mr.Float64, rv.Float64
			m.ByMonth = append(m.ByMonth, x)
		}
		rows.Close()
	}
	// stock side (every unit) and deal side (live deals) per key, merged in Go
	m2Groups := func(stockKey, dealKey string, decode map[string]string, order string) []M2Group {
		idx := map[string]int{}
		out := []M2Group{}
		lab := func(k sql.NullString) string {
			l := k.String
			if decode != nil {
				l = dec(decode, k)
			}
			if l == "" {
				l = "—"
			}
			return l
		}
		get := func(l string) *M2Group {
			i, ok := idx[l]
			if !ok {
				out = append(out, M2Group{Label: l})
				i = len(out) - 1
				idx[l] = i
			}
			return &out[i]
		}
		if rows, err := s.query(`SELECT ` + stockKey + ` k,
				SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatildi + ` THEN ISNULL(g.BrutM2,0) ELSE 0 END),
				SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketRezerve + ` THEN ISNULL(g.BrutM2,0) ELSE 0 END),
				SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatilik + ` THEN ISNULL(g.BrutM2,0) ELSE 0 END),
				SUM(ISNULL(g.BrutM2,0)),
				SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatildi + ` THEN ISNULL(g.NetM2,0) ELSE 0 END),
				SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketRezerve + ` THEN ISNULL(g.NetM2,0) ELSE 0 END),
				SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatilik + ` THEN ISNULL(g.NetM2,0) ELSE 0 END),
				SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatilik + ` THEN ISNULL(f.SatisFiyati01,0) ELSE 0 END)
			FROM T_Gayrimenkul g LEFT JOIN T_GayrimenkulFiyatListesi1 f ON f.GayrimenkulID=g.ID
			GROUP BY ` + stockKey + ` ORDER BY ` + order); err == nil {
			for rows.Next() {
				var k sql.NullString
				var a, b, c, d, e, f2, g2, h sql.NullFloat64
				rows.Scan(&k, &a, &b, &c, &d, &e, &f2, &g2, &h)
				x := get(lab(k))
				x.M2Sold, x.M2Res, x.M2Avail, x.M2Total = a.Float64, b.Float64, c.Float64, d.Float64
				x.NetSold, x.NetRes, x.NetAvail, x.StockList = e.Float64, f2.Float64, g2.Float64, h.Float64
			}
			rows.Close()
		}
		if rows, err := s.query(`SELECT ` + dealKey + ` k, COUNT(*), SUM(a.SatisFiyati),
				SUM(CASE WHEN ` + sold + ` THEN a.SatisFiyati ELSE 0 END),
				SUM(CASE WHEN ` + rez + ` THEN a.SatisFiyati ELSE 0 END),
				SUM(ISNULL(ag.ListeFiyati,0)), SUM(ISNULL(bk.Odedigi,0))
			FROM T_Aktivite a JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
			LEFT JOIN T_AktiviteBakiye bk ON bk.AktiviteID=a.ID
			WHERE ` + live + ` GROUP BY ` + dealKey); err == nil {
			for rows.Next() {
				var k sql.NullString
				var n int
				var rv, rs, rr, lv, col sql.NullFloat64
				rows.Scan(&k, &n, &rv, &rs, &rr, &lv, &col)
				x := get(lab(k))
				x.Deals, x.Revenue, x.RevSold, x.RevRes, x.ListValue, x.Collected = n, rv.Float64, rs.Float64, rr.Float64, lv.Float64, col.Float64
			}
			rows.Close()
		}
		return out
	}
	m.ByType = m2Groups("g.GayrimenkulTipDurumSecID_206", "g.GayrimenkulTipDurumSecID_206", s.dur, "SUM(ISNULL(g.BrutM2,0)) DESC")
	m.ByBlock = m2Groups("COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo)", "COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo)", nil, "k")
	// agents / agencies only have a deal side: m² they carry in live deals
	dealSide := func(key string, decode map[string]string) []M2Group {
		out := []M2Group{}
		if rows, err := s.query(`SELECT ` + key + ` k, COUNT(*), SUM(a.SatisFiyati),
				SUM(CASE WHEN ` + sold + ` THEN ISNULL(g.BrutM2,0) ELSE 0 END),
				SUM(CASE WHEN ` + rez + ` THEN ISNULL(g.BrutM2,0) ELSE 0 END),
				SUM(CASE WHEN ` + sold + ` THEN ISNULL(g.NetM2,0) ELSE 0 END),
				SUM(CASE WHEN ` + rez + ` THEN ISNULL(g.NetM2,0) ELSE 0 END),
				SUM(ISNULL(bk.Odedigi,0))
			FROM T_Aktivite a JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
			LEFT JOIN T_AktiviteBakiye bk ON bk.AktiviteID=a.ID
			WHERE ` + live + ` GROUP BY ` + key + ` ORDER BY 3 DESC`); err == nil {
			for rows.Next() {
				var k sql.NullString
				var x M2Group
				var rv, ms, mr, ns, nr, col sql.NullFloat64
				rows.Scan(&k, &x.Deals, &rv, &ms, &mr, &ns, &nr, &col)
				x.Label = dec(decode, k)
				if x.Label == "" {
					x.Label = "Doğrudan"
				}
				x.Revenue, x.M2Sold, x.M2Res, x.NetSold, x.NetRes, x.Collected = rv.Float64, ms.Float64, mr.Float64, ns.Float64, nr.Float64, col.Float64
				x.M2Total = x.M2Sold + x.M2Res
				out = append(out, x)
			}
			rows.Close()
		}
		return out
	}
	m.ByAgent = dealSide("a.SatisPersonelID1", s.per)
	m.ByAgency = dealSide("a.Araci1AcentaID", s.ace)
	if rows, err := s.query(`SELECT TOP 10 COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo), g.DaireNo, g.GayrimenkulTipDurumSecID_206, mu.MusteriAdiSoyadi,
			a.HareketTipiKisID_300, ISNULL(g.BrutM2,0), ISNULL(g.NetM2,0), a.SatisFiyati
		FROM T_Aktivite a JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		LEFT JOIN T_Musteri mu ON mu.ID=a.AnaMusteriID
		WHERE ` + live + ` ORDER BY g.BrutM2 DESC`); err == nil {
		for rows.Next() {
			var blk, num, typ, cust, hareket sql.NullString
			var x M2Deal
			rows.Scan(&blk, &num, &typ, &cust, &hareket, &x.BrutM2, &x.NetM2, &x.Price)
			x.Unit, x.UnitType, x.Customer = blk.String+"-"+num.String, dec(s.dur, typ), cust.String
			x.Stage = "reserved"
			if hareket.String == hareketSatildi {
				x.Stage = "sale"
			}
			m.Largest = append(m.Largest, x)
		}
		rows.Close()
	}

	// ---------------- customers ----------------
	c := &r.Customers
	c.Total = sm.Customers
	c.Buyers = sm.Buyers
	c.Companies = s.scanInt("SELECT COUNT(*) FROM T_Musteri WHERE SirketMi=1")
	c.ByCategory = s.labeledCount(`SELECT MusteriKategoriDurumSecID_90, COUNT(*) FROM T_Musteri GROUP BY MusteriKategoriDurumSecID_90 ORDER BY COUNT(*) DESC`, s.dur)
	c.BySource = s.labeledCount(`SELECT d.MusteriKaynagiID_1, COUNT(*) FROM T_MusteriDetay d GROUP BY d.MusteriKaynagiID_1 ORDER BY COUNT(*) DESC`, s.kay)
	c.ByCity = s.labeledCount(`SELECT TOP 10 NULLIF(LTRIM(RTRIM(i.AdresSehir)),''), COUNT(*) FROM T_MusteriDetayIletisim i GROUP BY NULLIF(LTRIM(RTRIM(i.AdresSehir)),'') ORDER BY COUNT(*) DESC`, nil)
	c.ByWeek = s.labeledCount(`SELECT CONVERT(varchar(10), DATEADD(day, 1-DATEPART(weekday, m.Tarih)+1, CONVERT(date, m.Tarih)), 23) wk, COUNT(*)
		FROM T_Musteri m WHERE YEAR(m.Tarih)>1901 GROUP BY DATEADD(day, 1-DATEPART(weekday, m.Tarih)+1, CONVERT(date, m.Tarih)) ORDER BY wk`, nil)
	c.ByAgent = s.labeledCount(`SELECT m.PersonelID_1, COUNT(*) FROM T_Musteri m GROUP BY m.PersonelID_1 ORDER BY COUNT(*) DESC`, s.per)

	// ---------------- cancellations ----------------
	x := &r.Cancellations
	x.Total = sm.Cancelled
	x.Refunds = sm.RefundsOwed
	x.Value = s.scanF(`SELECT SUM(a.SatisFiyati) FROM T_Aktivite a WHERE EXISTS (SELECT 1 FROM T_AktiviteIptal ip WHERE ip.AktiviteID=a.ID)`)
	x.AvgDays = s.scanF(`SELECT AVG(CAST(DATEDIFF(day, a.AktiviteTarihi, ip.IptalTarihi) AS float)) FROM T_AktiviteIptal ip JOIN T_Aktivite a ON a.ID=ip.AktiviteID WHERE YEAR(ip.IptalTarihi)>1901 AND YEAR(a.AktiviteTarihi)>1901`)
	x.ByMonth = s.labeledMoney(`SELECT FORMAT(ip.IptalTarihi,'yyyy-MM') m, COUNT(*), SUM(a.SatisFiyati) FROM T_AktiviteIptal ip JOIN T_Aktivite a ON a.ID=ip.AktiviteID WHERE YEAR(ip.IptalTarihi)>1901 GROUP BY FORMAT(ip.IptalTarihi,'yyyy-MM') ORDER BY m`, nil)
	// the "Kis" in the column name lies: reasons live in Tn_DurumSec (başlık 6)
	x.ByReason = s.labeledCount(`SELECT ip.IptalNedenIDKisID_6, COUNT(*) FROM T_AktiviteIptal ip GROUP BY ip.IptalNedenIDKisID_6 ORDER BY COUNT(*) DESC`, s.dur)
	x.ByKind = s.labeledCount(`SELECT ip.IptalTuruDurumSec_ID_49, COUNT(*) FROM T_AktiviteIptal ip GROUP BY ip.IptalTuruDurumSec_ID_49 ORDER BY COUNT(*) DESC`, s.dur)
	x.ByAgent = s.labeledCount(`SELECT a.SatisPersonelID1, COUNT(*) FROM T_AktiviteIptal ip JOIN T_Aktivite a ON a.ID=ip.AktiviteID GROUP BY a.SatisPersonelID1 ORDER BY COUNT(*) DESC`, s.per)
	x.ByType = s.labeledCount(`SELECT g.GayrimenkulTipDurumSecID_206, COUNT(*) FROM T_AktiviteIptal ip JOIN T_Aktivite a ON a.ID=ip.AktiviteID
		JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID GROUP BY g.GayrimenkulTipDurumSecID_206 ORDER BY COUNT(*) DESC`, s.dur)
	x.WithDeposit = s.scanInt(`SELECT COUNT(DISTINCT c.KayitID) FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551 AND EXISTS (SELECT 1 FROM T_AktiviteIptal ip WHERE ip.AktiviteID=c.KayitID)`)
	x.DepositAmount = sm.RefundsOwed

	// ---------------- plans ----------------
	pl := &r.Plans
	pl.ByType = s.labeledMoney(`SELECT a.OdemeTahutuKisID_301, COUNT(*), SUM(a.SatisFiyati) FROM T_Aktivite a WHERE `+live+` GROUP BY a.OdemeTahutuKisID_301 ORDER BY COUNT(*) DESC`, s.kis)
	pl.NPVTotal = s.scanF("SELECT SUM(a.NpvTutari) FROM T_Aktivite a WHERE " + live)
	pl.Discount = s.scanF("SELECT SUM(a.ToplamIndirimTutari) FROM T_Aktivite a WHERE " + live)
	if sm.ListValue > 0 {
		pl.AvgDiscount = sm.Discount / sm.ListValue * 100
	}
	pl.AvgDownPct = r.DownPayment.AvgDownPct
	pl.Installments = s.scanInt(`SELECT COUNT(*) FROM T_CariBorc b JOIN T_Aktivite a ON a.ID=b.KayitID
		WHERE b.CariIslemYeriKisID_310=550 AND b.TipKisID_304=380 AND b.BorcTutari>0 AND ` + live)
	pl.AvgTermMonth = s.scanF(`SELECT AVG(CAST(t.n AS float)) FROM (
		SELECT b.KayitID, COUNT(*) n FROM T_CariBorc b JOIN T_Aktivite a ON a.ID=b.KayitID
		WHERE b.CariIslemYeriKisID_310=550 AND b.TipKisID_304=380 AND b.BorcTutari>0 AND ` + live + ` GROUP BY b.KayitID) t`)

	return r, nil
}

// labeledMoney runs a (key, count, sum) query and decodes the key through m
// when given; a nil map keeps the key as the label.
func (s *Store) labeledMoney(q string, m map[string]string, args ...any) []LabeledMoney {
	out := []LabeledMoney{}
	rows, err := s.query(q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var k sql.NullString
		var c int
		var v sql.NullFloat64
		rows.Scan(&k, &c, &v)
		lab := k.String
		if m != nil {
			lab = dec(m, k)
		}
		if lab == "" {
			lab = "Belirtilmemiş"
		}
		out = append(out, LabeledMoney{lab, c, v.Float64})
	}
	return out
}

func (s *Store) labeledCount(q string, m map[string]string, args ...any) []LabeledCount {
	out := []LabeledCount{}
	rows, err := s.query(q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var k sql.NullString
		var c int
		rows.Scan(&k, &c)
		lab := k.String
		if m != nil {
			lab = dec(m, k)
		}
		if lab == "" {
			lab = "Belirtilmemiş"
		}
		out = append(out, LabeledCount{lab, c})
	}
	return out
}
