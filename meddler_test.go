package sqlscan

import (
	"testing"
)

type Item struct {
	ID     int             `sqlscan:"id,pk"`
	Stuff  map[string]bool `sqlscan:"stuff,json"`
	StuffZ map[string]bool `sqlscan:"stuffz,jsongzip"`
}

func TestJsonMeddler(t *testing.T) {
	once.Do(setup)

	// save a value
	elt := &Item{
		ID: 0,
		Stuff: map[string]bool{
			"hello": true,
			"world": true,
		},
		StuffZ: map[string]bool{
			"goodbyte": true,
			"cruel":    true,
			"world":    true,
		},
	}

	if err := Save(db, "item", elt); err != nil {
		t.Errorf("Save error: %v", err)
	}
	id := elt.ID

	// load it again
	elt = new(Item)
	if err := Load(db, "item", id, elt); err != nil {
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
	if !elt.StuffZ["goodbyte"] || !elt.StuffZ["cruel"] || !elt.StuffZ["world"] {
		t.Errorf("contents of stuffz wrong: %v", elt.StuffZ)
	}
}
