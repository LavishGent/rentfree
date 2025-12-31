package types

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// KeyValidationConfig contains configuration for cache key validation.
type KeyValidationConfig struct {
	ReservedPatterns  []string
	MaxKeyLength      int
	AllowEmpty        bool
	AllowControlChars bool
	AllowWhitespace   bool
}

// DefaultKeyValidationConfig returns a KeyValidationConfig with default values.
func DefaultKeyValidationConfig() KeyValidationConfig {
	return KeyValidationConfig{
		MaxKeyLength:      1024,
		AllowEmpty:        false,
		AllowControlChars: false,
		AllowWhitespace:   true,
		ReservedPatterns:  nil,
	}
}

// KeyValidator validates cache keys according to configured rules.
type KeyValidator struct {
	config KeyValidationConfig
}

// NewKeyValidator creates a new KeyValidator with the given configuration.
func NewKeyValidator(config KeyValidationConfig) *KeyValidator {
	return &KeyValidator{config: config}
}

// Validate checks if a cache key is valid according to the configured rules.
func (v *KeyValidator) Validate(key string) error {
	// Check empty
	if key == "" {
		if !v.config.AllowEmpty {
			return fmt.Errorf("%w: key cannot be empty", ErrInvalidKey)
		}
		return nil
	}

	// Check length
	if v.config.MaxKeyLength > 0 && len(key) > v.config.MaxKeyLength {
		return fmt.Errorf("%w: key length %d exceeds maximum %d bytes",
			ErrInvalidKey, len(key), v.config.MaxKeyLength)
	}

	// Check for valid UTF-8
	if !utf8.ValidString(key) {
		return fmt.Errorf("%w: key contains invalid UTF-8", ErrInvalidKey)
	}

	// Check for control characters and whitespace
	for i, r := range key {
		if r == utf8.RuneError {
			return fmt.Errorf("%w: key contains invalid UTF-8 at position %d", ErrInvalidKey, i)
		}

		// Control characters (ASCII 0-31 and 127)
		if !v.config.AllowControlChars && (r < 32 || r == 127) {
			return fmt.Errorf("%w: key contains control character at position %d", ErrInvalidKey, i)
		}

		// Whitespace (except space which is often allowed)
		if !v.config.AllowWhitespace && unicode.IsSpace(r) {
			return fmt.Errorf("%w: key contains whitespace at position %d", ErrInvalidKey, i)
		}
	}

	// Check reserved patterns
	for _, pattern := range v.config.ReservedPatterns {
		if strings.Contains(key, pattern) {
			return fmt.Errorf("%w: key contains reserved pattern %q", ErrInvalidKey, pattern)
		}
	}

	return nil
}

// ValidateKey validates a key using the default validator.
func ValidateKey(key string) error {
	return DefaultKeyValidator.Validate(key)
}

// DefaultKeyValidator is the default key validator instance.
var DefaultKeyValidator = NewKeyValidator(DefaultKeyValidationConfig())

// IsInvalidKey returns true if the error indicates an invalid key.
func IsInvalidKey(err error) bool {
	return err != nil && strings.Contains(err.Error(), ErrInvalidKey.Error())
}
