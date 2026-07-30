# PDF Document Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign PO, Delivery Note, and Kanban PDFs into clean, white, line-based documents using the Plant, Category, and Packing snapshots delivered in Phase 2.

**Architecture:** Keep the existing Go/fpdf rendering and document endpoints. Add small reusable drawing helpers, preserve snapshot-driven document projections, and cover layout/content invariants with PDF regression tests.

**Tech Stack:** Go 1.24, go-pdf/fpdf, embedded Go fonts, QR/barcode generation, Go tests.

## Global Constraints

- PDFs use white backgrounds and restrained line separators, without solid black or grey information blocks.
- Purchase Order shows company, supplier, destination Plant, Category, and Packing.
- Delivery Note uses an open header, prominent DN number, separate QR area, aligned materials/totals, and minimal signature borders.
- Kanban renders exactly three cards per A4 page, with a structured grid and `Card 1/N` below the primary QR on the left.
- Kanban numbering restarts for each material.
- Existing endpoints, authorization, QR payloads, and historical snapshot behavior remain compatible.

---

### Task 1: Purchase Order PDF

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Modify: `backend/internal/purchaseorder/pdf_test.go`

- [ ] Add failing assertions for Plant, Category, Packing, white/open section treatment, and readable PO header hierarchy.
- [ ] Run `go test ./internal/purchaseorder -run PurchaseOrderPDF` and confirm the new assertions fail.
- [ ] Implement the open PO header, two-column supplier/Plant details, expanded material table, restrained summary, and approval area.
- [ ] Run focused tests and commit.

### Task 2: Delivery Note PDF

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Modify: `backend/internal/purchaseorder/pdf_test.go`

- [ ] Add failing assertions for prominent DN number, Plant, Category/Packing, distinct QR area, and minimally bordered remarks/signatures.
- [ ] Run `go test ./internal/purchaseorder -run DeliveryNotePDF` and confirm failure.
- [ ] Implement the open DN header and aligned material, total, remarks, and signature sections.
- [ ] Run focused tests and commit.

### Task 3: Kanban Card PDF

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Modify: `backend/internal/purchaseorder/pdf_test.go`

- [ ] Add failing assertions for three cards per page, Plant/Category/Packing/order date content, large part hierarchy, and `Card 1/N` directly beneath the left QR.
- [ ] Add a mixed-material test proving card numbering restarts for each material.
- [ ] Run `go test ./internal/purchaseorder -run Kanban` and confirm failure.
- [ ] Implement the white structured grid and per-material card sequence.
- [ ] Run focused tests and commit.

### Task 4: Verification and integration

- [ ] Run `go test ./...` in `backend`.
- [ ] Run `npm test -- --run` and `npm run build` in `frontend`.
- [ ] Run `git diff --check`.
- [ ] Merge into `main`, repeat verification, push `origin main`, and remove the Phase 3 worktree.
