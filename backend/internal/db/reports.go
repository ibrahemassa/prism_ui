package db

import "database/sql"

// Reports is Prism's own reading of the book: a deal is what its hareket tipi
// says (Rezerve or Satıldı), a cancelled deal is out of every money figure, and
// the contract flag is never consulted — see the hareket* constants.
type Reports struct {
	Funnel       Funnel       `json:"funnel"`
	Collection   Collection   `json:"collection"`
	SalesByMonth []MonthPoint `json:"sales_by_month"`
	ByAgent      []AgentPerf  `json:"by_agent"`
	ByBlock      []BlockPoint `json:"by_block"`
	ByType       []TypeStatus `json:"by_type"`
}

type Funnel struct {
	Total       int     `json:"total"`
	Reserved    int     `json:"reserved"` // live, hareket Rezerve
	Sale        int     `json:"sale"`     // live, hareket Satıldı
	Cancelled   int     `json:"cancelled"`
	Negotiation int     `json:"negotiation"` // live, neither
	CancelRate  float64 `json:"cancel_rate"`
}

// Collection mirrors Prism's own Tahsilat card: cancelled aktivite are out of
// every figure (their receipts are refunded), and "to collect" spans every live
// deal — Prism's "Kalan".
type Collection struct {
	PipelineValue float64 `json:"pipeline_value"` // Σ SatisFiyati, live deals
	Collected     float64 `json:"collected"`      // Σ receipts on live deals
	ToCollect     float64 `json:"to_collect"`     // PipelineValue − Collected
	Remaining     float64 `json:"remaining"`      // kept for older clients; same as ToCollect
	RefundsOwed   float64 `json:"refunds_owed"`   // Σ receipts on cancelled deals
	KaporaPaid    float64 `json:"kapora_paid"`
	CollectPct    float64 `json:"collect_pct"`
	WithKapora    int     `json:"deals_with_kapora"`
}

// MonthPoint is live deals by the month they were opened (dealOpened:
// RezerveTarihi, AktiviteTarihi only where that is unset).
type MonthPoint struct {
	Month    string  `json:"month"`
	Count    int     `json:"count"`
	Reserved int     `json:"reserved"`
	Sold     int     `json:"sold"`
	Revenue  float64 `json:"revenue"`
}

// AgentPerf is per Satış Personeli (T_Aktivite.SatisPersonelID1): anyone with
// at least one live deal. Cancelled is its own column and never lists somebody
// on its own; revenue/collected count live deals only.
type AgentPerf struct {
	Agent     string  `json:"agent"`
	Reserved  int     `json:"reserved"`
	Sales     int     `json:"sales"`
	Cancelled int     `json:"cancelled"`
	Revenue   float64 `json:"revenue"`   // Σ SatisFiyati of live deals
	Collected float64 `json:"collected"` // Σ T_AktiviteBakiye.Odedigi of live deals
}

type BlockPoint struct {
	Label    string  `json:"label"`
	Count    int     `json:"count"`
	Reserved int     `json:"reserved"`
	Sold     int     `json:"sold"`
	Revenue  float64 `json:"revenue"`
}

// TypeStatus is the stock by unit type, each unit in the state Prism last
// moved it to (T_Gayrimenkul.SonHareketTipiKisID_300).
type TypeStatus struct {
	Label     string `json:"label"`
	Total     int    `json:"total"`
	Available int    `json:"available"`
	Reserved  int    `json:"reserved"`
	Sold      int    `json:"sold"`
	Closed    int    `json:"closed"`
}

// LabeledMoney is kept for older clients of the stats endpoint.
type LabeledMoney struct {
	Label   string  `json:"label"`
	Count   int     `json:"count"`
	Revenue float64 `json:"revenue"`
}

func (s *Store) scanF(q string) float64 {
	rows, err := s.query(q)
	if err != nil {
		return 0
	}
	defer rows.Close()
	var v sql.NullFloat64
	if rows.Next() {
		rows.Scan(&v)
	}
	return v.Float64
}

