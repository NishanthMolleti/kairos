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
        <p className="text-xs text-text-muted">Your data stays private — only you can access it.</p>
      </motion.div>
    </div>
  );
}
