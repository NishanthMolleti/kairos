'use client';

import { motion } from 'framer-motion';
import { TrendingUp, TrendingDown } from 'lucide-react';

interface MetricCardProps {
  label: string;
  value: number | string | null;
  unit?: string;
  trend?: number;
  subtitle?: string;
}

export default function MetricCard({ label, value, unit, trend, subtitle }: MetricCardProps) {
  return (
    <motion.div
      whileHover={{ y: -2, scale: 1.01 }}
      transition={{ type: 'spring', stiffness: 300, damping: 20 }}
      className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-5 flex flex-col gap-3"
    >
      <span className="text-text-muted text-xs font-medium uppercase tracking-wider">{label}</span>
      <div className="flex items-end gap-2">
        <span className="text-4xl font-bold text-text-primary leading-none">{value ?? '—'}</span>
        {unit && <span className="text-text-muted text-sm mb-1">{unit}</span>}
      </div>
      {trend !== undefined && (
        <div className={`flex items-center gap-1 text-xs font-medium ${trend >= 0 ? 'text-green-400' : 'text-red-400'}`}>
          {trend >= 0 ? <TrendingUp size={12} /> : <TrendingDown size={12} />}
          {Math.abs(trend).toFixed(1)}% vs last week
        </div>
      )}
      {subtitle && <p className="text-text-muted text-xs">{subtitle}</p>}
    </motion.div>
  );
}
