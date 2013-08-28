package meddler

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// DB is a generic database interface, matching both *sql.Db and *sql.Tx
type DB interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// Load loads a record using a query for the primary key field.
// Returns sql.ErrNoRows if not found.
func Load(db DB, table string, dst interface{}, pk int) error {
	columns, err := ColumnsQuoted(dst, true)
	if err != nil {
		return err
	}

	// make sure we have a primary key field
	pkName, _, err := PrimaryKey(dst)
	if err != nil {
		return err
	}
	if pkName == "" {
		return fmt.Errorf("meddler.Load: no primary key field found")
	}

	// run the query
	q := fmt.Sprintf("SELECT %s FROM %s WHERE %s = %s", columns, quoted(table), quoted(pkName), Placeholder)

	rows, err := db.Query(q, pk)
	if err != nil {
		return fmt.Errorf("meddler.Load: DB error in Query: %v", err)
	}

	// scan the row
	return ScanRow(rows, dst)
}

// Insert performs an INSERT query for the given record.
// If the record has a primary key flagged, it must be zero, and it
// will be set to the newly-allocated primary key value from the database
// as returned by LastInsertId.
func Insert(db DB, table string, src interface{}) error {
	pkName, pkValue, err := PrimaryKey(src)
	if err != nil {
		return err
	}
	if pkName != "" && pkValue != 0 {
		return fmt.Errorf("meddler.Insert: primary key must be zero")
	}

	// gather the query parts
	namesPart, err := ColumnsQuoted(src, false)
	if err != nil {
		return err
	}
	valuesPart, err := PlaceholdersString(src, false)
	if err != nil {
		return err
	}
	values, err := Values(src, false)
	if err != nil {
		return err
	}

	// run the query
	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quoted(table), namesPart, valuesPart)
	if PostgreSQL && pkName != "" {
		q += " RETURNING " + quoted(pkName)
		var newPk int
		err := db.QueryRow(q, values...).Scan(&newPk)
		if err != nil {
			return fmt.Errorf("meddler.Insert: DB error in QueryRow: %v", err)
		}
		if err = SetPrimaryKey(src, newPk); err != nil {
			return fmt.Errorf("meddler.Insert: Error saving updated pk: %v", err)
		}
	} else if pkName != "" {
		result, err := db.Exec(q, values...)
		if err != nil {
			return fmt.Errorf("meddler.Insert: DB error in Exec: %v", err)
		}

		// save the new primary key
		newPk, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("meddler.Insert: DB error getting new primary key value: %v", err)
		}
		if err = SetPrimaryKey(src, int(newPk)); err != nil {
			return fmt.Errorf("meddler.Insert: Error saving updated pk: %v", err)
		}
	} else {
		// no primary key, so no need to lookup new value
		_, err := db.Exec(q, values...)
		if err != nil {
			return fmt.Errorf("meddler.Insert: DB error in Exec: %v", err)
		}
	}

	return nil
}

// Update performs and UPDATE query for the given record.
// The record must have an integer primary key field that is non-zero,
// and it will be used to select the database row that gets updated.
func Update(db DB, table string, src interface{}) error {
	// gather the query parts
	names, err := Columns(src, false)
	if err != nil {
		return err
	}
	placeholders, err := Placeholders(src, false)
	if err != nil {
		return err
	}
	values, err := Values(src, false)
	if err != nil {
		return err
	}

	// form the column=placeholder pairs
	var pairs []string
	for i := 0; i < len(names) && i < len(placeholders); i++ {
		pair := fmt.Sprintf("%s=%s", quoted(names[i]), placeholders[i])
		pairs = append(pairs, pair)
	}

	pkName, pkValue, err := PrimaryKey(src)
	if err != nil {
		return err
	}
	if pkName == "" {
		return fmt.Errorf("meddler.Update: no primary key field")
	}
	if pkValue < 1 {
		return fmt.Errorf("meddler.Update: primary key must be an integer > 0")
	}
	ph := strings.Replace(Placeholder, "1", strconv.FormatInt(int64(len(placeholders)+1), 10), 1)

	// run the query
	q := fmt.Sprintf("UPDATE %s SET %s WHERE %s=%s", quoted(table),
		strings.Join(pairs, ","),
		quoted(pkName), ph)
	values = append(values, pkValue)

	if _, err := db.Exec(q, values...); err != nil {
		return fmt.Errorf("meddler.Update: DB error in Exec: %v", err)
	}

	return nil
}

// Save performs an INSERT or an UPDATE, depending on whether or not
// a primary keys exists and is non-zero.
func Save(db DB, table string, src interface{}) error {
	pkName, pkValue, err := PrimaryKey(src)
	if err != nil {
		return err
	}
	if pkName != "" && pkValue != 0 {
		return Update(db, table, src)
	} else {
		return Insert(db, table, src)
	}
}

// QueryOne performs the given query with the given arguments, scanning a
// single row of results into dst. Returns sql.ErrNoRows if there was no
// result row.
func QueryRow(db DB, dst interface{}, query string, args ...interface{}) error {
	// perform the query
	rows, err := db.Query(query, args...)
	if err != nil {
		return err
	}

	// gather the result
	return ScanRow(rows, dst)
}

// QueryAll performs the given query with the given arguments, scanning
// all results rows into dst.
func QueryAll(db DB, dst interface{}, query string, args ...interface{}) error {
	// perform the query
	rows, err := db.Query(query, args...)
	if err != nil {
		return err
	}

	// gather the results
	return ScanAll(rows, dst)
}
