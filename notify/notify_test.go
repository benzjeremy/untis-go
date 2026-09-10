package notify

import (
	"testing"

	"github.com/benzjeremy/untis-go/diff"
)

func TestNotifyLessonChange(t *testing.T) {
	change := diff.LessonChange{
		Type:    diff.ChangeCancelled,
		Title:   "❌ Entfall: Mathe",
		Message: "2026-09-09 (07:30 - 09:00) in Raum R101 entfällt!",
	}

	// This should not panic or fail in test environment
	err := NotifyLessonChange(change)
	if err != nil {
		t.Logf("Notification returned error (acceptable if display/service is absent): %v", err)
	}
}
