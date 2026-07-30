import type { ReactNode } from 'react';
import './globals.css';
import { Providers } from './providers';

export const metadata = { title: 'DETA MRP', description: 'Material Requirements Planning' };

export default function RootLayout({ children }: { children: ReactNode }) {
  return <html lang="id"><body><Providers>{children}</Providers></body></html>;
}
