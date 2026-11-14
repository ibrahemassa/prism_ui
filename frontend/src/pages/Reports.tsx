import { api, money } from "../api";
import { Card, PageHead, Loading, ErrorBox, useApi } from "../ui/kit";

// One palette for deal/unit state across the page: Prism's hareket tipi.
const TONE = { sold: "#4f46e5", reserved: "#f59e0b", available: "#cbd5e1", closed: "#64748b", cancelled: "#e11d48" };

function Stat({ label, value, sub, tone = "#4f46e5" }: { label: string; value: string; sub?: string; tone?: string }) {
  return (
    <Card className="p-5">
      <div className="text-sm text-slate-500 font-medium">{label}</div>
      <div className="text-2xl font-extrabold mt-1 tabular" style={{ color: tone }}>{value}</div>
      {sub && <div className="text-xs text-slate-400 mt-1">{sub}</div>}
    </Card>
  );
}

/** A row of stacked segments: reserved (amber) then sold (indigo) on one bar. */
function Stacked({ title, note, rows, max, legend }: {
  title: string;
  note?: string;
  rows: { label: string; segments: { key: keyof typeof TONE; value: number }[]; right: React.ReactNode }[];
  max: number;
  legend: { key: keyof typeof TONE; label: string }[];
}) {
  return (
    <Card className="p-5">
      <div className="flex items-baseline justify-between mb-1">
        <div className="font-bold">{title}</div>
        <div className="flex gap-3 text-xs text-slate-500">
          {legend.map((l) => (
            <span key={l.key} className="inline-flex items-center gap-1.5">
              <span className="inline-block w-2.5 h-2.5 rounded-sm" style={{ background: TONE[l.key] }} />{l.label}
            </span>
          ))}
        </div>
      </div>
      {note && <div className="text-xs text-slate-400 mb-4">{note}</div>}
      <div className="space-y-3">
        {rows.map((r) => (
          <div key={r.label}>
            <div className="flex justify-between text-sm mb-1">
              <span className="text-slate-600 truncate pr-2">{r.label}</span>
              <span className="tabular">{r.right}</span>
            </div>
            <div className="h-2.5 rounded-full bg-slate-100 overflow-hidden flex">
              {r.segments.map((sg) => (
                <div key={sg.key} className="h-full" style={{ width: `${(sg.value / max) * 100}%`, background: TONE[sg.key] }} title={`${sg.value}`} />
              ))}
            </div>
          </div>
        ))}
        {rows.length === 0 && <div className="text-sm text-slate-400">Veri yok</div>}
      </div>
    </Card>
  );
}

const monthName = (m: string) => {
  const [y, mm] = m.split("-").map(Number);
  return new Date(y, (mm || 1) - 1, 1).toLocaleDateString("tr-TR", { month: "long", year: "numeric" });
};

