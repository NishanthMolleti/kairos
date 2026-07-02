'use client';

import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import type { TooltipProps } from 'recharts';

interface TrendChartProps {
  data: Array<{ date: string; value: number | null }>;
  color?: string;
  label?: string;
  unit?: string;
}

function GlassTooltip({ active, payload, label }: TooltipProps<number, string>) {
  if (!active || !payload?.length) return null;
  return (
    <div className="bg-[#1a1a2e] border border-white/10 rounded-xl px-3 py-2 text-sm shadow-xl">
      <p className="text-text-muted text-xs mb-1">{label}</p>
      <p className="text-text-primary font-semibold">{payload[0]?.value ?? '—'}</p>
    </div>
  );
}

export default function TrendChart({ data, color = '#7c3aed', label, unit }: TrendChartProps) {
  const hasData = data.some(d => d.value != null);
  if (!hasData) {
    return (
      <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-5">
        {label && <p className="text-text-muted text-xs uppercase tracking-wider mb-4">{label}</p>}
        <div className="h-[200px] flex items-center justify-center text-text-muted text-sm">No data available</div>
      </div>
    );
  }

  const chartData = data.map((d) => ({ ...d, value: d.value ?? undefined }));
  return (
    <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-5">
      {label && <p className="text-text-muted text-xs uppercase tracking-wider mb-4">{label}</p>}
      <ResponsiveContainer width="100%" height={200}>
        <LineChart data={chartData} margin={{ top: 4, right: 8, left: -16, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
          <XAxis dataKey="date" tick={{ fill: '#94a3b8', fontSize: 11 }} axisLine={false} tickLine={false} tickFormatter={(v: string) => v.slice(5)} />
          <YAxis tick={{ fill: '#94a3b8', fontSize: 11 }} axisLine={false} tickLine={false} unit={unit} />
          <Tooltip content={<GlassTooltip />} />
          <Line type="monotone" dataKey="value" stroke={color} strokeWidth={2} dot={false} activeDot={{ r: 4, fill: color, stroke: '#0a0a0f', strokeWidth: 2 }} connectNulls />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
