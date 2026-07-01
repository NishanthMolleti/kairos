'use client';

import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import MetricCard from '@/components/ui/MetricCard';
import TrendChart from '@/components/ui/TrendChart';
import { getHRV, getSpO2, getReadiness } from '@/lib/api';
import type { DailyHRV, DailySpO2, DailyReadiness } from '@/lib/types';

export default function HeartPage() {
  const [hrv, setHRV] = useState<DailyHRV[]>([]);
  const [spo2, setSpo2] = useState<DailySpO2[]>([]);
  const [readiness, setReadiness] = useState<DailyReadiness[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const to = new Date().toISOString().slice(0, 10);
    const from = new Date(Date.now() - 30 * 86_400_000).toISOString().slice(0, 10);
    Promise.all([getHRV({ from, to }), getSpO2({ from, to }), getReadiness({ from, to })])
      .then(([h, sp, rd]) => { setHRV(h); setSpo2(sp); setReadiness(rd); })
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="h-64 bg-white/5 rounded-2xl animate-pulse" />;

  const latestHRV = hrv[hrv.length - 1];
  const latestSpo2 = spo2[spo2.length - 1];
  const hrvTrend = hrv.map((d) => ({ date: d.date, value: d.rmssd != null ? +d.rmssd.toFixed(1) : null }));
  const rhrTrend = readiness.map((d) => ({ date: d.date, value: d.resting_heart_rate }));
  const spo2Trend = spo2.map((d) => ({ date: d.date, value: d.avg_spo2 }));

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-6">
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard label="HRV (RMSSD)" value={latestHRV?.rmssd != null ? +latestHRV.rmssd.toFixed(1) : null} unit="ms" />
        <MetricCard label="BDI" value={latestHRV?.bdi != null ? +latestHRV.bdi.toFixed(1) : null} />
        <MetricCard label="Avg SpO2" value={latestSpo2?.avg_spo2 != null ? `${latestSpo2.avg_spo2.toFixed(1)}%` : null} />
        <MetricCard label="Min SpO2" value={latestSpo2?.min_spo2 != null ? `${latestSpo2.min_spo2.toFixed(1)}%` : null} />
      </div>
      <TrendChart data={hrvTrend} label="HRV RMSSD (30 days)" color="#7c3aed" unit="ms" />
      <TrendChart data={rhrTrend} label="Resting Heart Rate (30 days)" color="#06b6d4" unit="bpm" />
      <TrendChart data={spo2Trend} label="Avg Blood Oxygen SpO2 (30 days)" color="#10b981" unit="%" />
    </motion.div>
  );
}
