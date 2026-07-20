import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { LoginForm } from './login-form';

describe('LoginForm', () => {
  it('submits username and password and disables while waiting', async () => {
    const user = userEvent.setup();
    let resolveLogin: (() => void) | undefined;
    const login = vi.fn(() => new Promise<void>((resolve) => { resolveLogin = resolve; }));
    render(<LoginForm onLogin={login} />);

    await user.type(screen.getByLabelText('Username'), 'admin');
    await user.type(screen.getByLabelText('Password'), 'secret');
    await user.click(screen.getByRole('button', { name: 'Sign in' }));

    expect(login).toHaveBeenCalledWith('admin', 'secret');
    expect(screen.getByRole('button', { name: 'Signing in…' })).toBeDisabled();
    resolveLogin?.();
  });

  it('shows a friendly login error', async () => {
    const user = userEvent.setup();
    render(<LoginForm onLogin={async () => { throw new Error('Invalid username or password'); }} />);
    await user.type(screen.getByLabelText('Username'), 'admin');
    await user.type(screen.getByLabelText('Password'), 'wrong');
    await user.click(screen.getByRole('button', { name: 'Sign in' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Invalid username or password');
  });
});
