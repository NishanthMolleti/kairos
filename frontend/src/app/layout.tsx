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
