# Kairos — Plan 3: Frontend (Next.js)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Kairos health-dashboard frontend as a Next.js 14 static export that connects to the Go backend API and presents Oura ring data with a dark, glass-morphism UI powered by Sage (AI chat).
**Architecture:** All pages are client-side only (no SSR, no API routes); auth is JWT stored in localStorage and forwarded as `Authorization: Bearer <token>`; the built `/out` directory is served by Nginx on a GCP e2-micro VM. Data fetching happens in React client components using typed `fetch` wrappers, and SSE streaming is handled natively via `ReadableStream`.
**Tech Stack:** Next.js 14 (App Router, `output: 'export'`), TypeScript, Tailwind CSS, Framer Motion, Recharts, shadcn/ui, Inter font (Google Fonts).

---

## Project Layout

```
Kairos/frontend/
├── next.config.ts
├── package.json
├── tailwind.config.ts
├── tsconfig.json
├── src/
│   ├── app/
│   │   ├── layout.tsx
│   │   ├── page.tsx                   # /  dashboard
│   │   ├── sleep/page.tsx
│   │   ├── readiness/page.tsx
│   │   ├── activity/page.tsx
│   │   ├── heart/page.tsx
│   │   ├── chat/page.tsx
│   │   ├── settings/page.tsx
│   │   └── auth/
│   │       ├── login/page.tsx
│   │       └── callback/page.tsx
│   ├── components/
│   │   ├── layout/
│   │   │   ├── Sidebar.tsx
│   │   │   └── Header.tsx
│   │   ├── ui/
│   │   │   ├── MetricCard.tsx
│   │   │   ├── ScoreRing.tsx
│   │   │   ├── TrendChart.tsx
│   │   │   ├── SleepStagesChart.tsx
│   │   │   └── ActivityChart.tsx
│   │   └── chat/
│   │       ├── ChatWindow.tsx
│   │       ├── MessageBubble.tsx
│   │       └── SourceToggle.tsx
│   └── lib/
│       ├── api.ts
│       ├── auth.ts
│       └── types.ts
```

---

## Design Tokens

| Token | Value |
|---|---|
| Background | `#0a0a0f` |
| Card (glass) | `rgba(255,255,255,0.05)` + `backdrop-blur-md` + `border border-white/10` |
| Accent purple | `#7c3aed` |
| Accent cyan | `#06b6d4` |
| Text primary | `#f1f5f9` |
| Text muted | `#94a3b8` |
| Chart purple | `#7c3aed` |
| Chart cyan | `#06b6d4` |
| Chart green | `#10b981` |
| Chart amber | `#f59e0b` |

---

## Tasks

### Task 1 — Project scaffold

- [ ] Create `Kairos/frontend/` directory (if not present).
- [ ] Write `package.json`:

```json
{
  "name": "kairos-frontend",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint"
  },
  "dependencies": {
    "next": "14.2.5",
    "react": "^18",
    "react-dom": "^18",
    "framer-motion": "^11.3.8",
    "recharts": "^2.12.7",
    "@radix-ui/react-slot": "^1.1.0",
    "class-variance-authority": "^0.7.0",
    "clsx": "^2.1.1",
    "tailwind-merge": "^2.4.0",
    "lucide-react": "^0.411.0"
  },
  "devDependencies": {
    "typescript": "^5",
    "@types/node": "^20",
    "@types/react": "^18",
    "@types/react-dom": "^18",
    "tailwindcss": "^3.4.6",
    "postcss": "^8",
    "autoprefixer": "^10.4.19",
    "eslint": "^8",
    "eslint-config-next": "14.2.5"
  }
}
```

- [ ] Write `next.config.ts`:

```typescript
import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  output: 'export',
  trailingSlash: true,
  images: { unoptimized: true },
};

export default nextConfig;
```

- [ ] Write `tailwind.config.ts`:

```typescript
import type { Config } from 'tailwindcss';

const config: Config = {
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        background: '#0a0a0f',
        'accent-purple': '#7c3aed',
        'accent-cyan': '#06b6d4',
        'text-primary': '#f1f5f9',
        'text-muted': '#94a3b8',
      },
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
      },
    },
  },
  plugins: [],
};

export default config;
```

- [ ] Write `src/app/globals.css`:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

:root {
  --bg: #0a0a0f;
  --card: rgba(255, 255, 255, 0.05);
  --accent-purple: #7c3aed;
  --accent-cyan: #06b6d4;
  --text-primary: #f1f5f9;
  --text-muted: #94a3b8;
}

html,
body {
  background-color: var(--bg);
  color: var(--text-primary);
  font-family: 'Inter', sans-serif;
}

