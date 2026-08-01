# Deta MRP Client Proposal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate a polished, editable 16:9 PowerPoint proposal for a 10–15 minute Deta MRP client presentation that transitions into a live demo.

**Architecture:** Keep presentation content in a typed JavaScript data model and render it through a focused PptxGenJS generator. Use reusable layout helpers for typography, cards, process flows, timelines, and footer treatment, then validate both the content model and generated Office Open XML package with Node tests.

**Tech Stack:** Node.js, PptxGenJS, JavaScript ES modules, Node test runner, AdmZip, Microsoft PowerPoint `.pptx` output.

## Global Constraints

- Produce exactly 11 slides in 16:9 format.
- All visible presentation copy must be in English.
- Use white as the dominant background and light blue as the primary accent.
- Keep the tone short, confident, conversational, youthful, and credible for management.
- Position Deta MRP as a working, ready-to-use application that can be customized.
- Do not include pricing, commercial quotation, SAGE, accounting integration, unsupported ROI claims, or a technical architecture section.
- Preserve the qualified implementation estimate of 6–7 weeks.
- The final slide must transition directly into the live demonstration.

---

## File Structure

- `presentation/package.json` — isolated generation and test dependencies/scripts.
- `presentation/src/content.mjs` — single source of truth for deck metadata and approved slide copy.
- `presentation/src/theme.mjs` — palette, typography, spacing, and shared visual constants.
- `presentation/src/generate.mjs` — reusable slide primitives and the PowerPoint renderer.
- `presentation/test/deck.test.mjs` — content and generated-package validation.
- `presentation/output/Deta-MRP-Client-Proposal.pptx` — final editable presentation.

### Task 1: Content Model and Guardrails

**Files:**
- Create: `presentation/package.json`
- Create: `presentation/src/content.mjs`
- Create: `presentation/test/deck.test.mjs`

**Interfaces:**
- Produces: `deck` object with `{ title: string, subtitle: string, slides: Slide[] }`.
- Produces: `Slide` objects with `{ id: string, title: string, headline?: string, body?: string, bullets?: string[], kind: string }` and kind-specific data.
- Consumes: approved copy from `docs/superpowers/specs/2026-08-01-deta-mrp-client-proposal-design.md`.

- [ ] **Step 1: Add the isolated presentation package**

Create `presentation/package.json` with `type: module`, scripts `generate` and `test`, and exact dependencies for `pptxgenjs` and `adm-zip`.

- [ ] **Step 2: Write failing content tests**

Use `node:test` assertions that require exactly 11 slides, unique IDs, non-empty English titles, the 6–7 week estimate, the six demo steps, and absence of forbidden visible terms matching `/sage|accounting integration|price|pricing|quotation|tbd|todo/i`.

- [ ] **Step 3: Run tests and verify failure**

Run: `npm test --prefix presentation`

Expected: FAIL because `presentation/src/content.mjs` does not exist.

- [ ] **Step 4: Implement the approved content model**

Export the complete 11-slide copy from the approved spec. Use explicit kinds: `cover`, `challenge`, `needs`, `product`, `process`, `approval`, `modules`, `benefits`, `customization`, `timeline`, and `demo`.

- [ ] **Step 5: Run tests and verify content guardrails pass**

Run: `npm test --prefix presentation`

Expected: all content tests PASS.

- [ ] **Step 6: Commit**

```powershell
git add presentation/package.json presentation/src/content.mjs presentation/test/deck.test.mjs
git commit -m "test: define Deta MRP proposal content"
```

### Task 2: Visual System and PowerPoint Generation

**Files:**
- Create: `presentation/src/theme.mjs`
- Create: `presentation/src/generate.mjs`
- Modify: `presentation/test/deck.test.mjs`
- Create: `presentation/output/Deta-MRP-Client-Proposal.pptx`

**Interfaces:**
- Consumes: `deck` from `content.mjs`.
- Produces: `buildPresentation(deck): pptxgen` and `writePresentation(outputPath): Promise<void>`.
- Produces: shared theme constants `COLORS`, `FONTS`, `LAYOUT`, and `SHADOW`.

- [ ] **Step 1: Extend tests with generated-package requirements**

Generate the deck in a temporary test directory, open it with AdmZip, assert that `[Content_Types].xml` exists, and assert that `ppt/slides/slide1.xml` through `slide11.xml` exist while `slide12.xml` does not.

- [ ] **Step 2: Run tests and verify generation failure**

Run: `npm test --prefix presentation`

Expected: FAIL because `generate.mjs` does not exist.

- [ ] **Step 3: Define the visual theme**

Use a white canvas, dark navy text, light blue accent, pale blue panels, rounded rectangles, Aptos/Arial-compatible typography, restrained shadows, 16:9 layout, and a small Deta MRP footer marker. Keep body text at or above 16 pt and slide titles at or above 28 pt.

- [ ] **Step 4: Implement reusable layout helpers**

Implement `addHeader`, `addFooter`, `addPill`, `addCard`, `addBulletList`, `addProcessStep`, and `addSectionNumber`. Helpers must take a slide plus plain options and must not depend on slide-specific content.

- [ ] **Step 5: Implement all 11 slide layouts**

Use distinct but coherent layouts: bold cover, challenge contrast, needs checklist, product workspace mockup, connected process flow, remote approval highlight, grouped module grid, benefit cards, five-step customization path, horizontal implementation timeline, and demo checklist with a strong handoff statement.

- [ ] **Step 6: Generate the editable PowerPoint**

Run: `npm run generate --prefix presentation`

Expected: `presentation/output/Deta-MRP-Client-Proposal.pptx` exists and is non-empty.

- [ ] **Step 7: Run structural tests**

Run: `npm test --prefix presentation`

Expected: all content and package tests PASS with exactly 11 slide XML files.

- [ ] **Step 8: Commit**

```powershell
git add presentation/src/theme.mjs presentation/src/generate.mjs presentation/test/deck.test.mjs presentation/output/Deta-MRP-Client-Proposal.pptx
git commit -m "feat: generate Deta MRP client proposal deck"
```

### Task 3: Final Presentation Quality Review

**Files:**
- Modify if required: `presentation/src/content.mjs`
- Modify if required: `presentation/src/theme.mjs`
- Modify if required: `presentation/src/generate.mjs`
- Regenerate: `presentation/output/Deta-MRP-Client-Proposal.pptx`

**Interfaces:**
- Consumes: generated `.pptx` and acceptance criteria from the approved design.
- Produces: final verified presentation artifact.

- [ ] **Step 1: Inspect the generated package for overflow warnings**

Capture any PptxGenJS layout or write warnings during generation. Treat warnings about out-of-bounds positions, invalid shapes, or missing content as failures.

- [ ] **Step 2: Review slide copy against the approved spec**

Confirm all 11 titles, headlines, timeline stages, demo steps, and scope boundaries match the approved design. Confirm forbidden subjects appear nowhere in visible slide copy.

- [ ] **Step 3: Regenerate after any corrections**

Run: `npm run generate --prefix presentation`

Expected: generation completes without warnings and overwrites only the intended output file.

- [ ] **Step 4: Run final verification**

Run: `npm test --prefix presentation`

Expected: all tests PASS and the output contains exactly 11 slides.

- [ ] **Step 5: Confirm repository state**

Run: `git status --short`

Expected: no uncommitted changes related to the presentation implementation.

- [ ] **Step 6: Commit corrections if any**

```powershell
git add presentation
git commit -m "style: polish Deta MRP proposal presentation"
```
