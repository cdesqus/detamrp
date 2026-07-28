import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { RowMenuPortal } from './row-menu-portal';

function rect(left: number, top: number, width: number, height: number): DOMRect {
  return { left, top, width, height, right: left + width, bottom: top + height, x: left, y: top, toJSON: () => ({}) } as DOMRect;
}

describe('RowMenuPortal', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 800 });
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 600 });
  });

  it('renders under document.body and positions below the trigger', () => {
    const trigger = document.createElement('button');
    document.body.appendChild(trigger);
    trigger.getBoundingClientRect = () => rect(100, 80, 90, 24);

    render(<RowMenuPortal trigger={trigger} ariaLabel="Actions for PO-001" onClose={vi.fn()}><button>Open Detail</button></RowMenuPortal>);

    const menu = screen.getByRole('menu', { name: 'Actions for PO-001' });
    expect(menu.parentElement).toBe(document.body);
    expect(menu).toHaveStyle({ position: 'fixed', left: '100px', top: '108px' });
  });

  it('flips above and clamps inside the viewport', () => {
    const trigger = document.createElement('button');
    document.body.appendChild(trigger);
    trigger.getBoundingClientRect = () => rect(770, 560, 40, 24);
    const original = HTMLElement.prototype.getBoundingClientRect;
    HTMLElement.prototype.getBoundingClientRect = function () {
      return this.getAttribute('role') === 'menu' ? rect(0, 0, 158, 180) : original.call(this);
    };

    render(<RowMenuPortal trigger={trigger} ariaLabel="Documents" onClose={vi.fn()}><button>PO PDF</button></RowMenuPortal>);

    expect(screen.getByRole('menu')).toHaveStyle({ left: '634px', top: '376px' });
    HTMLElement.prototype.getBoundingClientRect = original;
  });

  it('dismisses outside and restores trigger focus only for Escape', () => {
    const trigger = document.createElement('button');
    document.body.appendChild(trigger);
    trigger.getBoundingClientRect = () => rect(20, 20, 80, 24);
    const onClose = vi.fn();
    const { rerender } = render(<RowMenuPortal trigger={trigger} ariaLabel="Actions" onClose={onClose}><button>Open</button></RowMenuPortal>);

    fireEvent.pointerDown(document.body);
    expect(onClose).toHaveBeenCalledWith(false);

    onClose.mockClear();
    rerender(<RowMenuPortal trigger={trigger} ariaLabel="Actions" onClose={onClose}><button>Open</button></RowMenuPortal>);
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledWith(true);
  });
});
