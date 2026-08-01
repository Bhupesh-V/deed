-- PostgreSQL Down Migration Script
-- Drops all 40 tables in strict reverse-dependency order.

BEGIN;

-- ============================================================================
-- SECTION 1: DROP INTERCONNECTED ENTITIES IN REVERSE DEPENDENCY ORDER
-- ============================================================================

-- Level 6
DROP TABLE IF EXISTS proof_verifications CASCADE;

-- Level 5
DROP TABLE IF EXISTS delivery_proofs CASCADE;

-- Level 4
DROP TABLE IF EXISTS shipment_tracking_events CASCADE;

-- Level 3
DROP TABLE IF EXISTS shipments CASCADE;
DROP TABLE IF EXISTS order_invoices CASCADE;
DROP TABLE IF EXISTS order_items CASCADE;
DROP TABLE IF EXISTS inventory CASCADE;
DROP TABLE IF EXISTS product_variants CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS reviews CASCADE;

-- Level 2
DROP TABLE IF EXISTS support_tickets CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS user_addresses CASCADE;
DROP TABLE IF EXISTS user_profiles CASCADE;

-- Level 1
DROP TABLE IF EXISTS users CASCADE;

-- ============================================================================
-- SECTION 2: DROP STANDALONE & UTILITY ENTITIES (24 TABLES)
-- ============================================================================

DROP TABLE IF EXISTS maintenance_windows CASCADE;
DROP TABLE IF EXISTS feature_flags CASCADE;
DROP TABLE IF EXISTS brand_assets CASCADE;
DROP TABLE IF EXISTS faq_articles CASCADE;
DROP TABLE IF EXISTS app_versions CASCADE;
DROP TABLE IF EXISTS email_logs CASCADE;
DROP TABLE IF EXISTS notification_templates CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS tax_rates CASCADE;
DROP TABLE IF EXISTS countries CASCADE;
DROP TABLE IF EXISTS currencies CASCADE;
DROP TABLE IF EXISTS coupons CASCADE;
DROP TABLE IF EXISTS shipping_carriers CASCADE;
DROP TABLE IF EXISTS warehouses CASCADE;
DROP TABLE IF EXISTS suppliers CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
DROP TABLE IF EXISTS tags CASCADE;
DROP TABLE IF EXISTS banners CASCADE;
DROP TABLE IF EXISTS newsletters CASCADE;
DROP TABLE IF EXISTS webhook_events CASCADE;
DROP TABLE IF EXISTS activity_logs CASCADE;
DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS system_settings CASCADE;

-- Optional: Clean up extension if created exclusively for this migration
-- DROP EXTENSION IF EXISTS "pgcrypto";

COMMIT;