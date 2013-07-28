package sqlscan

import (
	"fmt"
	"reflect"
	"time"
)

// Meddler is the interface for a field meddler. Implementations can be
// registered to convert struct fields being loaded and saved in the database.
type Meddler interface {
	// PreRead is called before a Scan operation. It is given a pointer to
	// the raw struct field, and returns the value that will be given to
	// the database driver.
	PreRead(fieldAddr interface{}) (scanTarget interface{}, err error)

	// PostRead is called after a Scan operation. It is given the value returned
	// by PreRead and a pointer to the raw struct field. It is expected to fill
	// in the struct field if the two are different.
	PostRead(fieldAddr interface{}, scanTarget interface{}) error

	// PreWrite is called before an Insert or Update operation. It is given
	// a pointer to the raw struct field, and returns the value that will be
	// given to the database driver.
	PreWrite(field interface{}) (saveValue interface{}, err error)
}

// Register sets up a meddler type. Meddlers get a chance to meddle with the
// data being loaded or saved when a field is annotated with the name of the meddler.
// The registry is global.
func Register(name string, m Meddler) {
	if name == "pk" {
		panic("sqlscan.Register: pk cannot be used as a meddler name")
	}
	registry[name] = m
}

var registry = make(map[string]Meddler)

func init() {
	Register("localtime", TimeMeddler{ZeroIsNull: false, Local: true})
	Register("localtimez", TimeMeddler{ZeroIsNull: true, Local: true})
	Register("utctime", TimeMeddler{ZeroIsNull: false, Local: false})
	Register("utctimez", TimeMeddler{ZeroIsNull: true, Local: false})
	Register("zeroisnull", ZeroIsNullMeddler(false))
	Register("identity", IdentityMeddler(false))
}

// IdentityMeddler is the default meddler, and it passes the original value through with
// no changes.
type IdentityMeddler bool

func (elt IdentityMeddler) PreRead(fieldAddr interface{}) (scanTarget interface{}, err error) {
	return fieldAddr, nil
}

func (elt IdentityMeddler) PostRead(fieldAddr, scanTarget interface{}) error {
	return nil
}

func (elt IdentityMeddler) PreWrite(field interface{}) (saveValue interface{}, err error) {
	return field, nil
}

// TimeMeddler provides useful operations on time.Time fields. It can convert the zero time
// to and from a null column, and it can convert the time zone to UTC on save and to Local on load.
type TimeMeddler struct {
	ZeroIsNull bool
	Local      bool
}

func (elt TimeMeddler) PreRead(fieldAddr interface{}) (scanTarget interface{}, err error) {
	switch tgt := fieldAddr.(type) {
	case *time.Time:
		if elt.ZeroIsNull {
			return &tgt, nil
		}
		return fieldAddr, nil
	case **time.Time:
		if elt.ZeroIsNull {
			return nil, fmt.Errorf("sqlscan.TimeMeddler cannot be used on a *time.Time field, only time.Time")
		}
		return fieldAddr, nil
	default:
		return nil, fmt.Errorf("sqlscan.TimeMeddler.PreRead: unknown struct field type: %T", fieldAddr)
	}
}

func (elt TimeMeddler) PostRead(fieldAddr, scanTarget interface{}) error {
	switch tgt := fieldAddr.(type) {
	case *time.Time:
		if elt.ZeroIsNull {
			src := scanTarget.(**time.Time)
			if *src == nil {
				*tgt = time.Time{}
			} else if elt.Local {
				*tgt = (*src).Local()
			} else {
				*tgt = (*src).UTC()
			}
			return nil
		}

		src := scanTarget.(*time.Time)
		if elt.Local {
			*tgt = src.Local()
		} else {
			*tgt = src.UTC()
		}

		return nil

	case **time.Time:
		if elt.ZeroIsNull {
			return fmt.Errorf("sqlscan TimeMeddler cannot be used on a *time.Time field, only time.Time")
		}
		src := scanTarget.(**time.Time)
		if *src == nil {
			*tgt = nil
		} else if elt.Local {
			**src = (*src).Local()
			*tgt = *src
		} else {
			**src = (*src).UTC()
			*tgt = *src
		}

		return nil

	default:
		return fmt.Errorf("sqlscan.TimeMeddler.PostRead: unknown struct field type: %T", fieldAddr)
	}
}

func (elt TimeMeddler) PreWrite(field interface{}) (saveValue interface{}, err error) {
	switch tgt := field.(type) {
	case time.Time:
		if elt.ZeroIsNull && tgt.IsZero() {
			return nil, nil
		}
		return tgt.UTC(), nil

	case *time.Time:
		if tgt == nil || elt.ZeroIsNull && tgt.IsZero() {
			return nil, nil
		}
		return tgt.UTC(), nil

	default:
		return nil, fmt.Errorf("sqlscan.TimeMeddler.PreWrite: unknown struct field type: %T", field)
	}
}

// ZeroIsNullMeddler converts zero value fields (integers both signed and unsigned, floats, complex numbers,
// and strings) to and from null database columns.
type ZeroIsNullMeddler bool

func (elt ZeroIsNullMeddler) PreRead(fieldAddr interface{}) (scanTarget interface{}, err error) {
	// create a pointer to this element
	// the database driver will set it to nil if the column value is null
	return reflect.New(reflect.TypeOf(fieldAddr)).Interface(), nil
}

func (elt ZeroIsNullMeddler) PostRead(fieldAddr, scanTarget interface{}) error {
	sv := reflect.ValueOf(scanTarget)
	fv := reflect.ValueOf(fieldAddr)
	if sv.Elem().IsNil() {
		// null column, so set target to be zero value
		fv.Elem().Set(reflect.Zero(fv.Elem().Type()))
	} else {
		// copy the value that scan found
		fv.Elem().Set(sv.Elem().Elem())
	}
	return nil
}

func (elt ZeroIsNullMeddler) PreWrite(field interface{}) (saveValue interface{}, err error) {
	val := reflect.ValueOf(field)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Int() == 0 {
			return nil, nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val.Uint() == 0 {
			return nil, nil
		}
	case reflect.Float32, reflect.Float64:
		if val.Float() == 0 {
			return nil, nil
		}
	case reflect.Complex64, reflect.Complex128:
		if val.Complex() == 0 {
			return nil, nil
		}
	case reflect.String:
		if val.String() == "" {
			return nil, nil
		}
	default:
		return nil, fmt.Errorf("ZeroIsNullMeddler.PreWrite: unknown struct field type: %T", field)
	}

	return field, nil
}
