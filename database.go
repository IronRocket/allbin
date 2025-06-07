package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Limits struct {
	Reads         int
	DownloadLimit int
}

func updateReads(id string, reads int, downloadLimit int) {
	fmt.Println("update reads:", id, reads, downloadLimit)
	query1 := `UPDATE Space SET reads = $1 WHERE id = $2`
	query2 := `UPDATE InvisibleSpace SET reads = $1 WHERE id = $2`

	res, err := db.Exec(query1, reads, id)
	if err != nil {
		log.Fatal(err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		// Try InvisibleSpace
		res, err = db.Exec(query2, reads, id)
		if err != nil {
			log.Fatal(err)
		}
	}
	if downloadLimit != 0 && reads > downloadLimit {
		purgeById(id)
	}
}

func purgingExpired() {
	for {
		_, err := db.Exec(`
		DELETE FROM Space
		WHERE expirationDate <= cast(unixepoch('subsec') * 1000 as INTEGER);

		DELETE FROM InvisibleSpace
		WHERE expirationDate <= cast(unixepoch('subsec') * 1000 as INTEGER);
		`)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Hour / 2)
	}
}

func purgeById(id string) {
	query1 := `Delete FROM Space WHERE id = ?`
	query2 := `Delete FROM InvisibleSpace WHERE id = ?`

	res, err := db.Exec(query1, id)
	if err != nil {
		log.Fatal(err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		// Try InvisibleSpace
		res, err = db.Exec(query2, id)
		if err != nil {
			log.Fatal(err)
		}
	}
	// Remove dimension from memory buffer if it exists
	for i := range publicDimensions {
		if id == publicDimensions[i].Id {
			publicDimensions[i] = Dimension{}
			logger.Info().Str("dimension-id", id).Msg("deleted dimension")
		}
	}
}

func spaceRead(id string, internelRead bool) (Dimension, bool, bool, error) {
	var dim Dimension
	var visibility bool
	var exists bool
	// Try Space table first
	querySpace := `
			SELECT title, encrypted, fileName,
			text, downloadLimit, reads, expirationDate
			FROM Space WHERE id = ?
		`
	visibility = true
	exists = true
	err := db.QueryRow(querySpace, id).Scan(
		&dim.Title, &dim.Encrypted, &dim.FileName,
		&dim.Text, &dim.DownloadLimit, &dim.Reads,
		&dim.ExpirationDate)
	if errors.Is(err, sql.ErrNoRows) {
		// Try InvisibleSpace table
		queryInvisible := `
				SELECT title, encrypted, fileName,
				text, downloadLimit, reads, expirationDate
				FROM InvisibleSpace WHERE id = ?
			`
		err = db.QueryRow(queryInvisible, id).Scan(
			&dim.Title, &dim.Encrypted, &dim.FileName, &dim.Text,
			&dim.DownloadLimit, &dim.Reads, &dim.ExpirationDate)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.Error().Err(err).Msg("no rows found")
			} else {
				logger.Error().Err(err).Msg("failed to read")
				return dim, false, false, err
			}
			return dim, false, false, err
		}
		visibility = false
		exists = true
	} else if err != nil {
		logger.Error().Err(err).Msg("failed to read")
		return dim, false, false, err
	}
	if exists && !internelRead {
		updateReads(id, dim.Reads+1, dim.DownloadLimit)
	}
	return dim, visibility, exists, nil
}

func printTable(db *sql.DB, tableName string) error {
	// 1. Run the query (no WHERE clause)
	query := fmt.Sprintf("SELECT * FROM %s;", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	// 2. Get column names
	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	// 3. Print header
	fmt.Println("Table:", tableName)
	fmt.Println("Columns:", cols)

	// Prepare containers for row values
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// 4. Iterate rows
	count := 0
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}
		count++
		// Convert each column’s value to a string for printing
		rowData := make([]string, len(cols))
		for i, val := range values {
			if val == nil {
				rowData[i] = "NULL"
			} else {
				rowData[i] = fmt.Sprintf("%v", val)
			}
		}
		fmt.Printf("Row %d: %v\n", count, rowData)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if count == 0 {
		fmt.Println("(no rows)")
	}
	return nil
}

func spaceWrite(id string, dim Dimension) error {
	var query string = `insert into InvisibleSpace (
		id, title, encrypted, fileName,
		text, downloadLimit, reads, expirationDate)
		values(?, ?, ?, ?, ?, ?, ?, ?)
		`
	if dim.Visibility {
		query = `insert into Space (
			id, title, encrypted, fileName,
			text, downloadLimit, reads, expirationDate)
			values(?, ?, ?, ?, ?, ?, ?, ?)`
	}

	stmt, err := db.Prepare(query)
	if err != nil {
		logger.Error().Err(err).Msg("failed to insert dimension into table")
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(
		id, dim.Title, dim.Encrypted,
		dim.FileName, dim.Text,
		dim.DownloadLimit, dim.Reads,
		dim.ExpirationDate)
	if err != nil {
		logger.Error().Err(err).Msg("failed to insert dimension into table")
		return err
	}
	return nil
}

func printDatabase(db *sql.DB) error {
	// 1. Find all user tables
	tblRows, err := db.Query(`
        SELECT name
        FROM sqlite_master
        WHERE type='table'
          AND name NOT LIKE 'sqlite_%';
    `)
	if err != nil {
		return fmt.Errorf("query tables: %w", err)
	}
	defer tblRows.Close()

	var tables []string
	for tblRows.Next() {
		var name string
		if err := tblRows.Scan(&name); err != nil {
			return fmt.Errorf("scan table name: %w", err)
		}
		tables = append(tables, name)
	}

	// 2. For each table, select * and print rows
	for _, tbl := range tables {
		fmt.Printf("\n--- TABLE: %s ---\n", tbl)

		// Get column names
		rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 1", tbl))
		if err != nil {
			return fmt.Errorf("query columns for %s: %w", tbl, err)
		}
		cols, err := rows.Columns()
		rows.Close()
		if err != nil {
			return fmt.Errorf("get columns for %s: %w", tbl, err)
		}

		// Now query all rows
		dataRows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", tbl))
		if err != nil {
			return fmt.Errorf("query data for %s: %w", tbl, err)
		}
		defer dataRows.Close()

		for dataRows.Next() {
			// Prepare a slice of interface{} to receive each column
			vals := make([]interface{}, len(cols))
			ptrs := make([]interface{}, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}

			if err := dataRows.Scan(ptrs...); err != nil {
				return fmt.Errorf("scan row in %s: %w", tbl, err)
			}

			// Print each column name = value
			for i, col := range cols {
				// Handle NULLs nicely
				var valStr string
				if vals[i] == nil {
					valStr = "NULL"
				} else {
					valStr = fmt.Sprintf("%v", vals[i])
				}
				fmt.Printf("%s: %s\t", col, valStr)
			}
			fmt.Println()
		}

		if err := dataRows.Err(); err != nil {
			return fmt.Errorf("iterate rows in %s: %w", tbl, err)
		}
	}

	return nil
}

func initDatebase() {
	os.Remove("./database.db")
	var err error
	db, err = sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatal(err)
	}

	sqlStmt := `
	create table InvisibleSpace (
		id tinytext not null primary key,
		title tinytext,
		encrypted boolean,
		fileName tinytext,
		text text,
		reads integer,
		downloadLimit integer,
		expirationDate integer
	);
	create table Space (
		id tinytext not null primary key,
		title tinytext,
		encrypted boolean,
		fileName tinytext,
		text text,
		reads integer,
		downloadLimit integer,
		expirationDate integer
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	//printDatabase(db)

	go purgingExpired()
}
