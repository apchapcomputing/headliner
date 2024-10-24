'use client';
import type { Metadata } from 'next';
import { SessionProvider } from 'next-auth/react';
import localFont from 'next/font/local';
import { Inter } from 'next/font/google';
import './globals.css';

const geistSans = localFont({
  src: './fonts/GeistVF.woff',
  variable: '--font-geist-sans',
  weight: '100 900'
});
const geistMono = localFont({
  src: './fonts/GeistMonoVF.woff',
  variable: '--font-geist-mono',
  weight: '100 900'
});

const inter = Inter({ subsets: ['latin'] });

const metadata: Metadata = {
  title: 'Headline Generator',
  description: 'AI-powered headline generation from your favorite content'
};

export default function RootLayout({
  children
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang='en'>
      <body className={inter.className}>
        <SessionProvider>
          <div className='min-h-screen bg-gray-50'>{children}</div>
        </SessionProvider>
      </body>
    </html>
  );
}
