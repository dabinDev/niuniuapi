-- 154_rename_tomato_brand_defaults.sql
-- Keep existing custom branding intact, but migrate prior project defaults to Tomato.

UPDATE settings
SET value = '番茄',
    updated_at = NOW()
WHERE key = 'site_name'
  AND value IN ('烂' || '番茄', 'LANFAN' || 'QIE STUDIO', '灵' || '犀' || '文创');

UPDATE settings
SET value = '番茄',
    updated_at = NOW()
WHERE key = 'payment_product_name_prefix'
  AND value IN ('烂' || '番茄', 'LANFAN' || 'QIE STUDIO', '灵' || '犀' || '文创');
