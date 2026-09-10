package diff

import (
	"fmt"
	"strings"

	"github.com/benzjeremy/untis-go/api"
)

type ChangeType string

const (
	ChangeCancelled      ChangeType = "CANCELLED"
	ChangeRoomChanged    ChangeType = "ROOM_CHANGED"
	ChangeTeacherChanged ChangeType = "TEACHER_CHANGED"
	ChangeSubjectChanged ChangeType = "SUBJECT_CHANGED"
	ChangeAdded          ChangeType = "ADDED"
	ChangeRemoved        ChangeType = "REMOVED"
)

type LessonChange struct {
	Type      ChangeType          `json:"type"`
	Title     string              `json:"title"`
	Message   string              `json:"message"`
	Date      string              `json:"date"`
	TimeRange string              `json:"timeRange"`
	OldLesson *api.EnrichedLesson `json:"oldLesson,omitempty"`
	NewLesson *api.EnrichedLesson `json:"newLesson,omitempty"`
}

// makeSlotKey creates a unique key for matching lessons in the timetable grid
func makeSlotKey(l *api.EnrichedLesson) string {
	return fmt.Sprintf("%s_%s_%s", l.Date, l.StartTimeStr, l.EndTimeStr)
}

// CompareLessons compares old and new lesson states and produces a list of meaningful changes
func CompareLessons(oldLessons, newLessons []api.EnrichedLesson) []LessonChange {
	oldMap := make(map[string]api.EnrichedLesson)
	for _, l := range oldLessons {
		oldMap[makeSlotKey(&l)] = l
	}

	newMap := make(map[string]api.EnrichedLesson)
	var changes []LessonChange

	for _, n := range newLessons {
		key := makeSlotKey(&n)
		newMap[key] = n

		old, exists := oldMap[key]
		if !exists {
			// Newly scheduled lesson
			changes = append(changes, LessonChange{
				Type:      ChangeAdded,
				Title:     fmt.Sprintf("📅 Neue Stunde: %s", n.Subject),
				Message:   fmt.Sprintf("%s (%s) in Raum %s bei %s", n.Date, n.TimeRange, n.Room, n.Teacher),
				Date:      n.Date,
				TimeRange: n.TimeRange,
				NewLesson: &n,
			})
			continue
		}

		// 1. Cancellation check
		if !old.IsCancelled && n.IsCancelled {
			changes = append(changes, LessonChange{
				Type:      ChangeCancelled,
				Title:     fmt.Sprintf("❌ Entfall: %s", n.Subject),
				Message:   fmt.Sprintf("%s (%s) in Raum %s entfällt!", n.Date, n.TimeRange, n.Room),
				Date:      n.Date,
				TimeRange: n.TimeRange,
				OldLesson: &old,
				NewLesson: &n,
			})
			continue
		}

		// 2. Room change check
		if old.Room != n.Room && n.Room != "" && old.Room != "" {
			changes = append(changes, LessonChange{
				Type:      ChangeRoomChanged,
				Title:     fmt.Sprintf("🚪 Raumänderung: %s", n.Subject),
				Message:   fmt.Sprintf("%s (%s): Neuer Raum %s (vorher %s)", n.Date, n.TimeRange, n.Room, old.Room),
				Date:      n.Date,
				TimeRange: n.TimeRange,
				OldLesson: &old,
				NewLesson: &n,
			})
		}

		// 3. Teacher change check
		if old.Teacher != n.Teacher && n.Teacher != "" && old.Teacher != "" {
			changes = append(changes, LessonChange{
				Type:      ChangeTeacherChanged,
				Title:     fmt.Sprintf("👤 Vertretung: %s", n.Subject),
				Message:   fmt.Sprintf("%s (%s): Vertretung durch %s (vorher %s)", n.Date, n.TimeRange, n.Teacher, old.Teacher),
				Date:      n.Date,
				TimeRange: n.TimeRange,
				OldLesson: &old,
				NewLesson: &n,
			})
		}

		// 4. Subject change check
		if old.Subject != n.Subject && n.Subject != "" && old.Subject != "" {
			changes = append(changes, LessonChange{
				Type:      ChangeSubjectChanged,
				Title:     fmt.Sprintf("🔄 Fachänderung: %s", n.Subject),
				Message:   fmt.Sprintf("%s (%s): Jetzt %s statt %s", n.Date, n.TimeRange, n.Subject, old.Subject),
				Date:      n.Date,
				TimeRange: n.TimeRange,
				OldLesson: &old,
				NewLesson: &n,
			})
		}
	}

	// Check for removed lessons (existed in old, completely missing in new)
	for _, o := range oldLessons {
		key := makeSlotKey(&o)
		if _, exists := newMap[key]; !exists {
			changes = append(changes, LessonChange{
				Type:      ChangeRemoved,
				Title:     fmt.Sprintf("🗑️ Entfallene Stunde: %s", o.Subject),
				Message:   fmt.Sprintf("%s (%s) wurde aus dem Plan entfernt.", o.Date, o.TimeRange),
				Date:      o.Date,
				TimeRange: o.TimeRange,
				OldLesson: &o,
			})
		}
	}

	return changes
}

// FormatSummary returns a concise summary of changes for logging or notification
func FormatSummary(changes []LessonChange) string {
	if len(changes) == 0 {
		return "Keine Änderungen im Stundenplan."
	}
	var lines []string
	for _, c := range changes {
		lines = append(lines, fmt.Sprintf("- %s: %s", c.Title, c.Message))
	}
	return strings.Join(lines, "\n")
}
