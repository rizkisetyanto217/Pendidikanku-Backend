package helpers

import (
	"errors"
	"strings"
)

// Validasi Register
func ValidateRegisterInput(name, email, password, securityAnswer string) error {
	name = sanitizeInput(name)
	email = sanitizeInput(email)
	password = sanitizeInput(password)
	securityAnswer = sanitizeInput(securityAnswer)

	if len(name) < 3 {
		return errors.New("Nama minimal 3 karakter")
	}

	// ⛔ Cek karakter aneh pada email
	if strings.ContainsAny(email, " <>(),;:\"[]") {
		return errors.New("Email mengandung karakter tidak valid")
	}

	if !isValidEmail(email) {
		return errors.New("Format email tidak valid")
	}

	if len(password) < 8 {
		return errors.New("Password minimal 8 karakter")
	}
	if !isAlphaNumeric(password) {
		return errors.New("Password harus mengandung huruf dan angka")
	}

	if len(securityAnswer) < 3 {
		return errors.New("Jawaban keamanan minimal 3 karakter")
	}

	// ⛔ Cek securityAnswer tidak sama persis
	if strings.EqualFold(securityAnswer, name) ||
		strings.EqualFold(securityAnswer, email) ||
		strings.EqualFold(securityAnswer, password) {
		return errors.New("Jawaban keamanan tidak boleh sama dengan nama, email, atau password")
	}

	return nil
}


// Validasi Login
func ValidateLoginInput(identifier, password string) error {
	if len(strings.TrimSpace(identifier)) < 3 {
		return errors.New("Email atau Username minimal 3 karakter")
	}
	if len(password) < 8 {
		return errors.New("Password minimal 8 karakter")
	}
	return nil
}


// Validasi Ganti Password
func ValidateChangePassword(oldPassword, newPassword string) error {
	if len(oldPassword) < 8 || len(newPassword) < 8 {
		return errors.New("Password minimal 8 karakter")
	}
	if oldPassword == newPassword {
		return errors.New("Password baru harus berbeda dengan password lama")
	}
	return nil
}

// Validasi Reset Password
func ValidateResetPassword(email, newPassword string) error {
	if !isValidEmail(email) {
		return errors.New("Format email tidak valid")
	}
	if len(newPassword) < 8 {
		return errors.New("Password baru minimal 8 karakter")
	}
	return nil
}

// Validasi untuk cek jawaban keamanan
func ValidateSecurityAnswerInput(email, answer string) error {
	if !isValidEmail(email) {
		return errors.New("Format email tidak valid")
	}
	if len(strings.TrimSpace(answer)) == 0 {
		return errors.New("Jawaban keamanan tidak boleh kosong")
	}
	return nil
}
