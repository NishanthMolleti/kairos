'use client';

import { motion } from 'framer-motion';

interface ScoreRingProps {
  score: number | null;
  size?: number;
  strokeWidth?: number;
  color?: string;
  label?: string;
}

export default function ScoreRing({ score, size = 120, strokeWidth = 10, color = '#7c3aed', label }: ScoreRingProps) {
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const progress = score != null ? (score / 100) * circumference : 0;
  const offset = circumference - progress;

  return (
    <div className="flex flex-col items-center gap-2">
      <div className="relative" style={{ width: size, height: size }}>
        <svg width={size} height={size} className="-rotate-90 absolute inset-0">
          <circle cx={size / 2} cy={size / 2} r={radius} fill="none" stroke="rgba(255,255,255,0.08)" strokeWidth={strokeWidth} />
          <motion.circle
            cx={size / 2} cy={size / 2} r={radius} fill="none" stroke={color}
            strokeWidth={strokeWidth} strokeLinecap="round"
            strokeDasharray={circumference}
            initial={{ strokeDashoffset: circumference }}
            animate={{ strokeDashoffset: offset }}
            transition={{ duration: 1, ease: 'easeOut' }}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-2xl font-bold text-text-primary">{score ?? '—'}</span>
          {label && <span className="text-text-muted text-xs mt-0.5">{label}</span>}
        </div>
      </div>
    </div>
  );
}
