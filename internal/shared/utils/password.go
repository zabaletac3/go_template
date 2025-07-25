// utils/password.go
package utils

import (
	"errors"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost define el costo de bcrypt (10-12 es recomendado para producción)
	BcryptCost = 12
)

// PasswordService maneja todas las operaciones relacionadas con contraseñas
type PasswordService struct {
	cost int
}

// NewPasswordService crea una nueva instancia del servicio de contraseñas
func NewPasswordService() *PasswordService {
	return &PasswordService{
		cost: BcryptCost,
	}
}

// NewPasswordServiceWithCost permite configurar un costo personalizado (útil para tests)
func NewPasswordServiceWithCost(cost int) *PasswordService {
	return &PasswordService{
		cost: cost,
	}
}

// HashPassword hashea una contraseña usando bcrypt
func (ps *PasswordService) HashPassword(password string) (string, error) {
	if err := ps.ValidatePassword(password); err != nil {
		return "", err
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), ps.cost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

// ComparePassword compara una contraseña plana con su hash
func (ps *PasswordService) ComparePassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// ValidatePassword valida la fortaleza de la contraseña
func (ps *PasswordService) ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return errors.New("password cannot exceed 128 characters")
	}

	// Verificar al menos una mayúscula, una minúscula y un dígito
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`\d`).MatchString(password)

	if !hasUpper || !hasLower || !hasDigit {
		return errors.New("password must contain at least one uppercase letter, one lowercase letter, and one digit")
	}

	return nil
}

// NeedsRehash verifica si un hash necesita ser rehashed (por cambio de costo)
func (ps *PasswordService) NeedsRehash(hashedPassword string) bool {
	cost, err := bcrypt.Cost([]byte(hashedPassword))
	if err != nil {
		return true // Si no podemos obtener el costo, asumir que necesita rehash
	}
	return cost != ps.cost
}

// Funciones de conveniencia globales para uso simple

var defaultPasswordService = NewPasswordService()

// HashPassword función global de conveniencia
func HashPassword(password string) (string, error) {
	return defaultPasswordService.HashPassword(password)
}

// ComparePassword función global de conveniencia
func ComparePassword(hashedPassword, password string) bool {
	return defaultPasswordService.ComparePassword(hashedPassword, password)
}

// ValidatePassword función global de conveniencia
func ValidatePassword(password string) error {
	return defaultPasswordService.ValidatePassword(password)
}

// NeedsRehash función global de conveniencia
func NeedsRehash(hashedPassword string) bool {
	return defaultPasswordService.NeedsRehash(hashedPassword)
}