func (s *Store) Reports() (Reports, error) {
	var r Reports
	const live = "a.HareketTipiKisID_300"

	// funnel — hareket tipi on live deals
	r.Funnel.Total = s.scanInt("SELECT COUNT(*) FROM T_Aktivite")
	r.Funnel.Cancelled = s.scanInt("SELECT COUNT(DISTINCT AktiviteID) FROM T_AktiviteIptal")
	r.Funnel.Sale = s.scanInt("SELECT COUNT(*) FROM T_Aktivite a WHERE " + live + "=" + hareketSatildi + " AND " + liveDeal)
	r.Funnel.Reserved = s.scanInt("SELECT COUNT(*) FROM T_Aktivite a WHERE " + live + "=" + hareketRezerve + " AND " + liveDeal)
	r.Funnel.Negotiation = r.Funnel.Total - r.Funnel.Sale - r.Funnel.Cancelled - r.Funnel.Reserved
	if r.Funnel.Negotiation < 0 {
		r.Funnel.Negotiation = 0
	}
	if r.Funnel.Total > 0 {
		r.Funnel.CancelRate = float64(r.Funnel.Cancelled) / float64(r.Funnel.Total) * 100
	}

	// collection — from the real receipts ledger (T_CariAlacak, işlem yeri 551),
	// the same source the deal page uses. Cancelled aktivite are excluded from
	// every figure, as Prism does: their receipts go back to the buyer. This is
	// what reconciles our Tahsilat/Kalan with Prism's dashboard to the lira.
	const liveReceipt = `NOT EXISTS (SELECT 1 FROM T_AktiviteIptal ip WHERE ip.AktiviteID = c.KayitID)`
	r.Collection.PipelineValue = s.scanF(`SELECT SUM(a.SatisFiyati) FROM T_Aktivite a WHERE ` + liveDeal)
	r.Collection.Collected = s.scanF(`SELECT SUM(c.AlacakTutari) FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551 AND ` + liveReceipt)
	r.Collection.RefundsOwed = s.scanF(`SELECT SUM(c.AlacakTutari) FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551 AND NOT ` + liveReceipt)
	r.Collection.KaporaPaid = s.scanF(`SELECT SUM(c.AlacakTutari) FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551 AND c.KaporaMi=1 AND ` + liveReceipt)
	r.Collection.WithKapora = s.scanInt(`SELECT COUNT(DISTINCT c.KayitID) FROM T_CariAlacak c WHERE c.CariIslemYeriKisID_310=551 AND c.KaporaMi=1 AND ` + liveReceipt)
	r.Collection.ToCollect = r.Collection.PipelineValue - r.Collection.Collected
	if r.Collection.ToCollect < 0 {
		r.Collection.ToCollect = 0
	}
	r.Collection.Remaining = r.Collection.ToCollect
	if r.Collection.PipelineValue > 0 {
		r.Collection.CollectPct = r.Collection.Collected / r.Collection.PipelineValue * 100
	}

	soldCase := "SUM(CASE WHEN " + live + "=" + hareketSatildi + " THEN 1 ELSE 0 END)"
	rezCase := "SUM(CASE WHEN " + live + "=" + hareketRezerve + " THEN 1 ELSE 0 END)"

	// live deals by the month they were opened
	if rows, err := s.query(`SELECT FORMAT(` + dealOpened + `,'yyyy-MM') m, COUNT(*), ` + rezCase + `, ` + soldCase + `, SUM(a.SatisFiyati)
		FROM T_Aktivite a WHERE YEAR(` + dealOpened + `)>1901 AND ` + liveDeal + `
		GROUP BY FORMAT(` + dealOpened + `,'yyyy-MM') ORDER BY m`); err == nil {
		for rows.Next() {
			var m sql.NullString
			var c, rez, sold int
			var rev sql.NullFloat64
			rows.Scan(&m, &c, &rez, &sold, &rev)
			r.SalesByMonth = append(r.SalesByMonth, MonthPoint{m.String, c, rez, sold, rev.Float64})
		}
		rows.Close()
	}

	// by agent — everyone holding a live deal; cancelled counted alongside
	if rows, err := s.query(`SELECT a.SatisPersonelID1,
			SUM(CASE WHEN ip.AktiviteID IS NULL AND ` + live + `=` + hareketSatildi + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL AND ` + live + `=` + hareketRezerve + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NOT NULL THEN 1 ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL THEN a.SatisFiyati ELSE 0 END),
			SUM(CASE WHEN ip.AktiviteID IS NULL THEN ISNULL(b.Odedigi,0) ELSE 0 END)
		FROM T_Aktivite a
		LEFT JOIN T_AktiviteIptal ip ON ip.AktiviteID=a.ID
		LEFT JOIN T_AktiviteBakiye b ON b.AktiviteID=a.ID
		GROUP BY a.SatisPersonelID1
		HAVING SUM(CASE WHEN ip.AktiviteID IS NULL THEN 1 ELSE 0 END) > 0
		ORDER BY 5 DESC`); err == nil {
		for rows.Next() {
			var id sql.NullString
			var sold, reserved, cancelled int
			var rev, col sql.NullFloat64
			rows.Scan(&id, &sold, &reserved, &cancelled, &rev, &col)
			lab := dec(s.per, id)
			if lab == "" {
				lab = "—"
			}
			r.ByAgent = append(r.ByAgent, AgentPerf{Agent: lab, Reserved: reserved, Sales: sold, Cancelled: cancelled, Revenue: rev.Float64, Collected: col.Float64})
		}
		rows.Close()
	}

	// live deals by block
	if rows, err := s.query(`SELECT COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo) blk, COUNT(*), ` + rezCase + `, ` + soldCase + `, SUM(a.SatisFiyati)
		FROM T_Aktivite a
		JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID
		JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		WHERE ` + liveDeal + `
		GROUP BY COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo) ORDER BY blk`); err == nil {
		for rows.Next() {
			var blk sql.NullString
			var c, rez, sold int
			var rev sql.NullFloat64
			rows.Scan(&blk, &c, &rez, &sold, &rev)
			r.ByBlock = append(r.ByBlock, BlockPoint{blk.String, c, rez, sold, rev.Float64})
		}
		rows.Close()
	}

	// stock by unit type, in the state Prism last moved each unit to
	if rows, err := s.query(`SELECT g.GayrimenkulTipDurumSecID_206, COUNT(*),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatilik + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketRezerve + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketSatildi + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN g.SonHareketTipiKisID_300=` + hareketKapali + ` THEN 1 ELSE 0 END)
		FROM T_Gayrimenkul g GROUP BY g.GayrimenkulTipDurumSecID_206 ORDER BY COUNT(*) DESC`); err == nil {
		for rows.Next() {
			var id sql.NullString
			var total, avail, rez, sold, closed int
			rows.Scan(&id, &total, &avail, &rez, &sold, &closed)
			lab := dec(s.dur, id)
			if lab == "" {
				lab = "—"
			}
			r.ByType = append(r.ByType, TypeStatus{lab, total, avail, rez, sold, closed})
		}
		rows.Close()
	}
	return r, nil
}
