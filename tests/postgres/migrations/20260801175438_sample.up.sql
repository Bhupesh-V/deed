-- PostgreSQL Up Migration Script
-- Schema: E-Commerce & Deep Logistics Pipeline (Refactored)

BEGIN;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- SECTION 1: CUSTOM TYPES & STANDALONE TABLES
-- ============================================================================

-- 1. Table with PostgreSQL Custom Enum Type
CREATE TYPE system_log_level AS ENUM ('DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL');

CREATE TABLE system_event_logs (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name VARCHAR(100) NOT NULL,
    log_level system_log_level NOT NULL DEFAULT 'INFO',
    message TEXT NOT NULL,
    logged_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 2. Table with String Arrays
CREATE TABLE search_index_keywords (
    keyword_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL,
    search_terms VARCHAR(100)[] NOT NULL,
    synonyms TEXT[] DEFAULT '{}'::TEXT[]
);

-- 3. Table with Integer Arrays
CREATE TABLE warehouse_shelf_grid (
    grid_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    zone_code VARCHAR(20) NOT NULL,
    allowed_weight_capacities_kg INT[] NOT NULL,
    matrix_coordinates INT[] NOT NULL
);

-- 4. Table with CHECK Constraint
CREATE TABLE promotional_discounts (
    discount_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    promo_code VARCHAR(30) NOT NULL UNIQUE,
    min_order_value NUMERIC(10, 2) NOT NULL,
    discount_value NUMERIC(10, 2) NOT NULL,
    CONSTRAINT chk_discount_amount_valid CHECK (discount_value > 0.00 AND discount_value < min_order_value)
);


-- ============================================================================
-- SECTION 2: CONNECTED LOOKUP & DOMAIN TABLES
-- ============================================================================

-- Categories (Referenced by products)
CREATE TABLE categories (
    category_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(120) NOT NULL UNIQUE,
    description TEXT,
    is_visible BOOLEAN NOT NULL DEFAULT TRUE
);

-- Warehouses (Referenced by inventory)
CREATE TABLE warehouses (
    warehouse_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    capacity_sqft NUMERIC(10, 2) NOT NULL,
    CONSTRAINT chk_warehouse_capacity CHECK (capacity_sqft > 0)
);

-- Shipping Carriers (Referenced by shipments)
CREATE TABLE shipping_carriers (
    carrier_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(30) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    support_phone VARCHAR(20),
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Currencies (Referenced by order_invoices & transactions)
CREATE TABLE currencies (
    currency_code CHAR(3) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    symbol VARCHAR(5) NOT NULL,
    exchange_rate_usd NUMERIC(12, 6) NOT NULL,
    CONSTRAINT chk_currency_rate CHECK (exchange_rate_usd > 0)
);

-- Countries (Referenced by user_addresses)
CREATE TABLE countries (
    country_code CHAR(2) PRIMARY KEY,
    name VARCHAR(90) NOT NULL UNIQUE,
    iso_numeric CHAR(3) UNIQUE NOT NULL
);

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

-- Level 2: User Addresses (1 -> M with Users, FK to Countries)
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

-- Level 2: Products (FK to Categories)
CREATE TABLE products (
    product_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category_id INT NOT NULL,
    sku VARCHAR(40) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    base_price NUMERIC(10, 2) NOT NULL,
    CONSTRAINT chk_product_price CHECK (base_price >= 0.00),
    CONSTRAINT fk_product_category FOREIGN KEY (category_id) REFERENCES categories (category_id) ON DELETE RESTRICT
);

-- Level 3: Product Variants (FK to Products)
CREATE TABLE product_variants (
    variant_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id INT NOT NULL,
    variant_sku VARCHAR(50) NOT NULL UNIQUE,
    attribute_json JSONB NOT NULL,
    additional_price NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    CONSTRAINT chk_variant_price CHECK (additional_price >= 0.00),
    CONSTRAINT fk_variant_product FOREIGN KEY (product_id) REFERENCES products (product_id) ON DELETE CASCADE
);

-- Level 4: Inventory Items (FK to Product Variants & Warehouses)
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

-- Level 2: Customer Orders (FK to Users)
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

-- Level 3: Order Items (FK to Orders & Product Variants)
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

-- Level 3: Order Invoices (FK to Orders & Currencies)
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

-- Level 3: Shipments (FK to Orders, Shipping Carriers & User Addresses)
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

-- Level 4: Shipment Tracking Events (FK to Shipments)
CREATE TABLE shipment_tracking_events (
    event_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    shipment_id UUID NOT NULL,
    location VARCHAR(150) NOT NULL,
    status_description VARCHAR(255) NOT NULL,
    event_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_tracking_event_shipment FOREIGN KEY (shipment_id) REFERENCES shipments (shipment_id) ON DELETE CASCADE
);

-- Level 5: Delivery Proofs (FK to Shipment Tracking Events)
CREATE TABLE delivery_proofs (
    proof_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id INT NOT NULL UNIQUE,
    recipient_signature_url VARCHAR(2048),
    photo_url VARCHAR(2048),
    delivered_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_delivery_proof_event FOREIGN KEY (event_id) REFERENCES shipment_tracking_events (event_id) ON DELETE CASCADE
);

-- Level 6: Delivery Proof Verification Logs (FK to Delivery Proofs)
CREATE TABLE proof_verifications (
    verification_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    proof_id UUID NOT NULL,
    verified_by_system BOOLEAN NOT NULL DEFAULT FALSE,
    confidence_score NUMERIC(5, 2) NOT NULL,
    verified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_confidence_score CHECK (confidence_score BETWEEN 0.00 AND 100.00),
    CONSTRAINT fk_verification_proof FOREIGN KEY (proof_id) REFERENCES delivery_proofs (proof_id) ON DELETE CASCADE
);

-- Level 3: Payment Transactions (FK to Orders & Currencies)
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

-- Level 3: Product Reviews (FK to Products & Users)
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

-- Level 2: Customer Support Tickets (FK to Users)
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