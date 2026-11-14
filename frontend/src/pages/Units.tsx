import { useMemo, useState } from "react";
import { api, money, unitLabel, hit } from "../api";
import { PageHead, Loading, ErrorBox, StatusChip, SearchBox, useApi, Card } from "../ui/kit";
import type { Unit } from "../types";
import { BRAND, dot } from "../brand";

const FILTERS = [
  { key: "all", label: "Tümü" },
  { key: "available", label: "Boşta" },
  { key: "reserved", label: "Rezerve" },
  { key: "sold", label: "Satıldı" },
];

const border: Record<string, string> = {
  available: "#a7f3d0",
  reserved: "#fde68a",
  sold: "#c7d2fe",
  unavailable: "#e2e8f0",
};

export default function Units() {
  const { data, error, loading } = useApi(() => api.units(), []);
  const [f, setF] = useState("all");
  const [q, setQ] = useState("");

  const groups = useMemo(() => {
    if (!data) return [];
    const rows = data.rows.filter(
      (u) => (f === "all" || u.status === f) && hit(q, unitLabel(u.block, u.number), u.type, u.view, u.gross_m2),
    );
    const by: Record<string, Unit[]> = {};
    for (const u of rows) (by[u.block || "—"] ??= []).push(u);
    return Object.entries(by).sort((a, b) => a[0].localeCompare(b[0], "tr"));
  }, [data, f]);

  if (loading) return <Loading />;
  if (error) return <ErrorBox error={error} />;
  const shown = groups.reduce((n, [, us]) => n + us.length, 0);

  return (
    <>
      <PageHead
        title="Daireler"
        subtitle={dot(q || f !== "all" ? `${shown} / ${data!.total} daire` : `${data!.total} daire`, BRAND.project)}
        right={
          <div className="flex items-center gap-2">
          <SearchBox value={q} onChange={setQ} placeholder="B2-13, tip, manzara…" />
          <div className="flex gap-1 bg-white border border-slate-200 rounded-xl p-1">
            {FILTERS.map((x) => (
              <button
                key={x.key}
                onClick={() => setF(x.key)}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium ${
                  f === x.key ? "bg-indigo-500 text-white" : "text-slate-600 hover:bg-slate-50"
                }`}
              >
                {x.label}
              </button>
            ))}
          </div>
          </div>
        }
      />
      <div className="space-y-6">
        {groups.map(([block, units]) => (
          <div key={block}>
            <div className="flex items-center gap-2 mb-3">
              <span className="text-sm font-bold text-slate-700">Blok {block}</span>
              <span className="text-xs text-slate-400">{units.length} daire</span>
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
              {units.map((u) => (
                <Card key={u.id} className="p-4" >
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="font-bold text-lg leading-none tabular whitespace-nowrap">{unitLabel(u.block, u.number)}</div>
                      <div className="text-xs text-slate-500 mt-1">{u.type || "—"}</div>
                    </div>
                    <div style={{ borderLeft: `3px solid ${border[u.status]}`, paddingLeft: 0 }} />
                    <StatusChip status={u.status} />
                  </div>
                  <div className="mt-3 text-sm text-slate-600 flex justify-between">
                    <span>{u.gross_m2 ? `${u.gross_m2} m²` : "—"}</span>
                    <span className="text-slate-400">{u.view}</span>
                  </div>
                  <div className="mt-2 font-bold tabular">{money(u.price, u.currency)}</div>
                </Card>
              ))}
            </div>
          </div>
        ))}
        {groups.length === 0 && <div className="text-slate-400 py-10 text-center">Kayıt yok</div>}
      </div>
    </>
  );
}
