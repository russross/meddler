package meddler

import (
	"testing"
	"time"
)

func TestLoadFromCache(t *testing.T) {
	once.Do(setup)
	insertAliceBob(t)

	cache := NewPermStmtCache(db)

	elt := new(Person)
	elt.Age = 50
	elt.Closed = time.Now()
	if err := Load(cache, "person", elt, 2); err != nil {
		t.Errorf("Load error on Bob: %v", err)
		return
	}
	bob.ID = 2
	personEqual(t, elt, bob)
	db.Exec("delete from person")
}

func TestSaveFromCache(t *testing.T) {
	once.Do(setup)
	insertAliceBob(t)

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
	cache := NewPermStmtCache(db)
	txcache := NewTxStmtCache(cache, tx)

	if err != nil {
		t.Errorf("DB error on begin: %v", err)
	}
	if err = Save(txcache, "person", chris); err != nil {
		t.Errorf("DB error on Save: %v", err)
	}

	id := chris.ID
	if id != 3 {
		t.Errorf("DB error on Save: expected ID of 3 but got %d", id)
	}

	chris.Email = "chris@chrischris.com"
	chris.Age = 27

	if err = Save(txcache, "person", chris); err != nil {
		t.Errorf("DB error on Save: %v", err)
	}
	if chris.ID != id {
		t.Errorf("ID mismatch: found %d when %d expected", chris.ID, id)
	}
	if err = tx.Commit(); err != nil {
		t.Errorf("Commit error: %v", err)
	}

	// now test if the data looks right
	rows, err := db.Query("select * from person where id = ?", id)
	if err != nil {
		t.Errorf("DB error on query: %v", err)
		return
	}

	p := new(Person)
	if err = ScanRow(rows, p); err != nil {
		t.Errorf("ScanRow error on Chris: %v", err)
		return
	}

	personEqual(t, p, &Person{3, "Chris", 0, "chris@chrischris.com", 0, 27, when, when, nil, &h})

	// delete this record so we don't confuse other tests
	if _, err = db.Exec("delete from person where id = ?", id); err != nil {
		t.Errorf("DB error on delete: %v", err)
	}
	db.Exec("delete from person")
}

func insertAliceBobBench(b *testing.B) {
	// insert Alice as row #1
	alice.ID = 0
	if err := Insert(db, "person", alice); err != nil {
		b.Errorf("Error inserting Alice: %v", err)
	}
	if alice.ID != 1 {
		b.Errorf("Alice ID is %d, expecting 1", alice.ID)
	}

	// insert Bob as row #2
	bob.ID = 0
	if err := Insert(db, "person", bob); err != nil {
		b.Errorf("Error inserting Bob: %v", err)
	}
	if bob.ID != 2 {
		b.Errorf("Bob ID is %d, expecting 2", bob.ID)
	}
}
