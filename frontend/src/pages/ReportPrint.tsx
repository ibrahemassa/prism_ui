import { useEffect, useMemo, useRef } from "react";
import { api, money, moneyShort, num, date } from "../api";
import { useApi } from "../ui/kit";
import type { FullReport, LabeledCount, LabeledMoney } from "../types";
import { VBars, LineArea, Donut, HBars, Legend, PALETTE as P } from "./charts";
import { BRAND, dot } from "../brand";
import "./report-print.css";

/* The long-form printed report. Opens in its own tab, waits for data + fonts,
   then hands the browser's print dialog over — "Save as PDF" gives a vector
   PDF with selectable Turkish text. ?noauto skips the auto-print. */

const T = { sold: P.forest, reserved: P.gold, available: P.line, closed: P.mute, cancelled: P.brick };
const rezSold = [{ label: "Rezerve", color: T.reserved }, { label: "Satıldı", color: T.sold }];

const monthName = (m: string) => {
  const [y, mm] = m.split("-").map(Number);
  return new Date(y, (mm || 1) - 1, 1).toLocaleDateString("tr-TR", { month: "long", year: "numeric" });
};
const monthShort = (m: string) => {
  const [y, mm] = m.split("-").map(Number);
  return new Date(y, (mm || 1) - 1, 1).toLocaleDateString("tr-TR", { month: "short", year: "2-digit" });
};
const dayShort = (d: string) => new Date(d).toLocaleDateString("tr-TR", { day: "numeric", month: "short" });
const pct = (n: number, d: number) => (d > 0 ? (n / d) * 100 : 0);
const pctS = (n: number, d: number, digits = 1) => `%${pct(n, d).toFixed(digits)}`;
const STAGE_TR: Record<string, string> = { sale: "Satıldı", reserved: "Rezerve", cancelled: "İptal", negotiation: "Görüşme" };

function isoWeekStart(d: string) {
  const dt = new Date(d + "T00:00:00");
  const day = (dt.getDay() + 6) % 7; // Monday = 0
  dt.setDate(dt.getDate() - day);
  return dt.toISOString().slice(0, 10);
}

function PageHead({ n, title, sub }: { n: string; title: string; sub?: string }) {
  return (
    <div className="rp-page-head">
      <div>
        <div className="num">BÖLÜM {n}</div>
        <h2>{title}</h2>
      </div>
      {sub && <div className="sub">{sub}</div>}
    </div>
  );
}

function Kpi({ l, v, s, tone }: { l: string; v: string; s?: string; tone?: "pos" | "gold" | "neg" | "deep" }) {
  const cls = tone === "pos" ? "rp-pos" : tone === "gold" ? "rp-gold" : tone === "neg" ? "rp-neg" : "rp-deep";
  const top = tone === "gold" ? "gold" : tone === "neg" ? "brick" : "";
  return (
    <div className={`rp-kpi ${top}`}>
      <div className="l">{l}</div>
      <div className={`v ${cls}`}>{v}</div>
      {s && <div className="s">{s}</div>}
    </div>
  );
}

function ChartTitle({ children, right }: { children: React.ReactNode; right?: React.ReactNode }) {
  return <div className="rp-chart-title"><span>{children}</span>{right}</div>;
}

