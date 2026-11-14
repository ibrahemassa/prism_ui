import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { api, money, date, hit } from "../api";
import { Card, PageHead, Loading, ErrorBox, SearchBox } from "../ui/kit";
import {
  STATUS, MISSING, NOTES, BASIS, STAGE, ALL, worth, inPeriod, sumBy, moneyCur, rateLabel, monthLabel, addMonth, today, exportUrl, printUrl,
  type CommissionLine, type CommissionsPayload, type CommissionStatus,
} from "../commissions";
import CommissionAgents from "./CommissionAgents";

type View = "month" | "waiting" | "overdue" | "paid" | "cancelled" | "all";

const payable = (l: CommissionLine) => l.status === "due" || l.status === "overdue" || l.status === "scheduled";

function CommissionChip({ status }: { status: CommissionStatus }) {
  const s = STATUS[status];
  return (
    <span className="chip whitespace-nowrap" style={{ background: s.bg, color: s.fg }}>
      <span style={{ width: 6, height: 6, borderRadius: 999, background: s.fg }} />
      {s.label}
    </span>
  );
}

function Kpi({ label, value, sub, tone }: { label: string; value: string; sub?: string; tone?: "good" | "warn" | "bad" }) {
  const color = tone === "good" ? "text-emerald-600" : tone === "warn" ? "text-amber-600" : tone === "bad" ? "text-rose-600" : "";
  return (
    <Card className="p-4">
      <div className="text-xs font-semibold text-slate-500 uppercase tracking-wide">{label}</div>
      <div className={`text-xl font-extrabold mt-1 tabular ${color}`}>{value}</div>
      {sub && <div className="text-xs text-slate-500 mt-0.5">{sub}</div>}
    </Card>
  );
}

function DownCell({ l }: { l: CommissionLine }) {
  if (l.down_paid_at) {
    return (
      <div className="whitespace-nowrap">
        <span className="text-emerald-600 font-semibold">✓</span> <span className="tabular">{date(l.down_paid_at)}</span>
      </div>
    );
  }
  const pct = l.down_required > 0 ? Math.min(100, (l.down_paid / l.down_required) * 100) : 0;
  return (
    <div className="min-w-32" title={`${money(l.down_paid, l.currency)} / ${money(l.down_required, l.currency)}`}>
      <div className="h-1.5 rounded-full bg-slate-100 overflow-hidden">
        <div className="h-full bg-amber-500" style={{ width: `${pct}%` }} />
      </div>
      <div className="text-[11px] text-slate-500 mt-1 tabular whitespace-nowrap">
        %{pct.toFixed(0)} · {money(l.down_paid, l.currency)} / {money(l.down_required, l.currency)}
      </div>
    </div>
  );
}

function ContractCell({ l }: { l: CommissionLine }) {
  if (l.contract_at)
    return (
      <span className="whitespace-nowrap">
        <span className="text-emerald-600 font-semibold">✓</span> <span className="tabular">{date(l.contract_at)}</span>
      </span>
    );
  if (l.contracted) return <span className="text-emerald-600 font-semibold whitespace-nowrap">✓ <span className="text-slate-400 font-normal">tarihsiz</span></span>;
  return <span className="text-rose-600 font-medium">Yok</span>;
}

