package sqlmarshal

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"reflect"
	"sync"
	"testing"
	"time"
)

var once sync.Once
var db *sql.DB
var when = time.Date(2013, 6, 23, 15, 30, 12, 0, time.UTC)

type Person struct {
	ID        int64  `sqlmarshal:"id,pk"`
	Name      string `sqlmarshal:"name"`
	private   int
	Email     string
	Ephemeral int        `sqlmarshal:"-"`
	Age       int        `sqlmarshal:",zeroisnull"`
	Opened    time.Time  `sqlmarshal:"opened,utctime"`
	Closed    time.Time  `sqlmarshal:"closed,utctimez"`
	Updated   *time.Time `sqlmarshal:"updated,localtime"`
	Height    *int       `sqlmarshal:"height"`
}

const schema = `create table person (
	id integer primary key,
	name text not null,
	Email text not null,
	Age integer,
	opened datetime not null,
	closed datetime,
	updated datetime,
	height integer
)`

func setup() {
	var err error

	// create the database
	db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic("error creating test database: " + err.Error())
	}

	// create the table
	if _, err = db.Exec(schema); err != nil {
		panic("error creating person table: " + err.Error())
	}

	// insert a few rows
	if _, err = db.Exec("insert into person values (null,'Alice','alice@alice.com',32,?,?,?,65)", when, when, when); err != nil {
		panic("error inserting row: " + err.Error())
	}
	if _, err = db.Exec("insert into person values (null,'Bob','bob@bob.com',null,?,null,null,null)", when); err != nil {
		panic("error inserting row: " + err.Error())
	}
}

func structFieldEqual(t *testing.T, elt *structField, ref *structField) {
	if elt == nil {
		t.Errorf("Missing field for %s", ref.column)
		return
	}
	if elt.column != ref.column {
		t.Errorf("Column %s column found as %v", ref.column, elt.column)
	}
	if elt.primaryKey != ref.primaryKey {
		t.Errorf("Column %s primaryKey found as %v", ref.column, elt.primaryKey)
	}
	if elt.index != ref.index {
		t.Errorf("Column %s index found as %v", ref.column, elt.index)
	}
	if elt.meddler != ref.meddler {
		t.Errorf("Column %s meddler mismatch", ref.column)
	}
}

func TestGetFields(t *testing.T) {
	fields, err := getFields(reflect.TypeOf((*Person)(nil)))
	if err != nil {
		t.Errorf("Error in getFields: %v", err)
		return
	}

	// see if everything checks out
	if len(fields) != 8 {
		t.Errorf("Found %d fields, expected 8", len(fields))
	}
	structFieldEqual(t, fields["id"], &structField{"id", 0, true, registry["identity"]})
	structFieldEqual(t, fields["name"], &structField{"name", 1, false, registry["identity"]})
	structFieldEqual(t, fields["Email"], &structField{"Email", 3, false, registry["identity"]})
	structFieldEqual(t, fields["Age"], &structField{"Age", 5, false, registry["zeroisnull"]})
	structFieldEqual(t, fields["opened"], &structField{"opened", 6, false, registry["utctime"]})
	structFieldEqual(t, fields["closed"], &structField{"closed", 7, false, registry["utctimez"]})
	structFieldEqual(t, fields["updated"], &structField{"updated", 8, false, registry["localtime"]})
	structFieldEqual(t, fields["height"], &structField{"height", 9, false, registry["identity"]})
}

