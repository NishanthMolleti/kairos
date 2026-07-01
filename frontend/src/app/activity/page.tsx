'use client';

import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import ScoreRing from '@/components/ui/ScoreRing';
import MetricCard from '@/components/ui/MetricCard';
import ActivityChart from '@/components/ui/ActivityChart';
import TrendChart from '@/components/ui/TrendChart';
import { getActivity, getWorkouts } from '@/lib/api';
import type { DailyActivity, Workout } from '@/lib/types';

export default function ActivityPage() {
  const [data, setData] = useState<DailyActivity[]>([]);
  const [workouts, setWorkouts] = useState<Workout[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const to = new Date().toISOString().slice(0, 10);
    const from = new Date(Date.now() - 30 * 86_400_000).toISOString().slice(0, 10);
    Promise.all([getActivity({ from, to }), getWorkouts({ from, to })])
      .then(([a, w]) => { setData(a); setWorkouts(w); })
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="h-64 bg-white/5 rounded-2xl animate-pulse" />;

  const today = data[data.length - 1];
  const calorieTrend = data.map((d) => ({ date: d.date, value: d.calories }));

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-6">
      <div className="flex flex-wrap gap-4 items-start">
        <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-6 flex flex-col items-center gap-2">
          <ScoreRing score={today?.score ?? null} color="#10b981" label="Score" />
          <p className="text-text-muted text-xs uppercase tracking-wider">Activity</p>
        </div>
        <div className="flex-1 grid grid-cols-2 sm:grid-cols-4 gap-4">
          <MetricCard label="Steps" value={today?.steps != null ? today.steps.toLocaleString() : null} />
          <MetricCard label="Calories" value={today?.calories ?? null} unit="kcal" />
          <MetricCard label="Active Cal" value={today?.active_calories ?? null} unit="kcal" />
          <MetricCard label="MET Minutes" value={today?.met_minutes ?? null} unit="min" />
        </div>
      </div>
      <ActivityChart data={data} />
      <TrendChart data={calorieTrend} label="30-Day Calories Burned" color="#10b981" unit="kcal" />
      {workouts.length > 0 && (
        <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl overflow-hidden">
          <p className="px-5 py-4 text-text-muted text-xs uppercase tracking-wider border-b border-white/10">Recent Workouts</p>
          <div className="divide-y divide-white/5">
            {[...workouts].reverse().slice(0, 10).map((w) => (
              <div key={w.id} className="px-5 py-3 flex items-center justify-between">
                <div>
                  <p className="text-text-primary text-sm font-medium capitalize">{w.activity ?? 'Workout'}</p>
                  <p className="text-text-muted text-xs">{new Date(w.start_datetime).toLocaleString()}</p>
                </div>
                <div className="text-right text-xs text-text-muted">
                  {w.calories != null && <p>{w.calories} kcal</p>}
                  {w.distance != null && <p>{(w.distance / 1000).toFixed(2)} km</p>}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </motion.div>
  );
}
