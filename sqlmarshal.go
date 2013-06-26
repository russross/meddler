package sqlmarshal

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type structField struct {
	column     string
	zeroIsNull bool
	primaryKey bool
	index int
}

const tagName = "sqlmarshal"

var typeOfTime = reflect.TypeOf(time.Time{})

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

		// the tag can override the field name
		name := f.Name
		if len(tag) > 0 && tag[0] != "" {
			name = tag[0]
		}

		// check for flags: zeroisnull and primarykey
		zeroIsNull := false
		primaryKey := false
		for j := 1; j < len(tag); j++ {
			switch tag[j] {
			case "zeroisnull":
				if f.Type.Kind() == reflect.Ptr {
					return nil, fmt.Errorf("sqlmarshal found field %s which is marked zeroisnull but is a pointer", f.Name)
				}
				zeroIsNull = true
			case "primarykey":
				if f.Type.Kind() == reflect.Ptr {
					return nil, fmt.Errorf("sqlmarshal found field %s which is marked as the primary key but is a pointer", f.Name)
				}
				if foundPrimary {
					return nil, fmt.Errorf("sqlmarshal found field %s which is marked as the primary key, but a primary key field was already found", f.Name)
				}
				primaryKey = true
				foundPrimary = true
			default:
				return nil, fmt.Errorf("sqlmarshal found unknown tag %s in field %s", tag[j], f.Name)
			}
		}

		if _, present := fields[name]; present {
			return nil, fmt.Errorf("sqlmarshal found multiple fields for column %s", name)
		}
		fields[name] = &structField{
			column:     name,
			zeroIsNull: zeroIsNull,
			primaryKey: primaryKey,
			index: i,
		}
	}

	return fields, nil
}

func scanRow(dst interface{}, rows *sql.Rows, fields map[string]*structField, columns []string) error {
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
			value := structVal.Field(field.index).Addr()
			if field.zeroIsNull {
				// create a pointer to this element
				ptr := reflect.New(value.Type()).Interface()
				targets = append(targets, ptr)
			} else {
				// point to the original element
				targets = append(targets, value.Interface())
			}
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
			value := structVal.Field(field.index).Addr()

			// convert null results to zero value
			if field.zeroIsNull {
				ptr := targets[i]
				v := reflect.ValueOf(ptr)
				if v.Elem().IsNil() {
					// null column, so set target to be zero value
					value.Elem().Set(reflect.Zero(value.Elem().Type()))
				} else {
					// copy the value that scan found
					value.Elem().Set(v.Elem().Elem())
				}
			}

			// convert time elements to local time zone
			if value.Elem().Type().ConvertibleTo(typeOfTime) {
				if t, okay := value.Elem().Interface().(time.Time); okay {
					value.Elem().Set(reflect.ValueOf(t.Local()))
				}
			}
		}
	}

	return rows.Err()
}

// ScanRow scans a single sql result row into a struct.
// It leaves rows ready to be scanned again for the next row.
// Returns sql.ErrNoRows if there is no data to read.
func ScanRow(dst interface{}, rows *sql.Rows) error {
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

	return scanRow(dst, rows, fields, columns)
}

// ScanOne scans a single sql result row into a struct.
// It reads exactly one result row and closes rows when finished.
// Returns sql.ErrNoRows if there is no result row.
func ScanOne(dst interface{}, rows *sql.Rows) error {
	// make sure we always close rows
	defer rows.Close()

	if err := ScanRow(dst, rows); err != nil {
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
func ScanAll(dst interface{}, rows *sql.Rows) error {
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
		if err := scanRow(elt, rows, fields, columns); err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}

		// add to the result slice
		sliceVal.Set(reflect.Append(sliceVal, eltVal))
	}
}
