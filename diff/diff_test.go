package diff

import (
	"testing"

	"github.com/benzjeremy/untis-go/api"
)

func TestCompareLessons(t *testing.T) {
	oldLessons := []api.EnrichedLesson{
		{
			ID:           1,
			Date:         "2026-09-09",
			StartTimeStr: "07:30",
			EndTimeStr:   "09:00",
			TimeRange:    "07:30 - 09:00",
			Subject:      "Mathe",
			Teacher:      "MUE",
			Room:         "R101",
			IsCancelled:  false,
		},
		{
			ID:           2,
			Date:         "2026-09-09",
			StartTimeStr: "09:15",
			EndTimeStr:   "10:45",
			TimeRange:    "09:15 - 10:45",
			Subject:      "Deutsch",
			Teacher:      "SCH",
			Room:         "R102",
			IsCancelled:  false,
		},
		{
			ID:           3,
			Date:         "2026-09-09",
			StartTimeStr: "11:00",
			EndTimeStr:   "12:30",
			TimeRange:    "11:00 - 12:30",
			Subject:      "Physik",
			Teacher:      "PHY",
			Room:         "R103",
			IsCancelled:  false,
		},
	}

	newLessons := []api.EnrichedLesson{
		{
			ID:           1,
			Date:         "2026-09-09",
			StartTimeStr: "07:30",
			EndTimeStr:   "09:00",
			TimeRange:    "07:30 - 09:00",
			Subject:      "Mathe",
			Teacher:      "MUE",
			Room:         "R101",
			IsCancelled:  true, // Cancelled!
		},
		{
			ID:           2,
			Date:         "2026-09-09",
			StartTimeStr: "09:15",
			EndTimeStr:   "10:45",
			TimeRange:    "09:15 - 10:45",
			Subject:      "Deutsch",
			Teacher:      "KLE", // Substitution!
			Room:         "R204", // Room change!
			IsCancelled:  false,
		},
		// Lesson 3 removed
		{
			ID:           4,
			Date:         "2026-09-09",
			StartTimeStr: "13:00",
			EndTimeStr:   "14:30",
			TimeRange:    "13:00 - 14:30",
			Subject:      "Sport", // Newly added!
			Teacher:      "SPO",
			Room:         "HALLE",
			IsCancelled:  false,
		},
	}

	changes := CompareLessons(oldLessons, newLessons)

	if len(changes) == 0 {
		t.Fatalf("Expected changes, got none")
	}

	hasCancelled := false
	hasRoomChanged := false
	hasTeacherChanged := false
	hasAdded := false
	hasRemoved := false

	for _, c := range changes {
		switch c.Type {
		case ChangeCancelled:
			hasCancelled = true
		case ChangeRoomChanged:
			hasRoomChanged = true
		case ChangeTeacherChanged:
			hasTeacherChanged = true
		case ChangeAdded:
			hasAdded = true
		case ChangeRemoved:
			hasRemoved = true
		}
	}

	if !hasCancelled {
		t.Errorf("Expected cancellation change")
	}
	if !hasRoomChanged {
		t.Errorf("Expected room change")
	}
	if !hasTeacherChanged {
		t.Errorf("Expected teacher change")
	}
	if !hasAdded {
		t.Errorf("Expected added lesson")
	}
	if !hasRemoved {
		t.Errorf("Expected removed lesson")
	}

	summary := FormatSummary(changes)
	if len(summary) == 0 {
		t.Errorf("Expected non-empty summary")
	}
}
