package cache

import (
	"encoding/json"

	"github.com/darrell-green/rentfree/internal/types"
)

// JSONSerializer implements Serializer using JSON encoding.
type JSONSerializer struct{}

// NewJSONSerializer creates a new JSON serializer.
func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{}
}

// Marshal serializes a value to JSON bytes.
func (s *JSONSerializer) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal deserializes JSON bytes into the destination.
func (s *JSONSerializer) Unmarshal(data []byte, dest any) error {
	return json.Unmarshal(data, dest)
}

var _ types.Serializer = (*JSONSerializer)(nil)
