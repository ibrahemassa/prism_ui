package db

import (
	"database/sql"
	"strings"
)

// Prism's own deal/unit state is the hareket tipi (Tn_Kis, ayıraç 300):
// Satılık 300 · Satışa Kapalı 301 · Rezerve 302 · Satıldı 303. Every count in
// this service reads that, never T_Aktivite.SozlesmeYapildiMi — the contract
// flag is bookkeeping the sales team does not work from.
const (
	hareketSatilik = "300"
	hareketKapali  = "301"
	hareketRezerve = "302"
	hareketSatildi = "303"
)

// liveDeal is the predicate that keeps cancelled aktivite out of a count.
const liveDeal = `NOT EXISTS (SELECT 1 FROM T_AktiviteIptal ip WHERE ip.AktiviteID = a.ID)`

// dealOpened is the date a deal counts from: RezerveTarihi, falling back to
// AktiviteTarihi only where the reservation date is unset (year 1900). Not
// AktiviteTarihi first: Prism re-stamps it whenever something happens to the
// deal (a reservation that later turns into a sale moves to the sale's date),
// so it reads as "last touched" and drags old deals into the current month.
const dealOpened = `CASE WHEN YEAR(a.RezerveTarihi)>1901 THEN a.RezerveTarihi ELSE a.AktiviteTarihi END`

type Stats struct {
	Customers     int            `json:"customers"`
	Units         int            `json:"units"`
	UnitsSold     int            `json:"units_sold"`
	UnitsReserved int            `json:"units_reserved"`
	UnitsFree     int            `json:"units_free"`
	Deals         int            `json:"deals"`
	DealsSale     int            `json:"deals_sale"`     // live, hareket Satıldı
	DealsReserved int            `json:"deals_reserved"` // live, hareket Rezerve
	Agencies      int            `json:"agencies"`
	Agents        int            `json:"agents"`
	SalesByAgent  []LabeledCount `json:"sales_by_agent"`
	LeadsByCat    []LabeledCount `json:"leads_by_category"`
}

type LabeledCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

func (s *Store) scanInt(q string, args ...any) int {
	rows, err := s.query(q, args...)
	if err != nil {
		return 0
	}
	defer rows.Close()
	n := 0
	if rows.Next() {
		rows.Scan(&n)
	}
	return n
}

func (s *Store) Stats() (Stats, error) {
	var st Stats
	st.Customers = s.scanInt("SELECT COUNT(*) FROM T_Musteri")
	st.Units = s.scanInt("SELECT COUNT(*) FROM T_Gayrimenkul")
	st.Deals = s.scanInt("SELECT COUNT(*) FROM T_Aktivite")
	st.DealsSale = s.scanInt("SELECT COUNT(*) FROM T_Aktivite a WHERE a.HareketTipiKisID_300=" + hareketSatildi + " AND " + liveDeal)
	st.DealsReserved = s.scanInt("SELECT COUNT(*) FROM T_Aktivite a WHERE a.HareketTipiKisID_300=" + hareketRezerve + " AND " + liveDeal)
	st.Agencies = s.scanInt("SELECT COUNT(*) FROM T_Acenta")
	st.Agents = s.scanInt("SELECT COUNT(*) FROM T_Personel WHERE AktifMi=1")
	st.UnitsSold = s.scanInt("SELECT COUNT(*) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketSatildi)
	st.UnitsReserved = s.scanInt("SELECT COUNT(*) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketRezerve)
	st.UnitsFree = s.scanInt("SELECT COUNT(*) FROM T_Gayrimenkul WHERE SonHareketTipiKisID_300=" + hareketSatilik)

	// live deals by agent (top 8)
	rows, err := s.query(`SELECT TOP 8 a.SatisPersonelID1, COUNT(*) c
		FROM T_Aktivite a WHERE ` + liveDeal + `
		GROUP BY a.SatisPersonelID1 ORDER BY c DESC`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id sql.NullString
			var c int
			rows.Scan(&id, &c)
			lab := dec(s.per, id)
			if lab == "" {
				lab = "—"
			}
			st.SalesByAgent = append(st.SalesByAgent, LabeledCount{lab, c})
		}
	}
	// leads by category
	r2, err := s.query(`SELECT MusteriKategoriDurumSecID_90, COUNT(*) c
		FROM T_Musteri GROUP BY MusteriKategoriDurumSecID_90 ORDER BY c DESC`)
	if err == nil {
		defer r2.Close()
		for r2.Next() {
			var id sql.NullString
			var c int
			r2.Scan(&id, &c)
			lab := dec(s.dur, id)
			if lab == "" {
				lab = "Belirtilmemiş"
			}
			st.LeadsByCat = append(st.LeadsByCat, LabeledCount{lab, c})
		}
	}
	return st, nil
}

type Lead struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Category  string `json:"category"`
	Agent     string `json:"agent"`
	Office    string `json:"office"`
	Agency    string `json:"agency"`
	Result    string `json:"result"`
	City      string `json:"city"`
	CreatedAt string `json:"created_at"`
}