/* custom scrollbar */
::-webkit-scrollbar { width: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: rgba(124, 58, 237, 0.4); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: rgba(124, 58, 237, 0.7); }
```

- [ ] Write `tsconfig.json` with strict mode, paths alias `@/*` -> `./src/*`.
- [ ] Run `npm install` to lock dependencies.

---

### Task 2 — `lib/types.ts`, `lib/auth.ts`, `lib/api.ts`

- [ ] Write `src/lib/types.ts` with all interfaces exactly matching Go model JSON output:

```typescript
export interface DailySleep {
  id: string;
  date: string;
  score: number | null;
  total_sleep_duration: number | null;
  efficiency: number | null;
  latency: number | null;
  rem_sleep_duration: number | null;
  deep_sleep_duration: number | null;
  light_sleep_duration: number | null;
  awake_time: number | null;
  restless_periods: number | null;
}

export interface DailyReadiness {
  id: string;
  date: string;
  score: number | null;
  hrv_balance: number | null;
  body_temperature: number | null;
  recovery_index: number | null;
  resting_heart_rate: number | null;
  activity_balance: number | null;
  sleep_balance: number | null;
}

export interface DailyActivity {
  id: string;
  date: string;
  score: number | null;
  steps: number | null;
  calories: number | null;
  active_calories: number | null;
  met_minutes: number | null;
  sedentary_time: number | null;
  low_activity: number | null;
  medium_activity: number | null;
  high_activity: number | null;
}

export interface DailyHRV {
  id: string;
  date: string;
  rmssd: number | null;
  bdi: number | null;
}

export interface HeartRate {
  id: string;
  timestamp: string;
  bpm: number;
  source: string;
}

export interface DailySpO2 {
  id: string;
  date: string;
  avg_spo2: number | null;
  min_spo2: number | null;
}

export interface DailyStress {
  id: string;
  date: string;
  stress_high: number | null;
  recovery_high: number | null;
  day_summary: string | null;
}

export interface Workout {
  id: string;
  start_datetime: string;
  end_datetime: string | null;
  activity: string | null;
  calories: number | null;
  distance: number | null;
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  sql_used: string | null;
  created_at: string;
}

export interface User {
  id: string;
  email: string;
  last_sync: string | null;
}
```

- [ ] Write `src/lib/auth.ts`:

```typescript
const TOKEN_KEY = 'kairos_token';

export function getToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}

export function isAuthenticated(): boolean {
  return !!getToken();
}
```

- [ ] Write `src/lib/api.ts` with typed fetch wrappers for every endpoint. All requests attach `Authorization: Bearer <token>` via `getToken()`. Throw on non-2xx. Full implementation:

```typescript
import { getToken } from './auth';
import type {
  User, DailySleep, DailyReadiness, DailyActivity,
  DailyHRV, HeartRate, DailySpO2, DailyStress, Workout, ChatMessage,
} from './types';

const BASE = 'https://kairos.nimoclaw.dev';

function authHeaders(): HeadersInit {
  const token = getToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...authHeaders(), ...init.headers },
  });
  if (!res.ok) throw new Error(`API error ${res.status}: ${await res.text()}`);
  return res.json() as Promise<T>;
}

type DateRange = { from: string; to: string };

function dateParams({ from, to }: DateRange): string {
  return `?from=${from}&to=${to}`;
}

// User
export const getUser = (): Promise<User> => apiFetch('/api/user');
export const postSync = (): Promise<{ message: string }> =>
  apiFetch('/api/sync', { method: 'POST' });

// Metrics
export const getSleep = (r: DateRange): Promise<DailySleep[]> =>
  apiFetch(`/api/metrics/sleep${dateParams(r)}`);
export const getReadiness = (r: DateRange): Promise<DailyReadiness[]> =>
  apiFetch(`/api/metrics/readiness${dateParams(r)}`);
export const getActivity = (r: DateRange): Promise<DailyActivity[]> =>
  apiFetch(`/api/metrics/activity${dateParams(r)}`);
export const getHRV = (r: DateRange): Promise<DailyHRV[]> =>
  apiFetch(`/api/metrics/hrv${dateParams(r)}`);
export const getHeartRate = (r: DateRange): Promise<HeartRate[]> =>
  apiFetch(`/api/metrics/heartrate${dateParams(r)}`);
export const getSpO2 = (r: DateRange): Promise<DailySpO2[]> =>
  apiFetch(`/api/metrics/spo2${dateParams(r)}`);
export const getStress = (r: DateRange): Promise<DailyStress[]> =>
  apiFetch(`/api/metrics/stress${dateParams(r)}`);
export const getWorkouts = (r: DateRange): Promise<Workout[]> =>
  apiFetch(`/api/metrics/workouts${dateParams(r)}`);

// Chat
export const createChatSession = (): Promise<{ id: string }> =>
  apiFetch('/api/chat/sessions', { method: 'POST' });
export const getChatMessages = (sessionId: string): Promise<ChatMessage[]> =>
  apiFetch(`/api/chat/sessions/${sessionId}/messages`);

// SSE streaming ask — returns raw Response for caller to consume ReadableStream
export function askStream(sessionId: string, message: string): Promise<Response> {
  return fetch(`${BASE}/api/chat/sessions/${sessionId}/ask`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify({ message }),
  });
}
```

---

### Task 3 — Root layout

- [ ] Write `src/app/layout.tsx`. Import Inter from `next/font/google`. Apply dark background. Render `<Sidebar />` on the left and `<main>` with `<Header />` on the right. Guard: if not authenticated and not on `/auth/*`, redirect to `/auth/login`. Since this is a static export, auth redirect must be done client-side (use `'use client'` + `useEffect`).

```typescript
'use client';

import './globals.css';
import { Inter } from 'next/font/google';
import { usePathname, useRouter } from 'next/navigation';
import { useEffect } from 'react';
import Sidebar from '@/components/layout/Sidebar';
import Header from '@/components/layout/Header';
import { isAuthenticated } from '@/lib/auth';

const inter = Inter({ subsets: ['latin'] });

export default function RootLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const isAuthRoute = pathname?.startsWith('/auth');

  useEffect(() => {
    if (!isAuthRoute && !isAuthenticated()) {
      router.replace('/auth/login');
    }
  }, [isAuthRoute, router]);

  if (isAuthRoute) {
    return (
      <html lang="en">
        <body className={`${inter.className} bg-background min-h-screen`}>{children}</body>
      </html>
    );
  }

  return (
    <html lang="en">
      <body className={`${inter.className} bg-[#0a0a0f] min-h-screen flex`}>
        <Sidebar />
        <div className="flex flex-col flex-1 min-h-screen overflow-hidden">
          <Header />
          <main className="flex-1 overflow-y-auto p-6">{children}</main>
        </div>
      </body>
    </html>
  );
}
```

---

### Task 4 — `Sidebar.tsx` + `Header.tsx`

- [ ] Write `src/components/layout/Sidebar.tsx`. Nav links: Dashboard (`/`), Sleep (`/sleep`), Readiness (`/readiness`), Activity (`/activity`), Heart (`/heart`), Chat (`/chat`), Settings (`/settings`). Active link highlighted with accent purple. Use Framer Motion for hover scale. Lucide icons for each route. Glass morphism panel on the left: `w-64 bg-white/5 backdrop-blur-md border-r border-white/10`.

```typescript
'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { motion } from 'framer-motion';
import {
  LayoutDashboard, Moon, Activity, Flame, Heart, MessageSquare, Settings,
} from 'lucide-react';

const NAV = [
  { label: 'Dashboard', href: '/', icon: LayoutDashboard },
  { label: 'Sleep', href: '/sleep', icon: Moon },
  { label: 'Readiness', href: '/readiness', icon: Activity },
  { label: 'Activity', href: '/activity', icon: Flame },
  { label: 'Heart', href: '/heart', icon: Heart },
  { label: 'Chat', href: '/chat', icon: MessageSquare },
  { label: 'Settings', href: '/settings', icon: Settings },
];

export default function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-64 h-screen sticky top-0 flex flex-col bg-white/5 backdrop-blur-md border-r border-white/10 z-20">
      {/* Logo */}
      <div className="px-6 py-5 border-b border-white/10">
        <span className="text-xl font-bold tracking-tight text-white">Kairos</span>
        <span className="ml-2 text-xs text-accent-cyan font-medium">+ Sage</span>
      </div>

      {/* Nav */}
      <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
        {NAV.map(({ label, href, icon: Icon }) => {
          const active = pathname === href || (href !== '/' && pathname?.startsWith(href));
          return (
            <motion.div key={href} whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}>
              <Link
                href={href}
                className={`flex items-center gap-3 px-4 py-2.5 rounded-xl text-sm font-medium transition-colors ${
                  active
                    ? 'bg-accent-purple/20 text-white border border-accent-purple/40'
                    : 'text-text-muted hover:text-white hover:bg-white/5'
                }`}
              >
                <Icon size={18} className={active ? 'text-accent-purple' : ''} />
                {label}
              </Link>
            </motion.div>
          );
        })}
      </nav>
    </aside>
  );
}
```

- [ ] Write `src/components/layout/Header.tsx`. Shows page title (derived from pathname), user email (fetched from `GET /api/user`), and last sync timestamp with a pulsing green dot if synced within 24 h.

```typescript
'use client';

