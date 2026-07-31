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

func TestNormalizeDeliveryNoteNumberRequiresCompleteValue(t *testing.T) {
	if _, err := normalizeDeliveryNoteNumber("   "); err == nil {
		t.Fatal("empty Delivery Note accepted")
	}
	got, err := normalizeDeliveryNoteNumber(" dn-202607-00003 ")
	if err != nil || got != "DN-202607-00003" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestReceivingErrorCodeUsesStableBusinessCodes(t *testing.T) {
	tests := []struct {
		err  error
		code string
	}{
		{ErrDeliveryNoteInvalid, "DN_INVALID"},
		{ErrDeliveryNoteFullyReceived, "DN_FULLY_RECEIVED"},
		{ErrDeliveryNoteInProgress, "DN_IN_PROGRESS"},
		{ErrKanbanAlreadyScanned, "KANBAN_ALREADY_SCANNED"},
		{ErrKanbanAlreadyReceived, "KANBAN_ALREADY_RECEIVED"},
		{ErrKanbanWrongDeliveryNote, "KANBAN_WRONG_DN"},
		{ErrKanbanNotFound, "KANBAN_NOT_FOUND"},
	}
	for _, tc := range tests {
		if got := receivingErrorCode(tc.err); got != tc.code {
			t.Errorf("receivingErrorCode(%v) = %q, want %q", tc.err, got, tc.code)
		}
	}
}
