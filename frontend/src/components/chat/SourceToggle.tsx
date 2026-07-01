'use client';

import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { ChevronDown } from 'lucide-react';

interface Props { sql: string }

export default function SourceToggle({ sql }: Props) {
  const [open, setOpen] = useState(false);
  return (
    <div className="mt-2">
      <button onClick={() => setOpen((v) => !v)} className="flex items-center gap-1 text-xs text-text-muted hover:text-text-primary transition-colors">
        <ChevronDown size={12} className={`transition-transform ${open ? 'rotate-180' : ''}`} />
        {open ? 'Hide SQL' : 'Show SQL'}
      </button>
      <AnimatePresence>
        {open && (
          <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }} exit={{ height: 0, opacity: 0 }} transition={{ duration: 0.2 }} className="overflow-hidden">
            <pre className="mt-2 p-3 bg-black/30 border border-white/10 rounded-lg text-xs text-[#06b6d4] overflow-x-auto whitespace-pre-wrap break-all">{sql}</pre>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
