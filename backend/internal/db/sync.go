package db

import (
	"database/sql"
	"strings"
)

// The CRM ingests what Prism's own exports carry, in the exports' own column
// vocabulary — so the rows below are keyed by the Turkish headers the CRM's
// importers already read (app/core/prism_deals.py, prism_cancellations.py),
// and a row here is indistinguishable from a row off the sheet. Nothing new
// to parse on the far side; this service only replaces the Excel file.
//
// Dates are ISO days ("2026-09-16") or ISO local timestamps — Prism stores the
// office's own wall clock and the CRM converts to UTC on arrival.

// ---------------- stock report (one row per unit) ----------------

const (
	hStockName       = "Müşteri Adı Soyadı"
	hStockAgency     = "Acenta"
	hStockMarketer   = "Müşteri Pazarlama Personeli"
	hStockSeller     = "Satış Personeli 1"
	hStockBlock      = "Blok No"
	hStockDoor       = "Daire No"
	hStockFloor      = "Kat Adı"
	hStockStatus     = "Aktivite Durumu"
	hStockDetail     = "Durum Detay"
	hStockActivityNo = "Aktivite No"
	hStockContracted = "Sözleşme Yapıldı Mı ?"
	hStockContractAt = "Sözleşme Tarihi"
	hStockReservedAt = "Rezerve Tarihi"
	hStockActivityAt = "Aktivite Tarihi"
	hStockPhone      = "Cep Tel 1"
	hStockEmail      = "Email"
	hStockListPrice  = "Liste Fiyatı"
	hStockSalePrice  = "Aktivite Satis Fiyati"
	hStockCurrency   = "Aktivite Satis Fiyati Doviz"
	hStockDeposit    = "Plan Peşinat"
	hStockInstalment = "Plan Vadeli"
	hStockPayment    = "Ödeme Biçimi"
	hStockSource     = "Müşteri Kaynağı 1"
	// The sale's own paperwork, as Prism's unit report heads it: its
	// reference (e.g. "AB-123", T_AktiviteDetay.SatisNo) and whether the DBS
	// (the bank's direct debit) is approved.
	hStockContractNo = "Sözleşme No"
	hStockDBS        = "DBS Onaylandı Mı ?"
)

