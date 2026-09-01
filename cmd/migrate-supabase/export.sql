-- Run this in the Supabase SQL editor and save the single JSON result as
-- supabase-export.json. The Go command consumes that file.
SELECT jsonb_build_object(
    'recaps', COALESCE(
        (SELECT jsonb_agg(to_jsonb(r) ORDER BY r.id)
         FROM public.recap_list AS r),
        '[]'::jsonb
    ),
    'recap_items', COALESCE(
        (SELECT jsonb_agg(to_jsonb(i) ORDER BY i.id)
         FROM public.recap_item AS i),
        '[]'::jsonb
    )
);
