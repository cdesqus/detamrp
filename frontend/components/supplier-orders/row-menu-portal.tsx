'use client';

import { CSSProperties, ReactNode, useCallback, useLayoutEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';

type RowMenuPortalProps = {
  trigger: HTMLElement;
  ariaLabel: string;
  onClose: (restoreFocus: boolean) => void;
  children: ReactNode;
};

const viewportMargin = 8;
const triggerGap = 4;

export function RowMenuPortal({ trigger, ariaLabel, onClose, children }: RowMenuPortalProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [style, setStyle] = useState<CSSProperties>({ position: 'fixed', visibility: 'hidden' });

  const position = useCallback(() => {
    const menu = menuRef.current;
    if (!menu || !trigger.isConnected) return;
    const anchor = trigger.getBoundingClientRect();
    const bounds = menu.getBoundingClientRect();
    const width = bounds.width || 158;
    const height = bounds.height;
    const below = window.innerHeight - anchor.bottom - triggerGap;
    const above = anchor.top - triggerGap;
    const openUpward = height > below && above > below;
    const top = openUpward ? Math.max(viewportMargin, anchor.top - triggerGap - height) : Math.min(window.innerHeight - viewportMargin - height, anchor.bottom + triggerGap);
    const left = Math.min(Math.max(viewportMargin, anchor.left), window.innerWidth - viewportMargin - width);
    setStyle({ position: 'fixed', top, left, visibility: 'visible' });
  }, [trigger]);

  useLayoutEffect(() => {
    position();
    const dismissOutside = (event: PointerEvent) => {
      const target = event.target as Node;
      if (!menuRef.current?.contains(target) && !trigger.contains(target)) onClose(false);
    };
    const dismissEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        onClose(true);
      }
    };
    document.addEventListener('pointerdown', dismissOutside);
    document.addEventListener('keydown', dismissEscape);
    window.addEventListener('resize', position);
    window.addEventListener('scroll', position, true);
    return () => {
      document.removeEventListener('pointerdown', dismissOutside);
      document.removeEventListener('keydown', dismissEscape);
      window.removeEventListener('resize', position);
      window.removeEventListener('scroll', position, true);
    };
  }, [onClose, position, trigger]);

  return createPortal(
    <div ref={menuRef} role="menu" aria-label={ariaLabel} data-row-menu-portal className="supplier-order-row-menu-portal row-menu-list" style={style}>
      {children}
    </div>,
    document.body,
  );
}
