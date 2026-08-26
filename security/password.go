package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"unicode"
	"strconv"
	"strings"
)

const (
	passwordHashPrefix = "crm$sha256"
	passwordIterations = 120000
	passwordSaltSize   = 16
)

func IsPasswordHashed(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), passwordHashPrefix+"$")
}

func ValidatePassword(password string) error {
	password = strings.TrimSpace(password)
	if len(password) < 8 {
		return fmt.Errorf("密码至少需要 8 位")
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return fmt.Errorf("密码必须同时包含大写字母、小写字母、数字和特殊字符")
	}
	return nil
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, passwordSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := derivePasswordHash(password, salt, passwordIterations)
	return fmt.Sprintf(
		"%s$%d$%s$%s",
		passwordHashPrefix,
		passwordIterations,
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(storedPassword, plainPassword string) (bool, error) {
	storedPassword = strings.TrimSpace(storedPassword)
	if !IsPasswordHashed(storedPassword) {
		return subtle.ConstantTimeCompare([]byte(storedPassword), []byte(plainPassword)) == 1, nil
	}

	parts := strings.Split(storedPassword, "$")
	if len(parts) != 5 {
		return false, fmt.Errorf("invalid password hash format")
	}

	iterations, err := strconv.Atoi(parts[2])
	if err != nil || iterations <= 0 {
		return false, fmt.Errorf("invalid password hash iterations")
	}

	salt, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, fmt.Errorf("invalid password hash salt")
	}
	expected, err := base64.StdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("invalid password hash digest")
	}

	actual := derivePasswordHash(plainPassword, salt, iterations)
	return subtle.ConstantTimeCompare(expected, actual) == 1, nil
}

func derivePasswordHash(password string, salt []byte, iterations int) []byte {
	seed := append(append([]byte{}, salt...), []byte(password)...)
	sum := sha256.Sum256(seed)
	digest := sum[:]
	for i := 1; i < iterations; i++ {
		nextSeed := make([]byte, 0, len(digest)+len(salt)+len(password))
		nextSeed = append(nextSeed, digest...)
		nextSeed = append(nextSeed, salt...)
		nextSeed = append(nextSeed, password...)
		next := sha256.Sum256(nextSeed)
		digest = next[:]
	}
	result := make([]byte, len(digest))
	copy(result, digest)
	return result
}