func personEqual(t *testing.T, elt *Person, ref *Person) {
	if elt == nil {
		t.Errorf("Person %s is nil", ref.Name)
		return
	}
	if elt.ID != ref.ID {
		t.Errorf("Person %s ID is %v", ref.Name, elt.ID)
	}
	if elt.Name != ref.Name {
		t.Errorf("Person %s Name is %v", ref.Name, elt.Name)
	}
	if elt.private != ref.private {
		t.Errorf("Person %s private is %v", ref.Name, elt.private)
	}
	if elt.Email != ref.Email {
		t.Errorf("Person %s Email is %v", ref.Name, elt.Email)
	}
	if elt.Ephemeral != ref.Ephemeral {
		t.Errorf("Person %s Ephemeral is %v", ref.Ephemeral, elt.Ephemeral)
	}
	if elt.Age != ref.Age {
		t.Errorf("Person %s Age is %v", ref.Name, elt.Age)
	}
	if !elt.Opened.Equal(ref.Opened) {
		t.Errorf("Person %s Opened is %v", ref.Name, elt.Opened)
	}
	if !elt.Closed.Equal(ref.Closed) {
		t.Errorf("Person %s Closed is %v", ref.Name, elt.Closed)
	}
	if (elt.Updated == nil) != (ref.Updated == nil) {
		t.Errorf("Person %s Updated == nil is %v", ref.Name, elt.Updated == nil)
	} else if elt.Updated != nil && !elt.Updated.Equal(*ref.Updated) {
		t.Errorf("Person %s Updated is %v", ref.Name, *elt.Updated)
	}
	if elt.Updated != nil {
		zone, _ := elt.Updated.Zone()
		local, _ := when.Local().Zone()
		if zone != local {
			t.Errorf("Person %s Updated in time zone %v, expected %v", ref.Name, zone, local)
		}
	}
	if (elt.Height == nil) != (ref.Height == nil) {
		t.Errorf("Person %s Height == nil is %v", ref.Name, elt.Height == nil)
	} else if elt.Height != nil && *elt.Height != *ref.Height {
		t.Errorf("Person %s Height is %v", ref.Name, *elt.Height)
	}
}

func TestScanOne(t *testing.T) {
	once.Do(setup)

	rows, err := db.Query("select * from person where id in (1,2) order by id")
	if err != nil {
		t.Errorf("DB error on query: %v", err)
		return
	}

	alice := new(Person)
	if err = ScanRow(rows, alice); err != nil {
		t.Errorf("ScanRow error on Alice: %v", err)
		return

	}

	bob := new(Person)
	bob.Age = 50
	bob.Closed = time.Now()
	bob.private = 14
	bob.Ephemeral = 16
	if err = ScanOne(rows, bob); err != nil {
		t.Errorf("ScanRow error on Bob: %v", err)
		return
	}

	height := 65
	personEqual(t, alice, &Person{1, "Alice", 0, "alice@alice.com", 0, 32, when, when, &when, &height})
	personEqual(t, bob, &Person{2, "Bob", 14, "bob@bob.com", 16, 0, when, time.Time{}, nil, nil})
}

func TestScanAll(t *testing.T) {
	once.Do(setup)

	rows, err := db.Query("select * from person order by id")
	if err != nil {
		t.Errorf("DB error on query: %v", err)
		return
	}

	var lst []*Person
	if err = ScanAll(rows, &lst); err != nil {
		t.Errorf("ScanAll error: %v", err)
		return
	}

	if len(lst) != 2 {
		t.Errorf("ScanAll found %d rows, expected 2", len(lst))
		return
	}

	height := 65
	personEqual(t, lst[0], &Person{1, "Alice", 0, "alice@alice.com", 0, 32, when, when, &when, &height})
	personEqual(t, lst[1], &Person{2, "Bob", 0, "bob@bob.com", 0, 0, when, time.Time{}, nil, nil})
}

func TestSave(t *testing.T) {
	once.Do(setup)

	h := 73
	chris := &Person{
		ID:        0,
		Name:      "Chris",
		Email:     "chris@chris.com",
		Ephemeral: 19,
		Age:       23,
		Opened:    when.Local(),
		Closed:    when,
		Updated:   nil,
		Height:    &h,
	}

	tx, err := db.Begin()
	if err != nil {
		t.Errorf("DB error on begin: %v", err)
	}
	if err = Save(tx, "person", chris); err != nil {
		t.Errorf("DB error on Save: %v", err)
	}

	id := chris.ID
	if id != 3 {
		t.Errorf("DB error on Save: expected ID of 3 but got %d", id)
	}

	chris.Email = "chris@chrischris.com"
	chris.Age = 27

	if err = Save(tx, "person", chris); err != nil {
		t.Errorf("DB error on Save: %v", err)
	}
	if chris.ID != id {
		t.Errorf("ID mismatch: found %d when %d expected", chris.ID, id)
	}
}
