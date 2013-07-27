package sqlmarshal

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const tagName = "sqlmarshal"

var Quote = "`"
var Placeholder = "?"

type structField struct {
	column     string
	index      int
	primaryKey bool
	meddler    Meddler
}

// Save performs a REPLACE query to insert or update the given record.
// If the record has a primary key flagged and it is zero, it will
// omitted, and the result of LastInsertId will be stored in the
// original struct for that field.
func Save(tx *sql.Tx, table string, src interface{}) error {
	// get the list of fields
	fields, err := getFields(reflect.TypeOf(src))
	if err != nil {
		return err
	}
	structVal := reflect.ValueOf(src).Elem()

	var columnNames []string
	var columnValues []string
	var values []interface{}
	count := int64(0)
	pkIndex := -1
	pkIsZero := false

	for _, field := range fields {
		columnNames = append(columnNames, Quote+field.column+Quote)

		// preprocess the value to be inserted
		saveVal, err := field.meddler.PreWrite(structVal.Field(field.index).Interface())
		if err != nil {
			return fmt.Errorf("sqlmarshal.Save: error on column %s: %v", field.column, err)
		}

		count++
		ph := strings.Replace(Placeholder, "1", strconv.FormatInt(count, 10), -1)
		columnValues = append(columnValues, ph)
		values = append(values, saveVal)

		if field.primaryKey {
			pkIndex = field.index

			// make sure we have an int value
			switch reflect.TypeOf(saveVal).Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			default:
				return fmt.Errorf("sqlmarshal.Save: primary key column %s has non-integer value", field.column)
			}

			// is the primary key currently zero?
			if reflect.ValueOf(saveVal).Int() == 0 {
				pkIsZero = true

				// undo the normal value insert
				count--
				columnNames = columnNames[:len(columnNames)-1]
				columnValues = columnValues[:len(columnValues)-1]
				values = values[:len(values)-1]
			}
		}
	}

	// form the query string
	q := fmt.Sprintf("REPLACE INTO %s%s%s (%s) values (%s)", Quote, table, Quote,
		strings.Join(columnNames, ","), strings.Join(columnValues, ","))

	// run the query
	result, err := tx.Exec(q, values...)
	if err != nil {
		return fmt.Errorf("sqlmarshal.Save: DB error in Exec: %v", err)
	}

	// check for updated primary key
	if pkIsZero {
		newPk, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("sqlmarshal.Save: DB error getting new primary key value: %v", err)
		}
		structVal.Field(pkIndex).SetInt(newPk)
	}

	return nil
}

// Columns returns a list of column names expected for its input struct.
// It also returns the name of the primary key column (empty string if none).
func Columns(src interface{}) (names []string, pk string, err error) {
	fields, err := getFields(reflect.TypeOf(src))
	if err != nil {
		return nil, "", err
	}
	names, pk = columns(fields)
	return
}

func columns(fields map[string]*structField) (names []string, pk string) {
	for _, elt := range fields {
		names = append(names, elt.column)
		if elt.primaryKey {
			pk = elt.column
		}
	}

	return
}

// ColumnList is similar to Columns, but it return the list of columns in the form:
//   "column1","column2",...
// using Quote as the quote character.
func ColumnList(src interface{}) (names string, pk string, err error) {
	var slice []string
	slice, pk, err = Columns(src)
	if err != nil {
		return
	}
	names = columnList(slice)
	return
}

func columnList(names []string) string {
	var quoted []string
	for _, elt := range names {
		quoted = append(quoted, Quote+elt+Quote)
	}

	return strings.Join(quoted, ",")
}

