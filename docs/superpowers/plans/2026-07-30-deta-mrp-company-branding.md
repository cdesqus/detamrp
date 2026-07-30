# DETA MRP Company Branding Implementation Plan

**Goal:** Replace the visible Order Stock identity with DETA MRP and let one installation manage its company name, logo, and login background from Company Settings.

**Architecture:** Store original image bytes and controlled MIME types in `tenant_settings`. Authenticated settings endpoints manage the media, while narrow public endpoints expose only login-safe branding. PDF loaders fetch branding only for document generation so normal list APIs remain small.

## Task 1: Persist and validate company media

- Add nullable logo/background bytes and MIME columns plus least-privilege grants.
- Extend company settings contracts with media availability and dedicated upload/reset operations.
- Validate decoded PNG, JPEG, or WebP content and enforce 2 MB logo / 5 MB background limits.
- Add service, HTTP, and store tests before implementation.

## Task 2: Serve safe public branding

- Add public metadata and image endpoints outside authentication middleware.
- Return only company name, availability/URLs, controlled content types, and safe cache headers.
- Add route tests covering absence, success, invalid media kind, and no private data leakage.

## Task 3: Upgrade Company Settings and login

- Add logo/background preview, upload/replace, reset, client-side type/size feedback, and accessible status.
- Load public branding on login and apply the configured background with readable overlay treatment.
- Keep DETA MRP fallbacks when branding is unset or unavailable.
- Add component tests before UI implementation.

## Task 4: Apply branding across the app and PDFs

- Replace visible Order Stock/OS copy in login, shell, metadata, settings defaults, and existing user-facing templates with DETA MRP.
- Load company logo only for PO, DN, and Kanban PDF generation, render proportionally, and retain a text fallback.
- Add PDF regression tests for logo rendering and DETA MRP fallback.

## Task 5: Verify and integrate

- Run backend tests, frontend tests, production build, formatting, and migration checks.
- Review the diff for accidental internal package/module renames and oversized API payloads.
- Merge the feature branch into `main`, remove the isolated worktree safely, and push `main`.