func (s *Store) Leads(search string, limit, offset int) ([]Lead, int, error) {
	where := ""
	args := []any{}
	if strings.TrimSpace(search) != "" {
		where = "WHERE m.MusteriAdiSoyadi LIKE @p1 OR di.CepTel1 LIKE @p1"
		args = append(args, "%"+search+"%")
	}
	total := s.scanInt("SELECT COUNT(*) FROM T_Musteri m LEFT JOIN T_MusteriDetayIletisim di ON di.MusteriID=m.ID "+where, args...)

	q := `SELECT m.ID, m.MusteriAdiSoyadi, di.CepTel1, di.EMail,
			m.MusteriKategoriDurumSecID_90, m.PersonelID_1, m.OfisID, m.AcentaID,
			m.MusteriSonucID, di.AdresSehir, m.SistemOlusturmaTarihi
		FROM T_Musteri m
		LEFT JOIN T_MusteriDetayIletisim di ON di.MusteriID=m.ID ` + where + `
		ORDER BY m.ID DESC OFFSET @off ROWS FETCH NEXT @lim ROWS ONLY`
	args = append(args, sql.Named("off", offset), sql.Named("lim", limit))
	rows, err := s.query(q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Lead{}
	for rows.Next() {
		var id, name, phone, email, cat, per, ofi, ace, son, city, created sql.NullString
		if err := rows.Scan(&id, &name, &phone, &email, &cat, &per, &ofi, &ace, &son, &city, &created); err != nil {
			return nil, 0, err
		}
		out = append(out, Lead{
			ID: id.String, Name: name.String, Phone: phone.String, Email: email.String,
			Category: dec(s.dur, cat), Agent: dec(s.per, per), Office: dec(s.ofi, ofi),
			Agency: dec(s.ace, ace), Result: dec(s.son, son), City: city.String,
			CreatedAt: created.String,
		})
	}
	return out, total, rows.Err()
}

type Unit struct {
	ID       string  `json:"id"`
	Project  string  `json:"project"`
	Block    string  `json:"block"`
	Number   string  `json:"number"`
	Type     string  `json:"type"`
	GrossM2  float64 `json:"gross_m2"`
	NetM2    float64 `json:"net_m2"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
	View     string  `json:"view"`
	Status   string  `json:"status"`
}

func (s *Store) Units() ([]Unit, error) {
	q := `SELECT g.ID, g.ProjeID, COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo), g.DaireNo, g.GayrimenkulTipDurumSecID_206,
			g.BrutM2, g.NetM2, f.SatisFiyati01, f.SatisFiyati01DovizID,
			g.ManzaraDurumSecID_2, g.SonHareketTipiKisID_300
		FROM T_Gayrimenkul g
		LEFT JOIN T_GayrimenkulFiyatListesi1 f ON f.GayrimenkulID=g.ID
		ORDER BY g.BolumAdi, g.DaireNo`
	rows, err := s.query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Unit{}
	for rows.Next() {
		var id, proj, block, num, typ, view, dov, hareket sql.NullString
		var gross, net, price sql.NullFloat64
		if err := rows.Scan(&id, &proj, &block, &num, &typ, &gross, &net, &price, &dov, &view, &hareket); err != nil {
			return nil, err
		}
		status := "available"
		switch hareket.String {
		case hareketSatildi:
			status = "sold"
		case hareketRezerve:
			status = "reserved"
		case hareketKapali:
			status = "unavailable"
		}
		pr := dec(s.proj, proj)
		if pr == "" {
			pr = proj.String
		}
		out = append(out, Unit{
			ID: id.String, Project: pr, Block: block.String, Number: num.String,
			Type: dec(s.dur, typ), GrossM2: gross.Float64, NetM2: net.Float64,
			Price: price.Float64, Currency: s.cur(dov), View: dec(s.dur, view), Status: status,
		})
	}
	return out, rows.Err()
}

type Deal struct {
	ID          string  `json:"id"`
	CustomerID  string  `json:"customer_id"`
	Customer    string  `json:"customer"`
	UnitID      string  `json:"unit_id"`
	Block       string  `json:"block"`
	UnitNumber  string  `json:"unit_number"`
	UnitType    string  `json:"unit_type"`
	Stage       string  `json:"stage"`
	SalePrice   float64 `json:"sale_price"`
	DownPayment float64 `json:"down_payment"`
	Currency    string  `json:"currency"`
	Agent       string  `json:"agent"`
	Agency      string  `json:"agency"`
	ReservedAt  string  `json:"reserved_at"`
	ContractAt  string  `json:"contract_at"`
	ActivityAt  string  `json:"activity_at"`
	CancelledAt string  `json:"cancelled_at"` // T_AktiviteIptal.IptalTarihi, empty unless cancelled
	// analytics from T_AktiviteBakiye (per-deal balance)
	KaporaPaid float64 `json:"kapora_paid"`
	PaidTotal  float64 `json:"paid_total"`
	Remaining  float64 `json:"remaining"`
	PaidPct    float64 `json:"paid_pct"`
}

const dealSelect = `SELECT a.ID, a.AnaMusteriID, m.MusteriAdiSoyadi,
		ag.GayrimenkulID, COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo), g.DaireNo, g.GayrimenkulTipDurumSecID_206,
		a.HareketTipiKisID_300, a.RezerveTarihi, ip.AktiviteID, ip.IptalTarihi,
		a.SatisFiyati, a.PlanPesinatToplam, a.SatisFiyatiDovizID,
		a.SatisPersonelID1, a.Araci1AcentaID, a.RezerveTarihi, a.SozlesmeTarihi, a.AktiviteTarihi,
		b.DepozitoOdenen, b.Odedigi, b.KalanBorcu, b.OdedigiYuzde
	FROM T_Aktivite a
	LEFT JOIN T_Musteri m ON m.ID=a.AnaMusteriID
	LEFT JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID
	LEFT JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
	LEFT JOIN T_AktiviteIptal ip ON ip.AktiviteID=a.ID
	LEFT JOIN T_AktiviteBakiye b ON b.AktiviteID=a.ID `

func (s *Store) scanDeal(rows *sql.Rows) (Deal, error) {
	var id, custID, cust, unitID, block, num, typ, hareket, rez2, iptal, iptalAt, per, ace, rez, contract, act, dov sql.NullString
	var price, down, kapora, paid, remaining, pct sql.NullFloat64
	if err := rows.Scan(&id, &custID, &cust, &unitID, &block, &num, &typ, &hareket, &rez2, &iptal, &iptalAt,
		&price, &down, &dov, &per, &ace, &rez, &contract, &act, &kapora, &paid, &remaining, &pct); err != nil {
		return Deal{}, err
	}
	stage := "negotiation"
	switch {
	case iptal.Valid && iptal.String != "":
		stage = "cancelled"
	case hareket.String == hareketSatildi:
		stage = "sale"
	case hareket.String == hareketRezerve:
		stage = "reserved"
	}
	return Deal{
		ID: id.String, CustomerID: custID.String, Customer: cust.String,
		UnitID: unitID.String, Block: block.String, UnitNumber: num.String, UnitType: dec(s.dur, typ),
		Stage: stage, SalePrice: price.Float64, DownPayment: down.Float64,
		Currency: s.cur(dov), Agent: dec(s.per, per), Agency: dec(s.ace, ace),
		ReservedAt: rez.String, ContractAt: contract.String, ActivityAt: act.String, CancelledAt: iptalAt.String,
		KaporaPaid: kapora.Float64, PaidTotal: paid.Float64, Remaining: remaining.Float64, PaidPct: pct.Float64,
	}, nil
}

func (s *Store) Deals() ([]Deal, error) {
	rows, err := s.query(dealSelect + "ORDER BY a.ID DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Deal{}
	for rows.Next() {
		d, err := s.scanDeal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) DealsForCustomer(mid string) ([]Deal, error) {
	rows, err := s.query(dealSelect+"WHERE a.AnaMusteriID=@p1 ORDER BY a.ID DESC", mid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Deal{}
	for rows.Next() {
		d, err := s.scanDeal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

type Agent struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Active bool   `json:"active"`
}

func (s *Store) Agents() ([]Agent, error) {
	rows, err := s.query(`SELECT p.ID, p.PersonelAdi, k.KullaniciMail, p.AktifMi
		FROM T_Personel p LEFT JOIN T_Kullanici k ON k.PersonelID=p.ID ORDER BY p.PersonelAdi`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Agent{}
	for rows.Next() {
		var id, name, email sql.NullString
		var active sql.NullBool
		rows.Scan(&id, &name, &email, &active)
		out = append(out, Agent{id.String, name.String, email.String, active.Bool})
	}
	return out, rows.Err()
}

type Agency struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Phone string  `json:"phone"`
	Email string  `json:"email"`
	City  string  `json:"city"`
	Rate  float64 `json:"rate"`
}

func (s *Store) Agencies() ([]Agency, error) {
	rows, err := s.query(`SELECT a.ID, a.AcentaAdi, d.CepTel1, d.Email1, d.AdresSehir, k.Komisyon1
		FROM T_Acenta a
		LEFT JOIN T_AcentaDetay d ON d.AcentaID=a.ID
		LEFT JOIN T_AcentaKomisyon k ON k.AcentaID=a.ID
		ORDER BY a.AcentaAdi`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Agency{}
	for rows.Next() {
		var id, name, phone, email, city sql.NullString
		var rate sql.NullFloat64
		rows.Scan(&id, &name, &phone, &email, &city, &rate)
		out = append(out, Agency{id.String, name.String, phone.String, email.String, city.String, rate.Float64})
	}
	return out, rows.Err()
}
