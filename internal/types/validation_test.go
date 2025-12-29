package types

import (
	"errors"
	"strings"
	"testing"
)

func TestDefaultKeyValidationConfig(t *testing.T) {
	cfg := DefaultKeyValidationConfig()

	if cfg.MaxKeyLength != 1024 {
		t.Errorf("MaxKeyLength = %d, want 1024", cfg.MaxKeyLength)
	}
	if cfg.AllowEmpty {
		t.Error("AllowEmpty = true, want false")
	}
	if cfg.AllowControlChars {
		t.Error("AllowControlChars = true, want false")
	}
	if !cfg.AllowWhitespace {
		t.Error("AllowWhitespace = false, want true")
	}
	if cfg.ReservedPatterns != nil {
		t.Error("ReservedPatterns should be nil by default")
	}
}

func TestNewKeyValidator(t *testing.T) {
	cfg := KeyValidationConfig{
		MaxKeyLength: 512,
		AllowEmpty:   true,
	}

	v := NewKeyValidator(cfg)

	if v == nil {
		t.Fatal("NewKeyValidator returned nil")
	}
	if v.config.MaxKeyLength != 512 {
		t.Errorf("config.MaxKeyLength = %d, want 512", v.config.MaxKeyLength)
	}
	if !v.config.AllowEmpty {
		t.Error("config.AllowEmpty = false, want true")
	}
}

func TestKeyValidator_Validate(t *testing.T) {
	t.Run("valid keys pass validation", func(t *testing.T) {
		v := NewKeyValidator(DefaultKeyValidationConfig())

		validKeys := []string{
			"simple",
			"user:123",
			"cache:key:with:colons",
			"key_with_underscores",
			"key-with-dashes",
			"key.with.dots",
			"MixedCaseKey",
			"key with spaces",
			"key123",
			"a",
			strings.Repeat("a", 1024), // max length
		}

		for _, key := range validKeys {
			if err := v.Validate(key); err != nil {
				t.Errorf("Validate(%q) = %v, want nil", key, err)
			}
		}
	})

	t.Run("empty key rejected by default", func(t *testing.T) {
		v := NewKeyValidator(DefaultKeyValidationConfig())

		err := v.Validate("")
		if err == nil {
			t.Error("Validate(\"\") = nil, want error")
		}
		if !errors.Is(err, ErrInvalidKey) {
			t.Errorf("error should wrap ErrInvalidKey, got: %v", err)
		}
	})

	t.Run("empty key allowed when configured", func(t *testing.T) {
		cfg := DefaultKeyValidationConfig()
		cfg.AllowEmpty = true
		v := NewKeyValidator(cfg)

		if err := v.Validate(""); err != nil {
			t.Errorf("Validate(\"\") = %v, want nil when AllowEmpty=true", err)
		}
	})

	t.Run("key exceeding max length rejected", func(t *testing.T) {
		v := NewKeyValidator(DefaultKeyValidationConfig())

		longKey := strings.Repeat("a", 1025)
		err := v.Validate(longKey)
		if err == nil {
			t.Error("Validate(long key) = nil, want error")
		}
		if !errors.Is(err, ErrInvalidKey) {
			t.Errorf("error should wrap ErrInvalidKey, got: %v", err)
		}
		if !strings.Contains(err.Error(), "exceeds maximum") {
			t.Errorf("error message should mention 'exceeds maximum', got: %v", err)
		}
	})

	t.Run("max length check disabled when zero", func(t *testing.T) {
		cfg := DefaultKeyValidationConfig()
		cfg.MaxKeyLength = 0
		v := NewKeyValidator(cfg)

		longKey := strings.Repeat("a", 10000)
		if err := v.Validate(longKey); err != nil {
			t.Errorf("Validate(long key) = %v, want nil when MaxKeyLength=0", err)
		}
	})

	t.Run("invalid UTF-8 rejected", func(t *testing.T) {
		v := NewKeyValidator(DefaultKeyValidationConfig())

		// Invalid UTF-8 sequence
		invalidUTF8 := string([]byte{0xff, 0xfe, 0xfd})
		err := v.Validate(invalidUTF8)
		if err == nil {
			t.Error("Validate(invalid UTF-8) = nil, want error")
		}
		if !errors.Is(err, ErrInvalidKey) {
			t.Errorf("error should wrap ErrInvalidKey, got: %v", err)
		}
	})

	t.Run("control characters rejected by default", func(t *testing.T) {
		v := NewKeyValidator(DefaultKeyValidationConfig())

		controlChars := []string{
			"key\x00value", // null
			"key\x01value", // SOH
			"key\nvalue",   // newline
			"key\rvalue",   // carriage return
			"key\tvalue",   // tab
			"key\x1bvalue", // escape
			"key\x7fvalue", // DEL
		}

		for _, key := range controlChars {
			err := v.Validate(key)
			if err == nil {
				t.Errorf("Validate(%q) = nil, want error for control char", key)
			}
			if !errors.Is(err, ErrInvalidKey) {
				t.Errorf("error should wrap ErrInvalidKey, got: %v", err)
			}
		}
	})

	t.Run("control characters allowed when configured", func(t *testing.T) {
		cfg := DefaultKeyValidationConfig()
		cfg.AllowControlChars = true
		v := NewKeyValidator(cfg)

		// Tab and newline should now be allowed
		keysWithControlChars := []string{
			"key\tvalue",
			"key\nvalue",
		}

		for _, key := range keysWithControlChars {
			if err := v.Validate(key); err != nil {
				t.Errorf("Validate(%q) = %v, want nil when AllowControlChars=true", key, err)
			}
		}
	})

	t.Run("whitespace allowed by default", func(t *testing.T) {
		v := NewKeyValidator(DefaultKeyValidationConfig())

		keysWithWhitespace := []string{
			"key with spaces",
			"  leading spaces",
			"trailing spaces  ",
			"multiple   spaces",
		}

		for _, key := range keysWithWhitespace {
			if err := v.Validate(key); err != nil {
				t.Errorf("Validate(%q) = %v, want nil (whitespace allowed by default)", key, err)
			}
		}
	})

	t.Run("whitespace rejected when configured", func(t *testing.T) {
		cfg := DefaultKeyValidationConfig()
		cfg.AllowWhitespace = false
		v := NewKeyValidator(cfg)

		keysWithWhitespace := []string{
			"key with spaces",
			"  leading",
			"trailing  ",
		}

		for _, key := range keysWithWhitespace {
			err := v.Validate(key)
			if err == nil {
				t.Errorf("Validate(%q) = nil, want error when AllowWhitespace=false", key)
			}
			if !errors.Is(err, ErrInvalidKey) {
				t.Errorf("error should wrap ErrInvalidKey, got: %v", err)
			}
		}
	})

	t.Run("reserved patterns rejected", func(t *testing.T) {
		cfg := DefaultKeyValidationConfig()
		cfg.ReservedPatterns = []string{"__internal__", "system:", ".."}
		v := NewKeyValidator(cfg)

		reservedKeys := []string{
			"cache:__internal__:data",
			"system:config",
			"path/../escape",
		}

		for _, key := range reservedKeys {
			err := v.Validate(key)
			if err == nil {
				t.Errorf("Validate(%q) = nil, want error for reserved pattern", key)
			}
			if !errors.Is(err, ErrInvalidKey) {
				t.Errorf("error should wrap ErrInvalidKey, got: %v", err)
			}
			if !strings.Contains(err.Error(), "reserved pattern") {
				t.Errorf("error message should mention 'reserved pattern', got: %v", err)
			}
		}
	})

	t.Run("keys without reserved patterns pass", func(t *testing.T) {
		cfg := DefaultKeyValidationConfig()
		cfg.ReservedPatterns = []string{"__internal__", "system:"}
		v := NewKeyValidator(cfg)

		validKeys := []string{
			"user:123",
			"cache:data",
			"normal_key",
		}

		for _, key := range validKeys {
			if err := v.Validate(key); err != nil {
				t.Errorf("Validate(%q) = %v, want nil", key, err)
			}
		}
	})

	t.Run("unicode keys supported", func(t *testing.T) {
		v := NewKeyValidator(DefaultKeyValidationConfig())

		unicodeKeys := []string{
			"用户:123",
			"キー:データ",
			"ключ:значение",
			"مفتاح",
			"🔑:emoji:key",
		}

		for _, key := range unicodeKeys {
			if err := v.Validate(key); err != nil {
				t.Errorf("Validate(%q) = %v, want nil for unicode key", key, err)
			}
		}
	})
}

