package server

import "encoding/json/v2"

// marshalJSON encodes Fiber JSON responses with encoding/json/v2. Deterministic
// output keeps map keys (fiber.Map) in a stable order, as encoding/json did.
func marshalJSON(v any) ([]byte, error) {
	return json.Marshal(v, json.Deterministic(true))
}

// unmarshalJSON decodes Fiber JSON request bodies with encoding/json/v2.
// Field names match case-sensitively.
func unmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
