package types

import "encoding/json"

// SecretString is a string type that redacts its value when marshaled to JSON
// or converted to a string. This prevents accidental leakage of sensitive values
// like passwords in logs, error messages, or config dumps.
type SecretString struct {
	value string
}

// NewSecretString creates a new SecretString with the provided value.
func NewSecretString(value string) SecretString {
	return SecretString{value: value}
}

// Value returns the underlying secret value.
func (s SecretString) Value() string {
	return s.value
}

func (s SecretString) String() string {
	if s.value == "" {
		return ""
	}
	return "[REDACTED]"
}

// MarshalJSON marshals the secret string as "[REDACTED]" to prevent leakage.
func (s SecretString) MarshalJSON() ([]byte, error) {
	if s.value == "" {
		return json.Marshal("")
	}
	return json.Marshal("[REDACTED]")
}

// UnmarshalJSON unmarshals a JSON value into the secret string.
func (s *SecretString) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	s.value = value
	return nil
}

// IsEmpty returns true if the secret string has no value.
func (s SecretString) IsEmpty() bool {
	return s.value == ""
}
