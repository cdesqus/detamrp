import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ToastProvider, useToast } from './toast-provider';

function Trigger() {
  const { showSuccess } = useToast();
  return <>
    <button onClick={() => showSuccess('First success')}>First</button>
    <button onClick={() => showSuccess('Second success')}>Second</button>
  </>;
}

describe('ToastProvider', () => {
  afterEach(() => vi.useRealTimers());

  it('shows one accessible success toast and replaces it with newer feedback', () => {
    render(<ToastProvider><Trigger /></ToastProvider>);

    fireEvent.click(screen.getByRole('button', { name: 'First' }));
    expect(screen.getByRole('status')).toHaveTextContent('First success');

    fireEvent.click(screen.getByRole('button', { name: 'Second' }));
    expect(screen.getByRole('status')).toHaveTextContent('Second success');
    expect(screen.queryByText('First success')).not.toBeInTheDocument();
  });

  it('dismisses manually or automatically after four seconds', () => {
    vi.useFakeTimers();
    const { rerender } = render(<ToastProvider><Trigger /></ToastProvider>);
    fireEvent.click(screen.getByRole('button', { name: 'First' }));
    fireEvent.click(screen.getByRole('button', { name: 'Dismiss notification' }));
    expect(screen.queryByRole('status')).not.toBeInTheDocument();

    rerender(<ToastProvider><Trigger /></ToastProvider>);
    fireEvent.click(screen.getByRole('button', { name: 'Second' }));
    act(() => vi.advanceTimersByTime(3_999));
    expect(screen.getByRole('status')).toHaveTextContent('Second success');
    act(() => vi.advanceTimersByTime(1));
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });

  it('resets the dismissal timer when a newer toast replaces the current one', () => {
    vi.useFakeTimers();
    render(<ToastProvider><Trigger /></ToastProvider>);
    fireEvent.click(screen.getByRole('button', { name: 'First' }));
    act(() => vi.advanceTimersByTime(3_000));
    fireEvent.click(screen.getByRole('button', { name: 'Second' }));
    act(() => vi.advanceTimersByTime(1_001));
    expect(screen.getByRole('status')).toHaveTextContent('Second success');
    act(() => vi.advanceTimersByTime(2_999));
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });
});
