-- PostgreSQL Up Migration Script
-- Schema: E-Commerce & Deep Logistics Pipeline
-- Total Tables: 40 | Connected Tables: 16 (40%) | FK Depth: 5 Levels

BEGIN;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- SECTION 1: STANDALONE & UTILITY ENTITIES (24 TABLES)
-- ============================================================================

-- Table 1: System Settings
CREATE TABLE system_settings (
    setting_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    setting_key VARCHAR(100) NOT NULL UNIQUE,
    setting_value TEXT NOT NULL,
    is_encrypted BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Table 2: Audit Logs
CREATE TABLE audit_logs (
    log_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    action_type VARCHAR(50) NOT NULL,
    entity_name VARCHAR(100) NOT NULL,
    entity_id VARCHAR(64) NOT NULL,
    ip_address INET,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_audit_action_length CHECK (LENGTH(action_type) >= 3)
);

-- Table 3: System Activity Logs
CREATE TABLE activity_logs (
    activity_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_name VARCHAR(120) NOT NULL,
    payload JSONB,
    severity VARCHAR(10) NOT NULL DEFAULT 'INFO',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_activity_severity CHECK (severity IN ('DEBUG', 'INFO', 'WARN', 'ERROR', 'CRITICAL'))
);

-- Table 4: Webhook Events
CREATE TABLE webhook_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_webhook_retry CHECK (retry_count >= 0 AND retry_count <= 5),
    CONSTRAINT chk_webhook_status CHECK (status IN ('PENDING', 'PROCESSING', 'DELIVERED', 'FAILED'))
);

