import { useEffect, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import { api, money, date } from "../api";
import { useApi } from "../ui/kit";
import {
  STATUS, MISSING, NOTES, BASIS, STAGE, ALL, worth, inPeriod, sumBy, moneyCur, rateLabel, monthLabel,
  type CommissionLine,
} from "../commissions";
import { BRAND } from "../brand";
import "./commissions-print.css";

/* The month's commission report (or every month's, for ?ay=all), laid out for A4 landscape. Like the main
   report it opens in its own tab and hands over to the print dialog —
   "Save as PDF" gives the PDF. ?noauto skips the auto-print. */

function DownText({ l }: { l: CommissionLine }) {
  if (l.down_paid_at) return <>{date(l.down_paid_at)}</>;
  return <span className="cp-warn">{money(l.down_paid, l.currency)} / {money(l.down_required, l.currency)}</span>;
}

const contractText = (l: CommissionLine) => (l.contract_at ? date(l.contract_at) : l.contracted ? "imzalı (tarihsiz)" : "yok");

export default function CommissionsPrint() {
  const [params] = useSearchParams();
  const { data, error, loading } = useApi(() => api.commissions(), []);
  const printed = useRef(false);
  const month = params.get("ay") || data?.month || "";
  const only = params.get("danisman") || "";

  useEffect(() => {
    const prev = document.documentElement.getAttribute("data-theme");
    document.documentElement.setAttribute("data-theme", "light");
    return () => {
      if (prev) document.documentElement.setAttribute("data-theme", prev);
      else document.documentElement.removeAttribute("data-theme");
    };
  }, []);

  useEffect(() => {
    if (month) document.title = `Komisyon Raporu — ${monthLabel(month)}`;
  }, [month]);

  useEffect(() => {
    if (!data || printed.current || params.has("noauto")) return;
    printed.current = true;
    const fonts = Promise.race([document.fonts.ready, new Promise((res) => setTimeout(res, 4000))]);
    fonts.then(() => setTimeout(() => window.print(), 400));
  }, [data, params]);

  if (loading) return <div className="cp-root cp-center">Rapor hazırlanıyor…</div>;
  if (error || !data) return <div className="cp-root cp-center cp-bad">Rapor alınamadı: {error}</div>;

  const agents = data.agents.filter((a) => !only || a.personnel_id === only);
  const lines = data.lines.filter((l) => !only || l.agent_id === only);
  const all = month === ALL;
  const inMonth = lines.filter((l) => inPeriod(l, month));
  const waiting = lines.filter((l) => l.status === "waiting");
  // for all time the overdue lines are already in the agents' tables
  const overdue = all ? [] : lines.filter((l) => l.status === "overdue" && l.bill_month !== month);
  const months = Array.from(new Set(inMonth.map((l) => l.bill_month))).sort().reverse();
  const clawback = lines.filter((l) => l.status === "clawback");
  const now = new Date();
  const stamp = now.toLocaleDateString("tr-TR", { day: "numeric", month: "long", year: "numeric" }) + " · " + now.toLocaleTimeString("tr-TR", { hour: "2-digit", minute: "2-digit" });

  const paidSum = (ls: CommissionLine[]) => sumBy(ls.filter((l) => l.payout), worth);
  const unpaidSum = (ls: CommissionLine[]) => sumBy(ls.filter((l) => !l.payout), worth);

  return (
    <div className="cp-root">
      <div className="cp-toolbar">
        <button className="cp-btn primary" onClick={() => window.print()}>PDF olarak kaydet / Yazdır</button>
        <button className="cp-btn" onClick={() => window.close()}>Kapat</button>
      </div>

      <article className="cp-sheet">
        <header className="cp-head">
          {BRAND.logo && <img src={BRAND.logo} alt={BRAND.project} className="cp-logo" />}
          <div className="cp-title">
            <h1>Komisyon Raporu</h1>
            <div className="cp-month">{monthLabel(month)}{only && agents[0] ? ` · ${agents[0].name}` : ""}</div>
            <div className="cp-muted">Oluşturma: {stamp} · Prism canlı verisi</div>
          </div>
        </header>

        <div className="cp-rule">
          <b>Kural:</b> Satışın peşinatı tamamen ödendiğinde ve sözleşmesi imzalandığında komisyon hak edilir; hak edişi izleyen ay faturalanır.
          Hak ediş tarihi, bu iki şarttan sonra gerçekleşenin tarihidir. İptal edilen satışlar komisyona girmez.
        </div>

        <h2>Özet — {monthLabel(month)}</h2>
        <table className="cp-table">
          <thead>
            <tr>
              <th>Danışman</th>
              <th>Oran</th>
              <th>Baz</th>
              <th className="r">Satış</th>
              <th className="r">Komisyon</th>
              <th className="r">Ödenen</th>
              <th className="r">Ödenecek</th>
              <th className="r">Şart bekleyen</th>
            </tr>
          </thead>
          <tbody>
            {agents.map((a) => {
              const mine = inMonth.filter((l) => l.agent_id === a.personnel_id);
              return (
                <tr key={a.personnel_id}>
                  <td className="b">{a.name}</td>
                  <td>{a.rate ? rateLabel(a.rate) : <span className="cp-warn">girilmedi</span>}</td>
                  <td>{BASIS[a.basis]}</td>
                  <td className="r">{mine.length}</td>
                  <td className="r b">{moneyCur(sumBy(mine, worth))}</td>
                  <td className="r">{moneyCur(paidSum(mine))}</td>
                  <td className="r">{moneyCur(unpaidSum(mine))}</td>
                  <td className="r">{waiting.filter((l) => l.agent_id === a.personnel_id).length}</td>
                </tr>
              );
            })}
          </tbody>
          <tfoot>
            <tr>
              <td colSpan={3}>Toplam</td>
              <td className="r">{inMonth.length}</td>
              <td className="r">{moneyCur(sumBy(inMonth, worth))}</td>
              <td className="r">{moneyCur(paidSum(inMonth))}</td>
              <td className="r">{moneyCur(unpaidSum(inMonth))}</td>
              <td className="r">{waiting.length}</td>
            </tr>
          </tfoot>
        </table>

        {all && months.length > 0 && (
          <>
            <h2>Aylara göre</h2>
            <table className="cp-table">
              <thead>
                <tr>
                  <th>Fatura ayı</th>
                  {agents.map((a) => (
                    <th key={a.personnel_id} className="r">{a.name}</th>
                  ))}
                  <th className="r">Satış</th>
                  <th className="r">Toplam</th>
                  <th className="r">Ödenen</th>
                  <th className="r">Ödenecek</th>
                </tr>
              </thead>
              <tbody>
                {months.map((m) => {
                  const ls = inMonth.filter((l) => l.bill_month === m);
                  return (
                    <tr key={m}>
                      <td className="b">{monthLabel(m)}</td>
                      {agents.map((a) => (
                        <td key={a.personnel_id} className="r">{moneyCur(sumBy(ls.filter((l) => l.agent_id === a.personnel_id), worth))}</td>
                      ))}
                      <td className="r">{ls.length}</td>
                      <td className="r b">{moneyCur(sumBy(ls, worth))}</td>
                      <td className="r">{moneyCur(paidSum(ls))}</td>
                      <td className="r">{moneyCur(unpaidSum(ls))}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </>
        )}

        {agents.map((a) => {
          const mine = inMonth.filter((l) => l.agent_id === a.personnel_id);
          return (
            <section key={a.personnel_id} className="cp-agent">
              <h2>
                {a.name} <span className="cp-muted">— {rateLabel(a.rate)} · {BASIS[a.basis].toLocaleLowerCase("tr")} üzerinden</span>
              </h2>
              {a.rule && <div className="cp-note">{a.rule}</div>}
              {mine.length === 0 ? (
                <div className="cp-muted cp-empty">{all ? "Henüz faturalanan komisyon yok." : `${monthLabel(month)} için faturalanacak komisyon yok.`}</div>
              ) : (
                <LineTable lines={mine} showMonth={all} />
              )}
            </section>
          );
        })}

        {overdue.length > 0 && (
          <section className="cp-agent">
            <h2>Geçmiş aylardan ödenmemiş</h2>
            <LineTable lines={overdue} showAgent showMonth />
          </section>
        )}

        {clawback.length > 0 && (
          <section className="cp-agent">
            <h2>Geri alınacak — ödendi, sonra iptal edildi</h2>
            <LineTable lines={clawback} showAgent showMonth />
          </section>
        )}

        {waiting.length > 0 && (
          <section className="cp-agent">
            <h2>Şart bekleyen satışlar <span className="cp-muted">— peşinat veya sözleşme tamamlanınca faturaya girer</span></h2>
            <table className="cp-table">
              <thead>
                <tr>
                  <th>Danışman</th>
                  <th>Müşteri</th>
                  <th>Daire</th>
                  <th>Satış tarihi</th>
                  <th className="r">Satış fiyatı</th>
                  <th className="r">Olası komisyon</th>
                  <th>Peşinat (tahsil / gereken)</th>
                  <th>Sözleşme</th>
                  <th>Eksik</th>
                </tr>
              </thead>
              <tbody>
                {waiting.map((l) => (
                  <tr key={l.deal_id}>
                    <td>{l.agent}</td>
                    <td className="b">{l.customer || "—"}</td>
                    <td>{l.unit || "—"}</td>
                    <td>{date(l.deal_date)}</td>
                    <td className="r">{money(l.sale_price, l.currency)}</td>
                    <td className="r">{money(l.amount, l.currency)}</td>
                    <td><DownText l={l} /></td>
                    <td>{contractText(l)}</td>
                    <td className="cp-warn">{l.missing.map((m) => MISSING[m] ?? m).join(", ")}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>
        )}

        <div className="cp-sign">
          <div><span>Hazırlayan</span></div>
          <div><span>Onaylayan</span></div>
        </div>
      </article>
    </div>
  );
}

function LineTable({ lines, showAgent, showMonth }: { lines: CommissionLine[]; showAgent?: boolean; showMonth?: boolean }) {
  const notes = Array.from(new Set(lines.flatMap((l) => l.notes)));
  return (
    <>
      <table className="cp-table">
        <thead>
          <tr>
            {showAgent && <th>Danışman</th>}
            <th>Müşteri</th>
            <th>Daire</th>
            <th>Durum</th>
            <th className="r">Satış fiyatı</th>
            <th>Peşinat tamam</th>
            <th>Sözleşme</th>
            <th>Hak ediş</th>
            {showMonth && <th>Fatura ayı</th>}
            <th className="r">Oran</th>
            <th className="r">Komisyon</th>
            <th>Ödeme</th>
          </tr>
        </thead>
        <tbody>
          {lines.map((l) => {
            const w = worth(l);
            return (
              <tr key={l.deal_id}>
                {showAgent && <td>{l.agent}</td>}
                <td className="b">{l.customer || "—"}{l.notes.length > 0 && <sup className="cp-warn"> *</sup>}</td>
                <td>{l.unit || "—"}{l.unit_type ? ` · ${l.unit_type}` : ""}</td>
                <td>{STAGE[l.stage] ?? l.stage}{l.agency ? ` · ${l.agency}` : ""}</td>
                <td className="r">{money(l.sale_price, l.currency)}</td>
                <td><DownText l={l} /></td>
                <td>{contractText(l)}</td>
                <td>{date(l.eligible_at)}</td>
                {showMonth && <td>{monthLabel(l.bill_month)}</td>}
                <td className="r">{rateLabel(l.rate)}</td>
                <td className="r b">{money(w.amount, w.currency)}</td>
                <td>{l.payout ? `Ödendi ${date(l.payout.paid_at)}` : <span style={{ color: STATUS[l.status].fg }}>{STATUS[l.status].label}</span>}</td>
              </tr>
            );
          })}
        </tbody>
        <tfoot>
          <tr>
            <td colSpan={8 + (showAgent ? 1 : 0) + (showMonth ? 1 : 0)}>Toplam · {lines.length} satış</td>
            <td className="r">{moneyCur(sumBy(lines, worth))}</td>
            <td></td>
          </tr>
        </tfoot>
      </table>
      {notes.length > 0 && (
        <div className="cp-foot">
          <span className="cp-warn">*</span> {notes.map((n) => NOTES[n] ?? n).join(" · ")}
        </div>
      )}
    </>
  );
}
