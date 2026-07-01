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

  if (error) return <div className="flex items-center justify-center h-[calc(100vh-120px)] text-[#94a3b8] text-sm">Failed to start session: {error}</div>;
  if (!sessionId) return <div className="flex items-center justify-center h-[calc(100vh-120px)]"><div className="w-8 h-8 border-2 border-[#7c3aed] border-t-transparent rounded-full animate-spin" /></div>;

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="h-[calc(100vh-120px)] flex flex-col bg-white/5 backdrop-blur-md border border-white/10 rounded-2xl overflow-hidden">
      <ChatWindow sessionId={sessionId} />
    </motion.div>
  );
}
