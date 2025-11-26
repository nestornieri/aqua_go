-- Tabla para precios personalizados por cliente y producto
CREATE TABLE IF NOT EXISTS customer_product_prices (
  customer_id BIGINT NOT NULL,
  product_id  BIGINT NOT NULL,
  price       DECIMAL(10,2) NOT NULL,
  is_active   BOOLEAN NOT NULL DEFAULT TRUE,
  created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (customer_id, product_id)
  -- Opcional si tu esquema usa FK:
  -- , CONSTRAINT fk_cpp_customer FOREIGN KEY (customer_id) REFERENCES users(id)
  -- , CONSTRAINT fk_cpp_product  FOREIGN KEY (product_id)  REFERENCES products(id)
);

-- Notas:
-- - El PRIMARY KEY permite usar INSERT ... ON DUPLICATE KEY UPDATE en el API.
-- - Usa DECIMAL(10,2) para almacenar precios con 2 decimales.
-- - Activa/desactiva un override con la columna is_active.

