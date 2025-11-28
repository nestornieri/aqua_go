package main

// MVP API para delivery de bidones
// Tecnologías: Go + Gin + MySQL (database/sql)
// Requisitos locales:
//   go mod init bidones-api
//   go get github.com/gin-gonic/gin github.com/go-sql-driver/mysql
// Variables de entorno sugeridas:
//   DB_DSN="user:pass@tcp(127.0.0.1:3306)/bidones?parseTime=true&charset=utf8mb4&loc=Local"
//   PORT=8080

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// MODELOS BÁSICOS (coinciden con la BD sugerida)

type User struct {
	ID        int64     `json:"id"`
	RoleID    int8      `json:"role_id"`
	FullName  string    `json:"full_name"`
	Phone     *string   `json:"phone,omitempty"`
	Email     *string   `json:"email,omitempty"`
	NumDoc    *string   `json:"num_doc,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt sql.NullTime `json:"created_at"`
}

type Address struct {
	ID        int64    `json:"id"`
	UserID    int64    `json:"user_id"`
	Label     *string  `json:"label,omitempty"`
	Street    string   `json:"street"`
	Reference *string  `json:"reference,omitempty"`
	Lat       *float64 `json:"lat,omitempty"`
	Lng       *float64 `json:"lng,omitempty"`
	IsDefault bool     `json:"is_default"`
}

type Product struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	CapacityLiters *float64 `json:"capacity_liters,omitempty"`
	Price          float64  `json:"price"`
	IsActive       bool     `json:"is_active"`
}

// Precio personalizado por cliente y producto
type CustomerPrice struct {
	CustomerID int64   `json:"customer_id"`
	ProductID  int64   `json:"product_id"`
	Price      float64 `json:"price"`
	IsActive   bool    `json:"is_active"`
}

type UpsertCustomerPriceReq struct {
	CustomerID int64   `json:"customer_id"`
	ProductID  int64   `json:"product_id"`
	Price      float64 `json:"price"`
	IsActive   *bool   `json:"is_active"`
}

type OrderItemReq struct {
	ProductID int64 `json:"product_id"`
	Qty       int   `json:"qty"`
}

type Order struct {
	ID               int64      `json:"id"`
	CustomerID       int64      `json:"customer_id"`
	AddressID        int64      `json:"address_id"`
	AssignedDriverID *int64     `json:"assigned_driver_id,omitempty"`
	Status           string     `json:"status"`
	Subtotal         float64    `json:"subtotal"`
	DeliveryFee      float64    `json:"delivery_fee"`
	Total            float64    `json:"total"`
	Notes            *string    `json:"notes,omitempty"`
	ScheduledAt      sql.NullTime  `json:"schedule_at"`
	DeliveredAt      sql.NullTime  `json:"delivered_at"`
	CreatedAt        sql.NullTime  `json:"created_at"`
}

type OrderWithItems struct {
	Order
	Items []OrderItem `json:"items"`
}

type OrderItem struct {
	ID        int64   `json:"id"`
	OrderID   int64   `json:"order_id"`
	ProductID int64   `json:"product_id"`
	Qty       int     `json:"qty"`
	UnitPrice float64 `json:"unit_price"`
	LineTotal float64 `json:"line_total"`
	// opcional: nombre del producto
	ProductName string   `json:"product_name"`
	Capacity    *float64 `json:"capacity_liters,omitempty"`
}

type StatusHistory struct {
	ID        int64     `json:"id"`
	OrderID   int64     `json:"order_id"`
	OldStatus *string   `json:"old_status,omitempty"`
	NewStatus string    `json:"new_status"`
	ChangedBy int64     `json:"changed_by"`
	ChangedAt  sql.NullTime  `json:"changed_at"`
	Note      *string   `json:"note,omitempty"`
}

// SOLICITUDES

type CreateUserReq struct {
	RoleID   int8    `json:"role_id"` // 1=encargado, 2=repartidor, 3=cliente
	FullName string  `json:"full_name"`
	Phone    *string `json:"phone"`
	Email    *string `json:"email"`
	NumDoc   *string `json:"num_doc"`
	Password string  `json:"password"` // Para MVP no haremos JWT, solo guardamos hash luego
}

type UpdateUserReq struct {
	RoleID   int8    `json:"role_id"`
	FullName string  `json:"full_name"`
	Phone    *string `json:"phone"`
	Email    *string `json:"email"`
	NumDoc   *string `json:"num_doc"`
	Password *string `json:"password"`  // opcional; si viene, se reemplaza
	IsActive *bool   `json:"is_active"` // opcional; por defecto true
}

type CreateAddressReq struct {
	UserID    int64    `json:"user_id"`
	Label     *string  `json:"label"`
	Street    string   `json:"street"`
	Reference *string  `json:"reference"`
	Lat       *float64 `json:"lat"`
	Lng       *float64 `json:"lng"`
	IsDefault bool     `json:"is_default"`
}

type CreateProductReq struct {
	Name           string   `json:"name"`
	CapacityLiters *float64 `json:"capacity_liters"`
	Price          float64  `json:"price"`
	IsActive       *bool    `json:"is_active"`
}

type CreateOrderReq struct {
	CustomerID  int64          `json:"customer_id"`
	AddressID   int64          `json:"address_id"`
	Items       []OrderItemReq `json:"items"`
	ScheduledAt  sql.NullTime  `json:"scheduled_at"`
	Notes       *string        `json:"notes"`
}

type AssignOrderReq struct {
	DriverID int64 `json:"driver_id"`
}

type UpdateStatusReq struct {
	NewStatus string  `json:"new_status"`
	Note      *string `json:"note"`
	ChangedBy int64   `json:"changed_by"`
}

// VARIABLES GLOBALES SIMPLES (para MVP didáctico)

var db *sql.DB

func init() {
	godotenv.Load()
}

func main() {
	// 1) Conexión a MySQL
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("Falta variable DB_DSN")
	}

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("Error al conectar DB:", err)
	}

	// 2) Router
	r := gin.Default()
	r.Use(simpleCORS())

	// Healthcheck
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	// Users (crear mínimo)
	r.GET("/api/v1/users", listUserHandler)
	r.POST("/api/v1/users", createUserHandler)
	r.PUT("/api/v1/users/:id", updateUserHandler)

	// Auth básica (login)
	r.GET("/api/v1/login", basicAuthLoginHandler)

	// Products
	r.GET("/api/v1/products", listProductsHandler) // opcional: ?customer_id= para precio efectivo
	r.POST("/api/v1/products", createProductHandler)
	r.PUT("/api/v1/products/:id", updateProductHandler)
	r.DELETE("/api/v1/products/:id", deleteProductHandler)

	// Customer Prices (precios personalizados)
	r.GET("/api/v1/customer_prices", listCustomerPricesHandler) // requiere ?customer_id=
	r.POST("/api/v1/customer_prices", upsertCustomerPriceHandler)
	r.DELETE("/api/v1/customer_prices", deleteCustomerPriceHandler) // requiere ?customer_id=&product_id=

	// Addresses
	r.GET("/api/v1/addresses", listAddressesHandler) // ?user_id=123
	r.POST("/api/v1/addresses", createAddressHandler)

	// Orders
	r.POST("/api/v1/orders", createOrderHandler)
	r.GET("/api/v1/orders", listOrdersHandler) // ?customer_id=, ?driver_id=
	r.GET("/api/v1/orders/:id", getOrderHandler)
	r.PATCH("/api/v1/orders/:id/assign", assignOrderHandler)
	r.PATCH("/api/v1/orders/:id/status", updateOrderStatusHandler)
	r.GET("/api/v1/orders/:id/history", listOrderHistoryHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("API escuchando en :" + port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// ==== MIDDLEWARE CORS MUY SIMPLE (solo para desarrollo) ====
func simpleCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// ==== HANDLERS ====

// PRODUCTS
func listProductsHandler(c *gin.Context) {
	customerID := c.Query("customer_id")
	var rows *sql.Rows
	var err error
	if customerID != "" {
		rows, err = db.Query(`
            SELECT p.id, p.name, p.capacity_liters,
                   COALESCE(cpp.price, p.price) AS price,
                   p.is_active
            FROM products p
            LEFT JOIN customer_product_prices cpp
              ON cpp.product_id = p.id AND cpp.customer_id = ? AND cpp.is_active = TRUE
            WHERE p.is_active = TRUE
            ORDER BY p.id`, customerID)
	} else {
		rows, err = db.Query(`SELECT id, name, capacity_liters, price, is_active FROM products WHERE is_active=TRUE ORDER BY id`)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var items []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.CapacityLiters, &p.Price, &p.IsActive); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, p)
	}
	c.JSON(http.StatusOK, items)
}

func createProductHandler(c *gin.Context) {
	var req CreateProductReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	res, err := db.Exec(`INSERT INTO products(name, capacity_liters, price, is_active) VALUES (?,?,?,?)`, req.Name, req.CapacityLiters, req.Price, active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func updateProductHandler(c *gin.Context) {
	id := c.Param("id")
	var req CreateProductReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
		return
	}
	// Si no envían is_active, asumimos true para mantener comportamiento explícito del recurso completo (PUT)
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}

	res, err := db.Exec(`UPDATE products SET name=?, capacity_liters=?, price=?, is_active=? WHERE id=?`, req.Name, req.CapacityLiters, req.Price, active, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "producto no encontrado"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func deleteProductHandler(c *gin.Context) {
	id := c.Param("id")
	// Borrado lógico para no romper historiales y joins: is_active = FALSE
	res, err := db.Exec(`UPDATE products SET is_active=FALSE WHERE id=?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "producto no encontrado"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// USERS
func createUserHandler(c *gin.Context) {
	var req CreateUserReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
		return
	}
	if req.FullName == "" || req.RoleID == 0 || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "full_name, role_id y password requeridos"})
		return
	}
	// Para MVP no hasheamos (peligroso en producción). Dejamos password_hash como el password directo.
	res, err := db.Exec(`INSERT INTO users(role_id, full_name, phone, email, num_doc, password_hash, is_active) VALUES (?,?,?,?,?,?,TRUE)`,
		req.RoleID, req.FullName, req.Phone, req.Email, req.NumDoc, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func updateUserHandler(c *gin.Context) {
	id := c.Param("id")
	var req UpdateUserReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
		return
	}
	if req.FullName == "" || req.RoleID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "full_name y role_id requeridos"})
		return
	}

	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}

	var (
		res sql.Result
		err error
	)
	if req.Password != nil {
		res, err = db.Exec(`UPDATE users SET role_id=?, full_name=?, phone=?, email=?, num_doc=?, password_hash=?, is_active=? WHERE id=?`,
			req.RoleID, req.FullName, req.Phone, req.Email, req.NumDoc, *req.Password, active, id)
	} else {
		res, err = db.Exec(`UPDATE users SET role_id=?, full_name=?, phone=?, email=?, num_doc=?, is_active=? WHERE id=?`,
			req.RoleID, req.FullName, req.Phone, req.Email, req.NumDoc, active, id)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// USERS
func listUserHandler(c *gin.Context) {
	rows, err := db.Query(`select id, role_id, full_name, phone, email, num_doc from users order by id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var items []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.RoleID, &u.FullName, &u.Phone, &u.Email, &u.NumDoc); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, u)
	}
	c.JSON(http.StatusOK, items)
}

// CUSTOMER PRICES
func listCustomerPricesHandler(c *gin.Context) {
	customerID := c.Query("customer_id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id requerido"})
		return
	}
	rows, err := db.Query(`
        SELECT customer_id, product_id, price, is_active
        FROM customer_product_prices
        WHERE customer_id = ?
        ORDER BY product_id`, customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var list []CustomerPrice
	for rows.Next() {
		var cp CustomerPrice
		if err := rows.Scan(&cp.CustomerID, &cp.ProductID, &cp.Price, &cp.IsActive); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, cp)
	}
	c.JSON(http.StatusOK, list)
}

func upsertCustomerPriceHandler(c *gin.Context) {
	var req UpsertCustomerPriceReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
		return
	}
	if req.CustomerID == 0 || req.ProductID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id y product_id requeridos"})
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	// Validar que el producto exista y esté activo (MVP: existencia basta)
	var exists int
	if err := db.QueryRow(`SELECT COUNT(1) FROM products WHERE id=?`, req.ProductID).Scan(&exists); err != nil || exists == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id inválido"})
		return
	}
	if err := db.QueryRow(`SELECT COUNT(1) FROM users WHERE id=?`, req.CustomerID).Scan(&exists); err != nil || exists == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id inválido"})
		return
	}
	// Upsert
	_, err := db.Exec(`
        INSERT INTO customer_product_prices(customer_id, product_id, price, is_active)
        VALUES (?,?,?,?)
        ON DUPLICATE KEY UPDATE price=VALUES(price), is_active=VALUES(is_active)`,
		req.CustomerID, req.ProductID, req.Price, active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func deleteCustomerPriceHandler(c *gin.Context) {
	customerID := c.Query("customer_id")
	productID := c.Query("product_id")
	if customerID == "" || productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id y product_id requeridos"})
		return
	}
	_, err := db.Exec(`DELETE FROM customer_product_prices WHERE customer_id=? AND product_id=?`, customerID, productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// AUTH BÁSICA
func basicAuthLoginHandler(c *gin.Context) {
	username, password, ok := c.Request.BasicAuth()
	if !ok {
		c.Header("WWW-Authenticate", "Basic realm=Login")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "credenciales requeridas"})
		return
	}

	var u User
	var stored string
	var active bool
	err := db.QueryRow(`SELECT id, role_id, full_name, phone, email, num_doc, password_hash, is_active FROM users WHERE (email=? OR phone=? OR num_doc=?) LIMIT 1`, username, username, username).
		Scan(&u.ID, &u.RoleID, &u.FullName, &u.Phone, &u.Email, &u.NumDoc, &stored, &active)
	if errors.Is(err, sql.ErrNoRows) {
		c.Header("WWW-Authenticate", "Basic realm=Login")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario o contraseña inválidos"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !active || stored != password {
		c.Header("WWW-Authenticate", "Basic realm=Login")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario o contraseña inválidos"})
		return
	}
	u.IsActive = active
	c.JSON(http.StatusOK, gin.H{"ok": true, "user": u})
}

// ADDRESSES
func listAddressesHandler(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id requerido"})
		return
	}
	rows, err := db.Query(`SELECT id, user_id, label, street, reference, lat, lng, is_default FROM addresses WHERE user_id=? ORDER BY id`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var list []Address
	for rows.Next() {
		var a Address
		if err := rows.Scan(&a.ID, &a.UserID, &a.Label, &a.Street, &a.Reference, &a.Lat, &a.Lng, &a.IsDefault); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, a)
	}
	c.JSON(http.StatusOK, list)
}

func createAddressHandler(c *gin.Context) {
	var req CreateAddressReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
		return
	}
	if req.UserID == 0 || req.Street == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id y street requeridos"})
		return
	}
	res, err := db.Exec(`INSERT INTO addresses(user_id, label, street, reference, lat, lng, is_default) VALUES (?,?,?,?,?,?,?)`,
		req.UserID, req.Label, req.Street, req.Reference, req.Lat, req.Lng, req.IsDefault)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// ORDERS
func createOrderHandler(c *gin.Context) {
	var req CreateOrderReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
		return
	}
	if req.CustomerID == 0 || req.AddressID == 0 || len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id, address_id e items requeridos"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	// Calcular subtotal con precio efectivo (personalizado si existe)
	subtotal := 0.0
	for _, it := range req.Items {
		var effPrice float64
		err := tx.QueryRow(`
            SELECT COALESCE(cpp.price, p.price) AS price
            FROM products p
            LEFT JOIN customer_product_prices cpp
              ON cpp.product_id=p.id AND cpp.customer_id=? AND cpp.is_active=TRUE
            WHERE p.id=? AND p.is_active=TRUE`, req.CustomerID, it.ProductID).Scan(&effPrice)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("producto %d no válido", it.ProductID)})
			return
		}
		subtotal += effPrice * float64(it.Qty)
	}
	deliveryFee := 0.0 // MVP: tarifa plana 0

	// Insert pedido
	res, err := tx.Exec(`INSERT INTO orders(customer_id, address_id, assigned_driver_id, status, subtotal, delivery_fee, notes, scheduled_at) VALUES (?,?,?,?,?,?,?,?)`,
		req.CustomerID, req.AddressID, nil, "por_atender", subtotal, deliveryFee, req.Notes, req.ScheduledAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	orderID, _ := res.LastInsertId()

	// Insert items con precio efectivo
	for _, it := range req.Items {
		var unitPrice float64
		err := tx.QueryRow(`
            SELECT COALESCE(cpp.price, p.price) AS price
            FROM products p
            LEFT JOIN customer_product_prices cpp
              ON cpp.product_id=p.id AND cpp.customer_id=? AND cpp.is_active=TRUE
            WHERE p.id=?`, req.CustomerID, it.ProductID).Scan(&unitPrice)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if _, err := tx.Exec(`INSERT INTO order_items(order_id, product_id, qty, unit_price) VALUES (?,?,?,?)`, orderID, it.ProductID, it.Qty, unitPrice); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	// Historial inicial
	if _, err := tx.Exec(`INSERT INTO order_status_history(order_id, old_status, new_status, changed_by, note) VALUES (?,?,?,?,?)`, orderID, nil, "por_atender", req.CustomerID, "Pedido creado"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"order_id": orderID})
}

func listOrdersHandler(c *gin.Context) {
	customerID := c.Query("customer_id")
	driverID := c.Query("driver_id")
	query := `SELECT id, customer_id, address_id, assigned_driver_id, status, subtotal, delivery_fee, (subtotal+delivery_fee) AS total, notes, scheduled_at, delivered_at, created_at FROM orders`
	var args []any
	if customerID != "" {
		query += " WHERE customer_id=? ORDER BY id DESC"
		args = append(args, customerID)
	} else if driverID != "" {
		query += " WHERE assigned_driver_id=? ORDER BY id DESC"
		args = append(args, driverID)
	} else {
		query += " ORDER BY id DESC LIMIT 50"
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var out []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.AddressID, &o.AssignedDriverID, &o.Status, &o.Subtotal, &o.DeliveryFee, &o.Total, &o.Notes, &o.ScheduledAt, &o.DeliveredAt, &o.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, o)
	}
	c.JSON(http.StatusOK, out)
}

func getOrderHandler(c *gin.Context) {
	id := c.Param("id")
	var o Order
	err := db.QueryRow(`SELECT id, customer_id, address_id, assigned_driver_id, status, subtotal, delivery_fee, (subtotal+delivery_fee) AS total, notes, scheduled_at, delivered_at, created_at FROM orders WHERE id=?`, id).
		Scan(&o.ID, &o.CustomerID, &o.AddressID, &o.AssignedDriverID, &o.Status, &o.Subtotal, &o.DeliveryFee, &o.Total, &o.Notes, &o.ScheduledAt, &o.DeliveredAt, &o.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "no encontrado"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Items
	rows, err := db.Query(`SELECT oi.id, oi.order_id, oi.product_id, oi.qty, oi.unit_price, (oi.qty*oi.unit_price) AS line_total, p.name, p.capacity_liters FROM order_items oi JOIN products p ON p.id=oi.product_id WHERE oi.order_id=?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var items []OrderItem
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.Qty, &it.UnitPrice, &it.LineTotal, &it.ProductName, &it.Capacity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, it)
	}
	c.JSON(http.StatusOK, OrderWithItems{Order: o, Items: items})
}

