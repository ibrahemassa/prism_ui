import { useMemo } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { api, money, date, unitLabel, hit } from "../api";
import { Card, PageHead, Loading, ErrorBox, StatusChip, useApi } from "../ui/kit";

/** One query, every list: customers come from the server search, the rest are
 *  small enough to filter in the browser. */
export default function Search() {
  const [params] = useSearchParams();
  const q = (params.get("q") ?? "").trim();
  const leads = useApi(() => (q ? api.leads(q, 10, 0) : Promise.resolve({ rows: [], total: 0 })), [q]);
  const deals = useApi(() => api.deals(), []);
  const units = useApi(() => api.units(), []);
  const agents = useApi(() => api.agents(), []);
  const agencies = useApi(() => api.agencies(), []);

  const dealHits = useMemo(
    () => (deals.data && q ? deals.data.rows.filter((d) => hit(q, d.customer, unitLabel(d.block, d.unit_number), d.unit_type, d.agent, d.agency)) : []),
    [deals.data, q],
  );
  const unitHits = useMemo(
    () => (units.data && q ? units.data.rows.filter((u) => hit(q, unitLabel(u.block, u.number), u.type, u.view)) : []),
    [units.data, q],
  );
  const agentHits = useMemo(() => (agents.data && q ? agents.data.rows.filter((a) => hit(q, a.name, a.email)) : []), [agents.data, q]);
  const agencyHits = useMemo(
    () => (agencies.data && q ? agencies.data.rows.filter((a) => hit(q, a.name, a.phone, a.email, a.city)) : []),
    [agencies.data, q],
  );

  const loading = leads.loading || deals.loading || units.loading || agents.loading || agencies.loading;
  const error = leads.error || deals.error || units.error || agents.error || agencies.error;
  const total = (leads.data?.total ?? 0) + dealHits.length + unitHits.length + agentHits.length + agencyHits.length;

  if (!q) return <PageHead title="Arama" subtitle="Üstteki kutuya bir isim, telefon, daire (B2-13) veya acenta yazın" />;
  if (error) return <ErrorBox error={error} />;

  return (
    <>
      <PageHead title={`"${q}"`} subtitle={loading ? "Aranıyor…" : `${total} sonuç`} />
      {loading ? (
        <Loading />
      ) : (
        <div className="space-y-5">
          <Section title="Müşteriler" count={leads.data!.total} more={leads.data!.total > 10 ? `/musteriler?q=${encodeURIComponent(q)}` : undefined}>
            {leads.data!.rows.map((l) => (
              <Row key={l.id} to={`/musteriler/${l.id}`} main={l.name || "—"} sub={[l.phone, l.category, l.agent].filter(Boolean).join(" · ")} right={date(l.created_at)} />
            ))}
          </Section>
          <Section title="Satışlar" count={dealHits.length}>
            {dealHits.slice(0, 20).map((d) => (
              <Row
                key={d.id}
                to={`/satislar/${d.id}`}
                main={`${d.customer || "—"} · ${unitLabel(d.block, d.unit_number)}`}
                sub={[d.unit_type, d.agent].filter(Boolean).join(" · ")}
                right={<span className="flex items-center gap-3"><StatusChip status={d.stage} /><span className="tabular font-semibold">{money(d.sale_price, d.currency)}</span></span>}
              />
            ))}
          </Section>
          <Section title="Daireler" count={unitHits.length} more={unitHits.length > 20 ? "/daireler" : undefined}>
            {unitHits.slice(0, 20).map((u) => (
              <Row key={u.id} to="/daireler" main={unitLabel(u.block, u.number)} sub={[u.type, u.gross_m2 ? `${u.gross_m2} m²` : "", u.view].filter(Boolean).join(" · ")}
                right={<span className="flex items-center gap-3"><StatusChip status={u.status} /><span className="tabular font-semibold">{money(u.price, u.currency)}</span></span>} />
            ))}
          </Section>
          <Section title="Personel" count={agentHits.length}>
            {agentHits.map((a) => <Row key={a.id} to="/personel" main={a.name} sub={a.email || "e-posta yok"} right={a.active ? "Aktif" : "Pasif"} />)}
          </Section>
          <Section title="Acentalar" count={agencyHits.length}>
            {agencyHits.map((a) => <Row key={a.id} to="/acentalar" main={a.name} sub={[a.phone, a.email, a.city].filter(Boolean).join(" · ")} right={a.rate ? `%${a.rate}` : ""} />)}
          </Section>
        </div>
      )}
    </>
  );
}

function Section({ title, count, more, children }: { title: string; count: number; more?: string; children: React.ReactNode }) {
  if (count === 0) return null;
  return (
    <Card className="overflow-hidden">
      <div className="flex items-center justify-between px-5 py-3 border-b border-slate-100">
        <div className="font-bold">{title} <span className="text-slate-400 font-normal text-sm">· {count}</span></div>
        {more && <Link to={more} className="text-sm text-indigo-600 hover:underline">Tümünü gör →</Link>}
      </div>
      <div>{children}</div>
    </Card>
  );
}

function Row({ to, main, sub, right }: { to: string; main: string; sub?: string; right?: React.ReactNode }) {
  return (
    <Link to={to} className="flex items-center justify-between gap-4 px-5 py-3 border-b border-slate-50 last:border-0 rowhover">
      <div className="min-w-0">
        <div className="font-medium text-indigo-700 truncate">{main}</div>
        {sub && <div className="text-xs text-slate-500 truncate">{sub}</div>}
      </div>
      <div className="text-sm text-slate-500 shrink-0">{right}</div>
    </Link>
  );
}
