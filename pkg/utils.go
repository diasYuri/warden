package warden

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// generateID generates a unique identifier
func generateID() string {
	return uuid.New().String()
}

// generateShortID generates a short ID using only random bytes
func generateShortID() string {
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	return hex.EncodeToString(randomBytes)
}

// generateExecutionID generates a unique execution identifier
func generateExecutionID(instanceID string) string {
	return fmt.Sprintf("%s_%s", instanceID, generateShortID())
}

// validateStateID validates a state ID
func validateStateID(stateID string) error {
	if stateID == "" {
		return fmt.Errorf("state ID cannot be empty")
	}

	if len(stateID) > 100 {
		return fmt.Errorf("state ID too long (max 100 characters)")
	}

	// Must start with a letter or underscore
	if !isValidIdentifierStart(rune(stateID[0])) {
		return fmt.Errorf("state ID must start with a letter or underscore")
	}

	// Must contain only valid identifier characters
	for _, r := range stateID {
		if !isValidIdentifierChar(r) {
			return fmt.Errorf("state ID contains invalid character: %c", r)
		}
	}

	return nil
}

// isValidIdentifierStart checks if a rune is valid for starting an identifier
func isValidIdentifierStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

// isValidIdentifierChar checks if a rune is valid for an identifier
func isValidIdentifierChar(r rune) bool {
	return isValidIdentifierStart(r) || (r >= '0' && r <= '9') || r == '-'
}

// validateInstanceID validates an instance ID
func validateInstanceID(instanceID string) error {
	if instanceID == "" {
		return fmt.Errorf("instance ID cannot be empty")
	}

	if len(instanceID) > 255 {
		return fmt.Errorf("instance ID too long (max 255 characters)")
	}

	// Checks for invalid characters
	if strings.ContainsAny(instanceID, " \t\n\r") {
		return fmt.Errorf("instance ID cannot contain whitespace")
	}

	return nil
}

// validateExecutionID validates an execution ID
func validateExecutionID(executionID string) error {
	if executionID == "" {
		return fmt.Errorf("execution ID cannot be empty")
	}
	return nil
}

// copyMap creates a deep copy of a map
func copyMap(original map[string]interface{}) map[string]interface{} {
	if original == nil {
		return nil
	}

	result := make(map[string]interface{})
	for k, v := range original {
		result[k] = copyValue(v)
	}
	return result
}

// copyValue creates a deep copy of a value
func copyValue(original interface{}) interface{} {
	if original == nil {
		return nil
	}

	switch v := original.(type) {
	case map[string]interface{}:
		return copyMap(v)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = copyValue(item)
		}
		return result
	case string, int, int32, int64, float32, float64, bool:
		return v
	default:
		// For other types, returns as is (may not be a deep copy)
		return v
	}
}

// mergeMap merges two maps, with the second map taking precedence
func mergeMap(base, overlay map[string]interface{}) map[string]interface{} {
	if base == nil && overlay == nil {
		return make(map[string]interface{})
	}

	if base == nil {
		return copyMap(overlay)
	}

	if overlay == nil {
		return copyMap(base)
	}

	result := copyMap(base)
	for k, v := range overlay {
		result[k] = copyValue(v)
	}

	return result
}

// containsString checks if a slice contains a string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// removeString removes a string from a slice
func removeString(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// uniqueStrings returns unique strings from a slice
func uniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// isEmptyMap checks if a map is empty or nil
func isEmptyMap(m map[string]interface{}) bool {
	return m == nil || len(m) == 0
}

// keysFromMap extracts keys from a map
func keysFromMap(m map[string]interface{}) []string {
	if m == nil {
		return []string{}
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

// valuesFromMap extracts values from a map
func valuesFromMap(m map[string]interface{}) []interface{} {
	if m == nil {
		return []interface{}{}
	}

	values := make([]interface{}, 0, len(m))
	for _, v := range m {
		values = append(values, copyValue(v))
	}

	return values
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000.0)
	}

	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000.0)
	}

	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}

	return d.String()
}

// getCurrentTimestamp returns the current timestamp
func getCurrentTimestamp() time.Time {
	return time.Now().UTC()
}

// isValidTransitionType checks if a transition type is valid
func isValidTransitionType(transitionType TransitionType) bool {
	switch transitionType {
	case TransitionOnSuccess, TransitionOnFailure, TransitionOnTimeout, TransitionOnCustom:
		return true
	default:
		return false
	}
}

// isValidStateType checks if a state type is valid
func isValidStateType(stateType StateType) bool {
	switch stateType {
	case StateTypeStart, StateTypeTask, StateTypeDecision,
		StateTypeParallel, StateTypeJoin, StateTypeEnd:
		return true
	default:
		return false
	}
}

// isValidInstanceStatus checks if an instance status is valid
func isValidInstanceStatus(status InstanceStatus) bool {
	switch status {
	case InstanceStatusCreated, InstanceStatusRunning, InstanceStatusCompleted,
		InstanceStatusFailed, InstanceStatusSuspended:
		return true
	default:
		return false
	}
}

// SafeString converts an interface{} to a safe string
func SafeString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// SafeInt64 converts an interface{} to int64 safely
func SafeInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}

	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		// Poderia usar strconv.ParseInt, mas por simplicidade retorna 0
		return 0
	default:
		return 0
	}
}

// SafeBool converts an interface{} to bool safely
func SafeBool(v interface{}) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
