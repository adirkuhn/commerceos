package events

import "encoding/json"

// marshalWithEvent encodes v (a struct) with an extra "event" field.
func marshalWithEvent(subject string, v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	name, err := json.Marshal(subject)
	if err != nil {
		return nil, err
	}
	fields["event"] = name
	return json.Marshal(fields)
}
