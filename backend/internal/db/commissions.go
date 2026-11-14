package db

import (
	"database/sql"
	"fmt"
	"maps"
	"strings"
	"time"

	"prismui/internal/commissions"
)

// Personnel is T_Personel id -> name, for picking which agents to track.
func (s *Store) Personnel() map[string]string { return maps.Clone(s.per) }

// CommissionDeals reads every deal whose Satış Personeli 1 is one of agentIDs,
// with what the commission rule needs: price, the planned peşinat and the
// receipts that paid it, and the contract.
//
// The rest of this service never reads SozlesmeYapildiMi (counts go by the
// hareket tipi); here the office's rule is literally "the contract is signed",
// so the contract date — or the flag when the date was left empty — is the
// thing being asked.
func (s *Store) CommissionDeals(agentIDs []string) ([]commissions.Deal, error) {
	if len(agentIDs) == 0 {
		return []commissions.Deal{}, nil
	}
	params := make([]string, len(agentIDs))
	args := make([]any, len(agentIDs))
	for i, id := range agentIDs {
		params[i] = fmt.Sprintf("@p%d", i+1)
		args[i] = id
	}
	in := strings.Join(params, ",")

	rows, err := s.query(`SELECT a.ID, a.SatisPersonelID1, m.MusteriAdiSoyadi,
			COALESCE(NULLIF(g.BolumAdi,''), g.BlokNo), g.DaireNo, g.GayrimenkulTipDurumSecID_206,
			a.HareketTipiKisID_300, ip.AktiviteID, a.Araci1AcentaID,
			a.SatisFiyati, a.SatisFiyatiDovizID, COALESCE(NULLIF(a.ListeFiyatiKur,0), ag.ListeFiyati),
			a.PlanPesinatToplam, b.TahsilatPesinat,
			a.SozlesmeYapildiMi, a.SozlesmeTarihi, a.AktiviteTarihi, a.RezerveTarihi
		FROM T_Aktivite a
		LEFT JOIN T_Musteri m ON m.ID=a.AnaMusteriID
		LEFT JOIN T_AktiviteGayrimenkul ag ON ag.AktiviteID=a.ID
		LEFT JOIN T_Gayrimenkul g ON g.ID=ag.GayrimenkulID
		LEFT JOIN T_AktiviteIptal ip ON ip.AktiviteID=a.ID
		LEFT JOIN T_AktiviteBakiye b ON b.AktiviteID=a.ID
		WHERE a.SatisPersonelID1 IN (`+in+`)
		ORDER BY a.ID`, args...)
	if err != nil {
		return nil, err
	}
	out := []commissions.Deal{}
	at := map[string]int{}
	for rows.Next() {
		var id, per, cust, block, door, typ, hareket, iptal, agency, dov sql.NullString
		var price, list, downPlan, downPrism sql.NullFloat64
		var contracted sql.NullBool
		var contractAt, activityAt, reservedAt sql.NullTime
		if err := rows.Scan(&id, &per, &cust, &block, &door, &typ, &hareket, &iptal, &agency,
			&price, &dov, &list, &downPlan, &downPrism,
			&contracted, &contractAt, &activityAt, &reservedAt); err != nil {
			rows.Close()
			return nil, err
		}
		if _, dup := at[id.String]; dup {
			continue // a second unit or iptal row on the same deal
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
		dealDate := realTime(activityAt)
		if dealDate.IsZero() {
			dealDate = realTime(reservedAt)
		}
		at[id.String] = len(out)
		out = append(out, commissions.Deal{
			ID: id.String, AgentID: strings.TrimSpace(per.String), Customer: strings.TrimSpace(cust.String),
			Unit: unitLabel(block.String, door.String), UnitType: dec(s.dur, typ), Agency: dec(s.ace, agency),
			Stage: stage, Currency: s.cur(dov), SalePrice: price.Float64, ListPrice: list.Float64,
			DownPlan: downPlan.Float64, DownPrism: downPrism.Float64,
			DealDate: dealDate, Contracted: contracted.Bool, ContractAt: realTime(contractAt),
		})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// receipts (the credit ledger, 551) on those deals, oldest first
	rc, err := s.query(`SELECT c.KayitID, c.Tarih, c.AlacakTutari
		FROM T_CariAlacak c JOIN T_Aktivite a ON a.ID=c.KayitID
		WHERE c.CariIslemYeriKisID_310=551 AND a.SatisPersonelID1 IN (`+in+`)
		ORDER BY c.KayitID, c.Tarih, c.ID`, args...)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	for rc.Next() {
		var deal sql.NullString
		var t sql.NullTime
		var amt sql.NullFloat64
		if err := rc.Scan(&deal, &t, &amt); err != nil {
			return nil, err
		}
		i, ok := at[deal.String]
		if !ok || realTime(t).IsZero() {
			continue
		}
		out[i].Receipts = append(out[i].Receipts, commissions.Receipt{Date: t.Time, Amount: amt.Float64})
	}
	return out, rc.Err()
}

// realTime drops Prism's "no date" placeholder (1900-01-01).
func realTime(t sql.NullTime) time.Time {
	if !t.Valid || t.Time.Year() <= 1901 {
		return time.Time{}
	}
	return t.Time
}

// unitLabel is "B2-13": block and door as the sales team says it.
func unitLabel(block, door string) string {
	b, d := strings.TrimSpace(block), strings.TrimSpace(door)
	switch {
	case b != "" && d != "":
		return b + "-" + d
	case b != "":
		return b
	default:
		return d
	}
}
