Precios personalizados por cliente

Resumen
- Permite definir un precio específico por `customer_id` y `product_id`.
- El API calcula un precio efectivo: si hay override activo, usa ese; de lo contrario usa el precio base del producto.

Endpoints
- `GET /api/v1/products?customer_id=123`
  - Devuelve productos con `price` ya resuelto para ese cliente.
- `GET /api/v1/customer_prices?customer_id=123`
  - Lista overrides definidos para el cliente.
- `POST /api/v1/customer_prices`
  - Body: `{ "customer_id": 123, "product_id": 45, "price": 12.5, "is_active": true }`
  - Upsert: crea o actualiza el precio y estado.
- `DELETE /api/v1/customer_prices?customer_id=123&product_id=45`
  - Elimina el override.

Pedidos
- `POST /api/v1/orders` ahora calcula `subtotal` y `order_items.unit_price` usando el precio efectivo para `customer_id`.

SQL
- Ver `migrations/001_customer_product_prices.sql` para crear la tabla `customer_product_prices`.

Notas
- Si no envías `customer_id` en `GET /api/v1/products`, se devuelven precios base.
- Para desactivar temporalmente un override sin borrarlo, reenvía el POST con `is_active=false`.

