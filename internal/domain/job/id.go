package job

import (
	"fmt"
	"strings"
)

type ID string

func ParseID(value string) (ID, error) {
	normalized := strings.TrimSpace(value)
	jobID := ID(normalized)

	if !jobID.IsValid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidID, value)
	}

	return jobID, nil
}

func (id ID) IsValid() bool {
	value := string(id)
	if len(value) != 36 {
		return false
	}

	for index, character := range value {
		switch index {
		case 8, 13, 18, 23:
			if character != '-' {
				return false
			}
		default:
			if !isHexadecimal(character) {
				return false
			}
		}
	}

	return true
}

func (id ID) String() string {
	return string(id)
}

func isHexadecimal(character rune) bool {
	return character >= '0' && character <= '9' ||
		character >= 'a' && character <= 'f' ||
		character >= 'A' && character <= 'F'
}
