package sqlmarshal

import (
	"fmt"
	"time"
	"reflect"
)

type Meddler interface {
	PreRead(fieldAddr interface{}) (scanTarget interface{}, err error)
	PostRead(fieldAddr interface{}, scanTarget interface{}) error
}

func Register(name string, m Meddler) {
	if name == "pk" {
		panic("sqlmarshal.Register: pk cannot be used as a meddler name")
	}
	registry[name] = m
}

var registry = make(map[string]Meddler)

func init() {
	Register("localtime", TimeMeddler{ ZeroIsNull: false, Local: true })
	Register("localtimez", TimeMeddler{ ZeroIsNull: true, Local: true })
	Register("utctime", TimeMeddler{ ZeroIsNull: false, Local: false })
	Register("utctimez", TimeMeddler{ ZeroIsNull: true, Local: false })
	Register("zeroisnull", ZeroIsNullMeddler(false))
	Register("identity", IdentityMeddler(false))
}

type IdentityMeddler bool

func (elt IdentityMeddler) PreRead(fieldAddr interface{}) (scanTarget interface{}, err error) {
	return fieldAddr, nil
}

func (elt IdentityMeddler) PostRead(fieldAddr, scanTarget interface{}) error {
	return nil
}

type TimeMeddler struct {
	ZeroIsNull bool
	Local bool
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
			return nil, fmt.Errorf("sqlmarshal.TimeMeddler cannot be used on a *time.Time field, only time.Time")
		}
		return fieldAddr, nil
	default:
		return nil, fmt.Errorf("sqlmarshal.TimeMeddler.PreRead: unknown struct field type: %T", fieldAddr)
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
			return fmt.Errorf("sqlmarshal TimeMeddler cannot be used on a *time.Time field, only time.Time")
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
		return fmt.Errorf("sqlmarshal.TimeMeddler.PostRead: unknown struct field type: %T", fieldAddr)
	}
}

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
