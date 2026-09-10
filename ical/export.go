package ical

import (
	"fmt"
	"strings"
	"time"

	"github.com/benzjeremy/untis-go/api"
)

// EscapeText escapes special characters according to RFC 5545 section 3.3.11
func EscapeText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\r\n", "\\n")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\n")
	return s
}

// FormatDateTime converts date ("2026-09-09") and time ("07:30") to iCal format "20260909T073000"
func FormatDateTime(dateStr, timeStr string) string {
	cleanDate := strings.ReplaceAll(dateStr, "-", "")
	cleanTime := strings.ReplaceAll(timeStr, ":", "")
	if len(cleanDate) != 8 {
		return ""
	}
	if len(cleanTime) == 4 {
		cleanTime += "00"
	}
	if len(cleanTime) != 6 {
		cleanTime = "000000"
	}
	return cleanDate + "T" + cleanTime
}

// ExportTimetable generates standard RFC 5545 iCalendar (.ics) content from enriched lessons
func ExportTimetable(lessons []api.EnrichedLesson, calendarTitle string) string {
	if calendarTitle == "" {
		calendarTitle = "Untis Stundenplan"
	}

	var b strings.Builder
	now := time.Now().UTC().Format("20060102T150405Z")

	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//Jeremy Benz//untis-go//DE\r\n")
	b.WriteString("CALSCALE:GREGORIAN\r\n")
	b.WriteString("METHOD:PUBLISH\r\n")
	b.WriteString(fmt.Sprintf("X-WR-CALNAME:%s\r\n", EscapeText(calendarTitle)))
	b.WriteString("X-WR-TIMEZONE:Europe/Berlin\r\n")

	for i, l := range lessons {
		dtStart := FormatDateTime(l.Date, l.StartTimeStr)
		dtEnd := FormatDateTime(l.Date, l.EndTimeStr)
		if dtStart == "" || dtEnd == "" {
			continue
		}

		// Unique ID for event
		uid := fmt.Sprintf("untis-%s-%d-%d@untis-go", strings.ReplaceAll(l.Date, "-", ""), l.ID, i)

		// Determine Summary / Title
		summary := l.Subject
		if summary == "" {
			summary = "Unterricht"
		}
		if l.SubjectLong != "" && l.SubjectLong != l.Subject {
			summary += " (" + l.SubjectLong + ")"
		}

		if l.IsCancelled {
			summary = "[ENTFALL] " + summary
		} else if l.IsSubstitution {
			summary = "[VERTRETUNG] " + summary
		} else if l.IsRoomChange {
			summary = "[RAUMWECHSEL] " + summary
		}

		// Location
		location := l.Room
		if l.RoomLong != "" && l.RoomLong != l.Room {
			location += " (" + l.RoomLong + ")"
		}

		// Build Description
		var descParts []string
		if l.Teacher != "" {
			t := l.Teacher
			if l.TeacherLong != "" && l.TeacherLong != l.Teacher {
				t += " (" + l.TeacherLong + ")"
			}
			descParts = append(descParts, "Lehrer: "+t)
		}
		if l.Class != "" {
			descParts = append(descParts, "Klasse: "+l.Class)
		}
		if l.Period != "" {
			descParts = append(descParts, "Stunde: "+l.Period)
		}
		if l.TimeRange != "" {
			descParts = append(descParts, "Zeit: "+l.TimeRange)
		}
		if l.SubstText != "" {
			descParts = append(descParts, "Vertretungstext: "+l.SubstText)
		}
		if l.Notes != "" {
			descParts = append(descParts, "Bemerkung: "+l.Notes)
		}
		if l.TeachingContent != "" {
			descParts = append(descParts, "Lehrstoff: "+l.TeachingContent)
		}
		if len(l.Homeworks) > 0 {
			descParts = append(descParts, "Hausaufgaben: "+strings.Join(l.Homeworks, ", "))
		}

		status := "CONFIRMED"
		if l.IsCancelled {
			status = "CANCELLED"
		}

		b.WriteString("BEGIN:VEVENT\r\n")
		b.WriteString(fmt.Sprintf("UID:%s\r\n", uid))
		b.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", now))
		b.WriteString(fmt.Sprintf("DTSTART;TZID=Europe/Berlin:%s\r\n", dtStart))
		b.WriteString(fmt.Sprintf("DTEND;TZID=Europe/Berlin:%s\r\n", dtEnd))
		b.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", EscapeText(summary)))
		if location != "" {
			b.WriteString(fmt.Sprintf("LOCATION:%s\r\n", EscapeText(location)))
		}
		if len(descParts) > 0 {
			b.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", EscapeText(strings.Join(descParts, "\n"))))
		}
		b.WriteString(fmt.Sprintf("STATUS:%s\r\n", status))
		b.WriteString("CLASS:PUBLIC\r\n")
		b.WriteString("END:VEVENT\r\n")
	}

	b.WriteString("END:VCALENDAR\r\n")
	return b.String()
}
