package types

import "encoding/json"

// SecretString is a string type that redacts its value when marshaled to JSON
// or converted to a string. This prevents accidental leakage of sensitive values
// like passwords in logs, error messages, or config dumps.
type SecretString struct {
	value string
}

func NewSecretString(value string) SecretString {
	return SecretString{value: value}
}

func (s SecretString) Value() string {
	return s.value
}

func (s SecretString) String() string {
	if s.value == "" {
		return ""
	}
	return "[REDACTED]"
}

func (s SecretString) MarshalJSON() ([]byte, error) {
	if s.value == "" {
		return json.Marshal("")
	}
	return json.Marshal("[REDACTED]")
}

func (s *SecretString) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	s.value = value
	return nil
}

func (s SecretString) IsEmpty() bool {
	return s.value == ""
}
