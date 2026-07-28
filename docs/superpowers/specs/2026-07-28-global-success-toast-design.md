# Global Success Toast Design

## Goal

Provide compact, non-blocking confirmation after successful purchase-order
email actions, including actions that navigate between pages.

## UX

- Display one compact toast at the top-right, below the application header.
- Use a green check indicator, concise text, and a small close button.
- Automatically dismiss after 4 seconds.
- A newer toast replaces the current toast and resets the dismissal timer.
- Clicking close dismisses it immediately.
- Use `role="status"` and `aria-live="polite"` for accessible feedback.
- Do not show success feedback for failed requests; existing inline error
  handling remains unchanged.
- Keep the component visually dense and consistent with the current UI.

## Messages

- Save & Send: `PO submitted and approval email sent.`
- Resend Approval: `Approval email sent successfully.`
- Send to Supplier: `Supplier email sent successfully.`

## Architecture

The application root layout owns a toast provider so feedback survives route
navigation between page-level App Shell instances. A `useToast()` hook exposes `showSuccess(message)`. The provider
renders a single toast and owns replacement, timer cleanup, manual dismissal,
and unmount cleanup.

Supplier Order Form calls `showSuccess` after save, submit, and approval-email
delivery all succeed, immediately before navigating to the index. Supplier
Order Index calls it after successful approval resend or supplier delivery.

## Testing

- Provider tests cover rendering, replacement, four-second dismissal, manual
  close, and timer cleanup.
- Supplier Order Form test proves success is emitted only after the third
  successful request and before navigation.
- Supplier Order Index tests cover approval resend and supplier delivery.
- Failure-path tests prove no success toast is emitted.
