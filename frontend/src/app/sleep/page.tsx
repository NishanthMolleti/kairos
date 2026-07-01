'use client';

import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import ScoreRing from '@/components/ui/ScoreRing';
import MetricCard from '@/components/ui/MetricCard';
import SleepStagesChart from '@/components/ui/SleepStagesChart';
import TrendChart from '@/components/ui/TrendChart';
import { getSleep } from '@/lib/api';
import type { DailySleep } from '@/lib/types';

function toHours(seconds: number | null): number | null {
  if (seconds == null) return null;
  return Math.round((seconds / 3600) * 10) / 10;
}

function toMinutes(seconds: number | null): number | null {
  if (seconds == null) return null;
  return Math.round(seconds / 60);
}

export default function SleepPage() {
  const [data, setData] = useState<DailySleep[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const to = new Date().toISOString().slice(0, 10);
    const from = new Date(Date.now() - 30 * 86_400_000).toISOString().slice(0, 10);
    getSleep({ from, to }).then(setData).finally(() => setLoading(false));
  }, []);

  if (loading) {
    return <div className="space-y-6 animate-pulse"><div className="h-48 bg-white/5 rounded-2xl" /><div className="h-56 bg-white/5 rounded-2xl" /></div>;
  }

  const today = data[data.length - 1];
  const durationTrend = data.map((d) => ({ date: d.date, value: toHours(d.total_sleep_duration) }));

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-6">
      <div className="flex flex-wrap gap-4 items-start">
        <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-6 flex flex-col items-center gap-2">
          <ScoreRing score={today?.score ?? null} color="#7c3aed" label="Score" />
          <p className="text-text-muted text-xs uppercase tracking-wider">Sleep Score</p>
        </div>
        <div className="flex-1 grid grid-cols-1 sm:grid-cols-3 gap-4">
          <MetricCard label="Total Sleep" value={toHours(today?.total_sleep_duration ?? null)} unit="h" />
          <MetricCard label="Efficiency" value={today?.efficiency ?? null} unit="%" />
          <MetricCard label="Latency" value={toMinutes(today?.latency ?? null)} unit="min" />
        </div>
      </div>
      <SleepStagesChart data={data} />
      <TrendChart data={durationTrend} label="Total Sleep Duration (30 days)" color="#06b6d4" unit="h" />
      <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-white/10">
                {['Date', 'Score', 'Total (h)', 'REM (min)', 'Deep (min)', 'Efficiency (%)'].map((h) => (
                  <th key={h} className="px-4 py-3 text-left text-text-muted font-medium text-xs uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {[...data].reverse().slice(0, 14).map((d) => (
                <tr key={d.id} className="border-b border-white/5 hover:bg-white/[0.03] transition-colors">
                  <td className="px-4 py-3 text-text-muted">{d.date}</td>
                  <td className="px-4 py-3 text-text-primary font-semibold">{d.score ?? '—'}</td>
                  <td className="px-4 py-3 text-text-primary">{toHours(d.total_sleep_duration) ?? '—'}</td>
                  <td className="px-4 py-3 text-text-primary">{toMinutes(d.rem_sleep_duration) ?? '—'}</td>
                  <td className="px-4 py-3 text-text-primary">{toMinutes(d.deep_sleep_duration) ?? '—'}</td>
                  <td className="px-4 py-3 text-text-primary">{d.efficiency ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </motion.div>
  );
}
