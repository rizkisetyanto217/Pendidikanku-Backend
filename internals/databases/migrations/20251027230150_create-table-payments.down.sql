

-- +migrate Down
BEGIN;

DROP TABLE IF EXISTS payment_gateway_events;
DROP TABLE IF EXISTS payment_items;
DROP TABLE IF EXISTS payments;

COMMIT;