import { useEffect, useState } from 'react';
import { usePathname } from 'next/navigation';
import { getUser } from '@/lib/api';
import type { User } from '@/lib/types';

const TITLES: Record<string, string> = {
  '/': 'Dashboard',
  '/sleep': 'Sleep',
  '/readiness': 'Readiness',
  '/activity': 'Activity',
  '/heart': 'Heart Rate',
  '/chat': 'Sage — AI Health Assistant',
  '/settings': 'Settings',
};

export default function Header() {
  const pathname = usePathname() ?? '/';
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    getUser().then(setUser).catch(() => null);
  }, []);

  const title = TITLES[pathname] ?? 'Kairos';

  const recentSync = user?.last_sync
    ? Date.now() - new Date(user.last_sync).getTime() < 86_400_000
    : false;

  return (
    <header className="flex items-center justify-between px-6 py-4 border-b border-white/10 bg-white/[0.02] backdrop-blur-sm">
      <h1 className="text-lg font-semibold text-text-primary">{title}</h1>
      <div className="flex items-center gap-4 text-sm text-text-muted">
        {user?.last_sync && (
          <span className="flex items-center gap-1.5">
            <span
              className={`w-2 h-2 rounded-full ${recentSync ? 'bg-green-400 animate-pulse' : 'bg-text-muted'}`}
            />
            Synced {new Date(user.last_sync).toLocaleString()}
          </span>
        )}
        {user?.email && (
          <span className="text-text-primary font-medium">{user.email}</span>
        )}
      </div>
    </header>
  );
}
```

---

### Task 5 — Auth pages

- [ ] Write `src/app/auth/login/page.tsx`. Centered card with Kairos logo, "Sign in with Oura" button. Button click navigates browser to `https://kairos.nimoclaw.dev/auth/login` (full page redirect, not Next.js routing). Use Framer Motion for card entrance animation.

```typescript
'use client';

import { motion } from 'framer-motion';

export default function LoginPage() {
  function handleLogin() {
    window.location.href = 'https://kairos.nimoclaw.dev/auth/login';
  }

  return (
    <div className="min-h-screen bg-[#0a0a0f] flex items-center justify-center px-4">
      <motion.div
        initial={{ opacity: 0, y: 24 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, ease: 'easeOut' }}
        className="w-full max-w-sm bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-8 text-center space-y-6"
      >
        <div>
          <h1 className="text-3xl font-bold text-white tracking-tight">Kairos</h1>
          <p className="mt-1 text-text-muted text-sm">Your AI-powered health dashboard</p>
        </div>

        <p className="text-text-muted text-sm leading-relaxed">
          Connect your Oura ring to unlock deep health insights powered by Sage, your personal AI.
        </p>

        <motion.button
          whileHover={{ scale: 1.03 }}
          whileTap={{ scale: 0.97 }}
          onClick={handleLogin}
          className="w-full py-3 rounded-xl font-semibold text-white bg-accent-purple hover:bg-[#6d28d9] transition-colors"
        >
          Sign in with Oura
        </motion.button>

        <p className="text-xs text-text-muted">
          Your data stays private — only you can access it.
        </p>
      </motion.div>
    </div>
  );
}
```

- [ ] Write `src/app/auth/callback/page.tsx`. Wrap inner component in `<Suspense>` (required for static export + `useSearchParams`). On mount, reads `?token=` from URL, calls `setToken()`, then pushes to `/`. Shows loading spinner while processing.

```typescript
'use client';

import { Suspense, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { setToken } from '@/lib/auth';

function CallbackInner() {
  const router = useRouter();
  const params = useSearchParams();

  useEffect(() => {
    const token = params.get('token');
    if (token) {
      setToken(token);
      router.replace('/');
    } else {
      router.replace('/auth/login');
    }
  }, [params, router]);

  return (
    <div className="min-h-screen bg-[#0a0a0f] flex items-center justify-center">
      <div className="text-center space-y-4">
        <div className="w-10 h-10 border-2 border-accent-purple border-t-transparent rounded-full animate-spin mx-auto" />
        <p className="text-text-muted text-sm">Signing you in...</p>
      </div>
    </div>
  );
}

export default function CallbackPage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen bg-[#0a0a0f] flex items-center justify-center">
          <div className="w-10 h-10 border-2 border-[#7c3aed] border-t-transparent rounded-full animate-spin" />
        </div>
      }
    >
      <CallbackInner />
    </Suspense>
  );
}
```

