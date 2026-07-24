# Live Email Integration Design

## Outcome

Order Stock sends polished transactional email through tenant-configured SMTP. Submitting a draft continues to create the in-app approval notification and also emails the configured approver. Approved orders can be emailed to the supplier with PO, Delivery Note, and Kanban Label PDFs.

## SMTP and security

- SMTP Settings stores host, port, security mode (`STARTTLS`, `TLS`, or `NONE`), username, encrypted password, sender name, and sender email.
- The SMTP password is encrypted with AES-GCM using `EMAIL_ENCRYPTION_KEY` from the server environment. The API never returns the password.
- Test Email sends to a user-entered address and records the attempt.
- Empty or invalid configuration produces a clear validation response without changing PO state.

## Approval email

- `Send to Approval` first submits the draft through the existing approval transaction, preserving the in-app notification.
- The resulting approval email contains a branded header, PO summary, material table, Approve and Reject actions, and `View PO Detail`.
- `View PO Detail` opens `/supplier-orders/{id}`. Authentication redirects to login and returns to the PO.
- Approval links contain a random, single-use token stored only as a SHA-256 hash. They expire after 72 hours and can act only while the approval remains pending.
- Approve completes immediately on a confirmation page. Reject requires a reason.
- A decision made in the app invalidates the email token because the same approval record is no longer pending.

## Supplier email

- Available for `APPROVED`, `PARTIALLY_RECEIVED`, and `FULLY_RECEIVED` orders with generated DN/Kanban documents.
- Sends the supplier a branded operational summary and three PDF attachments: PO, DN, and Kanban Labels.
- Attachment payload is capped at 20 MB.
- The supplier email contains no application link because suppliers do not have accounts.

## Email log

Every attempt records tenant, type, PO, recipient, subject, status (`PENDING`, `SENT`, `FAILED`), timestamps, retry count, and safe error text. Credentials and tokens never appear in the log. Email Log supports filtering and resend for failed/sent messages.

## Delivery model

For this prototype, explicit user actions enqueue a durable email job and return immediately. An in-process worker claims pending jobs, sends through SMTP, and updates Email Log. Restarting the API resumes pending jobs. PO submission and approval creation remain authoritative even if email delivery fails.

