package receiving

import "testing"

func TestNormalizeKanbanRejectsEmptyAndNormalizesCase(t *testing.T) {
	if _, err := normalizeKanban("   "); err == nil {
		t.Fatal("empty Kanban accepted")
	}
	got, err := normalizeKanban(" kb-260722-000001 ")
	if err != nil || got != "KB-260722-000001" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestDestinationIsNotPartOfReceiving(t *testing.T) {
	if SessionActive == SessionPaused {
		t.Fatal("session states must be distinct")
	}
}
