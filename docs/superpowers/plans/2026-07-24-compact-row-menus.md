# Compact Row Menus Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace wide Supplier Order icon groups with compact Action and Docs menus, and move the Receiving document control beside the row number.

**Architecture:** Refactor only frontend table rendering and menu state. Reuse the existing submit, cancel, navigation, and PDF URLs; do not add backend email behavior.

**Tech Stack:** React, TypeScript, Vitest, existing CSS.

## Constraints

- Use `Action ▾` and `Docs ▾`, not plus, ellipsis, or gear icons.
- Both action and document menus always show all entries, disabling unavailable entries.
- Receiving uses a direct `RCV PDF` control after Number.
- Reduce row, button, and pagination dimensions without reducing accessibility.

### Task 1: Supplier Order row menus

- [ ] Update failing Supplier Order tests for menu labels, states, endpoints, and existing submit/cancel behavior.
- [ ] Observe RED.
- [ ] Consolidate row state into one Action menu and one Docs menu.
- [ ] Observe GREEN.

### Task 2: Receiving document position and density

- [ ] Add failing tests for Number, Document, Status column order and direct RCV PDF link.
- [ ] Observe RED.
- [ ] Move and label the document control and compact the shared styles.
- [ ] Observe GREEN.

### Task 3: Verification

- [ ] Run all frontend tests serially, typecheck, lint, and production build.
- [ ] Commit the implementation.
- [ ] Merge after approval, rebuild frontend, and smoke-test both routes.
