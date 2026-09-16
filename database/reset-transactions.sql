-- Clears every transaction while keeping master data, users and roles.
--
-- Removed: production planning, orders, operations, daily entries, WIP ledger,
-- production periods, purchasing, receiving, outgoing, delivery notes, kanban
-- lots, sales orders, customer deliveries, the inventory ledger, the email log
-- and the activity log. Document numbering starts again at 1.
--
-- Kept: tenants and their settings, users, roles and permissions, units,
-- categories, packings, plants, warehouses, suppliers, customers, raw
-- materials, finished goods, bills of materials and routings.
--
-- This clears every tenant in the database, not one company.
--
--   docker compose exec -T postgres psql -U nextgen -d nextgen -v ON_ERROR_STOP=1 \
--     < database/reset-transactions.sql
--
-- Run it as the table owner: several of these tables reject ordinary deletes
-- because they are append-only, and TRUNCATE is what bypasses that by design.
-- Take a backup first; nothing here can be undone.
BEGIN;

-- Listing every table in one statement is deliberate: if a transactional table
-- is ever added and forgotten here, PostgreSQL refuses the whole statement
-- instead of quietly leaving orphans behind.
TRUNCATE TABLE
  production_wip_movements,
  production_entries,
  production_operations,
  production_orders,
  production_plan_lines,
  production_plans,
  production_periods,
  production_plan_counters,
  inventory_ledger_entries,
  integration_outbox,
  receiving_kanban_lots,
  receiving_session_scans,
  receivings,
  receiving_sessions,
  outgoing_kanban_lots,
  outgoing_session_scans,
  outgoing_documents,
  outgoing_sessions,
  kanban_lots,
  kanban_number_sequences,
  delivery_note_lines,
  delivery_notes,
  delivery_note_number_sequences,
  purchase_order_approvals,
  approval_email_tokens,
  purchase_order_lines,
  purchase_orders,
  purchase_order_number_sequences,
  customer_delivery_lines,
  customer_deliveries,
  sales_order_lines,
  sales_orders,
  email_logs,
  activity_logs
RESTART IDENTITY;

COMMIT;
