package ical

import (
	"strings"
	"testing"

	"github.com/benzjeremy/untis-go/api"
)

func TestEscapeText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello, World!", "Hello\\, World!"},
		{"Math; Physics", "Math\\; Physics"},
		{"Line 1\nLine 2", "Line 1\\nLine 2"},
		{"Backslash \\ here", "Backslash \\\\ here"},
	}

	for _, tt := range tests {
		got := EscapeText(tt.input)
		if got != tt.expected {
			t.Errorf("EscapeText(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatDateTime(t *testing.T) {
	tests := []struct {
		dateStr  string
		timeStr  string
		expected string
	}{
		{"2026-09-09", "07:30", "20260909T073000"},
		{"2026-10-15", "13:45", "20261015T134500"},
		{"invalid", "07:30", ""},
	}

	for _, tt := range tests {
		got := FormatDateTime(tt.dateStr, tt.timeStr)
		if got != tt.expected {
			t.Errorf("FormatDateTime(%q, %q) = %q, want %q", tt.dateStr, tt.timeStr, got, tt.expected)
		}
	}
}

func TestExportTimetable(t *testing.T) {
	lessons := []api.EnrichedLesson{
		{
			ID:           101,
			Date:         "2026-09-09",
			StartTimeStr: "07:30",
			EndTimeStr:   "09:00",
			Subject:      "Mathe",
			SubjectLong:  "Mathematik",
			Teacher:      "MUE",
			TeacherLong:  "Müller, Hans",
			Room:         "R101",
			RoomLong:     "Raum 101",
			Class:        "10A",
			Period:       "1. - 2. Stunde",
			IsCancelled:  false,
		},
		{
			ID:           102,
			Date:         "2026-09-09",
			StartTimeStr: "09:15",
			EndTimeStr:   "10:45",
			Subject:      "Englisch",
			Teacher:      "SMT",
			Room:         "R102",
			Class:        "10A",
			Period:       "3. - 4. Stunde",
			IsCancelled:  true,
		},
	}

	ics := ExportTimetable(lessons, "Klasse 10A Stundenplan")

	if !strings.HasPrefix(ics, "BEGIN:VCALENDAR\r\n") {
		t.Errorf("Expected VCALENDAR header, got: %s", ics)
	}
	if !strings.HasSuffix(ics, "END:VCALENDAR\r\n") {
		t.Errorf("Expected VCALENDAR footer, got: %s", ics)
	}
	if !strings.Contains(ics, "SUMMARY:Mathe (Mathematik)") {
		t.Errorf("Expected SUMMARY with Subject and SubjectLong")
	}
	if !strings.Contains(ics, "SUMMARY:[ENTFALL] Englisch") {
		t.Errorf("Expected [ENTFALL] in cancelled summary")
	}
	if !strings.Contains(ics, "STATUS:CANCELLED") {
		t.Errorf("Expected STATUS:CANCELLED for cancelled lesson")
	}
	if !strings.Contains(ics, "STATUS:CONFIRMED") {
		t.Errorf("Expected STATUS:CONFIRMED for regular lesson")
	}
	if !strings.Contains(ics, "LOCATION:R101 (Raum 101)") {
		t.Errorf("Expected LOCATION in event")
	}
}