export default function Commissions() {
  const nav = useNavigate();
  const [params, setParams] = useSearchParams();
  const [data, setData] = useState<CommissionsPayload | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [view, setView] = useState<View>("month");
  const [q, setQ] = useState("");
  const [payDate, setPayDate] = useState(today());
  const [busy, setBusy] = useState(false);
  const [flash, setFlash] = useState<{ ok: boolean; text: string } | null>(null);
  const [settings, setSettings] = useState(false);

  const load = () => api.commissions().then((d) => (setData(d), setError(null))).catch((e) => setError(String(e.message ?? e)));
  useEffect(() => {
    load();
  }, []);

  const month = params.get("ay") || data?.month || "";
  const all = month === ALL;
  const agent = params.get("danisman") || "";
  const setParam = (k: string, v: string) =>
    setParams((p) => {
      const n = new URLSearchParams(p);
      if (v) n.set(k, v);
      else n.delete(k);
      return n;
    }, { replace: true });

  const scoped = useMemo(() => (data ? data.lines.filter((l) => !agent || l.agent_id === agent) : []), [data, agent]);
  const views = useMemo(() => {
    const v: Record<View, CommissionLine[]> = {
      month: scoped.filter((l) => inPeriod(l, month)),
      waiting: scoped.filter((l) => l.status === "waiting"),
      overdue: scoped.filter((l) => l.status === "overdue"),
      paid: scoped.filter((l) => l.status === "paid"),
      cancelled: scoped.filter((l) => l.status === "cancelled" || l.status === "clawback"),
      all: scoped,
    };
    return v;
  }, [scoped, month]);
  const rows = useMemo(
    () => views[view].filter((l) => hit(q, l.agent, l.customer, l.unit, l.unit_type, l.agency, STATUS[l.status].label)),
    [views, view, q],
  );

  if (error && !data) return <ErrorBox error={error} />;
  if (!data) return <Loading />;

  const monthLines = views.month;
  const monthTotal = sumBy(monthLines, worth);
  const monthPaid = sumBy(monthLines.filter((l) => l.payout), worth);
  const monthUnpaid = sumBy(monthLines.filter((l) => !l.payout), worth);
  const waitingSum = sumBy(views.waiting, (l) => ({ amount: l.amount, currency: l.currency }));
  const overdueSum = sumBy(views.overdue, worth);
  const clawback = scoped.filter((l) => l.status === "clawback");
  const noRate = data.agents.filter((a) => !a.rate);
  // bulk marking covers what is billed by now; a future month's line is paid one by one
  const toPay = rows.filter((l) => l.status === "due" || l.status === "overdue");
  const basisOf = Object.fromEntries(data.agents.map((a) => [a.personnel_id, a.basis]));

  const act = async (ids: string[], paid: boolean, confirmText?: string) => {
    if (confirmText && !confirm(confirmText)) return;
    setBusy(true);
    setFlash(null);
    try {
      setData(await api.commissionPaid(ids, paid, payDate));
      setFlash({ ok: true, text: paid ? `${ids.length} komisyon ödendi olarak işaretlendi (${date(payDate)})` : "Ödeme kaydı kaldırıldı" });
    } catch (e) {
      setFlash({ ok: false, text: (e as Error).message });
    } finally {
      setBusy(false);
    }
  };

  const tabs: { key: View; label: string }[] = [
    { key: "month", label: all ? "Tüm faturalar" : `${monthLabel(month)} faturası` },
    { key: "waiting", label: "Bekleyen" },
    { key: "overdue", label: "Gecikmiş" },
    { key: "paid", label: "Ödenen" },
    { key: "cancelled", label: "İptal" },
    { key: "all", label: "Tümü" },
  ];

  return (
    <>
      <PageHead
        title="Komisyonlar"
        subtitle="Peşinat tamamen ödendi + sözleşme imzalandı → komisyon ertesi ay faturalanır"
        right={
          <div className="flex items-center gap-2 flex-wrap justify-end">
            <div className="flex items-center gap-1 bg-white border border-slate-200 rounded-xl p-1">
              {all ? (
                <button onClick={() => setParam("ay", "")} className="px-3 h-8 rounded-lg text-sm font-medium text-slate-600 hover:bg-slate-50" title="Aylık görünüme dön">
                  Aylık
                </button>
              ) : (
                <>
                  <button onClick={() => setParam("ay", addMonth(month, -1))} className="w-8 h-8 rounded-lg hover:bg-slate-50 text-slate-600" aria-label="Önceki ay">‹</button>
                  <div className="px-2 text-sm font-semibold min-w-28 text-center">{monthLabel(month)}</div>
                  <button onClick={() => setParam("ay", addMonth(month, 1))} className="w-8 h-8 rounded-lg hover:bg-slate-50 text-slate-600" aria-label="Sonraki ay">›</button>
                  {month !== data.month && (
                    <button onClick={() => setParam("ay", "")} className="px-2 h-8 rounded-lg text-xs font-semibold text-indigo-600 hover:bg-slate-50">Bu ay</button>
                  )}
                </>
              )}
              <button
                onClick={() => setParam("ay", all ? "" : ALL)}
                className={`px-3 h-8 rounded-lg text-sm font-medium ${all ? "bg-indigo-500 text-white" : "text-slate-600 hover:bg-slate-50"}`}
                title="Bütün ayların komisyonları"
              >
                Tüm zamanlar
              </button>
            </div>
            <a
              href={exportUrl(month, agent)}
              download
              className="px-3 py-2 rounded-xl text-sm font-semibold border border-slate-300 text-slate-700 hover:bg-slate-50"
              title={`${monthLabel(month)} komisyon raporunu Excel olarak indir`}
            >
              Excel
            </a>
            <a
              href={printUrl(month, agent)}
              target="_blank"
              rel="noreferrer"
              className="px-3 py-2 rounded-xl text-sm font-semibold border border-slate-300 text-slate-700 hover:bg-slate-50"
              title={`${monthLabel(month)} komisyon raporunu PDF olarak indir`}
            >
              PDF
            </a>
            <button onClick={() => setSettings(true)} className="px-3 py-2 rounded-xl text-sm font-semibold bg-indigo-600 text-white hover:bg-indigo-700">
              Danışmanlar ve kurallar
            </button>
          </div>
        }
      />

      {noRate.length > 0 && (
        <Card className="p-4 mb-4 border-amber-200 bg-amber-50 text-amber-800 text-sm">
          Komisyon oranı girilmemiş: <b>{noRate.map((a) => a.name).join(", ")}</b> — tutarlar sıfır görünür.{" "}
          <button onClick={() => setSettings(true)} className="underline font-semibold">Oranları gir</button>
        </Card>
      )}

      {data.agents.length === 0 ? (
        <Card className="p-10 text-center text-slate-500">
          Takip edilen danışman yok.{" "}
          <button onClick={() => setSettings(true)} className="text-indigo-600 font-semibold underline">Danışman ekle</button>
        </Card>
      ) : (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-4">
          {data.agents.map((a) => {
            const mine = data.lines.filter((l) => l.agent_id === a.personnel_id);
            const inMonth = mine.filter((l) => inPeriod(l, month));
            const waiting = mine.filter((l) => l.status === "waiting").length;
            const unpaid = sumBy(inMonth.filter((l) => !l.payout), worth);
            const active = agent === a.personnel_id;
            return (
              <button
                key={a.personnel_id}
                onClick={() => setParam("danisman", active ? "" : a.personnel_id)}
                className={`card p-4 text-left transition ${active ? "ring-2 ring-indigo-500" : "hover:shadow-md"}`}
                title={active ? "Filtreyi kaldır" : `Yalnızca ${a.name}`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="font-semibold truncate">{a.name}</div>
                  <span className="chip shrink-0" style={a.rate ? { background: "#eef2ff", color: "#4338ca" } : { background: "#fffbeb", color: "#b45309" }}>
                    {a.rate ? rateLabel(a.rate) : "oran yok"}
                  </span>
                </div>
                <div className="text-[11px] text-slate-500 mt-0.5">{BASIS[a.basis]} üzerinden{a.delay_months !== 1 ? ` · ${a.delay_months} ay sonra` : ""}</div>
                <div className="text-lg font-extrabold mt-2 tabular">{moneyCur(sumBy(inMonth, worth))}</div>
                <div className="text-xs text-slate-500">
                  {inMonth.length} satış · {monthLabel(month)}
                  {Object.keys(unpaid).length > 0 && inMonth.some((l) => l.payout) ? ` · ödenecek ${moneyCur(unpaid)}` : ""}
                </div>
                <div className="text-xs mt-1 text-amber-700">{waiting > 0 ? `${waiting} satış şart bekliyor` : <span className="text-slate-400">Bekleyen yok</span>}</div>
                {a.rule && <div className="text-[11px] text-slate-500 mt-2 line-clamp-2" title={a.rule}>{a.rule}</div>}
              </button>
            );
          })}
        </div>
      )}

      <div className="grid grid-cols-2 lg:grid-cols-5 gap-4 mb-6">
        <Kpi label={`${monthLabel(month)} komisyonu`} value={moneyCur(monthTotal)} sub={`${monthLines.length} satış`} />
        <Kpi label="Ödenen" value={moneyCur(monthPaid)} sub={`${monthLines.filter((l) => l.payout).length} satış`} tone="good" />
        <Kpi label="Ödenecek" value={moneyCur(monthUnpaid)} sub={`${monthLines.filter((l) => !l.payout).length} satış`} />
        <Kpi label="Şart bekleyen" value={moneyCur(waitingSum)} sub={`${views.waiting.length} satış · peşinat / sözleşme`} tone="warn" />
        <Kpi
          label="Geçmiş aylardan ödenmemiş"
          value={moneyCur(overdueSum)}
          sub={`${views.overdue.length} satış${clawback.length ? ` · ${clawback.length} geri alınacak` : ""}`}
          tone={views.overdue.length || clawback.length ? "bad" : undefined}
        />
      </div>

      <div className="flex items-center gap-2 mb-3 flex-wrap">
        <div className="flex gap-1 bg-white border border-slate-200 rounded-xl p-1 flex-wrap">
          {tabs.map((t) => (
            <button
              key={t.key}
              onClick={() => setView(t.key)}
              className={`px-3 py-1.5 rounded-lg text-sm font-medium ${view === t.key ? "bg-indigo-500 text-white" : "text-slate-600 hover:bg-slate-50"}`}
            >
              {t.label} <span className={`text-xs ${view === t.key ? "text-indigo-100" : "text-slate-400"}`}>{views[t.key].length}</span>
            </button>
          ))}
        </div>
        <SearchBox value={q} onChange={setQ} placeholder="Müşteri, daire, danışman…" className="ml-auto" />
      </div>

      {(toPay.length > 0 || flash) && (
        <div className="flex items-center gap-2 mb-3 flex-wrap text-sm">
          {toPay.length > 0 && (
            <>
              <label className="text-slate-500">Ödeme tarihi</label>
              <input
                type="date"
                value={payDate}
                onChange={(e) => setPayDate(e.target.value || today())}
                className="px-2 py-1.5 rounded-lg border border-slate-200 bg-white text-sm"
              />
              <button
                disabled={busy}
                onClick={() =>
                  act(
                    toPay.map((l) => l.deal_id),
                    true,
                    `Listedeki ${toPay.length} komisyon (${moneyCur(sumBy(toPay, worth))}) ${date(payDate)} tarihiyle ödendi olarak işaretlensin mi?`,
                  )
                }
                className="px-3 py-1.5 rounded-lg font-semibold bg-emerald-600 text-white hover:bg-emerald-700 disabled:opacity-60"
              >
                Listedeki {toPay.length} komisyonu ödendi işaretle
              </button>
            </>
          )}
          {flash && <span className={`ml-2 ${flash.ok ? "text-emerald-700" : "text-rose-600"}`}>{flash.text}</span>}
        </div>
      )}

      <Card className="overflow-hidden">
        <div className="overflow-x-auto scrolly">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-slate-500 border-b border-slate-100">
                <th className="px-4 py-3 font-semibold">Danışman</th>
                <th className="px-4 py-3 font-semibold">Müşteri / Daire</th>
                <th className="px-4 py-3 font-semibold text-right">Satış fiyatı</th>
                <th className="px-4 py-3 font-semibold text-right">Komisyon</th>
                <th className="px-4 py-3 font-semibold">Peşinat</th>
                <th className="px-4 py-3 font-semibold">Sözleşme</th>
                <th className="px-4 py-3 font-semibold">Hak ediş → Fatura</th>
                <th className="px-4 py-3 font-semibold">Durum</th>
                <th className="px-4 py-3 font-semibold text-right"></th>
              </tr>
            </thead>
            <tbody>
              {rows.map((l) => {
                const w = worth(l);
                return (
                  <tr key={l.deal_id} onClick={() => nav(`/satislar/${l.deal_id}`)} className="rowhover border-b border-slate-50 last:border-0 cursor-pointer align-top">
                    <td className="px-4 py-3 text-slate-600 whitespace-nowrap">{l.agent}</td>
                    <td className="px-4 py-3">
                      <div className="font-medium text-indigo-700">{l.customer || "—"}</div>
                      <div className="text-xs text-slate-500 tabular">
                        {l.unit || "—"}
                        {l.unit_type ? ` · ${l.unit_type}` : ""} · {STAGE[l.stage] ?? l.stage}
                        {l.agency ? ` · ${l.agency}` : ""}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-right tabular whitespace-nowrap">{money(l.sale_price, l.currency)}</td>
                    <td className="px-4 py-3 text-right whitespace-nowrap">
                      <div className="font-bold tabular">{money(w.amount, w.currency)}</div>
                      <div className="text-[11px] text-slate-500">
                        {rateLabel(l.rate)} · {basisOf[l.agent_id] === "list_price" && !l.notes.includes("no_list_price") ? "liste" : "satış"} fiyatı
                      </div>
                    </td>
                    <td className="px-4 py-3"><DownCell l={l} /></td>
                    <td className="px-4 py-3"><ContractCell l={l} /></td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      {l.bill_month ? (
                        <>
                          <div className="tabular text-slate-600">{date(l.eligible_at)}</div>
                          <div className="font-semibold">→ {monthLabel(l.bill_month)}</div>
                        </>
                      ) : (
                        <span className="text-slate-400">—</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1.5">
                        <CommissionChip status={l.status} />
                        {l.notes.length > 0 && (
                          <span
                            title={l.notes.map((n) => NOTES[n] ?? n).join("\n")}
                            className="w-5 h-5 grid place-items-center rounded-full bg-amber-100 text-amber-700 text-[11px] font-bold"
                          >
                            !
                          </span>
                        )}
                      </div>
                      {l.missing.length > 0 && <div className="text-[11px] text-amber-700 mt-1">{l.missing.map((m) => MISSING[m] ?? m).join(" · ")}</div>}
                      {l.payout && <div className="text-[11px] text-slate-500 mt-1">Ödendi: {date(l.payout.paid_at)}</div>}
                    </td>
                    <td className="px-4 py-3 text-right whitespace-nowrap" onClick={(e) => e.stopPropagation()}>
                      {payable(l) && (
                        <button
                          disabled={busy}
                          onClick={() => act([l.deal_id], true)}
                          className="px-2.5 py-1 rounded-lg text-xs font-semibold border border-emerald-300 text-emerald-700 hover:bg-emerald-50 disabled:opacity-60"
                          title={`${date(payDate)} tarihiyle ödendi işaretle`}
                        >
                          Ödendi
                        </button>
                      )}
                      {l.status === "paid" && (
                        <button
                          disabled={busy}
                          onClick={() => act([l.deal_id], false, `${l.customer} (${l.unit}) için ödeme kaydı kaldırılsın mı?`)}
                          className="px-2.5 py-1 rounded-lg text-xs text-slate-500 hover:bg-slate-100 disabled:opacity-60"
                        >
                          Geri al
                        </button>
                      )}
                      {l.status === "clawback" && (
                        <button
                          disabled={busy}
                          onClick={() => act([l.deal_id], false, `${l.customer} (${l.unit}): ödenen ${money(w.amount, w.currency)} geri alındı mı? Ödeme kaydı silinecek.`)}
                          className="px-2.5 py-1 rounded-lg text-xs font-semibold border border-rose-300 text-rose-700 hover:bg-rose-50 disabled:opacity-60"
                        >
                          Geri alındı
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })}
              {rows.length === 0 && (
                <tr>
                  <td colSpan={9} className="px-5 py-10 text-center text-slate-400">
                    {view === "month" ? (all ? "Henüz faturalanan komisyon yok" : `${monthLabel(month)} için faturalanacak komisyon yok`) : "Kayıt bulunamadı"}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>

      {settings && <CommissionAgents data={data} onClose={() => setSettings(false)} onSaved={(d) => (setData(d), setFlash({ ok: true, text: "Danışman kuralları kaydedildi" }))} />}
    </>
  );
}
