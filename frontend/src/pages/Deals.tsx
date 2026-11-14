import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api, money, date, unitLabel, hit } from "../api";
import { Card, PageHead, Loading, ErrorBox, StatusChip, SearchBox, useApi } from "../ui/kit";
import type { Deal } from "../types";

const STAGE_TR: Record<string, string> = { sale: "Satış", reserved: "Rezerve", negotiation: "Görüşme", cancelled: "İptal" };

const FILTERS = [
  { key: "all", label: "Tümü" },
  { key: "sale", label: "Satış" },
  { key: "reserved", label: "Rezerve" },
  { key: "negotiation", label: "Görüşme" },
  { key: "cancelled", label: "İptal" },
];

/** Prism stores "no date" as 1900-01-01 (or empty). */
function realDate(s?: string) {
  return s && !s.startsWith("1900") && !s.startsWith("0001") ? s : "";
}

/**
 * The date the deal was opened (RezerveTarihi). Not AktiviteTarihi: the office
 * re-stamps that one whenever anything happens to the deal, so it reads as
 * "last touched", not "sold". Activity/contract dates only fill in when the
 * reservation date is unset.
 */
const saleDate = (d: Deal) => realDate(d.reserved_at) || realDate(d.activity_at) || realDate(d.contract_at);

type SortKey = "sale_date" | "cancelled_at" | "sale_price" | "customer";
type Sort = { key: SortKey; dir: 1 | -1 };

const SORT_VALUE: Record<SortKey, (d: Deal) => string | number> = {
  sale_date: (d) => saleDate(d),
  cancelled_at: (d) => realDate(d.cancelled_at),
  sale_price: (d) => d.sale_price || 0,
  customer: (d) => (d.customer || "").toLocaleLowerCase("tr"),
};

function Th({
  label, k, sort, onSort, right,
}: { label: string; k?: SortKey; sort: Sort; onSort: (k: SortKey) => void; right?: boolean }) {
  if (!k) return <th className={`px-5 py-3 font-semibold ${right ? "text-right" : ""}`}>{label}</th>;
  const active = sort.key === k;
  return (
    <th className={`px-5 py-3 font-semibold ${right ? "text-right" : ""}`}>
      <button
        type="button"
        onClick={() => onSort(k)}
        className={`inline-flex items-center gap-1 hover:text-indigo-600 ${active ? "text-indigo-600" : ""}`}
        title="Sırala"
      >
        {label}
        <span className={`text-[10px] ${active ? "opacity-100" : "opacity-30"}`}>{active && sort.dir === 1 ? "▲" : "▼"}</span>
      </button>
    </th>
  );
}

export default function Deals() {
  const nav = useNavigate();
  const [f, setF] = useState("all");
  const [q, setQ] = useState("");
  const [sort, setSort] = useState<Sort>({ key: "sale_date", dir: -1 });
  const { data, error, loading } = useApi(() => api.deals(), []);

  const onSort = (k: SortKey) =>
    setSort((s) => (s.key === k ? { key: k, dir: s.dir === 1 ? -1 : 1 } : { key: k, dir: k === "customer" ? 1 : -1 }));

  const rows = useMemo(() => {
    if (!data) return [];
    const val = SORT_VALUE[sort.key];
    return data.rows
      .filter(
        (d) =>
          (f === "all" || d.stage === f) &&
          hit(q, d.customer, unitLabel(d.block, d.unit_number), d.unit_type, d.agent, d.agency, STAGE_TR[d.stage]),
      )
      .sort((a, b) => {
        const x = val(a), y = val(b);
        // empty dates always sink to the bottom whichever way we sort
        if (x === "" && y !== "") return 1;
        if (y === "" && x !== "") return -1;
        return (x < y ? -1 : x > y ? 1 : 0) * sort.dir;
      });
  }, [data, f, q, sort]);

  if (loading) return <Loading />;
  if (error) return <ErrorBox error={error} />;
  const showCancel = f === "all" || f === "cancelled";
  return (
    <>
      <PageHead
        title="Satışlar"
        subtitle={q || f !== "all" ? `${rows.length} / ${data!.total} aktivite` : `${data!.total} aktivite`}
        right={
          <div className="flex items-center gap-2">
            <SearchBox value={q} onChange={setQ} placeholder="Müşteri, daire (B2-13), danışman…" />
            <div className="flex gap-1 bg-white border border-slate-200 rounded-xl p-1">
              {FILTERS.map((x) => (
                <button
                  key={x.key}
                  onClick={() => {
                    setF(x.key);
                    if (x.key === "cancelled") setSort({ key: "cancelled_at", dir: -1 });
                    else if (sort.key === "cancelled_at") setSort({ key: "sale_date", dir: -1 });
                  }}
                  className={`px-3 py-1.5 rounded-lg text-sm font-medium ${f === x.key ? "bg-indigo-500 text-white" : "text-slate-600 hover:bg-slate-50"}`}
                >
                  {x.label}
                </button>
              ))}
            </div>
          </div>
        }
      />
      <Card className="overflow-hidden">
        <div className="overflow-x-auto scrolly">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-slate-500 border-b border-slate-100">
                <Th label="Müşteri" k="customer" sort={sort} onSort={onSort} />
                <Th label="Daire" sort={sort} onSort={onSort} />
                <Th label="Tip" sort={sort} onSort={onSort} />
                <Th label="Durum" sort={sort} onSort={onSort} />
                <Th label="Satış Fiyatı" k="sale_price" sort={sort} onSort={onSort} right />
                <Th label="Peşinat" sort={sort} onSort={onSort} right />
                <Th label="Danışman" sort={sort} onSort={onSort} />
                <Th label="Satış tarihi" k="sale_date" sort={sort} onSort={onSort} />
                {showCancel && <Th label="İptal tarihi" k="cancelled_at" sort={sort} onSort={onSort} />}
              </tr>
            </thead>
            <tbody>
              {rows.map((d) => (
                <tr key={d.id} onClick={() => nav(`/satislar/${d.id}`)} className="rowhover border-b border-slate-50 last:border-0 cursor-pointer">
                  <td className="px-5 py-3 font-medium text-indigo-700">{d.customer || "—"}</td>
                  <td className="px-5 py-3 text-slate-600 font-medium tabular whitespace-nowrap">{unitLabel(d.block, d.unit_number)}</td>
                  <td className="px-5 py-3 text-slate-600">{d.unit_type || "—"}</td>
                  <td className="px-5 py-3"><StatusChip status={d.stage} /></td>
                  <td className="px-5 py-3 text-right tabular font-semibold">{money(d.sale_price, d.currency)}</td>
                  <td className="px-5 py-3 text-right tabular text-slate-600">{money(d.down_payment, d.currency)}</td>
                  <td className="px-5 py-3 text-slate-600">{d.agent || "—"}</td>
                  <td className="px-5 py-3 text-slate-600 tabular whitespace-nowrap">{date(saleDate(d))}</td>
                  {showCancel && (
                    <td className={`px-5 py-3 tabular whitespace-nowrap ${d.stage === "cancelled" ? "text-rose-600" : "text-slate-300"}`}>
                      {d.stage === "cancelled" ? date(d.cancelled_at) : "—"}
                    </td>
                  )}
                </tr>
              ))}
              {rows.length === 0 && (
                <tr><td colSpan={9} className="px-5 py-10 text-center text-slate-400">Kayıt bulunamadı</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>
    </>
  );
}
