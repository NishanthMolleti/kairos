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
