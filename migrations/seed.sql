WITH seeded_recap AS (
    INSERT INTO recaps (
        name,
        status,
        bank_name,
        period,
        created_at,
        updated_at
    )
    SELECT
        'Monthly Expenses September 2025',
        'completed',
        'Bank Central Asia',
        'september',
        '2025-12-30T03:25:45.862124+00:00'::TIMESTAMPTZ,
        '2025-12-30T03:25:45.862124+00:00'::TIMESTAMPTZ
    WHERE NOT EXISTS (
        SELECT 1
        FROM recaps
        WHERE name = 'Monthly Expenses September 2025'
            AND bank_name = 'Bank Central Asia'
            AND period = 'september'
    )
    RETURNING id
), target_recap AS (
    SELECT id FROM seeded_recap
    UNION
    SELECT id
    FROM recaps
    WHERE name = 'Monthly Expenses September 2025'
        AND bank_name = 'Bank Central Asia'
        AND period = 'september'
    ORDER BY id
    LIMIT 1
)
INSERT INTO recap_items (
    recap_id,
    transaction_date,
    description,
    amount,
    balance,
    category,
    created_at,
    updated_at
)
SELECT
    target_recap.id,
    '2025-09-01',
    '"SALDO AWAL"',
    0.00,
    175654122.59,
    'Uncategorized',
    '2025-12-30T03:48:27.984584+00:00'::TIMESTAMPTZ,
    '2025-12-30T03:48:27.984584+00:00'::TIMESTAMPTZ
FROM target_recap
WHERE NOT EXISTS (
    SELECT 1
    FROM recap_items
    WHERE recap_id = target_recap.id
        AND transaction_date = '2025-09-01'
        AND description = '"SALDO AWAL"'
);