---

### Task 6 — `MetricCard.tsx` + `ScoreRing.tsx`

- [ ] Write `src/components/ui/MetricCard.tsx`. Props: `label: string`, `value: number | string | null`, `unit?: string`, `trend?: number` (positive = up, negative = down, percentage). Glass morphism card with Framer Motion whileHover lift. Trend indicator uses up/down arrow with green/red color.

```typescript
'use client';

import { motion } from 'framer-motion';
import { TrendingUp, TrendingDown } from 'lucide-react';

interface MetricCardProps {
  label: string;
  value: number | string | null;
  unit?: string;
  trend?: number; // percentage delta, positive = improvement
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
        <span className="text-4xl font-bold text-text-primary leading-none">
          {value ?? '—'}
        </span>
        {unit && <span className="text-text-muted text-sm mb-1">{unit}</span>}
      </div>

      {trend !== undefined && (
        <div
          className={`flex items-center gap-1 text-xs font-medium ${
            trend >= 0 ? 'text-green-400' : 'text-red-400'
          }`}
        >
          {trend >= 0 ? <TrendingUp size={12} /> : <TrendingDown size={12} />}
          {Math.abs(trend).toFixed(1)}% vs last week
        </div>
      )}

      {subtitle && <p className="text-text-muted text-xs">{subtitle}</p>}
    </motion.div>
  );
}
```

- [ ] Write `src/components/ui/ScoreRing.tsx`. SVG circular progress, 0-100 range. Props: `score: number | null`, `size?: number`, `strokeWidth?: number`, `color?: string`. Animated stroke-dashoffset on mount via Framer Motion.

```typescript
'use client';

import { motion } from 'framer-motion';

interface ScoreRingProps {
  score: number | null;
  size?: number;
  strokeWidth?: number;
  color?: string;
  label?: string;
}

export default function ScoreRing({
  score,
  size = 120,
  strokeWidth = 10,
  color = '#7c3aed',
  label,
}: ScoreRingProps) {
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const progress = score != null ? (score / 100) * circumference : 0;
  const offset = circumference - progress;

  return (
    <div className="flex flex-col items-center gap-2">
      <div className="relative" style={{ width: size, height: size }}>
        <svg width={size} height={size} className="-rotate-90 absolute inset-0">
          {/* Track */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke="rgba(255,255,255,0.08)"
            strokeWidth={strokeWidth}
          />
          {/* Progress */}
          <motion.circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke={color}
            strokeWidth={strokeWidth}
            strokeLinecap="round"
            strokeDasharray={circumference}
            initial={{ strokeDashoffset: circumference }}
            animate={{ strokeDashoffset: offset }}
            transition={{ duration: 1, ease: 'easeOut' }}
          />
        </svg>
        {/* Centred label */}
        <div
          className="absolute inset-0 flex flex-col items-center justify-center"
        >
          <span className="text-2xl font-bold text-text-primary">
            {score ?? '—'}
          </span>
          {label && <span className="text-text-muted text-xs mt-0.5">{label}</span>}
        </div>
      </div>
    </div>
  );
}
```

---

### Task 7 — Chart components

- [ ] Write `src/components/ui/TrendChart.tsx`. Recharts `LineChart` wrapper. Props: `data: Array<{ date: string; value: number | null }>`, `color?: string`, `label?: string`, `unit?: string`. Dark themed: transparent background, subtle grid in `rgba(255,255,255,0.06)`, custom `Tooltip` with glass style.

```typescript
'use client';

import {
  LineChart, Line, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer,
} from 'recharts';
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

export default function TrendChart({
  data,
  color = '#7c3aed',
  label,
  unit,
}: TrendChartProps) {
  const chartData = data.map((d) => ({ ...d, value: d.value ?? undefined }));

  return (
    <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-5">
      {label && (
        <p className="text-text-muted text-xs uppercase tracking-wider mb-4">{label}</p>
      )}
      <ResponsiveContainer width="100%" height={200}>
        <LineChart data={chartData} margin={{ top: 4, right: 8, left: -16, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
          <XAxis
            dataKey="date"
            tick={{ fill: '#94a3b8', fontSize: 11 }}
            axisLine={false}
            tickLine={false}
            tickFormatter={(v: string) => v.slice(5)}
          />
          <YAxis
            tick={{ fill: '#94a3b8', fontSize: 11 }}
            axisLine={false}
            tickLine={false}
            unit={unit}
          />
          <Tooltip content={<GlassTooltip />} />
          <Line
            type="monotone"
            dataKey="value"
            stroke={color}
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 4, fill: color, stroke: '#0a0a0f', strokeWidth: 2 }}
            connectNulls
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
```

- [ ] Write `src/components/ui/SleepStagesChart.tsx`. Stacked `BarChart` (REM / Deep / Light / Awake) in minutes. Props: `data: DailySleep[]`. Converts seconds to minutes. Colors: purple/cyan/green/amber.

```typescript
'use client';

import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid,
  Tooltip, Legend, ResponsiveContainer,
} from 'recharts';
import type { TooltipProps } from 'recharts';
import type { DailySleep } from '@/lib/types';

interface Props { data: DailySleep[] }

function GlassTooltip({ active, payload, label }: TooltipProps<number, string>) {
  if (!active || !payload?.length) return null;
  return (
    <div className="bg-[#1a1a2e] border border-white/10 rounded-xl px-3 py-2 text-sm shadow-xl space-y-1">
      <p className="text-text-muted text-xs">{label}</p>
      {payload.map((p) => (
        <p key={p.name} style={{ color: p.color }} className="font-medium">
          {p.name}: {p.value} min
        </p>
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
```

- [ ] Write `src/components/ui/ActivityChart.tsx`. `BarChart` of steps per day. Props: `data: DailyActivity[]`. Accent cyan bars with goal line at 10,000 steps using `ReferenceLine`.

