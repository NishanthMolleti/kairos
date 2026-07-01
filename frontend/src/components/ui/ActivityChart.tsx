'use client';

import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceLine } from 'recharts';
import type { TooltipProps } from 'recharts';
import type { DailyActivity } from '@/lib/types';

interface Props { data: DailyActivity[] }

function GlassTooltip({ active, payload, label }: TooltipProps<number, string>) {
  if (!active || !payload?.length) return null;
  return (
    <div className="bg-[#1a1a2e] border border-white/10 rounded-xl px-3 py-2 text-sm shadow-xl">
      <p className="text-text-muted text-xs">{label}</p>
      <p className="text-[#06b6d4] font-semibold">{(payload[0]?.value ?? 0).toLocaleString()} steps</p>
    </div>
  );
}

export default function ActivityChart({ data }: Props) {
  const chartData = data.map((d) => ({ date: d.date.slice(5), steps: d.steps ?? 0 }));
  return (
    <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-5">
      <p className="text-text-muted text-xs uppercase tracking-wider mb-4">Daily Steps</p>
      <ResponsiveContainer width="100%" height={200}>
        <BarChart data={chartData} margin={{ top: 4, right: 8, left: -16, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" vertical={false} />
          <XAxis dataKey="date" tick={{ fill: '#94a3b8', fontSize: 11 }} axisLine={false} tickLine={false} />
          <YAxis tick={{ fill: '#94a3b8', fontSize: 11 }} axisLine={false} tickLine={false} tickFormatter={(v: number) => `${(v / 1000).toFixed(0)}k`} />
          <Tooltip content={<GlassTooltip />} cursor={{ fill: 'rgba(255,255,255,0.03)' }} />
          <ReferenceLine y={10000} stroke="#f59e0b" strokeDasharray="4 4" label={{ value: 'Goal', fill: '#f59e0b', fontSize: 11 }} />
          <Bar dataKey="steps" fill="#06b6d4" radius={[4, 4, 0, 0]} maxBarSize={28} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
