package cache

import (
	"testing"
)

func TestNewJSONSerializer(t *testing.T) {
	s := NewJSONSerializer()
	if s == nil {
		t.Fatal("NewJSONSerializer() returned nil")
	}
}

func TestJSONSerializerMarshal(t *testing.T) {
	s := NewJSONSerializer()

	t.Run("marshals struct", func(t *testing.T) {
		//nolint:govet // Test struct - alignment not critical
		type User struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		user := User{ID: 1, Name: "Test"}

		data, err := s.Marshal(user)
		if err != nil {
			t.Errorf("Marshal() error = %v", err)
		}

		expected := `{"id":1,"name":"Test"}`
		if string(data) != expected {
			t.Errorf("Marshal() = %s, want %s", string(data), expected)
		}
	})

	t.Run("marshals map", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2}

		data, err := s.Marshal(m)
		if err != nil {
			t.Errorf("Marshal() error = %v", err)
		}

		if len(data) == 0 {
			t.Error("Marshal() returned empty data")
		}
	})

	t.Run("marshals slice", func(t *testing.T) {
		slice := []string{"one", "two", "three"}

		data, err := s.Marshal(slice)
		if err != nil {
			t.Errorf("Marshal() error = %v", err)
		}

		expected := `["one","two","three"]`
		if string(data) != expected {
			t.Errorf("Marshal() = %s, want %s", string(data), expected)
		}
	})

	t.Run("marshals primitives", func(t *testing.T) {
		// String
		data, err := s.Marshal("hello")
		if err != nil {
			t.Errorf("Marshal(string) error = %v", err)
		}
		if string(data) != `"hello"` {
			t.Errorf("Marshal(string) = %s, want \"hello\"", string(data))
		}

		// Int
		data, err = s.Marshal(42)
		if err != nil {
			t.Errorf("Marshal(int) error = %v", err)
		}
		if string(data) != "42" {
			t.Errorf("Marshal(int) = %s, want 42", string(data))
		}

		// Bool
		data, err = s.Marshal(true)
		if err != nil {
			t.Errorf("Marshal(bool) error = %v", err)
		}
		if string(data) != "true" {
			t.Errorf("Marshal(bool) = %s, want true", string(data))
		}
	})

	t.Run("marshals nil", func(t *testing.T) {
		data, err := s.Marshal(nil)
		if err != nil {
			t.Errorf("Marshal(nil) error = %v", err)
		}
		if string(data) != "null" {
			t.Errorf("Marshal(nil) = %s, want null", string(data))
		}
	})

	t.Run("returns error for invalid value", func(t *testing.T) {
		// Channels can't be marshaled to JSON
		ch := make(chan int)
		_, err := s.Marshal(ch)
		if err == nil {
			t.Error("Marshal(chan) should return error")
		}
	})
}

func TestJSONSerializerUnmarshal(t *testing.T) {
	s := NewJSONSerializer()

	t.Run("unmarshals to struct", func(t *testing.T) {
		//nolint:govet // Test struct - alignment not critical
		type User struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		data := []byte(`{"id":1,"name":"Test"}`)
		var user User

		err := s.Unmarshal(data, &user)
		if err != nil {
			t.Errorf("Unmarshal() error = %v", err)
		}
		if user.ID != 1 {
			t.Errorf("ID = %d, want 1", user.ID)
		}
		if user.Name != "Test" {
			t.Errorf("Name = %s, want Test", user.Name)
		}
	})

	t.Run("unmarshals to map", func(t *testing.T) {
		data := []byte(`{"a":1,"b":2}`)
		var m map[string]int

		err := s.Unmarshal(data, &m)
		if err != nil {
			t.Errorf("Unmarshal() error = %v", err)
		}
		if m["a"] != 1 {
			t.Errorf("m[a] = %d, want 1", m["a"])
		}
		if m["b"] != 2 {
			t.Errorf("m[b] = %d, want 2", m["b"])
		}
	})

	t.Run("unmarshals to slice", func(t *testing.T) {
		data := []byte(`["one","two","three"]`)
		var slice []string

		err := s.Unmarshal(data, &slice)
		if err != nil {
			t.Errorf("Unmarshal() error = %v", err)
		}
		if len(slice) != 3 {
			t.Errorf("len(slice) = %d, want 3", len(slice))
		}
		if slice[0] != "one" {
			t.Errorf("slice[0] = %s, want one", slice[0])
		}
	})

	t.Run("unmarshals primitives", func(t *testing.T) {
		// String
		var s1 string
		err := s.Unmarshal([]byte(`"hello"`), &s1)
		if err != nil {
			t.Errorf("Unmarshal(string) error = %v", err)
		}
		if s1 != "hello" {
			t.Errorf("Unmarshal(string) = %s, want hello", s1)
		}

		// Int
		var i int
		err = s.Unmarshal([]byte(`42`), &i)
		if err != nil {
			t.Errorf("Unmarshal(int) error = %v", err)
		}
		if i != 42 {
			t.Errorf("Unmarshal(int) = %d, want 42", i)
		}

		// Bool
		var b bool
		err = s.Unmarshal([]byte(`true`), &b)
		if err != nil {
			t.Errorf("Unmarshal(bool) error = %v", err)
		}
		if !b {
			t.Errorf("Unmarshal(bool) = %v, want true", b)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		var m map[string]string
		err := s.Unmarshal([]byte(`not valid json`), &m)
		if err == nil {
			t.Error("Unmarshal(invalid) should return error")
		}
	})

	t.Run("returns error for type mismatch", func(t *testing.T) {
		var i int
		err := s.Unmarshal([]byte(`"not a number"`), &i)
		if err == nil {
			t.Error("Unmarshal(type mismatch) should return error")
		}
	})
}

func TestJSONSerializerRoundTrip(t *testing.T) {
	s := NewJSONSerializer()

	//nolint:govet // Test struct - alignment not critical
	type ComplexType struct {
		ID       int               `json:"id"`
		Name     string            `json:"name"`
		Tags     []string          `json:"tags"`
		Metadata map[string]string `json:"metadata"`
		Nested   struct {
			Value int `json:"value"`
		} `json:"nested"`
	}

	original := ComplexType{
		ID:       42,
		Name:     "test",
		Tags:     []string{"tag1", "tag2"},
		Metadata: map[string]string{"key": "value"},
	}
	original.Nested.Value = 100

	// Marshal
	data, err := s.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal
	var result ComplexType
	err = s.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify
	if result.ID != original.ID {
		t.Errorf("ID = %d, want %d", result.ID, original.ID)
	}
	if result.Name != original.Name {
		t.Errorf("Name = %s, want %s", result.Name, original.Name)
	}
	if len(result.Tags) != len(original.Tags) {
		t.Errorf("len(Tags) = %d, want %d", len(result.Tags), len(original.Tags))
	}
	if result.Metadata["key"] != original.Metadata["key"] {
		t.Errorf("Metadata[key] = %s, want %s", result.Metadata["key"], original.Metadata["key"])
	}
	if result.Nested.Value != original.Nested.Value {
		t.Errorf("Nested.Value = %d, want %d", result.Nested.Value, original.Nested.Value)
	}
}
