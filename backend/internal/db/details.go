package db

import (
	"database/sql"
	"strings"
)

// ---------------- Lead detail ----------------

type Activity struct {
	Date     string `json:"date"`
	Agent    string `json:"agent"`
	Channel  string `json:"channel"`
	Notes    string `json:"notes"`
	Result   string `json:"result"`
	Reminder string `json:"reminder"`
	HasAudio bool   `json:"has_audio"`
}

type LeadDetail struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Phone       string     `json:"phone"`
	Phone2      string     `json:"phone2"`
	Email       string     `json:"email"`
	Category    string     `json:"category"`
	Agent       string     `json:"agent"`
	Office      string     `json:"office"`
	Agency      string     `json:"agency"`
	Result      string     `json:"result"`
	NationalID  string     `json:"national_id"`
	Birthday    string     `json:"birthday"`
	Nationality string     `json:"nationality"`
	Mother      string     `json:"mother"`
	Father      string     `json:"father"`
	Gender      string     `json:"gender"`
	City        string     `json:"city"`
	District    string     `json:"district"`
	Address     string     `json:"address"`
	Occupation  string     `json:"occupation"`
	Source      string     `json:"source"`
	Purpose     string     `json:"purpose"`
	Income      string     `json:"income"`
	CreatedAt   string     `json:"created_at"`
	Deals       []Deal     `json:"deals"`
	Activities  []Activity `json:"activities"`
	CRM         *CRMInfo   `json:"crm,omitempty"`
}

func (s *Store) LeadDetail(id string) (*LeadDetail, error) {
	q := `SELECT m.ID, m.MusteriAdiSoyadi, di.CepTel1, di.CepTel2, di.EMail,
			m.MusteriKategoriDurumSecID_90, m.PersonelID_1, m.OfisID, m.AcentaID, m.MusteriSonucID,
			m.TcKimlikNo, k.DogumTarihi, k.UyruguUlkeID, k.AnaAdi, k.BabaAdi, k.CisyetKisID_13,
			di.AdresSehir, di.AdresIlce, di.Adres, m.MeslekDurumSecID_205, m.SistemOlusturmaTarihi
		FROM T_Musteri m
		LEFT JOIN T_MusteriDetayIletisim di ON di.MusteriID=m.ID
		LEFT JOIN T_MusteriDetayKimlik  k  ON k.MusteriID=m.ID
		WHERE m.ID=@p1`
	row := struct {
		id, name, p1, p2, email, cat, per, ofi, ace, son, tc, dob, ulke, mom, dad, gen, city, ilce, addr, meslek, created sql.NullString
	}{}
	rows, err := s.query(q, id)
	if err != nil {
		return nil, err
	}
	found := false
	if rows.Next() {
		found = true
		rows.Scan(&row.id, &row.name, &row.p1, &row.p2, &row.email, &row.cat, &row.per, &row.ofi, &row.ace, &row.son,
			&row.tc, &row.dob, &row.ulke, &row.mom, &row.dad, &row.gen, &row.city, &row.ilce, &row.addr, &row.meslek, &row.created)
	}
	rows.Close()
	if !found {
		return nil, sql.ErrNoRows
	}

	d := &LeadDetail{
		ID: row.id.String, Name: row.name.String, Phone: row.p1.String, Phone2: row.p2.String, Email: row.email.String,
		Category: dec(s.dur, row.cat), Agent: dec(s.per, row.per), Office: dec(s.ofi, row.ofi),
		Agency: dec(s.ace, row.ace), Result: dec(s.son, row.son), NationalID: row.tc.String,
		Birthday: row.dob.String, Nationality: dec(s.ulke, row.ulke), Mother: row.mom.String, Father: row.dad.String,
		Gender: dec(s.kis, row.gen), City: row.city.String, District: row.ilce.String, Address: row.addr.String,
		Occupation: dec(s.dur, row.meslek), CreatedAt: row.created.String,
	}

	// latest intake row (source / purpose / income)
	ir, err := s.query(`SELECT TOP 1 MusteriKaynagiID_1, SatinAlmaAmaciDurumSecID_201, AylikGeliriDurumSecID_32
		FROM T_MusteriDetay WHERE MusteriID=@p1 ORDER BY ID DESC`, id)
	if err == nil {
		if ir.Next() {
			var src, pur, inc sql.NullString
			ir.Scan(&src, &pur, &inc)
			d.Source = dec(s.kis, src)
			d.Purpose = dec(s.dur, pur)
			d.Income = dec(s.dur, inc)
		}
		ir.Close()
	}

	d.Deals, _ = s.DealsForCustomer(id)

	// activity / call log
	ar, err := s.query(`SELECT Tarih, PersonelID, MusteriHaberlesmeTipiID, Notlar,
			MusteriSonucuID, HatirlatmaTarihi, SesDosyaAdi
		FROM T_MusteriIletisim WHERE MusteriID=@p1 ORDER BY Tarih DESC`, id)
	if err == nil {
		for ar.Next() {
			var t, per, ch, notes, res, rem, audio sql.NullString
			ar.Scan(&t, &per, &ch, &notes, &res, &rem, &audio)
			d.Activities = append(d.Activities, Activity{
				Date: t.String, Agent: dec(s.per, per), Channel: dec(s.kis, ch),
				Notes: notes.String, Result: dec(s.dur, res),
				Reminder: rem.String, HasAudio: strings.TrimSpace(audio.String) != "",
			})
		}
		ar.Close()
	}
	if d.Deals == nil {
		d.Deals = []Deal{}
	}
	if d.Activities == nil {
		d.Activities = []Activity{}
	}
	// live CRM enrichment (source + stage history), if the CRM is linked
	d.CRM = s.crm.enrich(d.Phone)
	return d, nil
}

