# Live Email Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Activate SMTP configuration, approval email, supplier document email, secure email decisions, and Email Log.

**Architecture:** A tenant-scoped emailing package owns encrypted SMTP settings, durable jobs/logs, rendering, SMTP transport, and public approval tokens. Purchase Order routes enqueue emails after existing domain actions; a background worker delivers jobs.

**Tech Stack:** Go, PostgreSQL 17, React/Next.js, SMTP, AES-GCM, SHA-256, Vitest

## Global Constraints

- SMTP secrets never leave the backend.
- Approval tokens are single-use and expire after 72 hours.
- Supplier attachments are capped at 20 MB.
- Existing in-app approvals remain authoritative.

---

### Task 1: Persistence and crypto

Create migration `010_email_integration.sql`, emailing domain/store, AES-GCM secret handling, and persistence tests.

### Task 2: SMTP settings and test delivery

Add authenticated SMTP GET/PUT/test endpoints and activate the SMTP Settings UI with validation tests.

### Task 3: Durable jobs, templates, and worker

Add branded HTML rendering, SMTP MIME delivery, durable job claiming, Email Log APIs, and worker lifecycle tests.

### Task 4: Approval and supplier email actions

Add enqueue endpoints, secure public approval/rejection endpoints, PDF attachments, and enable status-aware row actions.

### Task 5: Email Log UI and end-to-end verification

Activate filtering/resend, run backend/frontend suites, build Docker, migrate local DB, and verify SMTP-unconfigured behavior plus UI states.

