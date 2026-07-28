import { fireEvent, render, screen } from '@testing-library/react';
import { expect, it } from 'vitest';
import { useToast } from '../components/toast/toast-provider';
import { Providers } from './providers';

function Trigger() {
  const { showSuccess } = useToast();
  return <button onClick={() => showSuccess('Persistent success')}>Show success</button>;
}

it('provides persistent application-level toast feedback', () => {
  render(<Providers><Trigger /></Providers>);
  fireEvent.click(screen.getByRole('button', { name: 'Show success' }));
  expect(screen.getByRole('status')).toHaveTextContent('Persistent success');
});
