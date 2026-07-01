'use client';

import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import type { TooltipProps } from 'recharts';
import type { DailySleep } from '@/lib/types';

interface Props { data: DailySleep[] }

function GlassTooltip({ active, payload, label }: TooltipProps<number, string>) {
  if (!active || !payload?.length) return null;
  return (
    <div className="bg-[#1a1a2e] border border-white/10 rounded-xl px-3 py-2 text-sm shadow-xl space-y-1">
      <p className="text-text-muted text-xs">{label}</p>
      {payload.map((p) => (
        <p key={p.name} style={{ color: p.color }} className="font-medium">{p.name}: {p.value} min</p>
      ))}
    </div>
  );
}

export default function SleepStagesChart({ data }: Props) {
  const chartData = data.map((d) => ({
    date: d.date.slice(5),
    REM: d.rem_sleep_duration != null ? Math.round(d.rem_sleep_duration / 60) : 0,
    Deep: d.deep_sleep_duration != null ? Math.round(d.deep_sleep_duration / 60) : 0,
    Light: d.light_sleep_duration != null ? Math.round(d.light_sleep_duration / 60) : 0,
    Awake: d.awake_time != null ? Math.round(d.awake_time / 60) : 0,
  }));
  return (
    <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-5">
      <p className="text-text-muted text-xs uppercase tracking-wider mb-4">Sleep Stages</p>
      <ResponsiveContainer width="100%" height={220}>
        <BarChart data={chartData} margin={{ top: 4, right: 8, left: -16, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" vertical={false} />
          <XAxis dataKey="date" tick={{ fill: '#94a3b8', fontSize: 11 }} axisLine={false} tickLine={false} />
          <YAxis tick={{ fill: '#94a3b8', fontSize: 11 }} axisLine={false} tickLine={false} unit="m" />
          <Tooltip content={<GlassTooltip />} cursor={{ fill: 'rgba(255,255,255,0.03)' }} />
          <Legend wrapperStyle={{ fontSize: 12, color: '#94a3b8' }} />
          <Bar dataKey="REM" stackId="a" fill="#7c3aed" />
          <Bar dataKey="Deep" stackId="a" fill="#06b6d4" />
          <Bar dataKey="Light" stackId="a" fill="#10b981" />
          <Bar dataKey="Awake" stackId="a" fill="#f59e0b" radius={[3, 3, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
