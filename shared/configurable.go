package shared

import (
	"context"
	"fmt"
)

// Configurable is an optional contract a Module implements to expose its
// per-guild settings to the web dashboard. The web layer type-asserts each
// registered Module to Configurable; modules that don't implement it simply
// aren't editable in the dashboard. It is deliberately separate from Module so
// adding it is purely additive — existing modules keep compiling untouched.
type Configurable interface {
	// ConfigSchema describes the editable fields and their types/bounds.
	ConfigSchema() ConfigSchema
	// GetConfig returns the guild's current values keyed by field Key, using the
	// JSON-friendly representation documented on FieldType.
	GetConfig(ctx context.Context, guildID int64) (map[string]any, error)
	// SetConfig applies a partial patch (only the provided keys). Implementations
	// should validate with ConfigSchema.Normalize before persisting.
	SetConfig(ctx context.Context, guildID int64, patch map[string]any) error
}

// FieldType is the kind of a configurable field. The JSON representation the
// dashboard exchanges for each type:
//
//	FieldBool    -> bool
//	FieldInt     -> number (must stay within JS-safe integer range)
//	FieldString  -> string
//	FieldChannel -> string  (Discord snowflake, "" to clear — string avoids
//	FieldRole    -> string   the 2^53 precision loss of a JS number)
type FieldType string

const (
	FieldBool    FieldType = "bool"
	FieldInt     FieldType = "int"
	FieldString  FieldType = "string"
	FieldChannel FieldType = "channel"
	FieldRole    FieldType = "role"
)

// Field is one editable setting.
type Field struct {
	Key   string    `json:"key"`
	Label string    `json:"label"`
	Help  string    `json:"help,omitempty"`
	Type  FieldType `json:"type"`
	// Min and Max bound a FieldInt. Both zero means unbounded.
	Min int `json:"min,omitempty"`
	Max int `json:"max,omitempty"`
	// MaxLen caps a FieldString (0 = unbounded).
	MaxLen int `json:"maxLen,omitempty"`
}

// ConfigSchema is a module's editable surface for the dashboard.
type ConfigSchema struct {
	Module string  `json:"module"`
	Title  string  `json:"title"`
	Fields []Field `json:"fields"`
}

// field returns the schema field for key.
func (s ConfigSchema) field(key string) (Field, bool) {
	for _, f := range s.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return Field{}, false
}

// Normalize validates patch against the schema and returns a new map with each
// provided value coerced to its canonical Go type (bool, int or string).
// Unknown keys, wrong types and out-of-bounds values are rejected. JSON numbers
// arrive as float64, so FieldInt values are accepted as float64/int and
// returned as int.
func (s ConfigSchema) Normalize(patch map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(patch))
	for key, raw := range patch {
		f, ok := s.field(key)
		if !ok {
			return nil, fmt.Errorf("unknown field %q", key)
		}
		v, err := normalizeField(f, raw)
		if err != nil {
			return nil, err
		}
		out[key] = v
	}
	return out, nil
}

func normalizeField(f Field, raw any) (any, error) {
	switch f.Type {
	case FieldBool:
		b, ok := raw.(bool)
		if !ok {
			return nil, fmt.Errorf("field %q must be a boolean", f.Key)
		}
		return b, nil

	case FieldInt:
		n, err := toInt(raw)
		if err != nil {
			return nil, fmt.Errorf("field %q must be a whole number", f.Key)
		}
		if !(f.Min == 0 && f.Max == 0) {
			if n < f.Min || n > f.Max {
				return nil, fmt.Errorf("field %q must be between %d and %d", f.Key, f.Min, f.Max)
			}
		}
		return n, nil

	case FieldString:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("field %q must be a string", f.Key)
		}
		if f.MaxLen > 0 && len([]rune(s)) > f.MaxLen {
			return nil, fmt.Errorf("field %q is too long (max %d characters)", f.Key, f.MaxLen)
		}
		return s, nil

	case FieldChannel, FieldRole:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("field %q must be a snowflake string", f.Key)
		}
		if !isSnowflakeOrEmpty(s) {
			return nil, fmt.Errorf("field %q must be a Discord ID or empty", f.Key)
		}
		return s, nil

	default:
		return nil, fmt.Errorf("field %q has an unknown type %q", f.Key, f.Type)
	}
}

// toInt accepts the float64 JSON numbers decode to, as well as int variants.
func toInt(raw any) (int, error) {
	switch v := raw.(type) {
	case float64:
		if v != float64(int(v)) {
			return 0, fmt.Errorf("not a whole number")
		}
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("not a number")
	}
}

// isSnowflakeOrEmpty reports whether s is empty (clear) or all ASCII digits.
func isSnowflakeOrEmpty(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
