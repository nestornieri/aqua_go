go mod init bk_rep_agua
go get github.com/gin-gonic/gin github.com/go-sql-driver/mysql


set DB_DSN=root:@tcp(127.0.0.1:3606)/bd_rep_agua?parseTime=true&charset=utf8mb4&loc=Local
set PORT=8080
go run .


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



INTERPRETACION:
escribir que se quiere hallar
escribir los datos que se deducen del enunciado
escribir que estrategia o proceso se debe aplicar para hallar lo que se pide o responder a la pregunta del problem


REPRESENTACION:
definir variables con unidades y restrincciones
representar los datos y la pregunta de forma simbolica empleando las variables definidas
escribir las formulas a utilizar en la resolucion
escbozar un grafico o elaborar una tabla si es necesario

CALCULO:
plantear las ecuaciones necesarias y resolverlas utilizando algun aplicativo o calculadora cientifica
aplicar proceso coherentes y logicos que permitan llegar a una solucion correcta.

ANALISIS:
escribir la respues del problema justificando el resultado y especificando su validez


API - Cambios recientes
- Campo `num_doc` (VARCHAR(10) NULL) agregado a `users`. La API acepta y devuelve `num_doc` como opcional.
- Endpoint de login básico: `GET /api/v1/login` usando HTTP Basic Auth.
  - Usuario: puede ser `email`, `phone` o `num_doc` del usuario.
  - Contraseña: compara con `password_hash` almacenado (MVP: texto plano).
  - Respuesta 200: `{ ok: true, user: { ... } }`. 401 si inválido.

SQL sugerido para la columna:
```sql
ALTER TABLE users ADD COLUMN num_doc VARCHAR(10) NULL;
```
