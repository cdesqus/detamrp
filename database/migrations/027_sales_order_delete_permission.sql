BEGIN;

-- Draft sales-order deletion is restricted in the application, but the DB role
-- also needs DELETE for the ON DELETE CASCADE of its lines.
GRANT DELETE ON sales_orders, sales_order_lines TO nextgen_app;

COMMIT;