func TestValidateKey(t *testing.T) {
	t.Run("uses default validator", func(t *testing.T) {
		// Valid key
		if err := ValidateKey("valid:key"); err != nil {
			t.Errorf("ValidateKey(\"valid:key\") = %v, want nil", err)
		}

		// Empty key (should fail with default config)
		if err := ValidateKey(""); err == nil {
			t.Error("ValidateKey(\"\") = nil, want error")
		}

		// Key with control char (should fail with default config)
		if err := ValidateKey("key\x00value"); err == nil {
			t.Error("ValidateKey with null char = nil, want error")
		}
	})
}

func TestIsInvalidKey(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"nil error", nil, false},
		{"other error", errors.New("some error"), false},
		{"direct ErrInvalidKey", ErrInvalidKey, true},
		{"wrapped ErrInvalidKey", NewKeyValidator(DefaultKeyValidationConfig()).Validate(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInvalidKey(tt.err); got != tt.expect {
				t.Errorf("IsInvalidKey() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestDefaultKeyValidator(t *testing.T) {
	if DefaultKeyValidator == nil {
		t.Fatal("DefaultKeyValidator should not be nil")
	}

	// Should use default configuration
	if err := DefaultKeyValidator.Validate("valid:key"); err != nil {
		t.Errorf("DefaultKeyValidator.Validate(valid key) = %v, want nil", err)
	}

	if err := DefaultKeyValidator.Validate(""); err == nil {
		t.Error("DefaultKeyValidator.Validate(\"\") = nil, want error")
	}
}

// BenchmarkKeyValidator_Validate benchmarks key validation.
func BenchmarkKeyValidator_Validate(b *testing.B) {
	v := NewKeyValidator(DefaultKeyValidationConfig())
	key := "user:123:profile:data"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Validate(key)
	}
}

// BenchmarkKeyValidator_ValidateLongKey benchmarks validation of long keys.
func BenchmarkKeyValidator_ValidateLongKey(b *testing.B) {
	v := NewKeyValidator(DefaultKeyValidationConfig())
	key := strings.Repeat("a", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Validate(key)
	}
}

// BenchmarkKeyValidator_ValidateUnicode benchmarks validation of unicode keys.
func BenchmarkKeyValidator_ValidateUnicode(b *testing.B) {
	v := NewKeyValidator(DefaultKeyValidationConfig())
	key := "用户:123:キー:данные"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Validate(key)
	}
}

// BenchmarkKeyValidator_WithReservedPatterns benchmarks validation with reserved patterns.
func BenchmarkKeyValidator_WithReservedPatterns(b *testing.B) {
	cfg := DefaultKeyValidationConfig()
	cfg.ReservedPatterns = []string{"__internal__", "system:", "..", "__proto__"}
	v := NewKeyValidator(cfg)
	key := "user:123:profile:data"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Validate(key)
	}
}
