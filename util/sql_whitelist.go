package util

import "strings"

// Checks for ATTACH command in sql query string
func ContainsAttachStatement(query string) bool {

	statements := strings.Split(query, ";")

	for _, statement := range statements {
		trimmed := strings.TrimSpace(statement)

		if strings.HasPrefix(strings.ToUpper(trimmed), "ATTACH") {
			return true
		}
	}

	return false
}
