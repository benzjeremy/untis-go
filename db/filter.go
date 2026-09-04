package db

import (
	"errors"
	"strings"
	"unicode"
)

// ErrInappropriateContent is returned when a custom alias contains unacceptable terms
var ErrInappropriateContent = errors.New("dieser Name enthält unzulässige oder anstößige Begriffe. Bitte wähle eine respektvolle Bezeichnung")

// Blocked patterns for extremist, hate speech, violent or offensive terms
var blockedTerms = []string{
	"hitler",
	"nazi",
	"hakenkreuz",
	"swastika",
	"vergasen",
	"gasdusche",
	"gasunterricht",
	"gaskammer",
	"auschwitz",
	"holocaust",
	"zyklon",
	"arier",
	"aryan",
	"goebbels",
	"himmler",
	"eichmann",
	"neger",
	"kanake",
	"judensau",
	"siegheil",
	"heilhitler",
}

// ValidateCustomAlias checks if an alias contains prohibited, offensive, or hate-speech terms
func ValidateCustomAlias(alias string) error {
	trimmed := strings.TrimSpace(alias)
	if trimmed == "" {
		return errors.New("fachname darf nicht leer sein")
	}

	// Normalize: lowercase, remove non-letters/digits, map common leet substitutions
	lower := strings.ToLower(trimmed)
	var normalized strings.Builder
	for _, r := range lower {
		switch r {
		case '1', '!':
			normalized.WriteRune('i')
		case '0':
			normalized.WriteRune('o')
		case '3':
			normalized.WriteRune('e')
		case '4', '@':
			normalized.WriteRune('a')
		case '$', '5':
			normalized.WriteRune('s')
		case '7':
			normalized.WriteRune('t')
		default:
			if unicode.IsLetter(r) {
				normalized.WriteRune(r)
			}
		}
	}
	normStr := normalized.String()

	for _, term := range blockedTerms {
		if strings.Contains(lower, term) || strings.Contains(normStr, term) {
			return ErrInappropriateContent
		}
	}

	return nil
}
