# Email Delivery and Formatting Design

## Scope

Correct approval email number formatting, ensure **Save & Send for Approval**
actually sends the approval email, make public links configurable for deployment,
and verify the existing supplier email path and its three PDF attachments.

## Approval Email

- Display currency and quantities without database scale padding.
- Use Indonesian-readable separators: `2000000.000000` becomes `2.000.000`,
  `5.000000` becomes `5`, and `5.250000` becomes `5,25`.
- Preserve the exact database values; formatting changes presentation only.
- Build Approve, Reject, and View Detail URLs from `APP_BASE_URL`.
- A resend creates a new single-use token and invalidates the previous token.

## Create and Submit Flow

`Save & Send for Approval` performs these steps sequentially:

1. Create or update the purchase order.
2. Submit the draft for approval.
3. Send the approval email through the existing approval-email endpoint.
4. Return to the Supplier Orders index when all steps succeed.

If submission succeeds but email delivery fails, the purchase order remains
`PENDING_APPROVAL`. The form displays a delivery error and does not retry
automatically. The user can resend from the Supplier Orders Action menu, avoiding
duplicate delivery attempts.

## Deployment Configuration

Docker Compose reads:

```yaml
APP_BASE_URL: ${APP_BASE_URL:-http://localhost:3019}
```

Local development therefore continues to work without configuration. The server
must define:

```env
APP_BASE_URL=https://deta.kaumtech.com
```

No domain is hard-coded into application code.

## Supplier Email Verification

The supplier-email implementation remains manually triggered after approval.
Automated tests must verify:

- it is addressed to the supplier email snapshot/source selected for the PO;
- it identifies the correct PO and generated DN;
- it includes Purchase Order, Delivery Note, and Kanban Labels PDFs;
- each attachment has the expected filename and non-empty PDF content;
- failures are logged as failed and returned to the caller rather than reported
  as successful.

Supplier email sending is not added to the approval action automatically.

## Testing

- Go template tests cover formatted currency and decimal quantities.
- Frontend tests prove Save & Send invokes save, submit, and approval-email calls
  in order, including the partial-success email failure state.
- Service tests cover deployment URLs and supplier attachments.
- Full frontend tests, Go tests, lint, typecheck, production build, and Docker
  build must pass before deployment.