func assignOrderHandler(c *gin.Context) {
	id := c.Param("id")
	var req AssignOrderReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
		return
	}
	if req.DriverID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "driver_id requerido"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	// Leer estado actual
	var old string
	if err := tx.QueryRow(`SELECT status FROM orders WHERE id=? FOR UPDATE`, id).Scan(&old); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "pedido no existe"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if old != "por_atender" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "solo pedidos 'por_atender' pueden asignarse"})
		return
	}

	if _, err := tx.Exec(`UPDATE orders SET assigned_driver_id=?, status='asignado' WHERE id=?`, req.DriverID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Historial
	if _, err := tx.Exec(`INSERT INTO order_status_history(order_id, old_status, new_status, changed_by, note) VALUES (?,?,?,?,?)`, id, old, "asignado", req.DriverID, "Asignado a repartidor"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func updateOrderStatusHandler(c *gin.Context) {
	id := c.Param("id")
	var req UpdateStatusReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
		return
	}
	if req.NewStatus == "" || req.ChangedBy == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new_status y changed_by requeridos"})
		return
	}

	// Validaciones simples de transición
	valid := map[string][]string{
		"por_atender": {"asignado", "cancelado"},
		"asignado":    {"en_camino", "cancelado"},
		"en_camino":   {"entregado"},
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	var old string
	if err := tx.QueryRow(`SELECT status FROM orders WHERE id=? FOR UPDATE`, id).Scan(&old); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "pedido no existe"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	allowed := false
	for _, s := range valid[old] {
		if s == req.NewStatus {
			allowed = true
			break
		}
	}
	if !allowed {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("transición inválida %s → %s", old, req.NewStatus)})
		return
	}

	q := `UPDATE orders SET status=?`
	if req.NewStatus == "entregado" {
		q += `, delivered_at=NOW()`
	}
	q += ` WHERE id=?`
	if _, err := tx.Exec(q, req.NewStatus, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := tx.Exec(`INSERT INTO order_status_history(order_id, old_status, new_status, changed_by, note) VALUES (?,?,?,?,?)`, id, old, req.NewStatus, req.ChangedBy, req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func listOrderHistoryHandler(c *gin.Context) {
	id := c.Param("id")
	rows, err := db.Query(`SELECT id, order_id, old_status, new_status, changed_by, changed_at, note FROM order_status_history WHERE order_id=? ORDER BY id`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var hist []StatusHistory
	for rows.Next() {
		var h StatusHistory
		if err := rows.Scan(&h.ID, &h.OrderID, &h.OldStatus, &h.NewStatus, &h.ChangedBy, &h.ChangedAt, &h.Note); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		hist = append(hist, h)
	}
	c.JSON(http.StatusOK, hist)
}
