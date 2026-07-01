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
            <span className={`w-2 h-2 rounded-full ${recentSync ? 'bg-green-400 animate-pulse' : 'bg-text-muted'}`} />
            Synced {new Date(user.last_sync).toLocaleString()}
          </span>
        )}
        {user?.email && <span className="text-text-primary font-medium">{user.email}</span>}
      </div>
    </header>
  );
}