// ---------------- Deal detail ----------------

type PaymentRow struct {
	ID       int64   `json:"id"`
	Date     string  `json:"date"`
	Amount   float64 `json:"amount"`
	Paid     float64 `json:"paid"`
	Currency string  `json:"currency"`
	Kind     string  `json:"kind"` // Peşinat / Vadeli Ödeme / Banka Kredisi
	Type     string  `json:"type"` // Senet / Banka Havalesi / …
	Interim  bool    `json:"interim"`
	Note     string  `json:"note"`
}

type ReceiptRow struct {
	ID       int64   `json:"id"`
	Date     string  `json:"date"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Type     string  `json:"type"`
	Deposit  bool    `json:"deposit"`
	Account  string  `json:"account"`
	Note     string  `json:"note"`
}

type DealDetail struct {
	Deal
	Project        string       `json:"project"`
	UnitGrossM2    float64      `json:"unit_gross_m2"`
	UnitNetM2      float64      `json:"unit_net_m2"`
	UnitView       string       `json:"unit_view"`
	UnitBedrooms   string       `json:"unit_bedrooms"`
	ListPrice      float64      `json:"list_price"`
	FinancedAmount float64      `json:"financed_amount"`
	CommissionPct  float64      `json:"commission_pct"`
	CommissionAmt  float64      `json:"commission_amount"`
	CancelledAt    string       `json:"cancelled_at"`
	CancelReason   string       `json:"cancel_reason"`
	PaymentPlan    []PaymentRow `json:"payment_plan"`
	PlanTotal      float64      `json:"plan_total"`
	Receipts       []ReceiptRow `json:"receipts"`
	PaidTotal      float64      `json:"paid_total"`
}

func (s *Store) DealDetail(id string) (*DealDetail, error) {
	rows, err := s.query(dealSelect+"WHERE a.ID=@p1", id)
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		rows.Close()
		return nil, sql.ErrNoRows
	}
	base, err := s.scanDeal(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}
	d := &DealDetail{Deal: base}

	// unit extras + list price + term
	ur, err := s.query(`SELECT g.ProjeID, g.BrutM2, g.NetM2, g.ManzaraDurumSecID_2, g.OdaSayisi,
			ag.ListeFiyati, a.PlanVadeli
		FROM T_Aktivite a
		LEFT JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID
		LEFT JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		WHERE a.ID=@p1`, id)
	if err == nil {
		if ur.Next() {
			var proj, view, oda sql.NullString
			var gross, net, list, financed sql.NullFloat64
			ur.Scan(&proj, &gross, &net, &view, &oda, &list, &financed)
			p := dec(s.proj, proj)
			if p == "" {
				p = proj.String
			}
			d.Project = p
			d.UnitGrossM2 = gross.Float64
			d.UnitNetM2 = net.Float64
			d.UnitView = dec(s.dur, view)
			d.UnitBedrooms = oda.String
			d.ListPrice = list.Float64
			d.FinancedAmount = financed.Float64
		}
		ur.Close()
	}

	// commission
	cr, err := s.query(`SELECT TOP 1 AcentaninKomisyonYuzdesi, AcentaKomisyonu FROM T_AktivitePrim WHERE AktiviteID=@p1`, id)
	if err == nil {
		if cr.Next() {
			var pct, amt sql.NullFloat64
			cr.Scan(&pct, &amt)
			d.CommissionPct = pct.Float64
			d.CommissionAmt = amt.Float64
		}
		cr.Close()
	}

	// cancellation
	if base.Stage == "cancelled" {
		xr, err := s.query(`SELECT TOP 1 IptalTarihi, IptalAciklama FROM T_AktiviteIptal WHERE AktiviteID=@p1`, id)
		if err == nil {
			if xr.Next() {
				var t, r sql.NullString
				xr.Scan(&t, &r)
				d.CancelledAt = t.String
				d.CancelReason = r.String
			}
			xr.Close()
		}
	}

	// payment plan — the real debit ledger (T_CariBorc), not the audit log.
	// CariIslemYeriKisID_310=550 is the deal's installment plan; Prism keeps 10
	// empty peşinat slots + an empty banka-kredisi slot, so skip BorcTutari=0.
	// Paid-per-installment comes from the receipt→installment links.
	pr, err := s.query(`SELECT b.ID, b.Tarih, b.BorcTutari, b.BorcDovizID,
			b.TipKisID_304, b.OdemeTipiKisID_302, d.AraOdemeMi, d.Aciklama,
			(SELECT ISNULL(SUM(k.KapatilanTutar),0) FROM T_CariAlacakBorcKapatma k WHERE k.CariBorcID=b.ID)
		FROM T_CariBorc b
		LEFT JOIN T_CariBorcDetay d ON d.CariBorcID=b.ID
		WHERE b.KayitID=@p1 AND b.CariIslemYeriKisID_310=550 AND b.BorcTutari<>0
		ORDER BY b.Tarih, b.PesinatNo`, id)
	if err == nil {
		for pr.Next() {
			var bid sql.NullInt64
			var t, dov, kind, typ, note sql.NullString
			var amt, paid sql.NullFloat64
			var interim sql.NullBool
			pr.Scan(&bid, &t, &amt, &dov, &kind, &typ, &interim, &note, &paid)
			d.PaymentPlan = append(d.PaymentPlan, PaymentRow{
				ID: bid.Int64, Date: t.String, Amount: amt.Float64, Paid: paid.Float64,
				Currency: s.cur(dov), Kind: dec(s.kis, kind), Type: dec(s.kis, typ),
				Interim: interim.Bool, Note: note.String,
			})
			d.PlanTotal += amt.Float64
		}
		pr.Close()
	}
	if d.PaymentPlan == nil {
		d.PaymentPlan = []PaymentRow{}
	}

	// receipts — the credit ledger (T_CariAlacak), CariIslemYeriKisID_310=551.
	rc, err := s.query(`SELECT a.ID, a.Tarih, a.AlacakTutari, a.AlacakDovizID,
			a.OdemeTipiKisID_302, a.KaporaMi, h.HesapAciklama, ad.Aciklama
		FROM T_CariAlacak a
		LEFT JOIN T_HesapKartlari h ON h.ID=a.HesapKartlariID
		LEFT JOIN T_CariAlacakDetay ad ON ad.CariAlacakID=a.ID
		WHERE a.KayitID=@p1 AND a.CariIslemYeriKisID_310=551
		ORDER BY a.Tarih`, id)
	if err == nil {
		for rc.Next() {
			var aid sql.NullInt64
			var t, dov, typ, acct, note sql.NullString
			var amt sql.NullFloat64
			var kapora sql.NullBool
			rc.Scan(&aid, &t, &amt, &dov, &typ, &kapora, &acct, &note)
			d.Receipts = append(d.Receipts, ReceiptRow{
				ID: aid.Int64, Date: t.String, Amount: amt.Float64, Currency: s.cur(dov),
				Type: dec(s.kis, typ), Deposit: kapora.Bool, Account: acct.String, Note: note.String,
			})
			d.PaidTotal += amt.Float64
		}
		rc.Close()
	}
	if d.Receipts == nil {
		d.Receipts = []ReceiptRow{}
	}
	return d, nil
}