func getFields(dstType reflect.Type) (map[string]*structField, error) {
	// make sure dst is a non-nil pointer to a struct
	if dstType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("sqlmarshal called with non-pointer destination %v", dstType)
	}
	structType := dstType.Elem()
	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("sqlmarshal called with pointer to non-struct %v", dstType)
	}

	// gather the list of fields in the struct
	fields := make(map[string]*structField)

	foundPrimary := false
	for i := 0; i < structType.NumField(); i++ {
		f := structType.Field(i)

		// skip non-exported fields
		if f.PkgPath != "" {
			continue
		}

		// examine the tag for metadata
		tag := strings.Split(f.Tag.Get(tagName), ",")

		// was this field marked for skipping?
		if len(tag) > 0 && tag[0] == "-" {
			continue
		}

		// default to the field name
		name := f.Name

		// the tag can override the field name
		if len(tag) > 0 && tag[0] != "" {
			name = tag[0]
		}

		// check for a meddler
		primaryKey := false
		var meddler Meddler = registry["identity"]
		for j := 1; j < len(tag); j++ {
			if tag[j] == "pk" {
				if f.Type.Kind() == reflect.Ptr {
					return nil, fmt.Errorf("sqlmarshal found field %s which is marked as the primary key but is a pointer", f.Name)
				}

				// make sure it is an int of some kind
				switch f.Type.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				default:
					return nil, fmt.Errorf("sqlmarshal found field %s which is marked as the primary key, but is not an integer type", f.Name)
				}

				if foundPrimary {
					return nil, fmt.Errorf("sqlmarshal found field %s which is marked as the primary key, but a primary key field was already found", f.Name)
				}
				foundPrimary = true
				primaryKey = true
			} else if m, present := registry[tag[j]]; present {
				meddler = m
			} else {
				return nil, fmt.Errorf("sqlmarshal found field %s with meddler %s, but that meddler is not registered", f.Name, tag[j])
			}
		}

		if _, present := fields[name]; present {
			return nil, fmt.Errorf("sqlmarshal found multiple fields for column %s", name)
		}
		fields[name] = &structField{
			column:     name,
			primaryKey: primaryKey,
			index:      i,
			meddler:    meddler,
		}
	}

	return fields, nil
}

func scanRow(rows *sql.Rows, fields map[string]*structField, columns []string, dst interface{}) error {
	// check if there is data waiting
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	structVal := reflect.ValueOf(dst).Elem()

	// prepare a list of targets
	var targets []interface{}
	for _, column := range columns {
		if field, present := fields[column]; present {
			fieldAddr := structVal.Field(field.index).Addr().Interface()
			scanTarget, err := field.meddler.PreRead(fieldAddr)
			if err != nil {
				return fmt.Errorf("sqlmarshal.scanRow: error on column %s: %v", field.column, err)
			}
			targets = append(targets, scanTarget)
		} else {
			// no destination, so throw this away
			targets = append(targets, &sql.RawBytes{})
		}
	}

	// perform the scan
	if err := rows.Scan(targets...); err != nil {
		return err
	}

	// post-process
	for i, column := range columns {
		if field, present := fields[column]; present {
			fieldAddr := structVal.Field(field.index).Addr().Interface()
			err := field.meddler.PostRead(fieldAddr, targets[i])
			if err != nil {
				return fmt.Errorf("sqlmarshal.scanRow: error on column %s: %v", field.column, err)
			}
		}
	}

	return rows.Err()
}

// ScanRow scans a single sql result row into a struct.
// It leaves rows ready to be scanned again for the next row.
// Returns sql.ErrNoRows if there is no data to read.
func ScanRow(rows *sql.Rows, dst interface{}) error {
	// get the list of struct fields
	fields, err := getFields(reflect.TypeOf(dst))
	if err != nil {
		return err
	}

	// get the sql columns
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	return scanRow(rows, fields, columns, dst)
}

// ScanOne scans a single sql result row into a struct.
// It reads exactly one result row and closes rows when finished.
// Returns sql.ErrNoRows if there is no result row.
func ScanOne(rows *sql.Rows, dst interface{}) error {
	// make sure we always close rows
	defer rows.Close()

	if err := ScanRow(rows, dst); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	return nil
}

// ScanAll scans all sql result rows into a slice of structs.
// It reads all rows and closes rows when finished.
// dst should be a pointer to a slice of the appropriate type.
// The new results will be appended to any existing data in dst.
func ScanAll(rows *sql.Rows, dst interface{}) error {
	// make sure we always close rows
	defer rows.Close()

	// make sure dst is an appropriate type
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr || dstVal.IsNil() {
		return fmt.Errorf("ScanAll called with non-pointer destination: %T", dst)
	}
	sliceVal := dstVal.Elem()
	if sliceVal.Kind() != reflect.Slice {
		return fmt.Errorf("ScanAll called with pointer to non-slice: %T", dst)
	}
	ptrType := sliceVal.Type().Elem()
	if ptrType.Kind() != reflect.Ptr {
		return fmt.Errorf("ScanAll expects element to be pointers, found %T", dst)
	}
	eltType := ptrType.Elem()
	if eltType.Kind() != reflect.Struct {
		return fmt.Errorf("ScanAll expects element to be pointers to structs, found %T", dst)
	}

	// get the list of struct fields
	fields, err := getFields(ptrType)
	if err != nil {
		return err
	}

	// get the sql columns
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// gather the results
	for {
		// create a new element
		eltVal := reflect.New(eltType)
		elt := eltVal.Interface()

		// scan it
		if err := scanRow(rows, fields, columns, elt); err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}

		// add to the result slice
		sliceVal.Set(reflect.Append(sliceVal, eltVal))
	}
}