-- Table 5: Newsletter Subscriptions
CREATE TABLE newsletters (
    newsletter_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    is_subscribed BOOLEAN NOT NULL DEFAULT TRUE,
    subscribed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_newsletter_email CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

-- Table 6: Promotional Banners
CREATE TABLE banners (
    banner_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title VARCHAR(150) NOT NULL,
    image_url VARCHAR(2048) NOT NULL,
    target_url VARCHAR(2048),
    display_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    CONSTRAINT chk_banner_order CHECK (display_order >= 0)
);

-- Table 7: Tags
CREATE TABLE tags (
    tag_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    slug VARCHAR(60) NOT NULL UNIQUE,
    CONSTRAINT chk_tag_slug_format CHECK (slug ~* '^[a-z0-9-]+$')
);

-- Table 8: Product Categories
CREATE TABLE categories (
    category_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(120) NOT NULL UNIQUE,
    description TEXT,
    is_visible BOOLEAN NOT NULL DEFAULT TRUE
);

-- Table 9: Suppliers
CREATE TABLE suppliers (
    supplier_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_name VARCHAR(150) NOT NULL UNIQUE,
    contact_email VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    CONSTRAINT chk_supplier_phone CHECK (phone IS NULL OR LENGTH(phone) >= 7)
);

-- Table 10: Warehouses
CREATE TABLE warehouses (
    warehouse_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    capacity_sqft NUMERIC(10, 2) NOT NULL,
    CONSTRAINT chk_warehouse_capacity CHECK (capacity_sqft > 0)
);

-- Table 11: Shipping Carriers
CREATE TABLE shipping_carriers (
    carrier_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(30) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    support_phone VARCHAR(20),
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Table 12: Discount Coupons
CREATE TABLE coupons (
    coupon_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(25) NOT NULL UNIQUE,
    discount_percent NUMERIC(5, 2) NOT NULL,
    max_uses INT NOT NULL DEFAULT 100,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT chk_coupon_discount CHECK (discount_percent > 0 AND discount_percent <= 100),
    CONSTRAINT chk_coupon_max_uses CHECK (max_uses > 0)
);

-- Table 13: Currencies
CREATE TABLE currencies (
    currency_code CHAR(3) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    symbol VARCHAR(5) NOT NULL,
    exchange_rate_usd NUMERIC(12, 6) NOT NULL,
    CONSTRAINT chk_currency_rate CHECK (exchange_rate_usd > 0)
);

-- Table 14: Countries
CREATE TABLE countries (
    country_code CHAR(2) PRIMARY KEY,
    name VARCHAR(90) NOT NULL UNIQUE,
    iso_numeric CHAR(3) UNIQUE NOT NULL
);

-- Table 15: Tax Rates
CREATE TABLE tax_rates (
    tax_rate_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    region_code VARCHAR(10) NOT NULL UNIQUE,
    rate_percentage NUMERIC(5, 4) NOT NULL,
    tax_name VARCHAR(50) NOT NULL,
    CONSTRAINT chk_tax_percentage CHECK (rate_percentage >= 0 AND rate_percentage <= 1)
);

-- Table 16: Roles
CREATE TABLE roles (
    role_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    role_name VARCHAR(50) NOT NULL UNIQUE,
    description VARCHAR(255)
);

-- Table 17: Permissions
CREATE TABLE permissions (
    permission_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    permission_key VARCHAR(100) NOT NULL UNIQUE,
    module VARCHAR(50) NOT NULL
);

-- Table 18: Notification Templates
CREATE TABLE notification_templates (
    template_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    template_code VARCHAR(50) NOT NULL UNIQUE,
    subject_template VARCHAR(255) NOT NULL,
    body_template TEXT NOT NULL
);

-- Table 19: Email Dispatch Logs
CREATE TABLE email_logs (
    log_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    recipient_email VARCHAR(255) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    delivery_status VARCHAR(20) NOT NULL DEFAULT 'SENT',
    CONSTRAINT chk_email_status CHECK (delivery_status IN ('SENT', 'FAILED', 'BOUNCED'))
);

-- Table 20: Application Versions
CREATE TABLE app_versions (
    version_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    version_number VARCHAR(20) NOT NULL UNIQUE,
    is_critical_update BOOLEAN NOT NULL DEFAULT FALSE,
    released_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Table 21: FAQ Articles
CREATE TABLE faq_articles (
    faq_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    question VARCHAR(300) NOT NULL,
    answer TEXT NOT NULL,
    is_published BOOLEAN NOT NULL DEFAULT FALSE
);

-- Table 22: Brand Assets
CREATE TABLE brand_assets (
    asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_type VARCHAR(30) NOT NULL,
    file_path VARCHAR(500) NOT NULL UNIQUE,
    file_size_kb INT NOT NULL,
    CONSTRAINT chk_asset_size CHECK (file_size_kb > 0)
);

-- Table 23: Feature Flags
CREATE TABLE feature_flags (
    flag_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    flag_key VARCHAR(100) NOT NULL UNIQUE,
    is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    rollout_percentage INT NOT NULL DEFAULT 0,
    CONSTRAINT chk_rollout_range CHECK (rollout_percentage BETWEEN 0 AND 100)
);

-- Table 24: Scheduled Maintenance Windows
CREATE TABLE maintenance_windows (
    window_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT chk_window_times CHECK (end_time > start_time)
);


-- ============================================================================
-- SECTION 2: INTERCONNECTED DOMAIN ENTITIES (16 TABLES - 40%)
-- FEATURING A 5-LEVEL DEEP DEPENDENCY CHAIN:
-- Level 1: users
-- Level 2: orders
-- Level 3: shipments
-- Level 4: shipment_tracking_events
-- Level 5: delivery_proofs
-- ============================================================================

-- Level 1: Users
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(30) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_username_length CHECK (LENGTH(username) >= 3),
    CONSTRAINT chk_user_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

-- Level 2: User Profiles (1 -> 1 with Users)
CREATE TABLE user_profiles (
    profile_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    avatar_url VARCHAR(2048),
    CONSTRAINT fk_user_profile_user FOREIGN KEY (user_id) REFERENCES users (user_id) ON DELETE CASCADE
);

-- Level 2: User Addresses (1 -> M with Users)
CREATE TABLE user_addresses (
    address_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL,
    street_line1 VARCHAR(150) NOT NULL,
    city VARCHAR(80) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    country_code CHAR(2) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_user_address_user FOREIGN KEY (user_id) REFERENCES users (user_id) ON DELETE CASCADE,
    CONSTRAINT fk_user_address_country FOREIGN KEY (country_code) REFERENCES countries (country_code) ON DELETE RESTRICT
);

-- Level 2: Products
CREATE TABLE products (
    product_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category_id INT NOT NULL,
    sku VARCHAR(40) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    base_price NUMERIC(10, 2) NOT NULL,
    CONSTRAINT chk_product_price CHECK (base_price >= 0.00),
    CONSTRAINT fk_product_category FOREIGN KEY (category_id) REFERENCES categories (category_id) ON DELETE RESTRICT
);

-- Level 3: Product Variants (Depends on Products)
CREATE TABLE product_variants (
    variant_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id INT NOT NULL,
    variant_sku VARCHAR(50) NOT NULL UNIQUE,
    attribute_json JSONB NOT NULL,
    additional_price NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    CONSTRAINT chk_variant_price CHECK (additional_price >= 0.00),
    CONSTRAINT fk_variant_product FOREIGN KEY (product_id) REFERENCES products (product_id) ON DELETE CASCADE
);

-- Level 4: Inventory Items (Depends on Product Variants & Warehouses)
CREATE TABLE inventory (
    inventory_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    variant_id INT NOT NULL,
    warehouse_id INT NOT NULL,
    quantity_available INT NOT NULL DEFAULT 0,
    CONSTRAINT chk_inventory_qty CHECK (quantity_available >= 0),
    CONSTRAINT uq_variant_warehouse UNIQUE (variant_id, warehouse_id),
    CONSTRAINT fk_inventory_variant FOREIGN KEY (variant_id) REFERENCES product_variants (variant_id) ON DELETE CASCADE,
    CONSTRAINT fk_inventory_warehouse FOREIGN KEY (warehouse_id) REFERENCES warehouses (warehouse_id) ON DELETE RESTRICT
);

-- Level 2: Customer Orders (Depends on Users)
CREATE TABLE orders (
    order_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    order_number VARCHAR(30) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    total_amount NUMERIC(12, 2) NOT NULL,
    placed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_order_total CHECK (total_amount >= 0.00),
    CONSTRAINT chk_order_status CHECK (status IN ('PENDING', 'PAID', 'SHIPPED', 'COMPLETED', 'CANCELLED')),
    CONSTRAINT fk_order_user FOREIGN KEY (user_id) REFERENCES users (user_id) ON DELETE RESTRICT
);

-- Level 3: Order Items (Depends on Orders & Product Variants)
CREATE TABLE order_items (
    order_item_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id UUID NOT NULL,
    variant_id INT NOT NULL,
    unit_price NUMERIC(10, 2) NOT NULL,
    quantity INT NOT NULL,
    CONSTRAINT chk_item_qty CHECK (quantity > 0),
    CONSTRAINT chk_item_price CHECK (unit_price >= 0.00),
    CONSTRAINT fk_order_item_order FOREIGN KEY (order_id) REFERENCES orders (order_id) ON DELETE CASCADE,
    CONSTRAINT fk_order_item_variant FOREIGN KEY (variant_id) REFERENCES product_variants (variant_id) ON DELETE RESTRICT
);

-- Level 3: Order Invoices (Depends on Orders & Currencies)
CREATE TABLE order_invoices (
    invoice_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL UNIQUE,
    invoice_number VARCHAR(50) NOT NULL UNIQUE,
    currency CHAR(3) NOT NULL,
    amount_due NUMERIC(12, 2) NOT NULL,
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_invoice_amount CHECK (amount_due >= 0.00),
    CONSTRAINT fk_invoice_order FOREIGN KEY (order_id) REFERENCES orders (order_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_currency FOREIGN KEY (currency) REFERENCES currencies (currency_code) ON DELETE RESTRICT
);

-- Level 3: Shipments (Depends on Orders, Shipping Carriers & User Addresses)
CREATE TABLE shipments (
    shipment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL,
    carrier_id INT NOT NULL,
    address_id INT NOT NULL,
    tracking_number VARCHAR(100) NOT NULL UNIQUE,
    dispatched_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_shipment_order FOREIGN KEY (order_id) REFERENCES orders (order_id) ON DELETE CASCADE,
    CONSTRAINT fk_shipment_carrier FOREIGN KEY (carrier_id) REFERENCES shipping_carriers (carrier_id) ON DELETE RESTRICT,
    CONSTRAINT fk_shipment_address FOREIGN KEY (address_id) REFERENCES user_addresses (address_id) ON DELETE RESTRICT
);

-- Level 4: Shipment Tracking Events (Depends on Shipments)
CREATE TABLE shipment_tracking_events (
    event_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    shipment_id UUID NOT NULL,
    location VARCHAR(150) NOT NULL,
    status_description VARCHAR(255) NOT NULL,
    event_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_tracking_event_shipment FOREIGN KEY (shipment_id) REFERENCES shipments (shipment_id) ON DELETE CASCADE
);

-- Level 5: Delivery Proofs (Depends on Shipment Tracking Events)
CREATE TABLE delivery_proofs (
    proof_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id INT NOT NULL UNIQUE,
    recipient_signature_url VARCHAR(2048),
    photo_url VARCHAR(2048),
    delivered_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_delivery_proof_event FOREIGN KEY (event_id) REFERENCES shipment_tracking_events (event_id) ON DELETE CASCADE
);

-- Level 6: Delivery Proof Verification Logs (Depends on Delivery Proofs)
CREATE TABLE proof_verifications (
    verification_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    proof_id UUID NOT NULL,
    verified_by_system BOOLEAN NOT NULL DEFAULT FALSE,
    confidence_score NUMERIC(5, 2) NOT NULL,
    verified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_confidence_score CHECK (confidence_score BETWEEN 0.00 AND 100.00),
    CONSTRAINT fk_verification_proof FOREIGN KEY (proof_id) REFERENCES delivery_proofs (proof_id) ON DELETE CASCADE
);

-- Level 3: Payment Transactions (Depends on Orders & Currencies)
CREATE TABLE transactions (
    transaction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL,
    provider VARCHAR(40) NOT NULL,
    amount NUMERIC(12, 2) NOT NULL,
    currency CHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_trans_amount CHECK (amount > 0.00),
    CONSTRAINT chk_trans_status CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED', 'REFUNDED')),
    CONSTRAINT fk_trans_order FOREIGN KEY (order_id) REFERENCES orders (order_id) ON DELETE CASCADE,
    CONSTRAINT fk_trans_currency FOREIGN KEY (currency) REFERENCES currencies (currency_code) ON DELETE RESTRICT
);

-- Level 3: Product Reviews (Depends on Products & Users)
CREATE TABLE reviews (
    review_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id INT NOT NULL,
    user_id UUID NOT NULL,
    rating INT NOT NULL,
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_review_rating CHECK (rating BETWEEN 1 AND 5),
    CONSTRAINT uq_user_product_review UNIQUE (product_id, user_id),
    CONSTRAINT fk_review_product FOREIGN KEY (product_id) REFERENCES products (product_id) ON DELETE CASCADE,
    CONSTRAINT fk_review_user FOREIGN KEY (user_id) REFERENCES users (user_id) ON DELETE CASCADE
);

-- Level 2: Customer Support Tickets (Depends on Users)
CREATE TABLE support_tickets (
    ticket_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL,
    subject VARCHAR(200) NOT NULL,
    priority VARCHAR(10) NOT NULL DEFAULT 'MEDIUM',
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_ticket_priority CHECK (priority IN ('LOW', 'MEDIUM', 'HIGH', 'URGENT')),
    CONSTRAINT chk_ticket_status CHECK (status IN ('OPEN', 'IN_PROGRESS', 'RESOLVED', 'CLOSED')),
    CONSTRAINT fk_ticket_user FOREIGN KEY (user_id) REFERENCES users (user_id) ON DELETE CASCADE
);

COMMIT;