export default function ReportPrint() {
  const { data: r, error, loading } = useApi(() => api.reportFull(), []);
  const printed = useRef(false);

  useEffect(() => {
    document.title = [BRAND.project, "Satış, Tahsilat ve Pazarlama Raporu"].filter(Boolean).join(" — ");
    const prev = document.documentElement.getAttribute("data-theme");
    document.documentElement.setAttribute("data-theme", "light");
    return () => {
      if (prev) document.documentElement.setAttribute("data-theme", prev);
      else document.documentElement.removeAttribute("data-theme");
    };
  }, []);

  useEffect(() => {
    if (!r || printed.current || new URLSearchParams(location.search).has("noauto")) return;
    printed.current = true;
    const fonts = Promise.race([
      Promise.all([document.fonts.load('800 10.5pt "Instrument Sans"'), document.fonts.load('400 10.5pt "Instrument Sans"')]).then(() => document.fonts.ready),
      new Promise((res) => setTimeout(res, 5000)),
    ]);
    fonts.then(() => setTimeout(() => window.print(), 400));
  }, [r]);

  const derived = useMemo(() => (r ? derive(r) : null), [r]);

  if (loading) return <div className="rp-root" style={{ display: "grid", placeItems: "center" }}><div className="rp-muted">Rapor hazırlanıyor…</div></div>;
  if (error || !r || !derived) return <div className="rp-root" style={{ display: "grid", placeItems: "center" }}><div className="rp-neg">Rapor alınamadı: {error}</div></div>;

  const s = r.summary, d = derived;
  const now = new Date(r.generated_at);
  const stamp = now.toLocaleDateString("tr-TR", { day: "numeric", month: "long", year: "numeric" }) + " · " + now.toLocaleTimeString("tr-TR", { hour: "2-digit", minute: "2-digit" });
  const stampDate = now.toLocaleDateString("tr-TR", { day: "numeric", month: "long", year: "numeric" });
  const footer = dot(BRAND.project, BRAND.company, "Prism UI v2");

  const TOC = [
    ["1", "Yönetici özeti"], ["2", "Zaman çizelgesi — günlük ve haftalık"], ["3", "Aylık görünüm"], ["4", "Tahsilat ve nakit akışı"],
    ["5", "Ödeme planı yapısı"], ["6", "Satış personeli"], ["7", "Acentalar"], ["8", "Müşteri kaynakları"], ["9", "Reklam kaynaklı satışlar"],
    ["10", "Daire tipleri"], ["11", "m² analizi"], ["12", "Bloklar ve stok"], ["13", "Müşteriler"], ["14", "İptaller ve iadeler"], ["15", "Tanımlar ve yöntem"],
  ];

  return (
    <div className="rp-root">
      <div className="rp-toolbar">
        <button className="rp-btn primary" onClick={() => window.print()}>PDF olarak kaydet / Yazdır</button>
        <button className="rp-btn" onClick={() => window.close()}>Kapat</button>
      </div>

      <style>{`@page {
        @bottom-left { content: "${footer.replace(/"/g, "")}"; }
        @bottom-center { content: "Oluşturma: ${stamp.replace(/"/g, "")} · Prism canlı verisi"; font-family: "Instrument Sans", system-ui, sans-serif; font-size: 7.5pt; color: #66756c; }
      }`}</style>
      <article className="rp-sheet">
        {/* ------------------------------------------------ cover */}
        <div className="rp-cover">
          <header className="rp-head" style={{ borderBottom: 0, paddingBottom: 0 }}>
            {BRAND.logo && <img className="rp-logo" src={BRAND.logo} alt={BRAND.project} style={{ height: "18mm" }} />}
            <div className="rp-title"><div>{stampDate}</div></div>
          </header>
          <div className="band">
            <div className="k">SATIŞ · TAHSİLAT · PAZARLAMA</div>
            <h1>{BRAND.project && <>{BRAND.project}<br /></>}Proje Durum Raporu</h1>
            <p>Satış, rezervasyon, tahsilat, stok, acenta ve müşteri analizi — Prism canlı verisi üzerinden, iptaller hariç.</p>
            {BRAND.mark && <img src={BRAND.mark} alt={BRAND.company} />}
          </div>
          <div className="meta">
            <div><span>RAPOR TARİHİ</span><b>{stamp}</b></div>
            <div><span>VERİ KAYNAĞI</span><b>Prism PRISM_PLUS · canlı bağlantı</b></div>
            <div><span>KAPSAM</span><b>{s.deals} aktivite · {s.units} daire · {s.customers} müşteri</b></div>
          </div>
          <div className="rp-kpis" style={{ marginTop: "8mm" }}>
            <Kpi l="AÇIK SATIŞ" v={num(s.live)} s={`${s.sold} satıldı · ${s.reserved} rezerve`} tone="pos" />
            <Kpi l="TOPLAM BEDEL" v={moneyShort(s.pipeline_value)} s={`ort. ${moneyShort(s.avg_price)} / daire`} tone="deep" />
            <Kpi l="TAHSİL EDİLEN" v={moneyShort(s.collected)} s={pctS(s.collected, s.pipeline_value) + " tahsilat"} tone="pos" />
            <Kpi l="TAHSİL EDİLECEK" v={moneyShort(s.to_collect)} s="açık satışlardan kalan" tone="gold" />
          </div>
          <div className="rp-toc">
            {TOC.map(([n, t]) => <div key={n}><span><b style={{ color: P.goldInk }}>{n}</b>&nbsp; {t}</span><span>Bölüm {n}</span></div>)}
          </div>
          <div style={{ marginTop: "auto" }} className="rp-small">
            Hazırlayan: Prism UI v2 · Her sayfa altındaki zaman damgası raporun üretildiği anı gösterir. Rakamlar Prism'in kendi panosuyla birebir aynı kurallarla hesaplanmıştır (bkz. Bölüm 15).
          </div>
        </div>

        {/* ------------------------------------------------ 1 executive summary */}
        <section className="rp-page">
          <PageHead n="1" title="Yönetici özeti" sub={`${s.deals} aktivite · ${s.live} açık · ${s.cancelled} iptal`} />
          <div className="rp-kpis six">
            <Kpi l="AÇIK SATIŞ" v={num(s.live)} s={`${s.sold} satıldı · ${s.reserved} rezerve`} tone="pos" />
            <Kpi l="TOPLAM BEDEL" v={money(s.pipeline_value)} s={`liste ${moneyShort(s.list_value)} · indirim ${moneyShort(s.discount)} (${pctS(s.discount, s.list_value)})`} tone="deep" />
            <Kpi l="TAHSİL EDİLEN" v={money(s.collected)} s={`${pctS(s.collected, s.pipeline_value)} · kapora ${moneyShort(s.kapora_paid)}`} tone="pos" />
            <Kpi l="TAHSİL EDİLECEK" v={money(s.to_collect)} s={`peşinat ${moneyShort(r.down_payment.planned - r.down_payment.paid)} · vadeli ${moneyShort(r.down_payment.installments)}`} tone="gold" />
            <Kpi l="SATILAN / REZERVE m²" v={`${num(s.m2_sold + s.m2_reserved)} m²`} s={`${num(s.m2_sold)} satıldı · ${num(s.m2_reserved)} rezerve · brüt`} tone="deep" />
            <Kpi l="ORT. m² FİYATI" v={money(s.price_per_m2)} s="açık satışlar, brüt m²" tone="deep" />
            <Kpi l="ALICI" v={num(s.buyers)} s={`${s.customers} müşteriden · ${pctS(s.buyers, s.customers)} dönüşüm`} tone="deep" />
            <Kpi l="STOK" v={`${s.units_available} daire`} s={`${num(s.m2_available)} m² · liste ${moneyShort(s.stock_value)}`} tone="deep" />
            <Kpi l="İPTAL" v={`${s.cancelled} · ${pctS(s.cancelled, s.deals)}`} s={`${moneyShort(s.refunds_owed)} iade bekliyor`} tone="neg" />
          </div>

          <div className="rp-callout">
            <b>Öne çıkanlar</b>
            <ul>{d.insights.map((t, i) => <li key={i} dangerouslySetInnerHTML={{ __html: t }} />)}</ul>
          </div>

          <div className="rp-grid2">
            <div>
              <ChartTitle>Aktivite durumu</ChartTitle>
              <Donut center={num(s.deals)} parts={[
                { label: "Satıldı", value: s.sold, color: T.sold },
                { label: "Rezerve", value: s.reserved, color: T.reserved },
                { label: "İptal", value: s.cancelled, color: T.cancelled },
                { label: "Görüşme", value: s.deals - s.live - s.cancelled, color: T.closed },
              ]} />
            </div>
            <div>
              <ChartTitle>Stok durumu ({s.units} daire)</ChartTitle>
              <Donut center={`${pct(s.units_sold + s.units_reserved, s.units).toFixed(0)}%`} parts={[
                { label: "Satıldı", value: s.units_sold, color: T.sold },
                { label: "Rezerve", value: s.units_reserved, color: T.reserved },
                { label: "Boşta", value: s.units_available, color: T.available },
                { label: "Kapalı", value: s.units_closed, color: T.closed },
              ]} />
            </div>
          </div>

          <ChartTitle right={<Legend items={[{ label: "Tahsil edilen", color: T.sold }, { label: "Kalan", color: T.available }]} />}>Bedel ve tahsilat</ChartTitle>
          <HBars fmt={moneyShort} max={s.pipeline_value} rows={[
            { label: "Toplam bedel", parts: [{ value: s.collected, color: T.sold }, { value: s.to_collect, color: T.available }], right: <><b>{money(s.pipeline_value)}</b></> },
            { label: "Peşinat planı", parts: [{ value: r.down_payment.paid, color: T.sold }, { value: Math.max(0, r.down_payment.planned - r.down_payment.paid), color: T.available }], right: <><b>{money(r.down_payment.planned)}</b> <span className="rp-muted">· {pctS(r.down_payment.paid, r.down_payment.planned, 0)} ödendi</span></> },
            { label: "Vadeli (senet)", parts: [{ value: Math.max(0, s.collected - r.down_payment.paid), color: T.sold }, { value: r.down_payment.installments, color: T.available }], right: <><b>{money(r.down_payment.installments)}</b></> },
          ]} />
        </section>

        {/* ------------------------------------------------ 2 timeline */}
        <section className="rp-page">
          <PageHead n="2" title="Zaman çizelgesi" sub="Günlük açık satışlar, iptaller ve tahsilat · rezerve tarihine göre" />
          <ChartTitle right={<Legend items={[...rezSold, { label: "İptal", color: T.cancelled }]} />}>Günlük aktivite — rezerve ve satıldı (adet)</ChartTitle>
          <VBars labels={d.days.map((x) => dayShort(x.date))} labelEvery={Math.max(1, Math.ceil(d.days.length / 16))} rotate showValues={false} height={160}
            series={[{ name: "Rezerve", values: d.days.map((x) => x.reserved), color: T.reserved }, { name: "Satıldı", values: d.days.map((x) => x.sold), color: T.sold }]} />
          <ChartTitle>Günlük iptal (adet)</ChartTitle>
          <VBars labels={d.days.map((x) => dayShort(x.date))} labelEvery={Math.max(1, Math.ceil(d.days.length / 16))} rotate showValues={false} height={90}
            series={[{ name: "İptal", values: d.days.map((x) => x.cancelled), color: T.cancelled }]} />
          <ChartTitle right={<Legend items={[{ label: "Kümülatif açık satış", color: T.sold }, { label: "Kümülatif tahsilat", color: T.reserved }]} />}>Birikimli görünüm</ChartTitle>
          <div className="rp-grid2">
            <LineArea width={310} height={140} labels={d.days.map((x) => dayShort(x.date))} labelEvery={Math.max(1, Math.ceil(d.days.length / 6))}
              series={[{ name: "Açık satış", values: d.cumDeals, color: T.sold, area: true }]} />
            <LineArea width={310} height={140} fmt={moneyShort} labels={d.days.map((x) => dayShort(x.date))} labelEvery={Math.max(1, Math.ceil(d.days.length / 6))}
              series={[{ name: "Tahsilat", values: d.cumReceipts, color: T.reserved, area: true }]} />
          </div>
          <ChartTitle>Haftalık özet</ChartTitle>
          <table className="rp-table compact">
            <thead><tr><th>Hafta</th><th className="r">Rezerve</th><th className="r">Satıldı</th><th className="r">İptal</th><th className="r">Net</th><th className="r">Bedel</th><th className="r">Tahsilat</th></tr></thead>
            <tbody>
              {d.weeks.map((w) => (
                <tr key={w.week}>
                  <td>{dayShort(w.week)} haftası</td>
                  <td className="r rp-gold">{w.reserved || "—"}</td>
                  <td className="r rp-pos">{w.sold || "—"}</td>
                  <td className="r rp-neg">{w.cancelled || "—"}</td>
                  <td className={`r b ${w.reserved + w.sold - w.cancelled >= 0 ? "rp-pos" : "rp-neg"}`}>{w.reserved + w.sold - w.cancelled > 0 ? "+" : ""}{w.reserved + w.sold - w.cancelled}</td>
                  <td className="r">{w.revenue ? money(w.revenue) : "—"}</td>
                  <td className="r rp-pos">{w.receipts ? money(w.receipts) : "—"}</td>
                </tr>
              ))}
            </tbody>
            <tfoot><tr><td>Toplam</td><td className="r">{s.reserved}</td><td className="r">{s.sold}</td><td className="r">{s.cancelled}</td><td className="r">+{s.live - s.cancelled}</td><td className="r">{money(s.pipeline_value)}</td><td className="r">{money(s.collected)}</td></tr></tfoot>
          </table>
        </section>

        {/* ------------------------------------------------ 3 monthly */}
        <section className="rp-page">
          <PageHead n="3" title="Aylık görünüm" sub="Açık satışlar rezerve tarihine, tahsilat makbuz tarihine göre" />
          <div className="rp-grid2">
            <div>
              <ChartTitle right={<Legend items={rezSold} />}>Aylık açık satış (adet)</ChartTitle>
              <VBars width={310} height={150} labels={d.months.map((m) => monthShort(m.month))}
                series={[{ name: "Rezerve", values: d.months.map((m) => m.reserved), color: T.reserved }, { name: "Satıldı", values: d.months.map((m) => m.sold), color: T.sold }]} />
            </div>
            <div>
              <ChartTitle>Aylık bedel</ChartTitle>
              <VBars width={310} height={150} fmt={moneyShort} labels={d.months.map((m) => monthShort(m.month))}
                series={[{ name: "Bedel", values: d.months.map((m) => m.revenue), color: P.deep }]} />
            </div>
          </div>
          <div className="rp-grid2">
            <div>
              <ChartTitle>Aylık tahsilat</ChartTitle>
              <VBars width={310} height={150} fmt={moneyShort} labels={r.receipts.by_month.map((m) => monthShort(m.label))}
                series={[{ name: "Tahsilat", values: r.receipts.by_month.map((m) => m.revenue), color: T.sold }]} />
            </div>
            <div>
              <ChartTitle>Aylık iptal (adet)</ChartTitle>
              <VBars width={310} height={150} labels={r.cancellations.by_month.map((m) => monthShort(m.label))}
                series={[{ name: "İptal", values: r.cancellations.by_month.map((m) => m.count), color: T.cancelled }]} />
            </div>
          </div>
          <table className="rp-table compact">
            <thead><tr><th>Ay</th><th className="r">Rezerve</th><th className="r">Satıldı</th><th className="r">Açık</th><th className="r">İptal</th><th className="r">Bedel</th><th className="r">Ort. bedel</th><th className="r">Tahsilat</th></tr></thead>
            <tbody>
              {d.months.map((m) => (
                <tr key={m.month}>
                  <td>{monthName(m.month)}</td>
                  <td className="r rp-gold">{m.reserved || "—"}</td>
                  <td className="r rp-pos">{m.sold || "—"}</td>
                  <td className="r b">{m.reserved + m.sold}</td>
                  <td className="r rp-neg">{m.cancelled || "—"}</td>
                  <td className="r rp-deep b">{money(m.revenue)}</td>
                  <td className="r">{m.reserved + m.sold ? money(m.revenue / (m.reserved + m.sold)) : "—"}</td>
                  <td className="r rp-pos">{m.receipts ? money(m.receipts) : "—"}</td>
                </tr>
              ))}
            </tbody>
            <tfoot><tr><td>Toplam</td><td className="r">{s.reserved}</td><td className="r">{s.sold}</td><td className="r">{s.live}</td><td className="r">{s.cancelled}</td><td className="r">{money(s.pipeline_value)}</td><td className="r">{money(s.avg_price)}</td><td className="r">{money(s.collected)}</td></tr></tfoot>
          </table>
        </section>

        {/* ------------------------------------------------ 4 collection */}
        <section className="rp-page">
          <PageHead n="4" title="Tahsilat ve nakit akışı" sub="T_CariAlacak makbuzları ve T_CariBorc vade planı · iptaller hariç" />
          <div className="rp-kpis four">
            <Kpi l="TAHSİLAT" v={money(r.receipts.total)} s={`${r.receipts.count} makbuz · ort. ${moneyShort(r.receipts.average)}`} tone="pos" />
            <Kpi l="KAPORA" v={money(s.kapora_paid)} s={"tahsilatın " + pctS(s.kapora_paid, r.receipts.total) + "\u2019i"} tone="pos" />
            <Kpi l="EN BÜYÜK MAKBUZ" v={money(r.receipts.largest)} tone="deep" />
            <Kpi l="İADE BEKLEYEN" v={money(s.refunds_owed)} s={`${r.cancellations.with_deposit} iptal dosyasında`} tone="neg" />
          </div>
          <div className="rp-grid2">
            <div>
              <ChartTitle>Ödeme yöntemi</ChartTitle>
              <Donut fmt={moneyShort} parts={r.receipts.by_method.map((m, i) => ({ label: m.label, value: m.revenue, color: [T.sold, T.reserved, P.sage, T.closed][i % 4] }))} />
            </div>
            <div>
              <ChartTitle>Hesap</ChartTitle>
              <Donut fmt={moneyShort} parts={r.receipts.by_account.map((m, i) => ({ label: m.label, value: m.revenue, color: [P.deep, T.reserved, P.sage][i % 3] }))} />
            </div>
          </div>
          <ChartTitle>Peşinat ve vadeli</ChartTitle>
          <table className="rp-table compact">
            <thead><tr><th>Kalem</th><th className="r">Planlanan</th><th className="r">Ödenen</th><th className="r">Kalan</th><th className="r">Oran</th></tr></thead>
            <tbody>
              <tr><td>Peşinat</td><td className="r">{money(r.down_payment.planned)}</td><td className="r rp-pos">{money(r.down_payment.paid)}</td><td className="r rp-gold">{money(r.down_payment.planned - r.down_payment.paid)}</td><td className="r">{pctS(r.down_payment.paid, r.down_payment.planned)}</td></tr>
              <tr><td>Vadeli (senet)</td><td className="r">{money(r.down_payment.installments)}</td><td className="r rp-pos">{money(Math.max(0, s.collected - r.down_payment.paid))}</td><td className="r rp-gold">{money(r.down_payment.installments - Math.max(0, s.collected - r.down_payment.paid))}</td><td className="r">{pctS(Math.max(0, s.collected - r.down_payment.paid), r.down_payment.installments)}</td></tr>
              {r.down_payment.bank_loan > 0 && <tr><td>Banka kredisi</td><td className="r">{money(r.down_payment.bank_loan)}</td><td className="r">—</td><td className="r">—</td><td className="r">—</td></tr>}
            </tbody>
            <tfoot><tr><td>Toplam</td><td className="r">{money(s.pipeline_value)}</td><td className="r">{money(s.collected)}</td><td className="r">{money(s.to_collect)}</td><td className="r">{pctS(s.collected, s.pipeline_value)}</td></tr></tfoot>
          </table>
          <ChartTitle right={<Legend items={[{ label: "Ödenen", color: T.sold }, { label: "Bekleyen", color: T.reserved }]} />}>Vade planı — aya göre beklenen tahsilat (önümüzdeki 36 ay)</ChartTitle>
          <VBars fmt={moneyShort} height={160} rotate labelEvery={2} showValues={false} labels={d.schedule12.map((m) => monthShort(m.month))}
            series={[{ name: "Ödenen", values: d.schedule12.map((m) => m.paid), color: T.sold }, { name: "Bekleyen", values: d.schedule12.map((m) => m.remaining), color: T.reserved }]} />
          <div className="rp-grid2">
            {[d.schedule12.slice(0, 18), d.schedule12.slice(18, 36)].map((half, i) => (
              <table className="rp-table compact" key={i}>
                <thead><tr><th>Ay</th><th className="r">Taksit</th><th className="r">Vade</th><th className="r">Ödenen</th><th className="r">Bekleyen</th></tr></thead>
                <tbody>
                  {half.map((m) => (
                    <tr key={m.month}><td>{monthShort(m.month)}</td><td className="r">{m.count}</td><td className="r">{moneyShort(m.due)}</td><td className="r rp-pos">{m.paid ? moneyShort(m.paid) : "—"}</td><td className="r rp-gold b">{moneyShort(m.remaining)}</td></tr>
                  ))}
                </tbody>
                {i === 1 && <tfoot><tr><td>36 ay</td><td className="r">{d.schedule12.reduce((n, m) => n + m.count, 0)}</td><td className="r">{moneyShort(d.schedule12.reduce((n, m) => n + m.due, 0))}</td><td className="r">{moneyShort(d.schedule12.reduce((n, m) => n + m.paid, 0))}</td><td className="r">{moneyShort(d.schedule12.reduce((n, m) => n + m.remaining, 0))}</td></tr></tfoot>}
              </table>
            ))}
          </div>
          <div className="rp-small" style={{ marginTop: "1mm" }}>36 ay toplamı: {money(d.schedule12.reduce((n, m) => n + m.due, 0))} vade, {money(d.schedule12.reduce((n, m) => n + m.paid, 0))} ödenmiş, <b className="rp-gold">{money(d.schedule12.reduce((n, m) => n + m.remaining, 0))}</b> bekleyen.</div>
          <div className="rp-small" style={{ marginTop: "2mm" }}>Vade planının tamamı: {r.schedule.length} ay, {money(r.schedule.reduce((n, m) => n + m.due, 0))} ({r.schedule.reduce((n, m) => n + m.count, 0)} taksit); son vade {r.schedule.length ? monthName(r.schedule[r.schedule.length - 1].month) : "—"}. Geçmiş aylardan bekleyen: <b className="rp-neg">{money(d.overdue)}</b>.</div>
        </section>

        {/* ------------------------------------------------ 5 plan structure */}
        <section className="rp-page">
          <PageHead n="5" title="Ödeme planı yapısı" sub="Açık satışların plan tipi, peşinat oranı, vade ve indirim" />
          <div className="rp-kpis four">
            <Kpi l="ORT. PEŞİNAT ORANI" v={pctS(r.down_payment.planned, s.pipeline_value)} s="planlanan peşinat ÷ bedel" tone="deep" />
            <Kpi l="ORT. VADE" v={`${r.plans.avg_term_months.toFixed(1)} ay`} s={`${num(r.plans.installments)} bekleyen taksit`} tone="deep" />
            <Kpi l="NPV TOPLAMI" v={money(r.plans.npv_total)} s={`bedelin ${pctS(r.plans.npv_total, s.pipeline_value)}'i`} tone="deep" />
            <Kpi l="İNDİRİM" v={money(s.discount)} s={`liste fiyatına göre ${pctS(s.discount, s.list_value)}`} tone="gold" />
          </div>
          <div className="rp-grid2">
            <div>
              <ChartTitle>Plan tipi (adet)</ChartTitle>
              <Donut parts={r.plans.by_type.map((p, i) => ({ label: p.label, value: p.count, color: [T.sold, T.reserved, T.closed, P.sage][i % 4] }))} />
            </div>
            <div>
              <ChartTitle>Plan tipi (bedel)</ChartTitle>
              <Donut fmt={moneyShort} parts={r.plans.by_type.map((p, i) => ({ label: p.label, value: p.revenue, color: [T.sold, T.reserved, T.closed, P.sage][i % 4] }))} />
            </div>
          </div>
          <ChartTitle right={<Legend items={[{ label: "Ödenen", color: T.sold }, { label: "Bekleyen", color: T.reserved }]} />}>Yıllara göre vade planı</ChartTitle>
          <div className="rp-grid2">
            <VBars width={310} height={140} fmt={moneyShort} labels={d.years.map((y) => y.year)}
              series={[{ name: "Ödenen", values: d.years.map((y) => y.paid), color: T.sold }, { name: "Bekleyen", values: d.years.map((y) => y.remaining), color: T.reserved }]} />
            <table className="rp-table compact">
              <thead><tr><th>Yıl</th><th className="r">Taksit</th><th className="r">Vade tutarı</th><th className="r">Ödenen</th><th className="r">Bekleyen</th></tr></thead>
              <tbody>
                {d.years.map((y) => (
                  <tr key={y.year}><td className="b">{y.year}</td><td className="r">{y.count}</td><td className="r">{money(y.due)}</td><td className="r rp-pos">{y.paid ? money(y.paid) : "—"}</td><td className="r rp-gold b">{money(y.remaining)}</td></tr>
                ))}
              </tbody>
              <tfoot><tr><td>Toplam</td><td className="r">{d.years.reduce((n, y) => n + y.count, 0)}</td><td className="r">{money(d.years.reduce((n, y) => n + y.due, 0))}</td><td className="r">{money(d.years.reduce((n, y) => n + y.paid, 0))}</td><td className="r">{money(d.years.reduce((n, y) => n + y.remaining, 0))}</td></tr></tfoot>
            </table>
          </div>
          <ChartTitle right={<Legend items={[{ label: "Peşinat", color: T.sold }, { label: "Vadeli", color: T.reserved }, { label: "Banka", color: T.closed }]} />}>Bedelin dağılımı</ChartTitle>
          <HBars fmt={moneyShort} rows={[{ label: "Açık satışlar", parts: [{ value: r.down_payment.planned, color: T.sold }, { value: r.down_payment.installments, color: T.reserved }, { value: r.down_payment.bank_loan, color: T.closed }] }]} />
          <div className="rp-callout">
            <ul>
              <li>Açık satışların <b>{pctS(r.down_payment.planned, s.pipeline_value, 0)}</b>'i peşinat, <b>{pctS(r.down_payment.installments, s.pipeline_value, 0)}</b>'i vadeli olarak planlanmış; ortalama vade <b>{r.plans.avg_term_months.toFixed(0)} ay</b>.</li>
              <li>Planlanan peşinatın <b>{pctS(r.down_payment.paid, r.down_payment.planned, 0)}</b>'i tahsil edildi; peşinatı tamamlanmamış dosyalarda <b>{money(r.down_payment.planned - r.down_payment.paid)}</b> beklemede.</li>
              <li>Liste fiyatına göre toplam indirim <b>{money(s.discount)}</b> ({pctS(s.discount, s.list_value)}); Prism NPV toplamı bedelin <b>{pctS(r.plans.npv_total, s.pipeline_value, 0)}</b>'i.</li>
            </ul>
          </div>
        </section>

        {/* ------------------------------------------------ 6 agents */}
        <section className="rp-page">
          <PageHead n="6" title="Satış personeli" sub="Satış Personeli alanına göre · açık satışlar; iptal ayrı sayılır" />
          <table className="rp-table compact">
            <thead><tr><th>Danışman</th><th className="r">Rezerve</th><th className="r">Satıldı</th><th className="r">İptal</th><th className="r">Alıcı</th><th className="r">m²</th><th className="r">Toplam bedel</th><th className="r">Pay</th><th className="r">Tahsilat</th><th className="r">Tahsilat oranı</th></tr></thead>
            <tbody>
              {r.agents.map((a) => (
                <tr key={a.agent}>
                  <td className="b">{a.agent}</td>
                  <td className="r rp-gold">{a.reserved || "—"}</td>
                  <td className="r rp-pos">{a.sales || "—"}</td>
                  <td className="r rp-neg">{a.cancelled || "—"}</td>
                  <td className="r">{a.customers}</td>
                  <td className="r">{num(a.m2)}</td>
                  <td className="r rp-deep b">{money(a.revenue)}</td>
                  <td className="r rp-muted">{pctS(a.revenue, s.pipeline_value)}</td>
                  <td className="r rp-pos">{money(a.collected)}</td>
                  <td className="r">{pctS(a.collected, a.revenue)}</td>
                </tr>
              ))}
            </tbody>
            <tfoot><tr><td>Toplam</td><td className="r">{s.reserved}</td><td className="r">{s.sold}</td><td className="r">{r.agents.reduce((n, a) => n + a.cancelled, 0)}</td><td className="r">{s.buyers}</td><td className="r">{num(r.agents.reduce((n, a) => n + a.m2, 0))}</td><td className="r">{money(s.pipeline_value)}</td><td className="r">%100</td><td className="r">{money(s.collected)}</td><td className="r">{pctS(s.collected, s.pipeline_value)}</td></tr></tfoot>
          </table>
          <div className="rp-grid2" style={{ marginTop: "4mm" }}>
            <div>
              <ChartTitle right={<Legend items={rezSold} />}>Açık satış (adet)</ChartTitle>
              <HBars rows={r.agents.map((a) => ({ label: a.agent, parts: [{ value: a.reserved, color: T.reserved }, { value: a.sales, color: T.sold }] }))} />
            </div>
            <div>
              <ChartTitle>Tahsilat</ChartTitle>
              <HBars fmt={moneyShort} rows={r.agents.map((a) => ({ label: a.agent, parts: [{ value: a.collected, color: T.sold }] }))} />
            </div>
          </div>
          <div className="rp-grid2" style={{ marginTop: "3mm" }}>
            <div>
              <ChartTitle>İptal (adet)</ChartTitle>
              <HBars rows={r.cancellations.by_agent.map((a) => ({ label: a.label, parts: [{ value: a.count, color: T.cancelled }] }))} />
            </div>
            <div>
              <ChartTitle>Kayıtlı müşteri (adet)</ChartTitle>
              <HBars rows={r.customers.by_agent.slice(0, 8).map((a) => ({ label: a.label, parts: [{ value: a.count, color: P.sage }] }))} />
            </div>
          </div>
          <div className="rp-callout">
            <ul>
              <li>Bedelde lider <b>{r.agents[0]?.agent}</b> ({moneyShort(r.agents[0]?.revenue || 0)}, {pctS(r.agents[0]?.revenue || 0, s.pipeline_value, 0)}); tahsilatta lider <b>{d.topCollector?.agent}</b> ({moneyShort(d.topCollector?.collected || 0)}).</li>
              <li>Ekip toplam <b>{num(r.agents.reduce((n, a) => n + a.m2, 0))} m²</b> açık satış taşıyor; ortalama dosya bedeli <b>{money(s.avg_price)}</b>.</li>
              <li>İptal oranı en yüksek: <b>{d.worstCancel?.agent}</b> ({d.worstCancel ? pctS(d.worstCancel.cancelled, d.worstCancel.cancelled + d.worstCancel.reserved + d.worstCancel.sales, 0) : "—"} — {d.worstCancel?.cancelled} iptal / {d.worstCancel ? d.worstCancel.cancelled + d.worstCancel.reserved + d.worstCancel.sales : 0} aktivite).</li>
            </ul>
          </div>
        </section>

        {/* ------------------------------------------------ 7 agencies */}
        <section className="rp-page">
          <PageHead n="7" title="Acentalar" sub={`${s.agencies_active} acenta açık satış taşıyor · ${s.via_agency} açık satış acenta üzerinden (${pctS(s.via_agency, s.live, 0)})`} />
          <div className="rp-grid2">
            <div>
              <ChartTitle>Açık satış bedeli — doğrudan / acenta</ChartTitle>
              <Donut fmt={moneyShort} parts={[
                { label: "Doğrudan", value: d.direct?.revenue || 0, color: T.sold },
                { label: "Acenta", value: s.pipeline_value - (d.direct?.revenue || 0), color: T.reserved },
              ]} />
            </div>
            <div>
              <ChartTitle>Acentaların payı (bedel)</ChartTitle>
              <Donut fmt={moneyShort} parts={d.agencyOnly.slice(0, 6).map((a, i) => ({ label: a.agency, value: a.revenue, color: [T.sold, T.reserved, P.sage, P.deep, P.goldInk, T.closed][i % 6] }))} />
            </div>
          </div>
          <table className="rp-table compact">
            <thead><tr><th>Acenta</th><th className="r">Rezerve</th><th className="r">Satıldı</th><th className="r">İptal</th><th className="r">m²</th><th className="r">Toplam bedel</th><th className="r">Tahsilat</th><th className="r">Komisyon</th><th>Daire tipleri</th></tr></thead>
            <tbody>
              {r.agencies.map((a) => (
                <tr key={a.agency} style={a.agency.startsWith("Doğrudan") ? { background: P.band } : undefined}>
                  <td className="b">{a.agency}</td>
                  <td className="r rp-gold">{a.reserved || "—"}</td>
                  <td className="r rp-pos">{a.sold || "—"}</td>
                  <td className="r rp-neg">{a.cancelled || "—"}</td>
                  <td className="r">{a.m2 ? num(a.m2) : "—"}</td>
                  <td className="r rp-deep b">{a.revenue ? money(a.revenue) : "—"}</td>
                  <td className="r rp-pos">{a.collected ? money(a.collected) : "—"}</td>
                  <td className="r">{a.commission ? money(a.commission) : a.commission_pct ? `%${a.commission_pct.toFixed(1)}` : "—"}</td>
                  <td className="rp-muted" style={{ whiteSpace: "normal" }}>{a.types.map((t) => `${t.label} ×${t.count}`).join(", ") || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="rp-callout">
            <ul>
              <li>Açık satışların <b>{pctS(s.live - s.via_agency, s.live, 0)}</b>'i doğrudan, <b>{pctS(s.via_agency, s.live, 0)}</b>'i acenta üzerinden; bedelde acenta payı <b>{pctS(s.pipeline_value - (d.direct?.revenue || 0), s.pipeline_value, 0)}</b>.</li>
              <li>En büyük acenta <b>{d.agencyOnly[0]?.agency}</b>: {d.agencyOnly[0]?.reserved} rezerve, {d.agencyOnly[0]?.sold} satıldı, {moneyShort(d.agencyOnly[0]?.revenue || 0)}.</li>
              <li>Acenta kaynaklı iptal: <b>{d.agencyOnly.reduce((n, a) => n + a.cancelled, 0)}</b> — acenta aktivitelerinin {pctS(d.agencyOnly.reduce((n, a) => n + a.cancelled, 0), d.agencyOnly.reduce((n, a) => n + a.cancelled + a.reserved + a.sold, 0), 0)}'i (doğrudan: {d.direct ? pctS(d.direct.cancelled, d.direct.cancelled + d.direct.reserved + d.direct.sold, 0) : "—"}).</li>
            </ul>
          </div>
        </section>

        {/* ------------------------------------------------ 8 sources */}
        <section className="rp-page">
          <PageHead n="8" title="Müşteri kaynakları" sub="Aktivitenin Müşteri Kaynağı alanına göre (Tn_MusteriKaynagi)" />
          <div className="rp-grid2">
            <div>
              <ChartTitle right={<Legend items={rezSold} />}>Kaynağa göre açık satış (adet)</ChartTitle>
              <HBars rows={r.sources.map((x) => ({ label: x.source, parts: [{ value: x.reserved, color: T.reserved }, { value: x.sold, color: T.sold }] }))} />
            </div>
            <div>
              <ChartTitle>Kaynağa göre bedel</ChartTitle>
              <HBars fmt={moneyShort} rows={r.sources.map((x) => ({ label: x.source, parts: [{ value: x.revenue, color: P.deep }] }))} />
            </div>
          </div>
          <table className="rp-table compact">
            <thead><tr><th>Kaynak</th><th className="r">Rezerve</th><th className="r">Satıldı</th><th className="r">İptal</th><th className="r">İptal oranı</th><th className="r">Toplam bedel</th><th className="r">Pay</th><th className="r">Tahsilat</th></tr></thead>
            <tbody>
              {r.sources.map((x) => (
                <tr key={x.source}>
                  <td className="b">{x.source}</td>
                  <td className="r rp-gold">{x.reserved || "—"}</td>
                  <td className="r rp-pos">{x.sold || "—"}</td>
                  <td className="r rp-neg">{x.cancelled || "—"}</td>
                  <td className="r">{pctS(x.cancelled, x.cancelled + x.reserved + x.sold, 0)}</td>
                  <td className="r rp-deep b">{x.revenue ? money(x.revenue) : "—"}</td>
                  <td className="r rp-muted">{pctS(x.revenue, s.pipeline_value)}</td>
                  <td className="r rp-pos">{x.collected ? money(x.collected) : "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <ChartTitle>Kayıtlı müşteri havuzunun kaynağı ({r.customers.total} müşteri)</ChartTitle>
          <HBars rows={r.customers.by_source.map((x) => ({ label: x.label, parts: [{ value: x.count, color: P.sage }] }))} />
          <div className="rp-callout">
            <ul>
              <li>Müşteri havuzunda dijital kaynaklar (Landing Page, Web, Instagram) çoğunlukta; açık satışta ise <b>{r.sources[0]?.source}</b> öne çıkıyor ({r.sources[0]?.reserved} rezerve + {r.sources[0]?.sold} satıldı, {moneyShort(r.sources[0]?.revenue || 0)}).</li>
              <li>Kaynağı boş bırakılmış aktivite: <b>{r.sources.find((x) => x.source === "Belirtilmemiş")?.reserved ?? 0} açık</b> — kaynak girişi zorunlu tutulmalı.</li>
            </ul>
          </div>
        </section>

        {/* ------------------------------------------------ 9 ad sales */}
        <section className="rp-page">
          <PageHead n="9" title="Reklam kaynaklı satışlar" sub={`${date(r.ad_sales.since)} ve sonrası · acenta alanı boş`} />
          <div className="rp-callout">
            <b>Tanım.</b> {date(r.ad_sales.since)} tarihinden itibaren açılan ve üzerinde acenta bulunmayan aktiviteler. Müşteri kaynağı etiketine bakılmaz (referans dahil); acenta alanı dolu olan her aktivite acenta satışı sayılır. Bu dönemde toplam {r.ad_sales.all_since} aktivite açıldı; <b>{r.ad_sales.count}</b>'i ({pctS(r.ad_sales.count, r.ad_sales.all_since, 0)}) bu tanıma giriyor.
          </div>
          <div className="rp-kpis four">
            <Kpi l="AKTİVİTE" v={num(r.ad_sales.count)} s={`${r.ad_sales.reserved} rezerve · ${r.ad_sales.sold} satıldı · ${r.ad_sales.cancelled} iptal`} tone="deep" />
            <Kpi l="AÇIK SATIŞ" v={num(r.ad_sales.reserved + r.ad_sales.sold)} s={pctS(r.ad_sales.reserved + r.ad_sales.sold, r.ad_sales.count, 0) + " dönüşüm"} tone="pos" />
            <Kpi l="TOPLAM BEDEL" v={money(r.ad_sales.revenue)} s={`ort. ${moneyShort(r.ad_sales.revenue / Math.max(1, r.ad_sales.reserved + r.ad_sales.sold))}`} tone="deep" />
            <Kpi l="TAHSİLAT" v={money(r.ad_sales.collected)} s={"bedelin " + pctS(r.ad_sales.collected, r.ad_sales.revenue) + "\u2019i"} tone="pos" />
          </div>
          <div className="rp-grid3">
            <div>
              <ChartTitle>Kaynak etiketi (bilgi amaçlı, açık)</ChartTitle>
              <HBars rows={r.ad_sales.by_source.map((x) => ({ label: x.label, parts: [{ value: x.count, color: T.sold }] }))} />
            </div>
            <div>
              <ChartTitle>Danışman (açık)</ChartTitle>
              <HBars rows={r.ad_sales.by_agent.map((x) => ({ label: x.label, parts: [{ value: x.count, color: T.reserved }] }))} />
            </div>
            <div>
              <ChartTitle>Daire tipi (açık)</ChartTitle>
              <HBars rows={r.ad_sales.by_type.map((x) => ({ label: x.label, parts: [{ value: x.count, color: P.sage }] }))} />
            </div>
          </div>
          <ChartTitle>Aktivite listesi</ChartTitle>
          <table className="rp-table compact">
            <thead><tr><th>Tarih</th><th>Müşteri</th><th>Daire</th><th>Tip</th><th>Kaynak</th><th>Danışman</th><th>Durum</th><th className="r">Bedel</th><th className="r">Tahsilat</th></tr></thead>
            <tbody>
              {r.ad_sales.deals.map((x) => (
                <tr key={x.id} style={x.stage === "cancelled" ? { color: P.mute } : undefined}>
                  <td>{date(x.date)}</td>
                  <td>{x.customer || "—"}</td>
                  <td className="b">{x.unit}</td>
                  <td>{x.unit_type}</td>
                  <td>{x.source}</td>
                  <td>{x.agent || "—"}</td>
                  <td className={x.stage === "cancelled" ? "rp-neg" : x.stage === "sale" ? "rp-pos" : "rp-gold"}>{STAGE_TR[x.stage] ?? x.stage}</td>
                  <td className="r">{money(x.price)}</td>
                  <td className="r rp-pos">{x.collected ? money(x.collected) : "—"}</td>
                </tr>
              ))}
            </tbody>
            <tfoot><tr><td colSpan={7}>Açık toplam ({r.ad_sales.reserved + r.ad_sales.sold})</td><td className="r">{money(r.ad_sales.revenue)}</td><td className="r">{money(r.ad_sales.collected)}</td></tr></tfoot>
          </table>
        </section>

        {/* ------------------------------------------------ 10 types */}
        <section className="rp-page">
          <PageHead n="10" title="Daire tipleri" sub="Stok her dairenin Prism'deki son hareketine göre; bedel ve m² fiyatı açık satışlardan" />
          <div className="rp-grid2">
            <div>
              <ChartTitle right={<Legend items={[{ label: "Satıldı", color: T.sold }, { label: "Rezerve", color: T.reserved }, { label: "Boşta", color: T.available }, { label: "Kapalı", color: T.closed }]} />}>Tipe göre stok</ChartTitle>
              <HBars rows={r.types.map((t) => ({ label: t.label, parts: [{ value: t.sold, color: T.sold }, { value: t.reserved, color: T.reserved }, { value: t.available, color: T.available }, { value: t.closed, color: T.closed }], right: <><b>{t.total}</b> <span className="rp-muted">· {pctS(t.sold + t.reserved, t.total, 0)}</span></> }))} />
            </div>
            <div>
              <ChartTitle right={<Legend items={[{ label: "Satıldı", color: T.sold }, { label: "Rezerve", color: T.reserved }, { label: "Boşta", color: T.available }]} />}>Tipe göre m² (brüt)</ChartTitle>
              <HBars fmt={(n) => `${num(n)} m²`} rows={r.types.map((t) => ({ label: t.label, parts: [{ value: t.m2_sold, color: T.sold }, { value: t.m2_reserved, color: T.reserved }, { value: t.m2_available, color: T.available }], right: <><b>{num(t.m2_total)}</b> <span className="rp-muted">m²</span></> }))} />
            </div>
          </div>
          <ChartTitle>Ortalama m² fiyatı (açık satış)</ChartTitle>
          <HBars fmt={moneyShort} rows={r.types.filter((t) => t.price_per_m2 > 0).map((t) => ({ label: t.label, parts: [{ value: t.price_per_m2, color: P.deep }], right: <b>{money(t.price_per_m2)}</b> }))} />
          <ChartTitle>Tip bazında adet, m² ve bedel</ChartTitle>
          <table className="rp-table compact">
            <thead><tr><th>Tip</th><th className="r">Daire</th><th className="r">Satıldı</th><th className="r">Rezerve</th><th className="r">Boşta</th><th className="r">Doluluk</th><th className="r">Ort. m²</th><th className="r">m² satıldı</th><th className="r">m² rezerve</th><th className="r">m² boşta</th><th className="r">m² toplam</th><th className="r">Toplam bedel</th><th className="r">Ort. bedel</th><th className="r">m² fiyatı</th><th className="r">Tahsilat</th><th className="r">Stok liste</th></tr></thead>
            <tbody>
              {r.types.map((t) => (
                <tr key={t.label}>
                  <td className="b">{t.label}</td>
                  <td className="r">{t.total}</td>
                  <td className="r rp-pos">{t.sold || "—"}</td>
                  <td className="r rp-gold">{t.reserved || "—"}</td>
                  <td className="r">{t.available || "—"}{t.closed ? <span className="rp-muted"> +{t.closed}k</span> : null}</td>
                  <td className="r">{pctS(t.sold + t.reserved, t.total, 0)}</td>
                  <td className="r">{num(t.avg_m2, 1)}</td>
                  <td className="r rp-pos">{t.m2_sold ? num(t.m2_sold) : "—"}</td>
                  <td className="r rp-gold">{t.m2_reserved ? num(t.m2_reserved) : "—"}</td>
                  <td className="r">{t.m2_available ? num(t.m2_available) : "—"}</td>
                  <td className="r b">{num(t.m2_total)}</td>
                  <td className="r rp-deep b">{t.revenue ? money(t.revenue) : "—"}</td>
                  <td className="r">{t.avg_price ? money(t.avg_price) : "—"}</td>
                  <td className="r">{t.price_per_m2 ? money(t.price_per_m2) : "—"}</td>
                  <td className="r rp-pos">{t.collected ? money(t.collected) : "—"}</td>
                  <td className="r rp-muted">{t.stock_list ? money(t.stock_list) : "—"}</td>
                </tr>
              ))}
            </tbody>
            <tfoot><tr><td>Toplam</td><td className="r">{s.units}</td><td className="r">{s.units_sold}</td><td className="r">{s.units_reserved}</td><td className="r">{s.units_available}</td><td className="r">{pctS(s.units_sold + s.units_reserved, s.units, 0)}</td><td className="r">{num(s.m2_total / Math.max(1, s.units), 1)}</td><td className="r">{num(s.m2_sold)}</td><td className="r">{num(s.m2_reserved)}</td><td className="r">{num(s.m2_available)}</td><td className="r">{num(s.m2_total)}</td><td className="r">{money(s.pipeline_value)}</td><td className="r">{money(s.avg_price)}</td><td className="r">{money(s.price_per_m2)}</td><td className="r">{money(s.collected)}</td><td className="r">{money(s.stock_value)}</td></tr></tfoot>
          </table>
          <div className="rp-callout">
            <ul>
              <li><b>{d.bestType?.label}</b> en çok satılan tip: {d.bestType?.sold} satıldı + {d.bestType?.reserved} rezerve, {d.bestType ? pctS(d.bestType.sold + d.bestType.reserved, d.bestType.total, 0) : "—"} doluluk.</li>
              <li>Stokta en çok bekleyen tip <b>{d.slowType?.label}</b>: {d.slowType?.available} boş daire, liste değeri {moneyShort(d.slowType?.stock_list || 0)} — stok liste değerinin {d.slowType ? pctS(d.slowType.stock_list, s.stock_value, 0) : "—"}'i.</li>
              <li>m² fiyatı en yüksek tip <b>{d.priciest?.label}</b> ({money(d.priciest?.price_per_m2 || 0)}/m²); proje ortalaması {money(s.price_per_m2)}/m².</li>
              <li>Brüt m²: <b>{num(s.m2_sold)} m² satıldı</b>, <b>{num(s.m2_reserved)} m² rezerve</b>, <b>{num(s.m2_available)} m² boşta</b> — toplam {num(s.m2_total)} m²'nin {pctS(s.m2_sold + s.m2_reserved, s.m2_total, 0)}'i açık satışta.</li>
            </ul>
          </div>
        </section>

        {/* ------------------------------------------------ 11 m² */}
        <section className="rp-page">
          <PageHead n="11" title="m² analizi" sub="Brüt ve net m² · satılan, rezerve, boşta · m² başına bedel" />
          <div className="rp-kpis six">
            <Kpi l="TOPLAM BRÜT m²" v={`${num(s.m2_total)} m²`} s={`net ${num(r.m2.net_total)} m² · ${s.units} daire`} tone="deep" />
            <Kpi l="SATILAN m²" v={`${num(s.m2_sold)} m²`} s={`net ${num(r.m2.net_sold)} · toplamın ${pctS(s.m2_sold, s.m2_total)}`} tone="pos" />
            <Kpi l="REZERVE m²" v={`${num(s.m2_reserved)} m²`} s={`net ${num(r.m2.net_reserved)} · toplamın ${pctS(s.m2_reserved, s.m2_total)}`} tone="gold" />
            <Kpi l="BOŞTA m²" v={`${num(s.m2_available)} m²`} s={`net ${num(r.m2.net_available)} · toplamın ${pctS(s.m2_available, s.m2_total)}`} tone="deep" />
            <Kpi l="AÇIK SATIŞ m² FİYATI" v={money(s.price_per_m2)} s={`satıldı ${moneyShort(r.m2.sold_per_m2)} · rezerve ${moneyShort(r.m2.res_per_m2)}`} tone="pos" />
            <Kpi l="STOK İSTENEN m² FİYATI" v={money(r.m2.ask_per_m2)} s={`boştaki ${num(s.m2_available)} m² için liste`} tone="gold" />
          </div>

          <div className="rp-grid2">
            <div>
              <ChartTitle>Brüt m² dağılımı</ChartTitle>
              <Donut fmt={(n) => `${num(n)} m²`} center={`${pct(s.m2_sold + s.m2_reserved, s.m2_total).toFixed(0)}%`} parts={[
                { label: "Satıldı", value: s.m2_sold, color: T.sold },
                { label: "Rezerve", value: s.m2_reserved, color: T.reserved },
                { label: "Boşta", value: s.m2_available, color: T.available },
                { label: "Kapalı", value: r.m2.brut_closed, color: T.closed },
              ]} />
            </div>
            <div>
              <ChartTitle>Brüt / net</ChartTitle>
              <table className="rp-table compact">
                <thead><tr><th>Durum</th><th className="r">Brüt m²</th><th className="r">Net m²</th><th className="r">Net / brüt</th><th className="r">Daire</th><th className="r">Ort. brüt</th></tr></thead>
                <tbody>
                  <tr><td className="rp-pos b">Satıldı</td><td className="r">{num(s.m2_sold)}</td><td className="r">{num(r.m2.net_sold)}</td><td className="r">{pctS(r.m2.net_sold, s.m2_sold, 0)}</td><td className="r">{s.units_sold}</td><td className="r">{num(s.m2_sold / Math.max(1, s.units_sold))}</td></tr>
                  <tr><td className="rp-gold b">Rezerve</td><td className="r">{num(s.m2_reserved)}</td><td className="r">{num(r.m2.net_reserved)}</td><td className="r">{pctS(r.m2.net_reserved, s.m2_reserved, 0)}</td><td className="r">{s.units_reserved}</td><td className="r">{num(s.m2_reserved / Math.max(1, s.units_reserved))}</td></tr>
                  <tr><td className="b">Boşta</td><td className="r">{num(s.m2_available)}</td><td className="r">{num(r.m2.net_available)}</td><td className="r">{pctS(r.m2.net_available, s.m2_available, 0)}</td><td className="r">{s.units_available}</td><td className="r">{num(s.m2_available / Math.max(1, s.units_available))}</td></tr>
                  <tr><td className="rp-muted">Kapalı</td><td className="r">{num(r.m2.brut_closed)}</td><td className="r">{num(r.m2.net_closed)}</td><td className="r">{pctS(r.m2.net_closed, r.m2.brut_closed, 0)}</td><td className="r">{s.units_closed}</td><td className="r">{num(r.m2.brut_closed / Math.max(1, s.units_closed))}</td></tr>
                </tbody>
                <tfoot><tr><td>Toplam</td><td className="r">{num(s.m2_total)}</td><td className="r">{num(r.m2.net_total)}</td><td className="r">{pctS(r.m2.net_total, s.m2_total, 0)}</td><td className="r">{s.units}</td><td className="r">{num(s.m2_total / Math.max(1, s.units))}</td></tr></tfoot>
              </table>
            </div>
          </div>

          <ChartTitle right={<Legend items={rezSold} />}>Aylık açık satış m² (brüt, rezerve tarihine göre)</ChartTitle>
          <VBars height={140} fmt={(n) => `${num(n)} m²`} labels={r.m2.by_month.map((x) => monthShort(x.month))}
            series={[{ name: "Rezerve", values: r.m2.by_month.map((x) => x.m2_reserved), color: T.reserved }, { name: "Satıldı", values: r.m2.by_month.map((x) => x.m2_sold), color: T.sold }]} />

          <ChartTitle>m² fiyatı — satıldı, rezerve ve boşta (liste) · tipe göre</ChartTitle>
          <table className="rp-table compact">
            <thead><tr><th>Tip</th><th className="r">m² satıldı</th><th className="r">m² rezerve</th><th className="r">m² boşta</th><th className="r">m² toplam</th><th className="r">Net satıldı+rez.</th><th className="r">Satıldı ₺/m²</th><th className="r">Rezerve ₺/m²</th><th className="r">Boşta liste ₺/m²</th><th className="r">Bedel/m²</th><th className="r">Tahsilat/m²</th></tr></thead>
            <tbody>
              {r.m2.by_type.filter((t) => t.m2_total > 0).map((t) => (
                <tr key={t.label}>
                  <td className="b">{t.label}</td>
                  <td className="r rp-pos">{t.m2_sold ? num(t.m2_sold) : "—"}</td>
                  <td className="r rp-gold">{t.m2_reserved ? num(t.m2_reserved) : "—"}</td>
                  <td className="r">{t.m2_available ? num(t.m2_available) : "—"}</td>
                  <td className="r b">{num(t.m2_total)}</td>
                  <td className="r rp-muted">{num(t.net_sold + t.net_reserved)}</td>
                  <td className="r rp-pos">{t.m2_sold ? money(t.revenue_sold / t.m2_sold) : "—"}</td>
                  <td className="r rp-gold">{t.m2_reserved ? money(t.revenue_reserved / t.m2_reserved) : "—"}</td>
                  <td className="r">{t.m2_available && t.stock_list ? money(t.stock_list / t.m2_available) : "—"}</td>
                  <td className="r rp-deep b">{t.m2_sold + t.m2_reserved ? money(t.revenue / (t.m2_sold + t.m2_reserved)) : "—"}</td>
                  <td className="r">{t.m2_sold + t.m2_reserved ? money(t.collected / (t.m2_sold + t.m2_reserved)) : "—"}</td>
                </tr>
              ))}
            </tbody>
            <tfoot><tr><td>Toplam</td><td className="r">{num(s.m2_sold)}</td><td className="r">{num(s.m2_reserved)}</td><td className="r">{num(s.m2_available)}</td><td className="r">{num(s.m2_total)}</td><td className="r">{num(r.m2.net_sold + r.m2.net_reserved)}</td><td className="r">{money(r.m2.sold_per_m2)}</td><td className="r">{money(r.m2.res_per_m2)}</td><td className="r">{money(r.m2.ask_per_m2)}</td><td className="r">{money(s.price_per_m2)}</td><td className="r">{money(s.collected / Math.max(1, s.m2_sold + s.m2_reserved))}</td></tr></tfoot>
          </table>

          <div className="rp-callout">
            <ul>
              <li>Projenin <b>{num(s.m2_total)} m²</b> brüt alanının <b>{pctS(s.m2_sold + s.m2_reserved, s.m2_total, 0)}</b>'i açık satışta ({num(s.m2_sold)} satıldı + {num(s.m2_reserved)} rezerve); <b>{num(s.m2_available)} m²</b> boşta, <b>{num(r.m2.cancelled_m2)} m²</b> iptallerle stoka geri döndü.</li>
              <li>Satıldı dosyalarda gerçekleşen <b>{money(r.m2.sold_per_m2)}/m²</b>, rezervelerde <b>{money(r.m2.res_per_m2)}/m²</b>; boştaki stok liste fiyatıyla <b>{money(r.m2.ask_per_m2)}/m²</b> istiyor — açık satışların liste ortalaması {money(r.m2.list_per_m2)}/m², indirim sonrası {money(s.price_per_m2)}/m².</li>
              <li>Boştaki m²'nin <b>{pctS(d.stockTop?.m2_available || 0, s.m2_available, 0)}</b>'i tek tipte: <b>{d.stockTop?.label}</b> ({num(d.stockTop?.m2_available || 0)} m², liste {moneyShort(d.stockTop?.stock_list || 0)}).</li>
            </ul>
          </div>
        </section>

        <section className="rp-page">
          <PageHead n="11" title="m² analizi — devam" sub="Bloklara, danışmana ve acentaya göre m²; en büyük dosyalar" />
          <ChartTitle right={<Legend items={[{ label: "Satıldı", color: T.sold }, { label: "Rezerve", color: T.reserved }, { label: "Boşta", color: T.available }]} />}>Bloğa göre brüt m²</ChartTitle>
          <VBars height={140} fmt={(n) => `${num(n / 1000, 1)}k`} showValues={false} labels={r.m2.by_block.filter((b) => b.m2_total > 0).map((b) => b.label)}
            series={[{ name: "Satıldı", values: r.m2.by_block.filter((b) => b.m2_total > 0).map((b) => b.m2_sold), color: T.sold }, { name: "Rezerve", values: r.m2.by_block.filter((b) => b.m2_total > 0).map((b) => b.m2_reserved), color: T.reserved }, { name: "Boşta", values: r.m2.by_block.filter((b) => b.m2_total > 0).map((b) => b.m2_available), color: T.available }]} />
          <table className="rp-table compact">
            <thead><tr><th>Blok</th><th className="r">m² satıldı</th><th className="r">m² rezerve</th><th className="r">m² boşta</th><th className="r">m² toplam</th><th className="r">Açık pay</th><th className="r">Satıldı ₺/m²</th><th className="r">Rezerve ₺/m²</th><th className="r">Boşta liste ₺/m²</th><th className="r">Bedel/m²</th></tr></thead>
            <tbody>
              {r.m2.by_block.filter((b) => b.m2_total > 0).map((b) => (
                <tr key={b.label}>
                  <td className="b">{b.label}</td>
                  <td className="r rp-pos">{b.m2_sold ? num(b.m2_sold) : "—"}</td>
                  <td className="r rp-gold">{b.m2_reserved ? num(b.m2_reserved) : "—"}</td>
                  <td className="r">{b.m2_available ? num(b.m2_available) : "—"}</td>
                  <td className="r b">{num(b.m2_total)}</td>
                  <td className="r">{pctS(b.m2_sold + b.m2_reserved, b.m2_total, 0)}</td>
                  <td className="r rp-pos">{b.m2_sold ? money(b.revenue_sold / b.m2_sold) : "—"}</td>
                  <td className="r rp-gold">{b.m2_reserved ? money(b.revenue_reserved / b.m2_reserved) : "—"}</td>
                  <td className="r">{b.m2_available && b.stock_list ? money(b.stock_list / b.m2_available) : "—"}</td>
                  <td className="r rp-deep b">{b.m2_sold + b.m2_reserved ? money(b.revenue / (b.m2_sold + b.m2_reserved)) : "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="rp-grid2" style={{ marginTop: "3mm" }}>
            <div>
              <ChartTitle right={<Legend items={rezSold} />}>Danışmana göre m² (açık satış)</ChartTitle>
              <HBars fmt={(n) => `${num(n)} m²`} rows={r.m2.by_agent.map((a) => ({ label: a.label, parts: [{ value: a.m2_reserved, color: T.reserved }, { value: a.m2_sold, color: T.sold }], right: <><b>{num(a.m2_total)}</b> <span className="rp-muted">m² · {a.deals} dosya</span></> }))} />
            </div>
            <div>
              <ChartTitle right={<Legend items={rezSold} />}>Acentaya göre m² (açık satış)</ChartTitle>
              <HBars fmt={(n) => `${num(n)} m²`} rows={r.m2.by_agency.slice(0, 8).map((a) => ({ label: a.label, parts: [{ value: a.m2_reserved, color: T.reserved }, { value: a.m2_sold, color: T.sold }], right: <><b>{num(a.m2_total)}</b> <span className="rp-muted">m²</span></> }))} />
            </div>
          </div>
          <ChartTitle>En büyük 10 açık dosya (brüt m²)</ChartTitle>
          <table className="rp-table compact">
            <thead><tr><th>Daire</th><th>Tip</th><th>Müşteri</th><th>Durum</th><th className="r">Brüt m²</th><th className="r">Net m²</th><th className="r">Bedel</th><th className="r">₺/m²</th></tr></thead>
            <tbody>
              {r.m2.largest.map((x) => (
                <tr key={x.unit}><td className="b">{x.unit}</td><td>{x.unit_type}</td><td>{x.customer || "—"}</td><td className={x.stage === "sale" ? "rp-pos" : "rp-gold"}>{STAGE_TR[x.stage]}</td><td className="r b">{num(x.brut_m2, 1)}</td><td className="r">{num(x.net_m2, 1)}</td><td className="r">{money(x.price)}</td><td className="r">{x.brut_m2 ? money(x.price / x.brut_m2) : "—"}</td></tr>
              ))}
            </tbody>
          </table>
          <div className="rp-callout">
            <ul>
              <li>m²'de lider danışman <b>{r.m2.by_agent[0]?.label}</b> ({num(r.m2.by_agent[0]?.m2_total || 0)} m²); acentalar toplam <b>{num(r.m2.by_agency.filter((a) => a.label !== "Doğrudan").reduce((n, a) => n + a.m2_total, 0))} m²</b> taşıyor, doğrudan {num(r.m2.by_agency.find((a) => a.label === "Doğrudan")?.m2_total || 0)} m².</li>
              <li>Blok bazında en yüksek açık m² payı <b>{d.blockTopShare?.label}</b> ({d.blockTopShare ? pctS(d.blockTopShare.m2_sold + d.blockTopShare.m2_reserved, d.blockTopShare.m2_total, 0) : "—"}); en çok boş m² <b>{d.blockMostAvail?.label}</b> ({num(d.blockMostAvail?.m2_available || 0)} m²).</li>
            </ul>
          </div>
        </section>

        {/* ------------------------------------------------ 11 blocks */}
        <section className="rp-page">
          <PageHead n="12" title="Bloklar ve stok" sub="Her blokta daire durumu, m², bedel ve kalan stok" />
          <ChartTitle right={<Legend items={[{ label: "Satıldı", color: T.sold }, { label: "Rezerve", color: T.reserved }, { label: "Boşta", color: T.available }, { label: "Kapalı", color: T.closed }]} />}>Bloğa göre daire durumu (adet)</ChartTitle>
          <VBars height={150} labels={r.blocks.map((b) => b.label)}
            series={[{ name: "Satıldı", values: r.blocks.map((b) => b.sold), color: T.sold }, { name: "Rezerve", values: r.blocks.map((b) => b.reserved), color: T.reserved }, { name: "Boşta", values: r.blocks.map((b) => b.available), color: T.available }, { name: "Kapalı", values: r.blocks.map((b) => b.closed), color: T.closed }]} />
          <ChartTitle right={<Legend items={[{ label: "Satıldı", color: T.sold }, { label: "Rezerve", color: T.reserved }, { label: "Boşta", color: T.available }]} />}>Bloğa göre m² (brüt)</ChartTitle>
          <VBars height={130} fmt={(n) => `${num(n / 1000, 1)}k`} showValues={false} labels={r.blocks.map((b) => b.label)}
            series={[{ name: "Satıldı", values: r.blocks.map((b) => b.m2_sold), color: T.sold }, { name: "Rezerve", values: r.blocks.map((b) => b.m2_reserved), color: T.reserved }, { name: "Boşta", values: r.blocks.map((b) => b.m2_available), color: T.available }]} />
          <div className="rp-grid2">
            <div>
              <ChartTitle>Bloğa göre açık satış bedeli</ChartTitle>
              <VBars width={310} height={130} fmt={moneyShort} labels={r.blocks.map((b) => b.label)} series={[{ name: "Bedel", values: r.blocks.map((b) => b.revenue), color: P.deep }]} />
            </div>
            <div>
              <ChartTitle>Bloğa göre kalan stok (liste değeri)</ChartTitle>
              <VBars width={310} height={130} fmt={moneyShort} labels={r.blocks.map((b) => b.label)} series={[{ name: "Stok", values: r.blocks.map((b) => b.stock_list), color: T.reserved }]} />
            </div>
          </div>
          <table className="rp-table compact">
            <thead><tr><th>Blok</th><th className="r">Daire</th><th className="r">Satıldı</th><th className="r">Rezerve</th><th className="r">Boşta</th><th className="r">Doluluk</th><th className="r">m² satıldı</th><th className="r">m² rezerve</th><th className="r">m² boşta</th><th className="r">m² toplam</th><th className="r">Toplam bedel</th><th className="r">Tahsilat</th><th className="r">Tahsilat oranı</th><th className="r">Stok liste</th></tr></thead>
            <tbody>
              {r.blocks.map((b) => (
                <tr key={b.label}>
                  <td className="b">{b.label}</td>
                  <td className="r">{b.units}</td>
                  <td className="r rp-pos">{b.sold || "—"}</td>
                  <td className="r rp-gold">{b.reserved || "—"}</td>
                  <td className="r">{b.available || "—"}{b.closed ? <span className="rp-muted"> +{b.closed}k</span> : null}</td>
                  <td className="r">{pctS(b.sold + b.reserved, b.units, 0)}</td>
                  <td className="r rp-pos">{b.m2_sold ? num(b.m2_sold) : "—"}</td>
                  <td className="r rp-gold">{b.m2_reserved ? num(b.m2_reserved) : "—"}</td>
                  <td className="r">{b.m2_available ? num(b.m2_available) : "—"}</td>
                  <td className="r b">{num(b.m2_total)}</td>
                  <td className="r rp-deep b">{b.revenue ? money(b.revenue) : "—"}</td>
                  <td className="r rp-pos">{b.collected ? money(b.collected) : "—"}</td>
                  <td className="r">{b.revenue ? pctS(b.collected, b.revenue) : "—"}</td>
                  <td className="r rp-muted">{b.stock_list ? money(b.stock_list) : "—"}</td>
                </tr>
              ))}
            </tbody>
            <tfoot><tr><td>Toplam</td><td className="r">{s.units}</td><td className="r">{s.units_sold}</td><td className="r">{s.units_reserved}</td><td className="r">{s.units_available}</td><td className="r">{pctS(s.units_sold + s.units_reserved, s.units, 0)}</td><td className="r">{num(s.m2_sold)}</td><td className="r">{num(s.m2_reserved)}</td><td className="r">{num(s.m2_available)}</td><td className="r">{num(s.m2_total)}</td><td className="r">{money(s.pipeline_value)}</td><td className="r">{money(s.collected)}</td><td className="r">{pctS(s.collected, s.pipeline_value)}</td><td className="r">{money(s.stock_value)}</td></tr></tfoot>
          </table>
        </section>

        {/* ------------------------------------------------ 12 customers */}
        <section className="rp-page">
          <PageHead n="13" title="Müşteriler" sub="T_Musteri kayıtları · kategori, kaynak, şehir ve haftalık yeni kayıt" />
          <div className="rp-kpis four">
            <Kpi l="KAYITLI MÜŞTERİ" v={num(r.customers.total)} s={`${r.customers.companies} şirket`} tone="deep" />
            <Kpi l="ALICI" v={num(r.customers.buyers)} s={`${pctS(r.customers.buyers, r.customers.total)} dönüşüm`} tone="pos" />
            <Kpi l="AÇIK SATIŞ / ALICI" v={(s.live / Math.max(1, r.customers.buyers)).toFixed(2)} s="birden fazla daire alanlar" tone="deep" />
            <Kpi l="SON 4 HAFTA" v={num(d.newLast4w)} s="yeni müşteri kaydı" tone="deep" />
          </div>
          <div className="rp-grid2">
            <div>
              <ChartTitle>Kategori</ChartTitle>
              <Donut parts={r.customers.by_category.map((c, i) => ({ label: c.label, value: c.count, color: [T.sold, T.reserved, P.sage, T.closed, P.deep][i % 5] }))} />
            </div>
            <div>
              <ChartTitle>Şehir (belirtilenler)</ChartTitle>
              <HBars rows={r.customers.by_city.filter((c) => c.label !== "Belirtilmemiş").slice(0, 6).map((c) => ({ label: c.label, parts: [{ value: c.count, color: P.sage }] }))} />
              <div className="rp-small">Şehri boş: {r.customers.by_city.find((c) => c.label === "Belirtilmemiş")?.count ?? 0} müşteri</div>
            </div>
          </div>
          <ChartTitle>Haftalık yeni müşteri kaydı</ChartTitle>
          <VBars height={140} labels={d.custWeeks.map((w) => dayShort(w.label))} rotate labelEvery={1}
            series={[{ name: "Yeni müşteri", values: d.custWeeks.map((w) => w.count), color: P.sage }]} />
          <div className="rp-grid2">
            <div>
              <ChartTitle>Kaynak</ChartTitle>
              <HBars rows={r.customers.by_source.map((c) => ({ label: c.label, parts: [{ value: c.count, color: T.sold }] }))} />
            </div>
            <div>
              <ChartTitle>Danışman</ChartTitle>
              <HBars rows={r.customers.by_agent.slice(0, 8).map((c) => ({ label: c.label, parts: [{ value: c.count, color: T.reserved }] }))} />
            </div>
          </div>
        </section>

        {/* ------------------------------------------------ 13 cancellations */}
        <section className="rp-page">
          <PageHead n="14" title="İptaller ve iadeler" sub="T_AktiviteIptal · iptal edilen aktivitelerin bedeli hiçbir toplamda yer almaz" />
          <div className="rp-kpis four">
            <Kpi l="İPTAL" v={num(r.cancellations.total)} s={`aktivitelerin ${pctS(r.cancellations.total, s.deals)}\u2019i`} tone="neg" />
            <Kpi l="İPTAL EDİLEN BEDEL" v={money(r.cancellations.value)} s="tanımlı satış fiyatı" tone="neg" />
            <Kpi l="ORT. İPTAL SÜRESİ" v={`${r.cancellations.avg_days.toFixed(0)} gün`} s="aktivite → iptal" tone="deep" />
            <Kpi l="İADE BEKLEYEN" v={money(r.cancellations.refunds)} s={`${r.cancellations.with_deposit} dosya ödeme yapmıştı`} tone="gold" />
          </div>
          <ChartTitle>Aylık iptal (adet ve bedel)</ChartTitle>
          <div className="rp-grid2">
            <VBars width={310} height={130} labels={r.cancellations.by_month.map((m) => monthShort(m.label))} series={[{ name: "İptal", values: r.cancellations.by_month.map((m) => m.count), color: T.cancelled }]} />
            <VBars width={310} height={130} fmt={moneyShort} labels={r.cancellations.by_month.map((m) => monthShort(m.label))} series={[{ name: "Bedel", values: r.cancellations.by_month.map((m) => m.revenue), color: P.mute }]} />
          </div>
          <div className="rp-grid3">
            <div>
              <ChartTitle>Neden</ChartTitle>
              <Donut size={90} parts={r.cancellations.by_reason.map((x, i) => ({ label: x.label, value: x.count, color: [T.cancelled, T.closed, P.sand][i % 3] }))} />
            </div>
            <div>
              <ChartTitle>Tür</ChartTitle>
              <Donut size={90} parts={r.cancellations.by_kind.map((x, i) => ({ label: x.label, value: x.count, color: [P.mute, P.sand][i % 2] }))} />
            </div>
            <div>
              <ChartTitle>Daire tipi</ChartTitle>
              <HBars rows={r.cancellations.by_type.map((x) => ({ label: x.label, parts: [{ value: x.count, color: T.cancelled }] }))} />
            </div>
          </div>
          <ChartTitle>Danışmana göre iptal ve iptal oranı</ChartTitle>
          <table className="rp-table compact">
            <thead><tr><th>Danışman</th><th className="r">İptal</th><th className="r">Açık</th><th className="r">İptal oranı</th><th></th></tr></thead>
            <tbody>
              {r.agents.map((a) => (
                <tr key={a.agent}><td className="b">{a.agent}</td><td className="r rp-neg">{a.cancelled}</td><td className="r rp-pos">{a.reserved + a.sales}</td><td className="r">{pctS(a.cancelled, a.cancelled + a.reserved + a.sales, 0)}</td>
                  <td style={{ width: 160 }}><div className="rp-bar" style={{ display: "flex" }}><i style={{ width: `${pct(a.cancelled, a.cancelled + a.reserved + a.sales)}%`, background: T.cancelled }} /><i style={{ width: `${pct(a.reserved + a.sales, a.cancelled + a.reserved + a.sales)}%`, background: T.sold }} /></div></td></tr>
              ))}
            </tbody>
          </table>
          <div className="rp-callout">
            <ul>
              <li>Her <b>{(s.deals / Math.max(1, r.cancellations.total)).toFixed(1)}</b> aktiviteden biri iptal; iptaller ortalama <b>{r.cancellations.avg_days.toFixed(0)} gün</b> içinde geliyor.</li>
              <li>{r.cancellations.with_deposit} iptal dosyasında toplam <b>{money(r.cancellations.refunds)}</b> ödeme alınmıştı — iade takibi finansın gündeminde olmalı.</li>
              <li>İptal nedeni {r.cancellations.by_reason[0] ? <><b>{r.cancellations.by_reason[0].label}</b> ({pctS(r.cancellations.by_reason[0].count, r.cancellations.total, 0)})</> : "—"}; "Diğer" seçilenler için açıklama zorunlu tutulmalı.</li>
            </ul>
          </div>
        </section>

        {/* ------------------------------------------------ 14 definitions */}
        <section className="rp-page">
          <PageHead n="15" title="Tanımlar ve yöntem" sub="Rakamların nereden geldiği" />
          <dl className="rp-def">
            <dt>Açık satış</dt><dd>İptal edilmemiş her aktivite. Durum Prism'in hareket tipidir: <b>Rezerve</b> (302) veya <b>Satıldı</b> (303). Sözleşme bayrağı hiçbir hesaba girmez.</dd>
            <dt>İptal</dt><dd>T_AktiviteIptal'de kaydı olan aktivite. Bedeli, m²'si ve tahsilatı hiçbir toplamda yer almaz; alınan ödemeler "iade bekleyen" olarak ayrı gösterilir.</dd>
            <dt>Toplam bedel</dt><dd>Açık satışların Prism'deki satış fiyatı toplamı (T_Aktivite.SatisFiyati). Liste fiyatı T_AktiviteGayrimenkul.ListeFiyati; indirim ikisinin farkıdır.</dd>
            <dt>Tahsilat</dt><dd>T_CariAlacak makbuzları (işlem yeri 551), yalnızca açık satışlarda. Prism panosundaki "Tahsilat" ile birebir aynıdır.</dd>
            <dt>Tahsil edilecek</dt><dd>Toplam bedel − tahsilat. Prism'in "Kalan" değeriyle aynıdır.</dd>
            <dt>Peşinat / vadeli</dt><dd>Aktivitedeki plan alanları (PlanPesinatToplam, PlanVadeli, PlanBanka). Peşinat ödemesi, makbuzların peşinat borç satırlarına eşlenmesinden (T_CariAlacakBorcKapatma) gelir.</dd>
            <dt>Vade planı</dt><dd>T_CariBorc'taki tutarı sıfırdan büyük satırlar, vade ayına göre; ödenen kısım kapatma kayıtlarından.</dd>
            <dt>Stok durumu</dt><dd>Her dairenin T_Gayrimenkul.SonHareketTipi alanı: Satılık (boşta), Rezerve, Satıldı, Satışa Kapalı. Stok liste değeri boştaki dairelerin 1 no'lu fiyat listesi.</dd>
            <dt>m²</dt><dd>Brüt m² (T_Gayrimenkul.BrutM2) ve net m² (NetM2), her daire son hareketine göre sayılır. Satıldı ₺/m² = Satıldı dosyaların bedeli ÷ brüt m²'si; rezerve aynı şekilde; "boşta liste ₺/m²" = boş dairelerin liste fiyatı ÷ brüt m². Aylık m² rezerve tarihine göre.</dd>
            <dt>Satış personeli</dt><dd>T_Aktivite.SatisPersonelID1. Yalnızca en az bir açık satışı olan kişiler listelenir; iptal ayrı sütundur.</dd>
            <dt>Acenta</dt><dd>T_Aktivite.Araci1AcentaID. Komisyon T_AktivitePrim'den (girilmişse). "Doğrudan" = acenta alanı boş.</dd>
            <dt>Müşteri kaynağı</dt><dd>T_Aktivite.MusteriKaynagiID_1 → Tn_MusteriKaynagi. Müşteri havuzunun kaynağı ise T_MusteriDetay üzerinden.</dd>
            <dt>Reklam kaynaklı satış</dt><dd>Rezerve tarihi {date(r.ad_sales.since)} ve sonrası olan, acenta alanı boş aktiviteler. Aktivite tarihi kullanılmaz: Prism onu satışa dönüşte yeniden yazar. Müşteri Kaynağı etiketi hesaba katılmaz.</dd>
            <dt>Zaman</dt><dd>Aktiviteler rezerve tarihine (boşsa aktivite tarihine; Prism aktivite tarihini her harekette yeniden yazar), iptaller iptal tarihine, makbuzlar makbuz tarihine göre. Haftalar pazartesi başlar.</dd>
            <dt>Kaynak</dt><dd>Prism PRISM_PLUS veritabanı, canlı bağlantı, salt-okunur. Rapor üretim zamanı her sayfanın altındadır.</dd>
          </dl>
        </section>

        <footer className="rp-foot">
          <span>{footer}</span>
          <span className="stamp">Oluşturma: {stamp} · Prism canlı verisi, salt-okunur</span>
          {BRAND.mark && <img src={BRAND.mark} alt={BRAND.company} />}
        </footer>
      </article>
    </div>
  );
}

/* ---------- derived views the page needs but the API does not ship ---------- */
function derive(r: FullReport) {
  const s = r.summary;
  const days = r.daily.filter((x) => x.date >= "2026-01-01");
  let cd = 0, cr = 0;
  const cumDeals = days.map((x) => (cd += x.reserved + x.sold));
  const cumReceipts = days.map((x) => (cr += x.receipts));

  const wk = new Map<string, { week: string; reserved: number; sold: number; cancelled: number; revenue: number; receipts: number }>();
  for (const x of days) {
    const k = isoWeekStart(x.date);
    const w = wk.get(k) ?? { week: k, reserved: 0, sold: 0, cancelled: 0, revenue: 0, receipts: 0 };
    w.reserved += x.reserved; w.sold += x.sold; w.cancelled += x.cancelled; w.revenue += x.revenue; w.receipts += x.receipts;
    wk.set(k, w);
  }
  const weeks = [...wk.values()].sort((a, b) => a.week.localeCompare(b.week));

  const mo = new Map<string, { month: string; reserved: number; sold: number; cancelled: number; revenue: number; receipts: number }>();
  for (const x of days) {
    const k = x.date.slice(0, 7);
    const m = mo.get(k) ?? { month: k, reserved: 0, sold: 0, cancelled: 0, revenue: 0, receipts: 0 };
    m.reserved += x.reserved; m.sold += x.sold; m.cancelled += x.cancelled; m.revenue += x.revenue; m.receipts += x.receipts;
    mo.set(k, m);
  }
  const months = [...mo.values()].sort((a, b) => a.month.localeCompare(b.month));

  const thisMonth = new Date().toISOString().slice(0, 7);
  const yr = new Map<string, { year: string; count: number; due: number; paid: number; remaining: number }>();
  for (const m of r.schedule) {
    const k = m.month.slice(0, 4);
    const y = yr.get(k) ?? { year: k, count: 0, due: 0, paid: 0, remaining: 0 };
    y.count += m.count; y.due += m.due; y.paid += m.paid; y.remaining += m.remaining;
    yr.set(k, y);
  }
  const years = [...yr.values()].sort((a, b) => a.year.localeCompare(b.year));
  const schedule12 = r.schedule.filter((m) => m.month >= thisMonth).slice(0, 36); // contracts run 36 months
  const overdue = r.schedule.filter((m) => m.month < thisMonth).reduce((n, m) => n + m.remaining, 0);

  const direct = r.agencies.find((a) => a.agency.startsWith("Doğrudan"));
  const agencyOnly = r.agencies.filter((a) => !a.agency.startsWith("Doğrudan") && a.revenue > 0);

  const topCollector = [...r.agents].sort((a, b) => b.collected - a.collected)[0];
  const worstCancel = [...r.agents].filter((a) => a.cancelled + a.reserved + a.sales >= 5).sort((a, b) => pct(b.cancelled, b.cancelled + b.reserved + b.sales) - pct(a.cancelled, a.cancelled + a.reserved + a.sales))[0];
  const bestType = [...r.types].sort((a, b) => b.sold + b.reserved - (a.sold + a.reserved))[0];
  const slowType = [...r.types].sort((a, b) => b.stock_list - a.stock_list)[0];
  const priciest = [...r.types].filter((t) => t.price_per_m2 > 0).sort((a, b) => b.price_per_m2 - a.price_per_m2)[0];

  const stockTop = [...r.m2.by_type].sort((a, b) => b.m2_available - a.m2_available)[0];
  const blocksWithM2 = r.m2.by_block.filter((b) => b.m2_total > 0);
  const blockTopShare = [...blocksWithM2].sort((a, b) => pct(b.m2_sold + b.m2_reserved, b.m2_total) - pct(a.m2_sold + a.m2_reserved, a.m2_total))[0];
  const blockMostAvail = [...blocksWithM2].sort((a, b) => b.m2_available - a.m2_available)[0];
  const custWeeks = r.customers.by_week.filter((w) => w.label >= "2026-01-01");
  const newLast4w = custWeeks.slice(-4).reduce((n, w) => n + w.count, 0);

  const busiest = [...months].sort((a, b) => b.reserved + b.sold - (a.reserved + a.sold))[0];
  const bestWeek = [...weeks].sort((a, b) => b.reserved + b.sold - (a.reserved + a.sold))[0];
  const insights = [
    `<b>${num(s.live)} açık satış</b> (${s.sold} satıldı, ${s.reserved} rezerve), toplam bedel <b>${money(s.pipeline_value)}</b>; bunun <b>${pctS(s.collected, s.pipeline_value)}</b>'i (${money(s.collected)}) tahsil edildi, <b>${money(s.to_collect)}</b> bekliyor.`,
    busiest ? `En yoğun ay <b>${monthName(busiest.month)}</b>: ${busiest.reserved + busiest.sold} açık satış, ${moneyShort(busiest.revenue)}; en yoğun hafta ${bestWeek ? dayShort(bestWeek.week) + " haftası (" + (bestWeek.reserved + bestWeek.sold) + " açık satış)" : "—"}.` : "",
    `Stok: <b>${s.units_available} boş daire</b> (${num(s.m2_available)} m²), liste değeri <b>${moneyShort(s.stock_value)}</b>; doluluk ${pctS(s.units_sold + s.units_reserved, s.units, 0)}.`,
    `m²: <b>${num(s.m2_sold)} m² satıldı</b>, <b>${num(s.m2_reserved)} m² rezerve</b>, ${num(s.m2_available)} m² boşta (${num(s.m2_total)} m² brüt toplam). Gerçekleşen ₺/m² satıldıda ${moneyShort(r.m2.sold_per_m2)}, rezervede ${moneyShort(r.m2.res_per_m2)}; boş stok liste ile ${moneyShort(r.m2.ask_per_m2)}/m² istiyor.`,
    `Peşinat planının <b>${pctS(r.down_payment.paid, r.down_payment.planned, 0)}</b>'i ödendi; vade planında ${monthName(thisMonth)} öncesinden <b>${money(overdue)}</b> gecikmiş taksit var.`,
    `Reklam kaynaklı (${date(r.ad_sales.since)} sonrası, acentasız): <b>${r.ad_sales.reserved + r.ad_sales.sold} açık satış</b>, ${moneyShort(r.ad_sales.revenue)} bedel, ${moneyShort(r.ad_sales.collected)} tahsilat; dönemin ${pctS(r.ad_sales.count, r.ad_sales.all_since, 0)}'i.`,
    `İptal oranı <b>${pctS(s.cancelled, s.deals)}</b> (${s.cancelled} aktivite, ${moneyShort(r.cancellations.value)}); ortalama ${r.cancellations.avg_days.toFixed(0)} gün içinde. İade bekleyen ${money(s.refunds_owed)}.`,
    `Acenta üzerinden ${s.via_agency} açık satış (${pctS(s.via_agency, s.live, 0)}); en büyük acenta <b>${agencyOnly[0]?.agency ?? "—"}</b> (${moneyShort(agencyOnly[0]?.revenue || 0)}).`,
  ].filter(Boolean);

  return { days, cumDeals, cumReceipts, weeks, months, schedule12, overdue, years, direct, agencyOnly, topCollector, worstCancel, bestType, slowType, priciest, custWeeks, newLast4w, insights, stockTop, blockTopShare, blockMostAvail };
}

export type { LabeledCount, LabeledMoney };
