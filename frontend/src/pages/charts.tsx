import React from "react";

/* Print-safe SVG charts for the report. No library: everything is vector,
   inherits the sheet's font, and prints at any DPI. Sizes are in CSS px on a
   210 mm sheet (≈ 680 px usable), so widths default to what a column takes. */

export const PALETTE = {
  forest: "#0E5A33",
  deep: "#093B22",
  gold: "#C7A233",
  goldInk: "#8F6F14",
  brick: "#A23B2B",
  mute: "#66756C",
  line: "#D7DFD9",
  band: "#F2F5F3",
  sage: "#7FA88F",
  sand: "#E7D9A8",
};

type Fmt = (n: number) => string;
const defFmt: Fmt = (n) => n.toLocaleString("tr-TR", { maximumFractionDigits: 0 });

function ticks(max: number, n = 4): number[] {
  if (max <= 0) return [0];
  const raw = max / n;
  const mag = Math.pow(10, Math.floor(Math.log10(raw)));
  const step = [1, 2, 2.5, 5, 10].map((m) => m * mag).find((s) => s >= raw) ?? raw;
  const out: number[] = [];
  for (let v = 0; v <= max + step * 0.999; v += step) out.push(v);
  return out;
}

/** Vertical bars. `series` are stacked on one bar per label. */
export function VBars({
  labels, series, width = 640, height = 170, fmt = defFmt, showValues = true, labelEvery = 1, rotate = false,
}: {
  labels: string[];
  series: { name: string; values: number[]; color: string }[];
  width?: number; height?: number; fmt?: Fmt; showValues?: boolean; labelEvery?: number; rotate?: boolean;
}) {
  const padL = 44, padR = 8, padT = 14, padB = rotate ? 40 : 22;
  const W = width - padL - padR, H = height - padT - padB;
  const totals = labels.map((_, i) => series.reduce((s, sr) => s + (sr.values[i] || 0), 0));
  const max = Math.max(1, ...totals);
  const tk = ticks(max);
  const top = tk[tk.length - 1] || max;
  const n = labels.length;
  const slot = W / Math.max(1, n);
  const bw = Math.min(28, slot * 0.68);
  const y = (v: number) => padT + H - (v / top) * H;
  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} style={{ display: "block", maxWidth: "100%" }}>
      {tk.map((t) => (
        <g key={t}>
          <line x1={padL} x2={width - padR} y1={y(t)} y2={y(t)} stroke={PALETTE.line} strokeWidth={0.6} />
          <text x={padL - 4} y={y(t) + 3} fontSize={7.5} textAnchor="end" fill={PALETTE.mute}>{fmt(t)}</text>
        </g>
      ))}
      {labels.map((lab, i) => {
        const x = padL + slot * i + (slot - bw) / 2;
        let acc = 0;
        return (
          <g key={lab}>
            {series.map((sr) => {
              const v = sr.values[i] || 0;
              const y1 = y(acc + v), y0 = y(acc);
              acc += v;
              return v > 0 ? <rect key={sr.name} x={x} y={y1} width={bw} height={Math.max(0.5, y0 - y1)} fill={sr.color} /> : null;
            })}
            {showValues && totals[i] > 0 && n <= 16 && (
              <text x={x + bw / 2} y={y(totals[i]) - 3} fontSize={7.5} textAnchor="middle" fill={PALETTE.deep} fontWeight={700}>{fmt(totals[i])}</text>
            )}
            {i % labelEvery === 0 && (
              rotate
                ? <text x={x + bw / 2} y={padT + H + 8} fontSize={7} textAnchor="end" fill={PALETTE.mute} transform={`rotate(-45 ${x + bw / 2} ${padT + H + 8})`}>{lab}</text>
                : <text x={x + bw / 2} y={padT + H + 12} fontSize={7.5} textAnchor="middle" fill={PALETTE.mute}>{lab}</text>
            )}
          </g>
        );
      })}
      <line x1={padL} x2={width - padR} y1={padT + H} y2={padT + H} stroke={PALETTE.mute} strokeWidth={0.8} />
    </svg>
  );
}

/** Line with a soft area beneath; several series share the axis. */
export function LineArea({
  labels, series, width = 640, height = 170, fmt = defFmt, labelEvery = 1,
}: {
  labels: string[];
  series: { name: string; values: number[]; color: string; area?: boolean }[];
  width?: number; height?: number; fmt?: Fmt; labelEvery?: number;
}) {
  const padL = 44, padR = 10, padT = 12, padB = 22;
  const W = width - padL - padR, H = height - padT - padB;
  const max = Math.max(1, ...series.flatMap((s) => s.values));
  const tk = ticks(max);
  const top = tk[tk.length - 1] || max;
  const n = labels.length;
  const x = (i: number) => padL + (n <= 1 ? W / 2 : (i / (n - 1)) * W);
  const y = (v: number) => padT + H - (v / top) * H;
  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} style={{ display: "block", maxWidth: "100%" }}>
      {tk.map((t) => (
        <g key={t}>
          <line x1={padL} x2={width - padR} y1={y(t)} y2={y(t)} stroke={PALETTE.line} strokeWidth={0.6} />
          <text x={padL - 4} y={y(t) + 3} fontSize={7.5} textAnchor="end" fill={PALETTE.mute}>{fmt(t)}</text>
        </g>
      ))}
      {series.map((s) => {
        const pts = s.values.map((v, i) => `${x(i)},${y(v)}`).join(" ");
        return (
          <g key={s.name}>
            {s.area && n > 1 && <polygon points={`${x(0)},${y(0)} ${pts} ${x(n - 1)},${y(0)}`} fill={s.color} opacity={0.12} />}
            <polyline points={pts} fill="none" stroke={s.color} strokeWidth={1.6} strokeLinejoin="round" />
            {n <= 40 && s.values.map((v, i) => <circle key={i} cx={x(i)} cy={y(v)} r={1.6} fill={s.color} />)}
          </g>
        );
      })}
      {labels.map((lab, i) => (i % labelEvery === 0 ? <text key={lab + i} x={x(i)} y={padT + H + 12} fontSize={7} textAnchor="middle" fill={PALETTE.mute}>{lab}</text> : null))}
      <line x1={padL} x2={width - padR} y1={padT + H} y2={padT + H} stroke={PALETTE.mute} strokeWidth={0.8} />
    </svg>
  );
}

