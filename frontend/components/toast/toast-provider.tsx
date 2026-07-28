'use client';

import { createContext, ReactNode, useCallback, useContext, useEffect, useState } from 'react';

type ToastContextValue = {
  showSuccess: (message: string) => void;
};

type ToastState = {
  id: number;
  message: string;
};

const ToastContext = createContext<ToastContextValue>({ showSuccess: () => undefined });

export function useToast() {
  return useContext(ToastContext);
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toast, setToast] = useState<ToastState | null>(null);

  const showSuccess = useCallback((message: string) => {
    setToast(current => ({ id: (current?.id ?? 0) + 1, message }));
  }, []);

  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(() => setToast(current => current?.id === toast.id ? null : current), 4_000);
    return () => window.clearTimeout(timer);
  }, [toast]);

  return <ToastContext.Provider value={{ showSuccess }}>
    {children}
    {toast ? <div className="success-toast" role="status" aria-live="polite">
      <span className="success-toast-icon" aria-hidden="true">✓</span>
      <span>{toast.message}</span>
      <button type="button" aria-label="Dismiss notification" onClick={() => setToast(null)}>×</button>
    </div> : null}
  </ToastContext.Provider>;
}
