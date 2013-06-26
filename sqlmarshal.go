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
	value      reflect.Value
}

const tagName = "sqlmarshal"

var typeOfTime = reflect.TypeOf(time.Time{})

func getFields(dst interface{}) (map[string]*structField, error) {
	// make sure dst is a non-nil pointer to a struct
	dstType := reflect.TypeOf(dst)
	if dstType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("sqlmarshal called with non-pointer destination %T", dst)
	}
	dstVal := reflect.ValueOf(dst)
	if dstVal.IsNil() {
		return nil, fmt.Errorf("sqlmarshal called with nil pointer destination %T", dst)
	}
	structType := dstType.Elem()
	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("sqlmarshal called with pointer to non-struct %T", dst)
	}
	structVal := dstVal.Elem()

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
		value := structVal.Field(i)
		if !value.CanSet() {
			return nil, fmt.Errorf("sqlmarshal found field %s that cannot be set", f.Name)
		}
		fields[name] = &structField{
			column:     name,
			zeroIsNull: zeroIsNull,
			primaryKey: primaryKey,
			value:      structVal.Field(i).Addr(),
		}
	}

	return fields, nil
}

func ScanOne(dst interface{}, rows *sql.Rows) error {
	if err := ScanRow(dst, rows); err != nil {
		// make sure we always close rows
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	return nil
}

func ScanRow(dst interface{}, rows *sql.Rows) error {
	// get the list of struct fields
	fields, err := getFields(dst)
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

func scanRow(dst interface{}, rows *sql.Rows, fields map[string]*structField, columns []string) error {
	// check if there is data waiting
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	// prepare a list of targets
	var targets []interface{}
	for _, column := range columns {
		if field, present := fields[column]; present {
			if field.zeroIsNull {
				// create a pointer to this element
				ptr := reflect.New(field.value.Type()).Interface()
				targets = append(targets, ptr)
			} else {
				// point to the original element
				targets = append(targets, field.value.Interface())
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
			// convert null results to zero value
			if field.zeroIsNull {
				ptr := targets[i]
				v := reflect.ValueOf(ptr)
				if v.Elem().IsNil() {
					// null column, so set target to be zero value
					field.value.Elem().Set(reflect.Zero(field.value.Elem().Type()))
				} else {
					// copy the value that scan found
					field.value.Elem().Set(v.Elem().Elem())
				}
			}

			// convert time elements to local time zone
			if field.value.Elem().Type().ConvertibleTo(typeOfTime) {
				if t, okay := field.value.Elem().Interface().(time.Time); okay {
					field.value.Elem().Set(reflect.ValueOf(t.Local()))
				}
			}
		}
	}

	return rows.Err()
}
