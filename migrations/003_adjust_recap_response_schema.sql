ALTER TABLE recaps
    ALTER COLUMN period TYPE TEXT
    USING LOWER(TO_CHAR(period, 'FMMonth'));

ALTER TABLE recaps
    ADD CONSTRAINT recaps_period_check
    CHECK (
        period IN (
            'january',
            'february',
            'march',
            'april',
            'may',
            'june',
            'july',
            'august',
            'september',
            'october',
            'november',
            'december'
        )
    );

ALTER TABLE recap_items
    ALTER COLUMN amount TYPE NUMERIC(18, 2)
    USING amount::NUMERIC(18, 2),
    ALTER COLUMN balance TYPE NUMERIC(18, 2)
    USING balance::NUMERIC(18, 2);
