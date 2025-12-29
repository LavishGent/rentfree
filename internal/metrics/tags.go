package metrics

import "fmt"

// Tag creates a formatted DataDog tag string in "key:value" format.
func Tag(key, value string) string {
	return fmt.Sprintf("%s:%s", key, value)
}

// LevelTag creates a cache level tag.
func LevelTag(level string) string {
	return Tag("level", level)
}

// OperationTag creates an operation tag.
func OperationTag(op string) string {
	return Tag("operation", op)
}

// PatternTag creates a pattern tag for bulk operations.
func PatternTag(pattern string) string {
	return Tag("pattern", pattern)
}

// StatusTag creates a status tag (hit/miss/error).
func StatusTag(status string) string {
	return Tag("status", status)
}

// LayerTag creates a cache layer tag (memory/redis).
func LayerTag(layer string) string {
	return Tag("layer", layer)
}

// CircuitStateTag creates a circuit breaker state tag.
func CircuitStateTag(state string) string {
	return Tag("circuit_state", state)
}