```typescript
'use client';

import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, ReferenceLine,
} from 'recharts';
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
  const chartData = data.map((d) => ({
    date: d.date.slice(5),
    steps: d.steps ?? 0,
  }));

  return (
    <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-5">
      <p className="text-text-muted text-xs uppercase tracking-wider mb-4">Daily Steps</p>
      <ResponsiveContainer width="100%" height={200}>
        <BarChart data={chartData} margin={{ top: 4, right: 8, left: -16, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" vertical={false} />
          <XAxis dataKey="date" tick={{ fill: '#94a3b8', fontSize: 11 }} axisLine={false} tickLine={false} />
          <YAxis
            tick={{ fill: '#94a3b8', fontSize: 11 }}
            axisLine={false}
            tickLine={false}
            tickFormatter={(v: number) => `${(v / 1000).toFixed(0)}k`}
          />
          <Tooltip content={<GlassTooltip />} cursor={{ fill: 'rgba(255,255,255,0.03)' }} />
          <ReferenceLine
            y={10000}
            stroke="#f59e0b"
            strokeDasharray="4 4"
            label={{ value: 'Goal', fill: '#f59e0b', fontSize: 11 }}
          />
          <Bar dataKey="steps" fill="#06b6d4" radius={[4, 4, 0, 0]} maxBarSize={28} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
```

---

### Task 8 — Dashboard page (`/`)