/** Donut with a legend to the right. Segments are filled annular sectors, not
 *  thick strokes — Chromium's PDF backend flattens fat stroked arcs into
 *  visible facets, filled paths stay round. */
export function Donut({
  parts, size = 110, fmt = defFmt, center,
}: { parts: { label: string; value: number; color: string }[]; size?: number; fmt?: Fmt; center?: string }) {
  const total = parts.reduce((s, p) => s + p.value, 0) || 1;
  const cx = size / 2, cy = size / 2, R = size / 2 - 2, r = R * 0.62;
  const pt = (rad: number, ang: number) => [cx + rad * Math.cos(ang), cy + rad * Math.sin(ang)];
  let a0 = -Math.PI / 2;
  const arcs = parts.filter((p) => p.value > 0).map((p) => {
    const sweep = (p.value / total) * Math.PI * 2;
    const full = sweep >= Math.PI * 2 - 1e-6;
    const a1 = a0 + (full ? sweep - 1e-4 : sweep);
    const big = a1 - a0 > Math.PI ? 1 : 0;
    const [ox0, oy0] = pt(R, a0), [ox1, oy1] = pt(R, a1), [ix0, iy0] = pt(r, a0), [ix1, iy1] = pt(r, a1);
    const d = `M ${ox0} ${oy0} A ${R} ${R} 0 ${big} 1 ${ox1} ${oy1} L ${ix1} ${iy1} A ${r} ${r} 0 ${big} 0 ${ix0} ${iy0} Z`;
    a0 = a1;
    return { ...p, d };
  });
  return (
    <div style={{ display: "flex", alignItems: "center", gap: "4mm" }}>
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} style={{ flexShrink: 0 }}>
        <circle cx={cx} cy={cy} r={(R + r) / 2} fill="none" stroke={PALETTE.band} strokeWidth={R - r} />
        {arcs.map((a) => <path key={a.label} d={a.d} fill={a.color} />)}
        {center && <text x={cx} y={cy + 4} fontSize={size * 0.13} textAnchor="middle" fontWeight={800} fill={PALETTE.deep}>{center}</text>}
      </svg>
      <div style={{ fontSize: "0.85em", lineHeight: 1.5 }}>
        {parts.map((p) => (
          <div key={p.label} style={{ display: "flex", alignItems: "center", gap: "1.5mm", whiteSpace: "nowrap" }}>
            <span style={{ width: "2.4mm", height: "2.4mm", background: p.color, display: "inline-block" }} />
            <span style={{ color: PALETTE.mute }}>{p.label}</span>
            <b>{fmt(p.value)}</b>
            <span style={{ color: PALETTE.mute }}>%{((p.value / total) * 100).toFixed(0)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

/** Horizontal stacked bars with the label on the left and a value on the right. */
export function HBars({
  rows, max, fmt = defFmt, width = "100%",
}: { rows: { label: string; parts: { value: number; color: string }[]; right?: React.ReactNode }[]; max?: number; fmt?: Fmt; width?: string }) {
  const m = max ?? Math.max(1, ...rows.map((r) => r.parts.reduce((s, p) => s + p.value, 0)));
  return (
    <div style={{ width }}>
      {rows.map((r) => {
        const tot = r.parts.reduce((s, p) => s + p.value, 0);
        return (
          <div key={r.label} style={{ display: "grid", gridTemplateColumns: "28% 1fr auto", gap: "2mm", alignItems: "center", marginBottom: "1.4mm", fontSize: "0.85em" }}>
            <span style={{ color: PALETTE.mute, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{r.label}</span>
            <div style={{ height: "2.4mm", background: PALETTE.band, display: "flex", overflow: "hidden" }}>
              {r.parts.map((p, i) => <i key={i} style={{ display: "block", height: "100%", width: `${(p.value / m) * 100}%`, background: p.color }} />)}
            </div>
            <span style={{ fontVariantNumeric: "tabular-nums", whiteSpace: "nowrap" }}>{r.right ?? <b>{fmt(tot)}</b>}</span>
          </div>
        );
      })}
    </div>
  );
}

export function Legend({ items }: { items: { label: string; color: string }[] }) {
  return (
    <span style={{ display: "inline-flex", gap: "3mm", fontSize: "0.82em", color: PALETTE.mute }}>
      {items.map((it) => (
        <span key={it.label} style={{ display: "inline-flex", alignItems: "center", gap: "1mm" }}>
          <span style={{ width: "2.2mm", height: "2.2mm", background: it.color, display: "inline-block" }} />{it.label}
        </span>
      ))}
    </span>
  );
}
