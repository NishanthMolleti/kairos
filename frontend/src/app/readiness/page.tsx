'use client';

import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import ScoreRing from '@/components/ui/ScoreRing';
import MetricCard from '@/components/ui/MetricCard';
import TrendChart from '@/components/ui/TrendChart';
import { getReadiness } from '@/lib/api';
import type { DailyReadiness } from '@/lib/types';

export default function ReadinessPage() {
  const [data, setData] = useState<DailyReadiness[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const to = new Date().toISOString().slice(0, 10);
    const from = new Date(Date.now() - 30 * 86_400_000).toISOString().slice(0, 10);
    getReadiness({ from, to }).then(setData).finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="h-64 bg-white/5 rounded-2xl animate-pulse" />;

  const today = data[data.length - 1];
  const trend = data.map((d) => ({ date: d.date, value: d.score }));
  const tempStr = today?.body_temperature != null
    ? `${today.body_temperature > 0 ? '+' : ''}${today.body_temperature.toFixed(2)}`
    : null;

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-6">
      <div className="flex flex-wrap gap-4 items-start">
        <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-6 flex flex-col items-center gap-2">
          <ScoreRing score={today?.score ?? null} color="#06b6d4" label="Score" />
          <p className="text-text-muted text-xs uppercase tracking-wider">Readiness</p>
        </div>
        <div className="flex-1 grid grid-cols-2 sm:grid-cols-3 gap-4">
          <MetricCard label="HRV Balance" value={today?.hrv_balance ?? null} />
          <MetricCard label="Body Temp" value={tempStr} unit="°C" />
          <MetricCard label="Resting HR" value={today?.resting_heart_rate ?? null} unit="bpm" />
          <MetricCard label="Activity Balance" value={today?.activity_balance ?? null} />
          <MetricCard label="Sleep Balance" value={today?.sleep_balance ?? null} />
          <MetricCard label="Recovery Index" value={today?.recovery_index ?? null} />
        </div>
      </div>
      <TrendChart data={trend} label="30-Day Readiness Score" color="#06b6d4" />
    </motion.div>
  );
}
