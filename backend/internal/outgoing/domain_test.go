package outgoing

import "testing"

func TestNormalizeDestination(t *testing.T) {
	if value, e := normalizeDestination(" "); e != nil || value != "" {
		t.Fatalf("empty destination = %q, %v", value, e)
	}
	v, e := normalizeDestination(" Line A ")
	if e != nil || v != "Line A" {
		t.Fatalf("%q %v", v, e)
	}
	if _, e := normalizeDestination(string(make([]byte, 121))); e == nil {
		t.Fatal("overlength destination accepted")
	}
}
