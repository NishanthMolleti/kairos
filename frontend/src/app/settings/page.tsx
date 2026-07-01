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
    setSyncing(true); setBanner(null);
    try {
      const res = await postSync();
      setBanner({ text: res.message ?? 'Sync started', ok: true });
      const updated = await getUser();
      setUser(updated);
    } catch (e) {
      setBanner({ text: (e as Error).message, ok: false });
    } finally { setSyncing(false); }
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
      <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-6 space-y-4">
        <p className="text-[#94a3b8] text-xs uppercase tracking-wider">Account</p>
        <div className="space-y-2 text-sm">
          <div className="flex justify-between"><span className="text-[#94a3b8]">Email</span><span className="text-[#f1f5f9] font-medium">{user?.email ?? '—'}</span></div>
          <div className="flex justify-between"><span className="text-[#94a3b8]">Last Sync</span><span className="text-[#f1f5f9]">{user?.last_sync ? new Date(user.last_sync).toLocaleString() : 'Never'}</span></div>
        </div>
      </div>
      <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl p-6 space-y-3">
        <p className="text-[#94a3b8] text-xs uppercase tracking-wider mb-4">Actions</p>
        <motion.button whileHover={{ scale: 1.01 }} whileTap={{ scale: 0.99 }} onClick={handleSync} disabled={syncing} className="w-full flex items-center justify-center gap-2 py-3 rounded-xl bg-[#7c3aed]/20 border border-[#7c3aed]/40 text-white text-sm font-medium hover:bg-[#7c3aed]/30 disabled:opacity-50 transition">
          <RefreshCw size={16} className={syncing ? 'animate-spin' : ''} />
          {syncing ? 'Syncing...' : 'Sync Oura Data Now'}
        </motion.button>
        <motion.button whileHover={{ scale: 1.01 }} whileTap={{ scale: 0.99 }} onClick={handleResetChat} className="w-full flex items-center justify-center gap-2 py-3 rounded-xl bg-white/5 border border-white/10 text-[#94a3b8] text-sm font-medium hover:text-white hover:border-white/20 transition">
          <MessageSquare size={16} />
          Start New Sage Chat Session
        </motion.button>
        <motion.button whileHover={{ scale: 1.01 }} whileTap={{ scale: 0.99 }} onClick={handleDisconnect} className="w-full flex items-center justify-center gap-2 py-3 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm font-medium hover:bg-red-500/20 transition">
          <LogOut size={16} />
          Disconnect &amp; Sign Out
        </motion.button>
      </div>
      {banner && (
        <motion.div initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} className={`px-4 py-3 rounded-xl text-sm font-medium border ${banner.ok ? 'bg-green-500/10 border-green-500/30 text-green-400' : 'bg-red-500/10 border-red-500/30 text-red-400'}`}>
          {banner.text}
        </motion.div>
      )}
    </motion.div>
  );
}
