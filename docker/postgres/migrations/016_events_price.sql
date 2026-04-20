-- Add price fields to events. price_amount=0 means the event is free (use /tickets/register).
-- Paid events (price_amount > 0) require payment via /payments/initiate before a ticket is issued.
ALTER TABLE events
    ADD COLUMN IF NOT EXISTS price_amount  bigint  NOT NULL DEFAULT 0 CHECK (price_amount >= 0),
    ADD COLUMN IF NOT EXISTS price_currency char(3) NOT NULL DEFAULT 'KZT';
