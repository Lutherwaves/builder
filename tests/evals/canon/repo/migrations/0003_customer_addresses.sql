CREATE TABLE customer_addresses (id BIGSERIAL PRIMARY KEY, customer_id BIGINT NOT NULL REFERENCES customers(id), line1 TEXT NOT NULL, city TEXT NOT NULL, postcode TEXT NOT NULL);
