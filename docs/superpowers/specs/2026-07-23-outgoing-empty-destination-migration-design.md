# Outgoing Empty Destination Migration Design

## Problem

The MVP Outgoing flow no longer asks users for a destination. The frontend and backend therefore create sessions with an empty destination, but the existing PostgreSQL constraint `outgoing_sessions_destination_check` requires a trimmed length between 1 and 120. The insert fails with SQLSTATE `23514`, and the UI reports that the outgoing session could not be created.

## Design

Add a forward-only, idempotent database migration that:

- drops the existing `outgoing_sessions_destination_check`;
- recreates it as `length(trim(destination)) <= 120`;
- preserves the `NOT NULL` column requirement;
- does not rewrite existing session or document data;
- leaves the completed outgoing-document destination column unchanged because it already accepts an empty string.

The application continues to send an empty destination and display an em dash for historical empty values. No placeholder data and no destination input are introduced.

## Verification

- A migration regression test proves the new constraint accepts an empty destination and still caps the value at 120 characters.
- The outgoing domain test continues to accept an empty destination and reject values longer than 120 characters.
- Rebuilding Docker applies the migration through the existing migration service.
- A live authenticated `POST /outgoing-sessions` with an empty destination must return HTTP `201`.
- The created test session is removed from the local development database after the smoke test so it does not leave demo noise.
