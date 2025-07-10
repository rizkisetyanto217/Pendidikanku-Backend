package helpers

import (
	"regexp"
)

func isAlphaNumeric(s string) bool {
	hasLetter := regexp.MustCompile(`[A-Za-z]`).MatchString(s)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(s)
	return hasLetter && hasNumber
}

// Validasi Email (regex simple)
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}
