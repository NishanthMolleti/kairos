'use client';

import { motion } from 'framer-motion';
import SourceToggle from './SourceToggle';
import type { ChatMessage } from '@/lib/types';

interface Props { message: ChatMessage }

export default function MessageBubble({ message }: Props) {
  const isUser = message.role === 'user';
  return (
    <motion.div initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} className={`flex ${isUser ? 'justify-end' : 'justify-start'} mb-4`}>
      <div className={`max-w-[75%] px-4 py-3 rounded-2xl text-sm leading-relaxed ${isUser ? 'bg-[#7c3aed] text-white rounded-br-sm' : 'bg-white/5 backdrop-blur-md border border-white/10 text-[#f1f5f9] rounded-bl-sm'}`}>
        {message.content.split('\n').map((line, i, arr) => (
          <span key={i}>{line}{i < arr.length - 1 && <br />}</span>
        ))}
        {!isUser && message.sql_used && <SourceToggle sql={message.sql_used} />}
      </div>
    </motion.div>
  );
}
