import type { ReactNode } from 'react';
import './globals.css';
import { Providers } from './providers';

export default function RootLayout({ children }: { children: ReactNode }) {
  return <html lang="id"><body><Providers>{children}</Providers></body></html>;
}
