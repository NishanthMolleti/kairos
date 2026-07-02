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

  useEffect(() => { getChatMessages(sessionId).then(data => setMessages(data ?? [])).catch(() => null); }, [sessionId]);
  useEffect(() => { bottomRef.current?.scrollIntoView({ behavior: 'smooth' }); }, [messages]);

  const handleSend = useCallback(async () => {
    const text = input.trim();
    if (!text || streaming) return;
    setInput('');

    const userMsg: ChatMessage = { id: crypto.randomUUID(), role: 'user', content: text, sql_used: null, created_at: new Date().toISOString() };
    setMessages((prev) => [...prev, userMsg]);

    const assistantId = crypto.randomUUID();
    setMessages((prev) => [...prev, { id: assistantId, role: 'assistant', content: '', sql_used: null, created_at: new Date().toISOString() }]);
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
          const raw = line.slice(6);
          if (raw.trim() === '[DONE]') break;
          setMessages((prev) => prev.map((m) => m.id === assistantId ? { ...m, content: m.content + raw } : m));
        }
      }
    } catch (err) {
      setMessages((prev) => prev.map((m) => m.id === assistantId ? { ...m, content: `Error: ${(err as Error).message}` } : m));
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
            <p className="text-sm">Hi, I&apos;m Sage — your AI health assistant.</p>
            <p className="text-xs">Ask me anything about your sleep, readiness, or activity data.</p>
          </div>
        )}
        {messages.map((m) => <MessageBubble key={m.id} message={m} />)}
        {streaming && (
          <div className="flex gap-1 mb-4 pl-1">
            {[0, 1, 2].map((i) => (
              <motion.span key={i} className="w-2 h-2 bg-[#7c3aed] rounded-full" animate={{ opacity: [0.3, 1, 0.3] }} transition={{ duration: 1, repeat: Infinity, delay: i * 0.2 }} />
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
            onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend(); } }}
            placeholder="Ask Sage about your health data..."
            rows={1}
            className="flex-1 resize-none bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-sm text-[#f1f5f9] placeholder-[#94a3b8] focus:outline-none focus:border-[#7c3aed]/50 transition-colors"
          />
          <motion.button whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }} onClick={handleSend} disabled={!input.trim() || streaming} className="p-3 rounded-xl bg-[#7c3aed] hover:bg-[#6d28d9] disabled:opacity-40 disabled:cursor-not-allowed transition-colors">
            <Send size={18} className="text-white" />
          </motion.button>
        </div>
      </div>
    </div>
  );
}
