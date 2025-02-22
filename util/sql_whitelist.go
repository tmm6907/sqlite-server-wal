package util

import "strings"

// Checks for ATTACH command in sql query string
func ContainsAttachStatement(query string) (string, bool) {
	hasATTACH := false
	statements := strings.Split(query, "\n")

	for _, statement := range statements {

		if strings.HasPrefix(strings.ToUpper(statement), "ATTACH") {
			hasATTACH = true
		}
	}

	return strings.Join(statements, ""), hasATTACH
}
