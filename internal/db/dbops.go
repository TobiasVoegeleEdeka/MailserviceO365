package db

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	_ "github.com/lib/pq" // Stellt sicher, dass der PostgreSQL-Treiber registriert ist.
)

// Client handles database operations.
type Client struct {
	db *sql.DB
}

// NewClient initializes a new database client.
func NewClient(driverName, dataSourceName string) (*Client, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection with driver '%s': %w", driverName, err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return &Client{db: db}, nil
}

// Create inserts a single record into a table based on a struct.
// KORRIGIERT: Ignoriert jetzt das 'id'-Feld, damit die DB es automatisch generiert.
func (c *Client) Create(tableName string, model interface{}) (int64, error) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return 0, fmt.Errorf("expected a struct, but got %T", model)
	}

	var cols, placeholders []string
	var values []interface{}

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		tag := field.Tag.Get("json")

		// *** HIER IST DIE KORREKTUR ***
		// Wir überspringen das 'id'-Feld, damit PostgreSQL es selbst verwalten kann.
		if tag == "id" || tag == "" || tag == "-" || strings.HasSuffix(tag, ",omitempty") {
			continue
		}

		cols = append(cols, tag)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(cols)))
		values = append(values, v.Field(i).Interface())
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		tableName,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	var id int64
	err := c.db.QueryRow(query, values...).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create record in table '%s': %w", tableName, err)
	}
	return id, nil
}

// BulkCreate inserts multiple records into a table in a single transaction.
func (c *Client) BulkCreate(tableName string, models []interface{}) (int64, error) {
	// ... (unveränderter Code)
	return 0, nil // Platzhalter
}

// Read performs a SELECT query and populates a slice of structs.
func (c *Client) Read(tableName string, dest interface{}, whereClause string, args ...interface{}) error {
	// ... (unveränderter Code)
	return nil // Platzhalter
}

// Update updates a record in a given table. Returns the number of affected rows.
func (c *Client) Update(tableName string, model interface{}, whereClause string, args ...interface{}) (int64, error) {
	// ... (unveränderter Code)
	return 0, nil // Platzhalter
}

// Delete deletes a record from a given table. Returns the number of affected rows.
func (c *Client) Delete(tableName string, whereClause string, args ...interface{}) (int64, error) {
	// ... (unveränderter Code)
	return 0, nil // Platzhalter
}