- [ ] Write `src/app/page.tsx` as a `'use client'` component. Fetches last 30 days of sleep, readiness, HRV, activity, SpO2, stress, and user. Layout:
  - **Top row:** 4 `MetricCard`s (today's sleep score, readiness score, HRV rmssd, steps).
  - **Middle:** `TrendChart` showing 30-day sleep score trend (purple).
  - **Bottom row:** 3 smaller glass cards (last sync time, today's avg SpO2, today's stress summary text).
- Loading state: skeleton shimmer via `animate-pulse`.
- Error state: inline error with retry button.

```typescript
'use client';

import { useEffect, useState, useCallback } from 'react';
import { motion } from 'framer-motion';
import MetricCard from '@/components/ui/MetricCard';
import TrendChart from '@/components/ui/TrendChart';
import {
  getSleep, getReadiness, getHRV, getActivity,
  getUser, getSpO2, getStress,
} from '@/lib/api';
import type {
  DailySleep, DailyReadiness, DailyHRV, DailyActivity,
  User, DailySpO2, DailyStress,
} from '@/lib/types';

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

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const r = dateRange(30);
      const [s, rd, h, a, u, sp, st] = await Promise.all([
        getSleep(r), getReadiness(r), getHRV(r),
        getActivity(r), getUser(), getSpO2(r), getStress(r),
      ]);
      setSleep(s); setReadiness(rd); setHRV(h); setActivity(a);
      setUser(u); setSpo2(sp); setStress(st);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

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
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-32 bg-white/5 rounded-2xl" />
          ))}
        </div>
        <div className="h-56 bg-white/5 rounded-2xl" />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {[...Array(3)].map((_, i) => (
            <div key={i} className="h-24 bg-white/5 rounded-2xl" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-64 gap-4 text-text-muted">
        <p>Failed to load data: {error}</p>
        <button
          onClick={load}
          className="px-4 py-2 rounded-xl bg-accent-purple/20 border border-accent-purple/40 text-white text-sm hover:bg-accent-purple/30 transition"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4 }}
      className="space-y-6"
    >
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard label="Sleep Score" value={todaySleep?.score ?? null} unit="/100" />
        <MetricCard label="Readiness" value={todayReadiness?.score ?? null} unit="/100" />
        <MetricCard
          label="HRV (RMSSD)"
          value={todayHRV?.rmssd != null ? Math.round(todayHRV.rmssd) : null}
          unit="ms"
        />
        <MetricCard
          label="Steps"
          value={todayActivity?.steps != null ? todayActivity.steps.toLocaleString() : null}
        />
      </div>

      <TrendChart data={sleepTrend} label="30-Day Sleep Score Trend" color="#7c3aed" />

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-5">
          <p className="text-text-muted text-xs uppercase tracking-wider mb-2">Last Sync</p>
          <p className="text-text-primary text-sm font-medium">
            {user?.last_sync ? new Date(user.last_sync).toLocaleString() : '—'}
          </p>
        </div>
        <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-5">
          <p className="text-text-muted text-xs uppercase tracking-wider mb-2">SpO2 (avg)</p>
          <p className="text-text-primary text-2xl font-bold">
            {todaySpo2?.avg_spo2 != null ? `${todaySpo2.avg_spo2.toFixed(1)}%` : '—'}
          </p>
        </div>
        <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-5">
          <p className="text-text-muted text-xs uppercase tracking-wider mb-2">Stress Summary</p>
          <p className="text-text-primary text-sm">{todayStress?.day_summary ?? '—'}</p>
        </div>
      </div>
    </motion.div>
  );
}
```

---

### Task 9 — Sleep page (`/sleep`)

- [ ] Write `src/app/sleep/page.tsx`. 30-day date range. Shows:
  - `ScoreRing` (sleep score, purple) + 3 `MetricCard`s (total sleep hours, efficiency %, latency min).
  - `SleepStagesChart` (stacked bars).
  - `TrendChart` of total sleep duration in hours.
  - Scrollable data table: last 14 nights — Date / Score / Total / REM / Deep / Efficiency.

```typescript
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
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-48 bg-white/5 rounded-2xl" />
        <div className="h-56 bg-white/5 rounded-2xl" />
      </div>
    );
  }

  const today = data[data.length - 1];
  const durationTrend = data.map((d) => ({
    date: d.date,
    value: toHours(d.total_sleep_duration),
  }));

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

      <TrendChart
        data={durationTrend}
        label="Total Sleep Duration (30 days)"
        color="#06b6d4"
        unit="h"
      />

      <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-white/10">
                {['Date', 'Score', 'Total (h)', 'REM (min)', 'Deep (min)', 'Efficiency (%)'].map((h) => (
                  <th
                    key={h}
                    className="px-4 py-3 text-left text-text-muted font-medium text-xs uppercase tracking-wider"
                  >
                    {h}
                  </th>
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
```

---

### Task 10 — Readiness page (`/readiness`)

- [ ] Write `src/app/readiness/page.tsx`. `ScoreRing` (cyan) + contributor `MetricCard`s: HRV Balance, Body Temp, Resting HR, Activity Balance, Sleep Balance, Recovery Index. 30-day `TrendChart` in cyan.

```typescript
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

  const tempStr =
    today?.body_temperature != null
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
```

---

### Task 11 — Activity page (`/activity`)

- [ ] Write `src/app/activity/page.tsx`. `ScoreRing` (green) + `MetricCard`s: steps, calories, active calories, MET minutes. `ActivityChart` (30 days). `TrendChart` of calories (green). Workout list from `GET /api/metrics/workouts`.

```typescript
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
          <MetricCard
            label="Steps"
            value={today?.steps != null ? today.steps.toLocaleString() : null}
          />
          <MetricCard label="Calories" value={today?.calories ?? null} unit="kcal" />
          <MetricCard label="Active Cal" value={today?.active_calories ?? null} unit="kcal" />
          <MetricCard label="MET Minutes" value={today?.met_minutes ?? null} unit="min" />
        </div>
      </div>

      <ActivityChart data={data} />
      <TrendChart data={calorieTrend} label="30-Day Calories Burned" color="#10b981" unit="kcal" />

      {workouts.length > 0 && (
        <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl overflow-hidden">
          <p className="px-5 py-4 text-text-muted text-xs uppercase tracking-wider border-b border-white/10">
            Recent Workouts
          </p>
          <div className="divide-y divide-white/5">
            {[...workouts].reverse().slice(0, 10).map((w) => (
              <div key={w.id} className="px-5 py-3 flex items-center justify-between">
                <div>
                  <p className="text-text-primary text-sm font-medium capitalize">
                    {w.activity ?? 'Workout'}
                  </p>
                  <p className="text-text-muted text-xs">
                    {new Date(w.start_datetime).toLocaleString()}
                  </p>
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
```

---

### Task 12 — Heart rate page (`/heart`)

- [ ] Write `src/app/heart/page.tsx`. Fetches HRV, SpO2, and readiness (for resting HR) for 30 days. Shows `MetricCard`s for latest RMSSD, BDI, avg SpO2, min SpO2, then three `TrendChart`s: RMSSD, resting HR, avg SpO2.

```typescript
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

  const hrvTrend = hrv.map((d) => ({
    date: d.date,
    value: d.rmssd != null ? +d.rmssd.toFixed(1) : null,
  }));
  const rhrTrend = readiness.map((d) => ({ date: d.date, value: d.resting_heart_rate }));
  const spo2Trend = spo2.map((d) => ({ date: d.date, value: d.avg_spo2 }));

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-6">
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          label="HRV (RMSSD)"
          value={latestHRV?.rmssd != null ? +latestHRV.rmssd.toFixed(1) : null}
          unit="ms"
        />
        <MetricCard
          label="BDI"
          value={latestHRV?.bdi != null ? +latestHRV.bdi.toFixed(1) : null}
        />
        <MetricCard
          label="Avg SpO2"
          value={latestSpo2?.avg_spo2 != null ? `${latestSpo2.avg_spo2.toFixed(1)}%` : null}
        />
        <MetricCard
          label="Min SpO2"
          value={latestSpo2?.min_spo2 != null ? `${latestSpo2.min_spo2.toFixed(1)}%` : null}
        />
      </div>

      <TrendChart data={hrvTrend} label="HRV RMSSD (30 days)" color="#7c3aed" unit="ms" />
      <TrendChart data={rhrTrend} label="Resting Heart Rate (30 days)" color="#06b6d4" unit="bpm" />
      <TrendChart data={spo2Trend} label="Avg Blood Oxygen SpO2 (30 days)" color="#10b981" unit="%" />
    </motion.div>
  );
}
```

---

### Task 13 — Chat components

- [ ] Write `src/components/chat/SourceToggle.tsx`. Collapsible SQL block, animated via Framer Motion `AnimatePresence`.

```typescript
'use client';

import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { ChevronDown } from 'lucide-react';

interface Props { sql: string }

export default function SourceToggle({ sql }: Props) {
  const [open, setOpen] = useState(false);

  return (
    <div className="mt-2">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1 text-xs text-text-muted hover:text-text-primary transition-colors"
      >
        <ChevronDown
          size={12}
          className={`transition-transform ${open ? 'rotate-180' : ''}`}
        />
        {open ? 'Hide SQL' : 'Show SQL'}
      </button>
      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="overflow-hidden"
          >
            <pre className="mt-2 p-3 bg-black/30 border border-white/10 rounded-lg text-xs text-[#06b6d4] overflow-x-auto whitespace-pre-wrap break-all">
              {sql}
            </pre>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
```

- [ ] Write `src/components/chat/MessageBubble.tsx`. User messages: right-aligned, purple bg (`bg-accent-purple`). Assistant messages: left-aligned, glass (`bg-white/5 backdrop-blur-md border border-white/10`). Renders `sql_used` via `SourceToggle`.

```typescript
'use client';

import { motion } from 'framer-motion';
import SourceToggle from './SourceToggle';
import type { ChatMessage } from '@/lib/types';

interface Props { message: ChatMessage }

export default function MessageBubble({ message }: Props) {
  const isUser = message.role === 'user';

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      className={`flex ${isUser ? 'justify-end' : 'justify-start'} mb-4`}
    >
      <div
        className={`max-w-[75%] px-4 py-3 rounded-2xl text-sm leading-relaxed ${
          isUser
            ? 'bg-[#7c3aed] text-white rounded-br-sm'
            : 'bg-white/5 backdrop-blur-md border border-white/10 text-[#f1f5f9] rounded-bl-sm'
        }`}
      >
        {message.content.split('\n').map((line, i, arr) => (
          <span key={i}>
            {line}
            {i < arr.length - 1 && <br />}
          </span>
        ))}
        {!isUser && message.sql_used && (
          <SourceToggle sql={message.sql_used} />
        )}
      </div>
    </motion.div>
  );
}
```

- [ ] Write `src/components/chat/ChatWindow.tsx`. Loads existing messages. Sends via `askStream()`. Parses SSE `data: ` lines, appends tokens. Handles `[DONE]`. Shows typing dots while streaming.

```typescript
'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { motion } from 'framer-motion';
import { Send } from 'lucide-react';
import MessageBubble from './MessageBubble';
import { getChatMessages, askStream } from '@/lib/api';
import type { ChatMessage } from '@/lib/types';

interface Props { sessionId: string }

export default function ChatWindow({ sessionId }: Props) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [streaming, setStreaming] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    getChatMessages(sessionId).then(setMessages).catch(() => null);
  }, [sessionId]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSend = useCallback(async () => {
    const text = input.trim();
    if (!text || streaming) return;
    setInput('');

    const userMsg: ChatMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content: text,
      sql_used: null,
      created_at: new Date().toISOString(),
    };
    setMessages((prev) => [...prev, userMsg]);

    const assistantId = crypto.randomUUID();
    setMessages((prev) => [
      ...prev,
      { id: assistantId, role: 'assistant', content: '', sql_used: null, created_at: new Date().toISOString() },
    ]);
    setStreaming(true);

    try {
      const res = await askStream(sessionId, text);
      if (!res.body) throw new Error('No response body');

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() ?? '';

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue;
          const raw = line.slice(6).trim();
          if (raw === '[DONE]') break;

          let token = raw;
          let sql: string | null = null;

          try {
            const parsed = JSON.parse(raw) as { token?: string; content?: string; sql?: string };
            token = parsed.token ?? parsed.content ?? '';
            sql = parsed.sql ?? null;
          } catch {
            // plain text token, use raw as-is
          }

          if (token || sql) {
            setMessages((prev) =>
              prev.map((m) =>
                m.id === assistantId
                  ? { ...m, content: m.content + token, sql_used: sql ?? m.sql_used }
                  : m,
              ),
            );
          }
        }
      }
    } catch (err) {
      setMessages((prev) =>
        prev.map((m) =>
          m.id === assistantId
            ? { ...m, content: `Error: ${(err as Error).message}` }
            : m,
        ),
      );
    } finally {
      setStreaming(false);
    }
  }, [input, sessionId, streaming]);

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto px-4 py-4">
        {messages.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full text-center text-[#94a3b8] gap-3">
            <p className="text-4xl">🌿</p>
            <p className="text-sm">Hi, I'm Sage — your AI health assistant.</p>
            <p className="text-xs">Ask me anything about your sleep, readiness, or activity data.</p>
          </div>
        )}
        {messages.map((m) => (
          <MessageBubble key={m.id} message={m} />
        ))}
        {streaming && (
          <div className="flex gap-1 mb-4 pl-1">
            {[0, 1, 2].map((i) => (
              <motion.span
                key={i}
                className="w-2 h-2 bg-[#7c3aed] rounded-full"
                animate={{ opacity: [0.3, 1, 0.3] }}
                transition={{ duration: 1, repeat: Infinity, delay: i * 0.2 }}
              />
            ))}
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      <div className="border-t border-white/10 p-4 bg-white/[0.02]">
        <div className="flex gap-3 items-end">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend(); }
            }}
            placeholder="Ask Sage about your health data..."
            rows={1}
            className="flex-1 resize-none bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-sm text-[#f1f5f9] placeholder-[#94a3b8] focus:outline-none focus:border-[#7c3aed]/50 transition-colors"
          />
          <motion.button
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            onClick={handleSend}
            disabled={!input.trim() || streaming}
            className="p-3 rounded-xl bg-[#7c3aed] hover:bg-[#6d28d9] disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            <Send size={18} className="text-white" />
          </motion.button>
        </div>
      </div>
    </div>
  );
}
```

---

### Task 14 — Chat page (`/chat`)

- [ ] Write `src/app/chat/page.tsx`. Creates or reuses session from `sessionStorage`. Full-height layout.

```typescript
'use client';

import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import ChatWindow from '@/components/chat/ChatWindow';
import { createChatSession } from '@/lib/api';

const SESSION_KEY = 'kairos_chat_session_id';

export default function ChatPage() {
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const cached = sessionStorage.getItem(SESSION_KEY);
    if (cached) { setSessionId(cached); return; }
    createChatSession()
      .then(({ id }) => { sessionStorage.setItem(SESSION_KEY, id); setSessionId(id); })
      .catch((e) => setError((e as Error).message));
  }, []);

  if (error) {
    return (
      <div className="flex items-center justify-center h-[calc(100vh-120px)] text-[#94a3b8] text-sm">
        Failed to start session: {error}
      </div>
    );
  }

  if (!sessionId) {
    return (
      <div className="flex items-center justify-center h-[calc(100vh-120px)]">
        <div className="w-8 h-8 border-2 border-[#7c3aed] border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      className="h-[calc(100vh-120px)] flex flex-col bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl overflow-hidden"
    >
      <ChatWindow sessionId={sessionId} />
    </motion.div>
  );
}
```

---

### Task 15 — Settings page (`/settings`)

- [ ] Write `src/app/settings/page.tsx`. User info, sync button, chat reset, disconnect.

```typescript
'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { motion } from 'framer-motion';
import { RefreshCw, LogOut, MessageSquare } from 'lucide-react';
import { getUser, postSync } from '@/lib/api';
import { clearToken } from '@/lib/auth';
import type { User } from '@/lib/types';

export default function SettingsPage() {
  const [user, setUser] = useState<User | null>(null);
  const [syncing, setSyncing] = useState(false);
  const [banner, setBanner] = useState<{ text: string; ok: boolean } | null>(null);
  const router = useRouter();

  useEffect(() => { getUser().then(setUser).catch(() => null); }, []);

  async function handleSync() {
    setSyncing(true);
    setBanner(null);
    try {
      const res = await postSync();
      setBanner({ text: res.message ?? 'Sync started', ok: true });
      const updated = await getUser();
      setUser(updated);
    } catch (e) {
      setBanner({ text: (e as Error).message, ok: false });
    } finally {
      setSyncing(false);
    }
  }

  function handleDisconnect() {
    clearToken();
    sessionStorage.removeItem('kairos_chat_session_id');
    router.replace('/auth/login');
  }

  function handleResetChat() {
    sessionStorage.removeItem('kairos_chat_session_id');
    setBanner({ text: 'Chat session reset. A new session will start on next visit.', ok: true });
  }

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="max-w-lg space-y-6">
      {/* Account info */}
      <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-6 space-y-4">
        <p className="text-[#94a3b8] text-xs uppercase tracking-wider">Account</p>
        <div className="space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-[#94a3b8]">Email</span>
            <span className="text-[#f1f5f9] font-medium">{user?.email ?? '—'}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-[#94a3b8]">Last Sync</span>
            <span className="text-[#f1f5f9]">
              {user?.last_sync ? new Date(user.last_sync).toLocaleString() : 'Never'}
            </span>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-6 space-y-3">
        <p className="text-[#94a3b8] text-xs uppercase tracking-wider mb-4">Actions</p>

        <motion.button
          whileHover={{ scale: 1.01 }}
          whileTap={{ scale: 0.99 }}
          onClick={handleSync}
          disabled={syncing}
          className="w-full flex items-center justify-center gap-2 py-3 rounded-xl bg-[#7c3aed]/20 border border-[#7c3aed]/40 text-white text-sm font-medium hover:bg-[#7c3aed]/30 disabled:opacity-50 transition"
        >
          <RefreshCw size={16} className={syncing ? 'animate-spin' : ''} />
          {syncing ? 'Syncing...' : 'Sync Oura Data Now'}
        </motion.button>

        <motion.button
          whileHover={{ scale: 1.01 }}
          whileTap={{ scale: 0.99 }}
          onClick={handleResetChat}
          className="w-full flex items-center justify-center gap-2 py-3 rounded-xl bg-white/5 border border-white/10 text-[#94a3b8] text-sm font-medium hover:text-white hover:border-white/20 transition"
        >
          <MessageSquare size={16} />
          Start New Sage Chat Session
        </motion.button>

        <motion.button
          whileHover={{ scale: 1.01 }}
          whileTap={{ scale: 0.99 }}
          onClick={handleDisconnect}
          className="w-full flex items-center justify-center gap-2 py-3 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm font-medium hover:bg-red-500/20 transition"
        >
          <LogOut size={16} />
          Disconnect &amp; Sign Out
        </motion.button>
      </div>

      {banner && (
        <motion.div
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          className={`px-4 py-3 rounded-xl text-sm font-medium border ${
            banner.ok
              ? 'bg-green-500/10 border-green-500/30 text-green-400'
              : 'bg-red-500/10 border-red-500/30 text-red-400'
          }`}
        >
          {banner.text}
        </motion.div>
      )}
    </motion.div>
  );
}
```

---

### Task 16 — Static export build + deploy

- [ ] Confirm `next.config.ts` has `output: 'export'` and `images: { unoptimized: true }`.
- [ ] Verify `auth/callback/page.tsx` wraps `useSearchParams` consumer in `<Suspense>` (already included in Task 5).
- [ ] Build:

```bash
cd Kairos/frontend
npm run build
# Expected: /out directory with all routes pre-rendered as static HTML
```

- [ ] Deploy via rsync:

```bash
rsync -avz --delete out/ <user>@<VM_IP>:/var/www/kairos/
```

- [ ] Confirm Nginx config includes SPA fallback — create or update `/etc/nginx/sites-available/kairos`:

```nginx
server {
    listen 80;
    server_name kairos.nimoclaw.dev;
    root /var/www/kairos;
    index index.html;

    location / {
        try_files $uri $uri/ $uri.html /index.html;
    }
}
```

- [ ] Test routes: `/`, `/sleep`, `/readiness`, `/activity`, `/heart`, `/chat`, `/settings`, `/auth/callback?token=test` — all must resolve without Nginx 404.
- [ ] Add `/out` to `Kairos/frontend/.gitignore`.

---

## Self-Review Checklist

- [ ] **No SSR / API routes** — every page is `'use client'`; no server actions; no `export const dynamic`.
- [ ] **Static export compatibility** — `useSearchParams()` in callback page is wrapped in `<Suspense>`. No `next/headers`, no `cookies()`.
- [ ] **Auth guard** — `layout.tsx` redirects unauthenticated users via `useEffect`; auth routes bypass guard via `pathname.startsWith('/auth')` check.
- [ ] **Type consistency** — all API return types match `types.ts` interfaces; no `any`; null fields handled with `?? null` or `?? '—'`.
- [ ] **No placeholders** — every component has real Tailwind classes, real Recharts props, real API calls with proper endpoint paths.
- [ ] **SSE streaming** — `ChatWindow` uses `ReadableStream`, parses `data: ` prefix, handles JSON token payloads with plain-text fallback, stops on `[DONE]`.
- [ ] **Null safety** — all `number | null` fields guarded before arithmetic (`!= null`) or display (`?? '—'`).
- [ ] **Design tokens consistent** — `#0a0a0f` background, `bg-white/5 backdrop-blur-md border border-white/10` glass cards, `#7c3aed` purple, `#06b6d4` cyan used across all pages.
- [ ] **ScoreRing layout fixed** — inner label uses `absolute inset-0` overlay, not negative margin hack.
- [ ] **Recharts imports** — `TooltipProps` imported as type from `'recharts'` in every chart component.
- [ ] **Lucide icons verified** — `LayoutDashboard`, `Moon`, `Activity`, `Flame`, `Heart`, `MessageSquare`, `Settings`, `TrendingUp`, `TrendingDown`, `ChevronDown`, `Send`, `RefreshCw`, `LogOut` all exist in `lucide-react@0.411.0`.
- [ ] **Inter font** — loaded via `next/font/google` in `layout.tsx`; applied as `font-family` class on `<body>`.
- [ ] **Build artifact excluded** — `/out` in `.gitignore`; only source files committed.
- [ ] **Nginx SPA fallback** — `try_files $uri $uri/ $uri.html /index.html` covers static export `trailingSlash: true` paths.
