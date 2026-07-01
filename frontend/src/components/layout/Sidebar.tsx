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
      <div className="px-6 py-5 border-b border-white/10">
        <span className="text-xl font-bold tracking-tight text-white">Kairos</span>
        <span className="ml-2 text-xs text-accent-cyan font-medium">+ Sage</span>
      </div>
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
