# NextGen Logistics MVP Delivery Roadmap

## Delivery slices

1. **Platform, Identity, Master Data, dan Supplier Order** — runnable monorepo, local login, RBAC, tenant isolation, compact application shell, master CRUD, PO draft/submission.
2. **Approval, Auto DN/Kanban, Documents, dan Supplier Email** — dual-channel approval, atomic generation, PDF, SMTP settings/log, notification outbox.
3. **Receiving dan Inventory Ledger** — exclusive receiving sessions, barcode validation, partial receipts, receiving PDF, stock projection.
4. **Outgoing, Movement, Adjustment, dan Stock Taking** — full-Kanban inventory lifecycle and audit.
5. **Sage Outbox dan On-Premise Agent** — idempotent job protocol, dependencies, retries, reconciliation, integration monitor.
6. **Dashboard, Reports, Export, dan Hardening** — role-aware analytics, supplier filters, exports, performance, backup/restore, production deployment.

Setiap slice memiliki plan, test suite, migration, verification, dan commit sendiri. Slice berikutnya hanya mulai setelah interface slice sebelumnya stabil.
