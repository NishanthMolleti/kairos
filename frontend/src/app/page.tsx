'use client';

import { useEffect, useState, useCallback } from 'react';
import { motion } from 'framer-motion';
import MetricCard from '@/components/ui/MetricCard';
import TrendChart from '@/components/ui/TrendChart';
import { getSleep, getReadiness, getHRV, getActivity, getUser, getSpO2, getStress, postSync } from '@/lib/api';
import type { DailySleep, DailyReadiness, DailyHRV, DailyActivity, User, DailySpO2, DailyStress } from '@/lib/types';

function dateRange(days: number) {
  const to = new Date().toISOString().slice(0, 10);
  const from = new Date(Date.now() - days * 86_400_000).toISOString().slice(0, 10);
  return { from, to };
}

export default function DashboardPage() {
  const [sleep, setSleep] = useState<DailySleep[]>([]);
  const [readiness, setReadiness] = useState<DailyReadiness[]>([]);
  const [hrv, setHRV] = useState<DailyHRV[]>([]);
  const [activity, setActivity] = useState<DailyActivity[]>([]);
  const [user, setUser] = useState<User | null>(null);
  const [spo2, setSpo2] = useState<DailySpO2[]>([]);
  const [stress, setStress] = useState<DailyStress[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [syncing, setSyncing] = useState(false);
  const [days, setDays] = useState<number>(() => {
    if (typeof window !== 'undefined') {
      return Number(localStorage.getItem('kairos_days') ?? '30');
    }
    return 30;
  });

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const r = dateRange(days);
      const [s, rd, h, a, u, sp, st] = await Promise.all([
        getSleep(r), getReadiness(r), getHRV(r), getActivity(r), getUser(), getSpO2(r), getStress(r),
      ]);
      setSleep(s); setReadiness(rd); setHRV(h); setActivity(a); setUser(u); setSpo2(sp); setStress(st);

      // Bug 7 / Bug 11 — auto-sync if last_sync is null or older than 6 hours
      const sixHoursAgo = Date.now() - 6 * 60 * 60 * 1000;
      const needsSync = !u.last_sync || new Date(u.last_sync).getTime() < sixHoursAgo;
      if (needsSync) {
        setSyncing(true);
        try {
          await postSync();
        } finally {
          setSyncing(false);
        }
        // Refresh data after sync (without triggering another sync check)
        const r2 = dateRange(days);
        const [s2, rd2, h2, a2, u2, sp2, st2] = await Promise.all([
          getSleep(r2), getReadiness(r2), getHRV(r2), getActivity(r2), getUser(), getSpO2(r2), getStress(r2),
        ]);
        setSleep(s2); setReadiness(rd2); setHRV(h2); setActivity(a2); setUser(u2); setSpo2(sp2); setStress(st2);
      }
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [days]);

  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('kairos_days', String(days));
    }
  }, [days]);

  useEffect(() => { load(); }, [load]);

  const last = <T,>(arr: T[]): T | undefined => arr[arr.length - 1];
  const todaySleep = last(sleep);
  const todayReadiness = last(readiness);
  const todayHRV = last(hrv);
  const todayActivity = last(activity);
  const todaySpo2 = last(spo2);
  const todayStress = last(stress);
  const sleepTrend = sleep.map((d) => ({ date: d.date, value: d.score }));

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {[...Array(4)].map((_, i) => <div key={i} className="h-32 bg-white/5 rounded-2xl" />)}
        </div>
        <div className="h-56 bg-white/5 rounded-2xl" />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {[...Array(3)].map((_, i) => <div key={i} className="h-24 bg-white/5 rounded-2xl" />)}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-64 gap-4 text-text-muted">
        <p>Failed to load data: {error}</p>
        <button onClick={load} className="px-4 py-2 rounded-xl bg-accent-purple/20 border border-accent-purple/40 text-white text-sm hover:bg-accent-purple/30 transition">
          Retry
        </button>
      </div>
    );
  }

  return (
    <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4 }} className="space-y-6">
      <div className="flex items-center justify-between flex-wrap gap-2">
        {syncing && (
          <div className="flex items-center gap-2 text-xs text-text-muted bg-white/5 border border-white/10 rounded-xl px-4 py-2">
            <div className="w-3 h-3 border border-accent-purple border-t-transparent rounded-full animate-spin" />
            Syncing your Oura data...
          </div>
        )}
        <div className="flex gap-2 ml-auto">
          {[7, 30, 90].map(d => (
            <button key={d} onClick={() => setDays(d)}
              className={`px-3 py-1 rounded-lg text-xs font-medium transition ${days === d ? 'bg-accent-purple text-white' : 'bg-white/5 text-text-muted hover:bg-white/10'}`}>
              {d}d
            </button>
          ))}
        </div>
      </div>
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard label="Sleep Score" value={todaySleep?.score ?? null} unit="/100" />
        <MetricCard label="Readiness" value={todayReadiness?.score ?? null} unit="/100" />
        <MetricCard label="HRV (RMSSD)" value={todayHRV?.rmssd != null ? Math.round(todayHRV.rmssd) : null} unit="ms" />
        <MetricCard label="Steps" value={todayActivity?.steps != null ? todayActivity.steps.toLocaleString() : null} />
      </div>
      <TrendChart data={sleepTrend} label={`${days}-Day Sleep Score Trend`} color="#7c3aed" />
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-5">
          <p className="text-text-muted text-xs uppercase tracking-wider mb-2">Last Sync</p>
          <p className="text-text-primary text-sm font-medium">{user?.last_sync ? new Date(user.last_sync).toLocaleString() : '—'}</p>
        </div>
        <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-5">
          <p className="text-text-muted text-xs uppercase tracking-wider mb-2">SpO2 (avg)</p>
          <p className="text-text-primary text-2xl font-bold">{todaySpo2?.avg_spo2 != null ? `${todaySpo2.avg_spo2.toFixed(1)}%` : '—'}</p>
        </div>
        <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-5">
          <p className="text-text-muted text-xs uppercase tracking-wider mb-2">Stress Summary</p>
          <p className="text-text-primary text-sm">{todayStress?.day_summary ?? '—'}</p>
        </div>
      </div>
    </motion.div>
  );
}
