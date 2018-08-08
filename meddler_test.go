package meddler

import (
	"testing"
)

type ItemJson struct {
	ID     int64           `meddler:"id,pk"`
	Stuff  map[string]bool `meddler:"stuff,json"`
	StuffZ map[string]bool `meddler:"stuffz,jsongzip"`
}

type ItemGob struct {
	ID     int64           `meddler:"id,pk"`
	Stuff  map[string]bool `meddler:"stuff,gob"`
	StuffZ map[string]bool `meddler:"stuffz,gobgzip"`
}

type ItemZeroes struct {
	ID      int64      `meddler:"id,pk"`
	Int     int        `meddler:"nullint,zeroisnull"`
	Float   float64    `meddler:"nullfloat,zeroisnull"`
	Complex complex128 `meddler:"nullcomplex,zeroisnull"`
	String  string     `meddler:"nullstring,zeroisnull"`
	Bool    bool       `meddler:"nullbool,zeroisnull"`
}

func TestZeroIsNullMeddler(t *testing.T) {
	once.Do(setup)

	before := &ItemZeroes{}
	if err := Save(db, "null_item", before); err != nil {
		t.Errorf("Save error: %v", err)
	}
	id := before.ID

	after := new(ItemZeroes)
	if err := Load(db, "null_item", after, id); err != nil {
		t.Errorf("Load error: %v", err)
	}

	if before.String != after.String {
		t.Errorf("before.String: expected %s, got %s", before.String, after.String)
	}
	if before.Int != after.Int {
		t.Errorf("before.Int: expected %d, got %d", before.Int, after.Int)
	}
	if before.Float != after.Float {
		t.Errorf("before.Float: expected %#v, got %#v", before.Float, after.Float)
	}
	if before.Bool != after.Bool {
		t.Errorf("before.Bool: expected %#v, got %#v", before.Bool, after.Bool)
	}
	if before.Complex != after.Complex {
		t.Errorf("before.Complex: expected %#v, got %#v", before.Complex, after.Complex)
	}
}

func TestJsonMeddler(t *testing.T) {
	once.Do(setup)

	// save a value
	elt := &ItemJson{
		ID: 0,
		Stuff: map[string]bool{
			"hello": true,
			"world": true,
		},
		StuffZ: map[string]bool{
			"goodbye": true,
			"cruel":   true,
			"world":   true,
		},
	}

	if err := Save(db, "item", elt); err != nil {
		t.Errorf("Save error: %v", err)
	}
	id := elt.ID

	// load it again
	elt = new(ItemJson)
	if err := Load(db, "item", elt, id); err != nil {
		t.Errorf("Load error: %v", err)
	}

	if elt.ID != id {
		t.Errorf("expected id of %d, found %d", id, elt.ID)
	}
	if len(elt.Stuff) != 2 {
		t.Errorf("expected %d items in Stuff, found %d", 2, len(elt.Stuff))
	}
	if !elt.Stuff["hello"] || !elt.Stuff["world"] {
		t.Errorf("contents of stuff wrong: %v", elt.Stuff)
	}
	if len(elt.StuffZ) != 3 {
		t.Errorf("expected %d items in StuffZ, found %d", 3, len(elt.StuffZ))
	}
	if !elt.StuffZ["goodbye"] || !elt.StuffZ["cruel"] || !elt.StuffZ["world"] {
		t.Errorf("contents of stuffz wrong: %v", elt.StuffZ)
	}
	if _, err := db.Exec("delete from `item`"); err != nil {
		t.Errorf("error wiping item table: %v", err)
	}
}

func TestGobMeddler(t *testing.T) {
	once.Do(setup)

	// save a value
	elt := &ItemGob{
		ID: 0,
		Stuff: map[string]bool{
			"hello": true,
			"world": true,
		},
		StuffZ: map[string]bool{
			"goodbye": true,
			"cruel":   true,
			"world":   true,
		},
	}

	if err := Save(db, "item", elt); err != nil {
		t.Errorf("Save error: %v", err)
	}
	id := elt.ID

	// load it again
	elt = new(ItemGob)
	if err := Load(db, "item", elt, id); err != nil {
		t.Errorf("Load error: %v", err)
	}

	if elt.ID != id {
		t.Errorf("expected id of %d, found %d", id, elt.ID)
	}
	if len(elt.Stuff) != 2 {
		t.Errorf("expected %d items in Stuff, found %d", 2, len(elt.Stuff))
	}
	if !elt.Stuff["hello"] || !elt.Stuff["world"] {
		t.Errorf("contents of stuff wrong: %v", elt.Stuff)
	}
	if len(elt.StuffZ) != 3 {
		t.Errorf("expected %d items in StuffZ, found %d", 3, len(elt.StuffZ))
	}
	if !elt.StuffZ["goodbye"] || !elt.StuffZ["cruel"] || !elt.StuffZ["world"] {
		t.Errorf("contents of stuffz wrong: %v", elt.StuffZ)
	}
	if _, err := db.Exec("delete from `item`"); err != nil {
		t.Errorf("error wiping item table: %v", err)
	}
}
