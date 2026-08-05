CREATE TABLE recap_items (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    recap_id BIGINT NOT NULL
        REFERENCES recaps(id)
        ON DELETE CASCADE,

    transaction_date DATE NOT NULL,
    description TEXT NOT NULL,
    amount BIGINT NOT NULL,
    balance BIGINT,
    category TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX recap_items_recap_id_date_idx
    ON recap_items (recap_id, transaction_date, id);