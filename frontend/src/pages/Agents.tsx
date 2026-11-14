import { useMemo, useState } from "react";
import { api, hit } from "../api";
import { Card, PageHead, Loading, ErrorBox, SearchBox, useApi } from "../ui/kit";

function initials(name: string) {
  return name.split(/\s+/).filter(Boolean).slice(0, 2).map((s) => s[0]).join("").toUpperCase();
}

export default function Agents() {
  const { data, error, loading } = useApi(() => api.agents(), []);
  const [q, setQ] = useState("");
  const rows = useMemo(() => (data ? data.rows.filter((a) => hit(q, a.name, a.email, a.active ? "aktif" : "pasif")) : []), [data, q]);
  if (loading) return <Loading />;
  if (error) return <ErrorBox error={error} />;
  return (
    <>
      <PageHead
        title="Personel"
        subtitle={q ? `${rows.length} / ${data!.total} kişi` : `${data!.total} kişi`}
        right={<SearchBox value={q} onChange={setQ} placeholder="İsim veya e-posta…" />}
      />
      <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
        {rows.length === 0 && <div className="col-span-full text-slate-400 py-10 text-center">Kayıt bulunamadı</div>}
        {rows.map((a) => (
          <Card key={a.id} className="p-5 flex items-center gap-4">
            <div className="w-12 h-12 rounded-2xl bg-indigo-100 text-indigo-700 grid place-items-center font-bold">
              {initials(a.name)}
            </div>
            <div className="min-w-0">
              <div className="font-semibold truncate">{a.name}</div>
              <div className="text-xs text-slate-500 truncate">{a.email || "e-posta yok"}</div>
              <span className={`chip mt-1 ${a.active ? "" : ""}`} style={{ background: a.active ? "#ecfdf5" : "#f1f5f9", color: a.active ? "#047857" : "#64748b" }}>
                {a.active ? "Aktif" : "Pasif"}
              </span>
            </div>
          </Card>
        ))}
      </div>
    </>
  );
}
