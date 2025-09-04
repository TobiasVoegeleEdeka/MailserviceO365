package db

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"
)

// Client handles database operations.
type Client struct {
	db *sql.DB
}

// NewClient initializes a new database client. It is now database-agnostic
// and accepts a driver name (e.g., "postgres", "mysql").
func NewClient(driverName, dataSourceName string) (*Client, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection with driver '%s': %w", driverName, err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Successfully connected to '%s' database.", driverName)
	return &Client{db: db}, nil
}

// Create inserts a single record into a table based on a struct.
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
		if tag == "" || tag == "-" || strings.HasSuffix(tag, ",omitempty") {
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
	if len(models) == 0 {
		return 0, nil
	}

	tx, err := c.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	v := reflect.ValueOf(models[0])
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var cols []string
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" || strings.HasSuffix(tag, ",omitempty") {
			continue
		}
		cols = append(cols, tag)
	}

	colStr := strings.Join(cols, ", ")
	placeholders := make([]string, len(cols))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	placeholderStr := strings.Join(placeholders, ", ")

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, colStr, placeholderStr)
	stmt, err := tx.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare bulk insert statement: %w", err)
	}
	defer stmt.Close()

	var totalRowsAffected int64
	for _, model := range models {
		v := reflect.ValueOf(model)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		var values []interface{}
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			tag := field.Tag.Get("json")
			if tag == "" || tag == "-" || strings.HasSuffix(tag, ",omitempty") {
				continue
			}
			values = append(values, v.Field(i).Interface())
		}

		res, err := stmt.Exec(values...)
		if err != nil {
			return 0, fmt.Errorf("failed to bulk create record: %w", err)
		}
		rowsAffected, _ := res.RowsAffected()
		totalRowsAffected += rowsAffected
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return totalRowsAffected, nil
}

// Read performs a SELECT query and populates a slice of structs.
func (c *Client) Read(tableName string, dest interface{}, whereClause string, args ...interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("destination must be a pointer to a slice")
	}

	sliceValue := destValue.Elem()
	structType := sliceValue.Type().Elem()
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	cols := make([]string, 0, structType.NumField())
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tag := field.Tag.Get("json")
		if tag != "" && tag != "-" {
			cols = append(cols, tag)
		}
	}

	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(cols, ", "), tableName)
	if whereClause != "" {
		query = fmt.Sprintf("%s WHERE %s", query, whereClause)
	}

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		newStruct := reflect.New(structType)
		var values []interface{}
		for i := 0; i < newStruct.Elem().NumField(); i++ {
			values = append(values, newStruct.Elem().Field(i).Addr().Interface())
		}
		if err := rows.Scan(values...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		sliceValue = reflect.Append(sliceValue, newStruct.Elem())
	}

	destValue.Elem().Set(sliceValue)

	return rows.Err()
}

// Update updates a record in a given table. Returns the number of affected rows.
func (c *Client) Update(tableName string, model interface{}, whereClause string, args ...interface{}) (int64, error) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return 0, fmt.Errorf("expected a struct, but got %T", model)
	}

	var setClauses []string
	var values []interface{}

	i := 1
	for j := 0; j < v.NumField(); j++ {
		field := v.Type().Field(j)
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" || strings.HasSuffix(tag, ",omitempty") {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", tag, i))
		values = append(values, v.Field(j).Interface())
		i++
	}

	query := fmt.Sprintf("UPDATE %s SET %s", tableName, strings.Join(setClauses, ", "))
	if whereClause != "" {
		query = fmt.Sprintf("%s WHERE %s", query, whereClause)
		for j := range args {
			values = append(values, args[j])
		}
	}

	res, err := c.db.Exec(query, values...)
	if err != nil {
		return 0, fmt.Errorf("failed to update record in table '%s': %w", tableName, err)
	}
	return res.RowsAffected()
}

// Delete deletes a record from a given table. Returns the number of affected rows.
func (c *Client) Delete(tableName string, whereClause string, args ...interface{}) (int64, error) {
	query := fmt.Sprintf("DELETE FROM %s", tableName)
	if whereClause != "" {
		query = fmt.Sprintf("%s WHERE %s", query, whereClause)
	}

	res, err := c.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete record from table '%s': %w", tableName, err)
	}
	return res.RowsAffected()
}
