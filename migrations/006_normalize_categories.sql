-- Keep category values consistent with the API and frontend vocabulary.
UPDATE recap_items
SET amount = NULL
WHERE LOWER(description) LIKE '%saldo awal%';

UPDATE recap_items
SET category = 'Uncategorized'
WHERE LOWER(description) LIKE '%saldo awal%';

UPDATE recap_items
SET category = 'Food'
WHERE category IS NOT NULL
  AND LOWER(BTRIM(category)) IN ('f&b', 'food & beverage', 'food');

UPDATE recap_items
SET category = 'Uncategorized'
WHERE category IS NULL
   OR BTRIM(category) = ''
   OR LOWER(BTRIM(category)) NOT IN (
       'food',
       'groceries',
       'bills',
       'transport',
       'e-wallet',
       'shopping',
       'income',
       'fees',
       'transfer',
       'uncategorized'
   );

ALTER TABLE recap_items
    ADD CONSTRAINT recap_items_category_check
    CHECK (
        category IN (
            'Food',
            'Groceries',
            'Bills',
            'Transport',
            'E-Wallet',
            'Shopping',
            'Income',
            'Fees',
            'Transfer',
            'Uncategorized'
        )
    );
