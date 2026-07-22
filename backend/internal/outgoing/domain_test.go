package outgoing

import "testing"

func TestNormalizeDestination(t *testing.T) {
	if _, e := normalizeDestination(" "); e == nil {
		t.Fatal("empty accepted")
	}
	v, e := normalizeDestination(" Line A ")
	if e != nil || v != "Line A" {
		t.Fatalf("%q %v", v, e)
	}
}