export default function Reports() {
  const { data: r, error, loading } = useApi(() => api.reports(), []);
  if (loading) return <Loading />;
  if (error) return <ErrorBox error={error} />;
  const f = r!.funnel, c = r!.collection;
  const open = f.sale + f.reserved;
  const agentMax = Math.max(1, ...r!.by_agent.map((a) => a.revenue));
  const monthMax = Math.max(1, ...r!.sales_by_month.map((m) => m.count));
  const blockMax = Math.max(1, ...r!.by_block.map((b) => b.count));
  const typeMax = Math.max(1, ...r!.by_type.map((t) => t.total));
  const Sum = (k: "reserved" | "sales" | "cancelled" | "revenue" | "collected") => r!.by_agent.reduce((n, a) => n + a[k], 0);

  return (
    <>
      <PageHead
        title="Raporlar"
        subtitle="Satış ve tahsilat analizi · iptaller hariç"
        right={
          <a
            href="/raporlar/pdf"
            target="_blank"
            rel="noopener"
            className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-500 text-white text-sm font-semibold hover:bg-indigo-600"
          >
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M14 3H6a2 2 0 00-2 2v14a2 2 0 002 2h12a2 2 0 002-2V9z" /><path d="M14 3v6h6" /><path d="M12 12v6m-3-3l3 3 3-3" />
            </svg>
            PDF olarak indir
          </a>
        }
      />

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-5">
        <Stat label="Açık satışlar" value={open.toString()} sub={`${f.sale} satıldı · ${f.reserved} rezerve`} tone="#059669" />
        <Stat label="İptal oranı" value={`%${f.cancel_rate.toFixed(1)}`} sub={`${f.cancelled} iptal · ${f.total} aktivite`} tone="#e11d48" />
        <Stat label="Tahsil edilen" value={money(c.collected)} sub={`%${c.collect_pct.toFixed(1)} · iptaller hariç`} tone="#059669" />
        <Stat label="Tahsil edilecek" value={money(c.to_collect)} sub="açık satışlardan kalan" tone="#e11d48" />
      </div>

      <div className="grid md:grid-cols-2 gap-5 mb-5">
        <Card className="p-5">
          <div className="font-bold mb-4">Aktivite durumu</div>
          <div className="space-y-3">
            {[
              { label: "Satıldı", val: f.sale, tone: TONE.sold },
              { label: "Rezerve", val: f.reserved, tone: TONE.reserved },
              { label: "Görüşme", val: f.negotiation, tone: TONE.closed },
              { label: "İptal", val: f.cancelled, tone: TONE.cancelled },
            ].map((x) => (
              <div key={x.label}>
                <div className="flex justify-between text-sm mb-1">
                  <span className="text-slate-600">{x.label}</span>
                  <span className="font-semibold tabular">{x.val}<span className="text-slate-400 font-normal"> · %{f.total ? ((x.val / f.total) * 100).toFixed(0) : 0}</span></span>
                </div>
                <div className="h-3 rounded-full bg-slate-100 overflow-hidden">
                  <div className="h-full rounded-full" style={{ width: `${(x.val / Math.max(1, f.total)) * 100}%`, background: x.tone }} />
                </div>
              </div>
            ))}
          </div>
        </Card>
        <Card className="p-5">
          <div className="font-bold mb-1">Tahsilat</div>
          <div className="text-xs text-slate-400 mb-4">İptal edilen aktiviteler hariç — Prism ile birebir</div>
          <div className="space-y-3 text-sm">
            <div className="flex justify-between"><span className="text-slate-500">Toplam bedel ({open} açık satış)</span><span className="font-semibold tabular">{money(c.pipeline_value)}</span></div>
            <div className="flex justify-between"><span className="text-slate-500">Tahsil edilen</span><span className="font-semibold tabular text-emerald-600">{money(c.collected)}</span></div>
            <div className="flex justify-between"><span className="text-slate-500">Kapora tahsil</span><span className="font-semibold tabular text-sky-600">{money(c.kapora_paid)}<span className="text-slate-400 font-normal"> · {c.deals_with_kapora} aktivite</span></span></div>
            <div className="flex justify-between"><span className="text-slate-500">Tahsil edilecek</span><span className="font-semibold tabular text-rose-600">{money(c.to_collect)}</span></div>
            <div className="h-3 rounded-full bg-slate-100 overflow-hidden mt-2">
              <div className="h-full bg-emerald-500" style={{ width: `${Math.min(100, c.collect_pct)}%` }} />
            </div>
            <div className="text-xs text-slate-400">%{c.collect_pct.toFixed(1)} tahsil edildi</div>
            {c.refunds_owed > 0 && (
              <div className="flex justify-between pt-3 mt-1 border-t border-slate-100">
                <span className="text-slate-500">İptal edilenlerdeki ödemeler (iade)</span>
                <span className="font-semibold tabular text-slate-500">{money(c.refunds_owed)}</span>
              </div>
            )}
          </div>
        </Card>
      </div>

      <Card className="overflow-hidden mb-5">
        <div className="flex items-baseline justify-between px-5 py-4 border-b border-slate-100">
          <div className="font-bold">Satış personeli</div>
          <div className="text-xs text-slate-400">açık satışlar · iptal ayrı sayılır · tahsilat oranı = tahsilat ÷ bedel</div>
        </div>
        <div className="overflow-x-auto scrolly">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-slate-500 border-b border-slate-100">
                <th className="px-5 py-3 font-semibold">Danışman</th>
                <th className="px-5 py-3 font-semibold text-right">Rezerve</th>
                <th className="px-5 py-3 font-semibold text-right">Satıldı</th>
                <th className="px-5 py-3 font-semibold text-right">İptal</th>
                <th className="px-5 py-3 font-semibold text-right">Toplam bedel</th>
                <th className="px-5 py-3 font-semibold w-[22%]"></th>
                <th className="px-5 py-3 font-semibold text-right">Tahsilat</th>
                <th className="px-5 py-3 font-semibold text-right whitespace-nowrap">Tahsilat oranı</th>
              </tr>
            </thead>
            <tbody>
              {r!.by_agent.map((a) => (
                <tr key={a.agent} className="rowhover border-b border-slate-50 last:border-0">
                  <td className="px-5 py-3 font-medium whitespace-nowrap">{a.agent}</td>
                  <td className="px-5 py-3 text-right tabular text-amber-600">{a.reserved || "—"}</td>
                  <td className="px-5 py-3 text-right tabular text-indigo-600">{a.sales || "—"}</td>
                  <td className="px-5 py-3 text-right tabular text-rose-500">{a.cancelled || "—"}</td>
                  <td className="px-5 py-3 text-right tabular font-semibold whitespace-nowrap">{money(a.revenue)}</td>
                  <td className="px-5 py-3">
                    <div className="h-2 rounded-full bg-slate-100 overflow-hidden">
                      <div className="h-full rounded-full bg-emerald-500" style={{ width: `${(a.revenue / agentMax) * 100}%` }} />
                    </div>
                  </td>
                  <td className="px-5 py-3 text-right tabular text-emerald-600 whitespace-nowrap">{money(a.collected)}</td>
                  <td className="px-5 py-3 text-right tabular text-slate-500">{a.revenue ? `%${((a.collected / a.revenue) * 100).toFixed(1)}` : "—"}</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr className="border-t border-slate-100 bg-slate-50 font-semibold">
                <td className="px-5 py-3">Toplam</td>
                <td className="px-5 py-3 text-right tabular">{Sum("reserved")}</td>
                <td className="px-5 py-3 text-right tabular">{Sum("sales")}</td>
                <td className="px-5 py-3 text-right tabular">{Sum("cancelled")}</td>
                <td className="px-5 py-3 text-right tabular">{money(Sum("revenue"))}</td>
                <td />
                <td className="px-5 py-3 text-right tabular">{money(Sum("collected"))}</td>
                <td className="px-5 py-3 text-right tabular">%{c.collect_pct.toFixed(1)}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </Card>

      <div className="grid md:grid-cols-2 gap-5 mb-5">
        <Stacked
          title="Aylık açık satışlar"
          note="Rezerve tarihine göre"
          max={monthMax}
          legend={[{ key: "reserved", label: "Rezerve" }, { key: "sold", label: "Satıldı" }]}
          rows={r!.sales_by_month.map((m) => ({
            label: monthName(m.month),
            segments: [{ key: "reserved", value: m.reserved }, { key: "sold", value: m.sold }],
            right: <><b>{m.count}</b><span className="text-slate-400"> · {money(m.revenue)}</span></>,
          }))}
        />
        <Stacked
          title="Bloklara göre açık satışlar"
          max={blockMax}
          legend={[{ key: "reserved", label: "Rezerve" }, { key: "sold", label: "Satıldı" }]}
          rows={r!.by_block.map((b) => ({
            label: `Blok ${b.label}`,
            segments: [{ key: "reserved", value: b.reserved }, { key: "sold", value: b.sold }],
            right: <><b>{b.count}</b><span className="text-slate-400"> · {money(b.revenue)}</span></>,
          }))}
        />
      </div>

      <Stacked
        title="Daire tipine göre durum"
        note="Tüm stok — her daire Prism'deki son hareketiyle"
        max={typeMax}
        legend={[{ key: "sold", label: "Satıldı" }, { key: "reserved", label: "Rezerve" }, { key: "available", label: "Boşta" }, { key: "closed", label: "Satışa kapalı" }]}
        rows={r!.by_type.map((t) => ({
          label: t.label,
          segments: [{ key: "sold", value: t.sold }, { key: "reserved", value: t.reserved }, { key: "available", value: t.available }, { key: "closed", value: t.closed }],
          right: (
            <>
              <span style={{ color: TONE.sold }}>{t.sold}</span>
              <span className="text-slate-300"> / </span>
              <span style={{ color: TONE.reserved }}>{t.reserved}</span>
              <span className="text-slate-300"> / </span>
              <span className="text-slate-500">{t.available}</span>
              {t.closed > 0 && <span className="text-slate-400"> · {t.closed} kapalı</span>}
              <span className="text-slate-400"> · {t.total} daire</span>
            </>
          ),
        }))}
      />
    </>
  );
}