// StockRows is the stock report: every unit, and the live activity on the
// taken ones. A unit whose last activity was cancelled reads as available with
// nobody on it, which is what Prism's own export does.
func (s *Store) StockRows() ([]map[string]any, error) {
	q := `SELECT g.ID, COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo), g.DaireNo, k.KatAdi,
			g.SonHareketTipiKisID_300, f.SatisFiyati01, a.ListeFiyatiKur,
			a.ID, a.AktiviteStatuKisID_308, a.SozlesmeYapildiMi, a.SozlesmeTarihi, a.RezerveTarihi, a.AktiviteTarihi,
			a.SatisFiyati, a.SatisFiyatiDovizID, a.PlanPesinatToplam, a.PlanVadeli, a.OdemeTahutuKisID_301,
			a.SatisPersonelID1, a.Araci1AcentaID, a.MusteriKaynagiID_1,
			m.MusteriAdiSoyadi, m.PersonelID_1, di.CepTel1, di.EMail,
			ad.SatisNo, ad.DbsOnaylandiMi
		FROM T_Gayrimenkul g
		LEFT JOIN T_GayrimenkulKat k ON k.ID=g.GayrimenkulKatID
		LEFT JOIN T_GayrimenkulFiyatListesi1 f ON f.GayrimenkulID=g.ID
		LEFT JOIN T_Aktivite a ON a.ID=g.SonAktiviteID
			AND g.SonHareketTipiKisID_300 IN (` + hareketRezerve + `,` + hareketSatildi + `)
			AND NOT EXISTS (SELECT 1 FROM T_AktiviteIptal ip WHERE ip.AktiviteID=a.ID)
		LEFT JOIN T_Musteri m ON m.ID=a.AnaMusteriID
		LEFT JOIN T_MusteriDetayIletisim di ON di.MusteriID=m.ID
		LEFT JOIN T_AktiviteDetay ad ON ad.ID=(SELECT MAX(ID) FROM T_AktiviteDetay WHERE AktiviteID=a.ID)
		ORDER BY g.BlokNo, g.DaireNo`
	rows, err := s.query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var unitID, block, door, floor, hareket, actID, statu, dov, payment, seller, agency, source, name, marketer, phone, email, contractNo sql.NullString
		var listPrice, listAtSale, salePrice, deposit, instalment sql.NullFloat64
		var contracted, dbs sql.NullBool
		var contractAt, reservedAt, activityAt sql.NullTime
		if err := rows.Scan(&unitID, &block, &door, &floor, &hareket, &listPrice, &listAtSale,
			&actID, &statu, &contracted, &contractAt, &reservedAt, &activityAt,
			&salePrice, &dov, &deposit, &instalment, &payment,
			&seller, &agency, &source, &name, &marketer, &phone, &email, &contractNo, &dbs); err != nil {
			return nil, err
		}
		status := dec(s.kis, hareket)
		if hareket.String == "" {
			status = "Satılık"
		}
		row := map[string]any{
			hStockBlock:     block.String,
			hStockDoor:      door.String,
			hStockFloor:     floor.String,
			hStockStatus:    status,
			hStockListPrice: listPrice.Float64,
			"prism_unit_id": unitID.String,
		}
		if actID.Valid {
			taken := hareket.String == hareketRezerve || hareket.String == hareketSatildi
			// The list price the sale was written against, not today's
			// price list: the developer raises the list every few weeks and
			// a deal's list price is a fact about the day it was made.
			if listAtSale.Valid && listAtSale.Float64 > 0 {
				row[hStockListPrice] = listAtSale.Float64
			}
			row[hStockActivityNo] = actID.String
			row[hStockDetail] = dec(s.kis, statu)
			if contracted.Bool || contractAt.Valid {
				row[hStockContracted] = "Sözleşme Yapıldı"
			} else {
				row[hStockContracted] = "Sözleşme Yapılmadı"
			}
			row[hStockContractAt] = isoDay(contractAt)
			row[hStockReservedAt] = isoDay(reservedAt)
			row[hStockActivityAt] = isoDay(activityAt)
			row[hStockSalePrice] = salePrice.Float64
			row[hStockCurrency] = s.cur(dov)
			row[hStockDeposit] = deposit.Float64
			row[hStockInstalment] = instalment.Float64
			row[hStockPayment] = dec(s.kis, payment)
			row[hStockSeller] = dec(s.per, seller)
			row[hStockMarketer] = dec(s.per, marketer)
			row[hStockAgency] = dec(s.ace, agency)
			row[hStockSource] = dec(s.kay, source)
			if taken {
				row[hStockName] = strings.TrimSpace(name.String)
				row[hStockPhone] = strings.TrimSpace(phone.String)
				row[hStockEmail] = strings.TrimSpace(email.String)
				row[hStockContractNo] = strings.TrimSpace(contractNo.String)
				// Absent rather than false when Prism has no detail row:
				// the CRM reads a missing key as "Prism does not say".
				if dbs.Valid {
					row[hStockDBS] = dbs.Bool
				}
			}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ---------------- cancellations report ----------------

const (
	hCancelBlock      = "Blok No"
	hCancelDoor       = "Daire No"
	hCancelName       = "Müşteri"
	hCancelNames      = "Toplu Müşteri İsimleri"
	hCancelPhone      = "Cep Tel 1"
	hCancelEmail      = "Email"
	hCancelStatus     = "Durumu"
	hCancelDetail     = "Durum Detay"
	hCancelActivityAt = "Tarih"
	hCancelContractAt = "Sözleşme Tarihi"
	hCancelContracted = "Sözleşme Yapıldı Mı ?"
	hCancelAt         = "İptal Tarihi"
	hCancelReason     = "İptal Nedeni"
	hCancelType       = "İptal Türü"
	hCancelNote       = "İptal Açıklama"
	hCancelBy         = "İptal Eden Personel"
	hCancelContractNo = "Sözleşme No"
	// The sale's own reference (e.g. "AB-123", T_AktiviteDetay.SatisNo). Prism's
	// iptal export calls the activity number "Sözleşme No", so the LB number
	// travels under its Prism field name instead; blank when Prism has none.
	hCancelSaleNo    = "Satış No"
	hCancelSeller    = "Satış Personeli"
	hCancelAgency    = "Acenta 1"
	hCancelOffice    = "Satış Ofisi"
	hCancelDeposit   = "Tahsil Edilen Kapora"
	hCancelDownPay   = "Tahsil Edilen Peşinat"
	hCancelCollected = "Toplam Tahsilat"
)

// CancellationRows is the iptal list: one row per activity that was called
// off, newest first.
//
// Each row also carries what the sale was written at — the price, its
// currency and the list price it was made against, under the stock report's
// own headers. Prism's iptal export has no such column, but the CRM values a
// cancelled sale by it, and without it every cancellation reads as a deal
// with no price at all.
func (s *Store) CancellationRows() ([]map[string]any, error) {
	q := `SELECT ip.AktiviteID, COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo), g.DaireNo,
			m.MusteriAdiSoyadi, a.MusteriAdSoyadlariToplu, di.CepTel1, di.EMail,
			a.HareketTipiKisID_300, a.AktiviteStatuKisID_308, a.AktiviteTarihi, a.SozlesmeTarihi, a.SozlesmeYapildiMi,
			ip.IptalTarihi, ip.IptalNedenIDKisID_6, ip.IptalTuruDurumSec_ID_49, ip.IptalAciklama, ip.IptalEdenPersonel,
			a.SatisPersonelID1, a.Araci1AcentaID, a.OfisID,
			b.DepozitoOdenen, b.TahsilatPesinat, b.Odedigi,
			a.SatisFiyati, a.SatisFiyatiDovizID, a.ListeFiyatiKur, ad.SatisNo
		FROM T_AktiviteIptal ip
		JOIN T_Aktivite a ON a.ID=ip.AktiviteID
		LEFT JOIN T_AktiviteDetay ad ON ad.AktiviteID=a.ID
		LEFT JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID
		LEFT JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		LEFT JOIN T_Musteri m ON m.ID=a.AnaMusteriID
		LEFT JOIN T_MusteriDetayIletisim di ON di.MusteriID=m.ID
		LEFT JOIN T_AktiviteBakiye b ON b.AktiviteID=a.ID
		ORDER BY ip.IptalTarihi DESC`
	rows, err := s.query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var actID, block, door, name, names, phone, email, hareket, statu, reason, typ, note, by, seller, agency, office, dov, saleNo sql.NullString
		var activityAt, contractAt, cancelAt sql.NullTime
		var contracted sql.NullBool
		var deposit, downPay, collected, salePrice, listAtSale sql.NullFloat64
		if err := rows.Scan(&actID, &block, &door, &name, &names, &phone, &email,
			&hareket, &statu, &activityAt, &contractAt, &contracted,
			&cancelAt, &reason, &typ, &note, &by, &seller, &agency, &office,
			&deposit, &downPay, &collected, &salePrice, &dov, &listAtSale, &saleNo); err != nil {
			return nil, err
		}
		contractedText := "Sözleşme Yapılmadı"
		if contracted.Bool || contractAt.Valid {
			contractedText = "Sözleşme Yapıldı"
		}
		out = append(out, map[string]any{
			hCancelBlock:      block.String,
			hCancelDoor:       door.String,
			hCancelName:       strings.TrimSpace(name.String),
			hCancelNames:      strings.TrimSpace(names.String),
			hCancelPhone:      strings.TrimSpace(phone.String),
			hCancelEmail:      strings.TrimSpace(email.String),
			hCancelStatus:     dec(s.kis, hareket),
			hCancelDetail:     dec(s.kis, statu),
			hCancelActivityAt: isoDay(activityAt),
			hCancelContractAt: isoDay(contractAt),
			hCancelContracted: contractedText,
			hCancelAt:         isoDay(cancelAt),
			hCancelReason:     dec(s.dur, reason),
			hCancelType:       dec(s.dur, typ),
			hCancelNote:       strings.TrimSpace(note.String),
			hCancelBy:         strings.TrimSpace(by.String),
			hCancelContractNo: actID.String,
			hCancelSaleNo:     strings.TrimSpace(saleNo.String),
			hCancelSeller:     dec(s.per, seller),
			hCancelAgency:     dec(s.ace, agency),
			hCancelOffice:     dec(s.ofi, office),
			hCancelDeposit:    deposit.Float64,
			hCancelDownPay:    downPay.Float64,
			hCancelCollected:  collected.Float64,
			hStockSalePrice:   salePrice.Float64,
			hStockCurrency:    s.cur(dov),
			hStockListPrice:   listAtSale.Float64,
		})
	}
	return out, rows.Err()
}

// ---------------- people ----------------

// SyncLead is one Prism customer with everything the CRM has a column for.
type SyncLead struct {
	PrismID          string `json:"prism_id"`
	Name             string `json:"name"`
	Phone            string `json:"phone"`
	Phone2           string `json:"phone2"`
	Email            string `json:"email"`
	NationalID       string `json:"national_id"` // TC kimlik no / YKN only
	Passport         string `json:"passport"`
	Birthday         string `json:"birthday"` // ISO day
	Nationality      string `json:"nationality"`
	Mother           string `json:"mother"`
	Father           string `json:"father"`
	City             string `json:"city"`
	District         string `json:"district"`
	Neighborhood     string `json:"neighborhood"`
	Address          string `json:"address"`
	Occupation       string `json:"occupation"`
	Source           string `json:"source"` // Tn_MusteriKaynagi label
	Agency           string `json:"agency"` // T_Acenta name
	Agent            string `json:"agent"`  // T_Personel name
	Office           string `json:"office"`
	Category         string `json:"category"`           // Tn_DurumSec label (Kriter)
	Result           string `json:"result"`             // Tn_MusteriSonuc label
	CreatedAt        string `json:"created_at"`         // ISO local timestamp
	FirstContactType string `json:"first_contact_type"` // Haberleşme tipi of first contact
}

// LeadRows is every customer in Prism, one row each.
func (s *Store) LeadRows() ([]SyncLead, error) {
	q := `SELECT m.ID, m.MusteriAdiSoyadi, di.CepTel1, di.CepTel2, di.EMail,
			m.TcKimlikNo, m.PasaportNo, k.DogumTarihi, k.UyruguUlkeID, k.AnaAdi, k.BabaAdi,
			di.AdresSehir, di.AdresIlce, di.AdresSemt, di.Adres, m.MeslekDurumSecID_205,
			md.MusteriKaynagiID_1, m.AcentaID, m.PersonelID_1, m.OfisID,
			m.MusteriKategoriDurumSecID_90, m.MusteriSonucID, m.SistemOlusturmaTarihi,
			fi.MusteriHaberlesmeTipiID
		FROM T_Musteri m
		LEFT JOIN T_MusteriDetayIletisim di ON di.MusteriID=m.ID
		LEFT JOIN T_MusteriDetayKimlik k ON k.MusteriID=m.ID
		LEFT JOIN T_MusteriDetay md ON md.ID=(SELECT MAX(ID) FROM T_MusteriDetay WHERE MusteriID=m.ID)
		LEFT JOIN T_MusteriIletisim fi ON fi.ID=m.IlkMusteriIletisimID
		ORDER BY m.ID`
	rows, err := s.query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SyncLead{}
	for rows.Next() {
		var id, name, p1, p2, email, tc, passport, ulke, mom, dad, city, ilce, semt, addr, meslek, src, ace, per, ofi, cat, son, ftype sql.NullString
		var dob, created sql.NullTime
		if err := rows.Scan(&id, &name, &p1, &p2, &email, &tc, &passport, &dob, &ulke, &mom, &dad,
			&city, &ilce, &semt, &addr, &meslek, &src, &ace, &per, &ofi, &cat, &son, &created, &ftype); err != nil {
			return nil, err
		}
		// The two documents travel apart: the CRM has a passport column of
		// its own (leads.passport_no) and moves a passport an older build
		// sent as the ID number into it.
		nationalID := strings.TrimSpace(tc.String)
		out = append(out, SyncLead{
			PrismID: id.String, Name: strings.TrimSpace(name.String),
			Phone: strings.TrimSpace(p1.String), Phone2: strings.TrimSpace(p2.String), Email: strings.TrimSpace(email.String),
			NationalID: nationalID, Passport: strings.TrimSpace(passport.String),
			Birthday: isoDay(dob), Nationality: dec(s.ulke, ulke),
			Mother: strings.TrimSpace(mom.String), Father: strings.TrimSpace(dad.String),
			City: strings.TrimSpace(city.String), District: strings.TrimSpace(ilce.String),
			Neighborhood: strings.TrimSpace(semt.String), Address: strings.TrimSpace(addr.String),
			Occupation: dec(s.dur, meslek), Source: dec(s.kay, src), Agency: dec(s.ace, ace),
			Agent: dec(s.per, per), Office: dec(s.ofi, ofi), Category: dec(s.dur, cat), Result: dec(s.son, son),
			CreatedAt: isoStamp(created), FirstContactType: dec(s.kis, ftype),
		})
	}
	return out, rows.Err()
}

// ---------------- ledger (payment plans and receipts) ----------------

// LedgerRow is one instalment of a deal's plan (T_CariBorc, işlem yeri 550)
// with what has been applied to it, or one receipt (T_CariAlacak, 551).
type LedgerRow struct {
	PrismID string  `json:"prism_id"`
	No      int64   `json:"no,omitempty"`      // Peşinat slot number on a down payment
	Date    string  `json:"date"`              // due day for a plan row, receipt day for a receipt
	Amount  float64 `json:"amount"`            // owed (plan) or received (receipt)
	Paid    float64 `json:"paid,omitempty"`    // plan rows: what receipts have been applied
	Kind    string  `json:"kind,omitempty"`    // plan rows: Peşinat | Vadeli Ödeme | Banka Kredisi
	Method  string  `json:"method,omitempty"`  // Senet, Banka Havalesi, Nakit, …
	Interim bool    `json:"interim,omitempty"` // plan rows: ara ödeme
	Deposit bool    `json:"deposit,omitempty"` // receipts: kapora
	Account string  `json:"account,omitempty"` // receipts: which cash/bank account
	Note    string  `json:"note,omitempty"`
}

// DealLedger is one activity's plan and receipts, keyed the way the stock
// rows are so the CRM finds the deal by unit and buyer.
type DealLedger struct {
	ActivityNo string      `json:"activity_no"`
	Block      string      `json:"block"`
	Door       string      `json:"door"`
	Customer   string      `json:"customer"`
	Phone      string      `json:"phone"`
	Cancelled  bool        `json:"cancelled"`
	SalePrice  float64     `json:"sale_price"`
	Plan       []LedgerRow `json:"plan"`
	Receipts   []LedgerRow `json:"receipts"`
}

// Ledgers is every activity that has a plan row or a receipt: live sales,
// and cancelled ones too, because money received on a cancelled sale is
// exactly what somebody has to chase.
func (s *Store) Ledgers() ([]DealLedger, error) {
	acts := map[string]*DealLedger{}
	order := []string{}
	head, err := s.query(`SELECT a.ID, COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo), g.DaireNo, m.MusteriAdiSoyadi, di.CepTel1,
			a.SatisFiyati, CASE WHEN EXISTS (SELECT 1 FROM T_AktiviteIptal ip WHERE ip.AktiviteID=a.ID) THEN 1 ELSE 0 END
		FROM T_Aktivite a
		LEFT JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID
		LEFT JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		LEFT JOIN T_Musteri m ON m.ID=a.AnaMusteriID
		LEFT JOIN T_MusteriDetayIletisim di ON di.MusteriID=m.ID
		WHERE EXISTS (SELECT 1 FROM T_CariBorc b WHERE b.KayitID=a.ID AND b.CariIslemYeriKisID_310=550 AND b.BorcTutari<>0)
		   OR EXISTS (SELECT 1 FROM T_CariAlacak c WHERE c.KayitID=a.ID AND c.CariIslemYeriKisID_310=551)
		ORDER BY a.ID`)
	if err != nil {
		return nil, err
	}
	for head.Next() {
		var id, block, door, name, phone sql.NullString
		var price sql.NullFloat64
		var cancelled int
		if err := head.Scan(&id, &block, &door, &name, &phone, &price, &cancelled); err != nil {
			head.Close()
			return nil, err
		}
		acts[id.String] = &DealLedger{
			ActivityNo: id.String, Block: block.String, Door: door.String,
			Customer: strings.TrimSpace(name.String), Phone: strings.TrimSpace(phone.String),
			Cancelled: cancelled == 1, SalePrice: price.Float64,
			Plan: []LedgerRow{}, Receipts: []LedgerRow{},
		}
		order = append(order, id.String)
	}
	head.Close()

	plan, err := s.query(`SELECT b.KayitID, b.ID, b.PesinatNo, b.Tarih, b.BorcTutari, b.TipKisID_304, b.OdemeTipiKisID_302,
			d.AraOdemeMi, d.Aciklama,
			(SELECT ISNULL(SUM(k.KapatilanTutar),0) FROM T_CariAlacakBorcKapatma k WHERE k.CariBorcID=b.ID)
		FROM T_CariBorc b
		LEFT JOIN T_CariBorcDetay d ON d.CariBorcID=b.ID
		WHERE b.CariIslemYeriKisID_310=550 AND b.BorcTutari<>0
		ORDER BY b.KayitID, b.Tarih, b.PesinatNo, b.ID`)
	if err != nil {
		return nil, err
	}
	for plan.Next() {
		var act, id, kind, method, note sql.NullString
		var no sql.NullInt64
		var due sql.NullTime
		var amount, paid sql.NullFloat64
		var interim sql.NullBool
		if err := plan.Scan(&act, &id, &no, &due, &amount, &kind, &method, &interim, &note, &paid); err != nil {
			plan.Close()
			return nil, err
		}
		if a, ok := acts[act.String]; ok {
			a.Plan = append(a.Plan, LedgerRow{
				PrismID: id.String, No: no.Int64, Date: isoDay(due), Amount: amount.Float64, Paid: paid.Float64,
				Kind: dec(s.kis, kind), Method: dec(s.kis, method), Interim: interim.Bool,
				Note: strings.TrimSpace(note.String),
			})
		}
	}
	plan.Close()

	rec, err := s.query(`SELECT a.KayitID, a.ID, a.Tarih, a.AlacakTutari, a.OdemeTipiKisID_302, a.KaporaMi,
			h.HesapAciklama, ad.Aciklama
		FROM T_CariAlacak a
		LEFT JOIN T_HesapKartlari h ON h.ID=a.HesapKartlariID
		LEFT JOIN T_CariAlacakDetay ad ON ad.CariAlacakID=a.ID
		WHERE a.CariIslemYeriKisID_310=551
		ORDER BY a.KayitID, a.Tarih, a.ID`)
	if err != nil {
		return nil, err
	}
	for rec.Next() {
		var act, id, method, account, note sql.NullString
		var day sql.NullTime
		var amount sql.NullFloat64
		var deposit sql.NullBool
		if err := rec.Scan(&act, &id, &day, &amount, &method, &deposit, &account, &note); err != nil {
			rec.Close()
			return nil, err
		}
		if a, ok := acts[act.String]; ok {
			a.Receipts = append(a.Receipts, LedgerRow{
				PrismID: id.String, Date: isoDay(day), Amount: amount.Float64,
				Method: dec(s.kis, method), Deposit: deposit.Bool,
				Account: strings.TrimSpace(account.String), Note: strings.TrimSpace(note.String),
			})
		}
	}
	rec.Close()

	out := make([]DealLedger, 0, len(order))
	for _, id := range order {
		out = append(out, *acts[id])
	}
	return out, nil
}

func isoDay(t sql.NullTime) string {
	if !t.Valid || t.Time.Year() < 2000 {
		return ""
	}
	return t.Time.Format("2006-01-02")
}

func isoStamp(t sql.NullTime) string {
	if !t.Valid || t.Time.Year() < 2000 {
		return ""
	}
	return t.Time.Format("2006-01-02T15:04:05")
}
