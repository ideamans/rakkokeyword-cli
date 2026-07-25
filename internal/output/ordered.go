package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Object is a JSON object that remembers the order its keys arrived in.
//
// Column order in table and CSV output comes straight from the API response,
// so it matches the order the API documentation lists the fields in. Decoding
// into a plain map[string]any would throw that away and leave the columns
// shuffling from run to run.
type Object struct {
	Keys   []string
	Values map[string]any
}

// Get returns the value stored under key.
func (o *Object) Get(key string) (any, bool) {
	if o == nil {
		return nil, false
	}
	v, ok := o.Values[key]
	return v, ok
}

func (o *Object) set(key string, value any) {
	if o.Values == nil {
		o.Values = map[string]any{}
	}
	if _, exists := o.Values[key]; !exists {
		o.Keys = append(o.Keys, key)
	}
	o.Values[key] = value
}

// MarshalJSON re-emits the object in its original key order.
func (o *Object) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.Keys {
		if i > 0 {
			b.WriteByte(',')
		}
		key, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		b.Write(key)
		b.WriteByte(':')
		val, err := json.Marshal(o.Values[k])
		if err != nil {
			return nil, err
		}
		b.Write(val)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// Decode parses JSON into ordered objects, json.Number scalars, []any arrays,
// strings, bools and nil.
func Decode(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decodeValue(dec)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return decodeFrom(dec, tok)
}

func decodeFrom(dec *json.Decoder, tok json.Token) (any, error) {
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			obj := &Object{}
			for {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				if d, ok := keyTok.(json.Delim); ok && d == '}' {
					return obj, nil
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, fmt.Errorf("unexpected object key %v", keyTok)
				}
				val, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				obj.set(key, val)
			}
		case '[':
			arr := []any{}
			for {
				elemTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				if d, ok := elemTok.(json.Delim); ok && d == ']' {
					return arr, nil
				}
				val, err := decodeFrom(dec, elemTok)
				if err != nil {
					return nil, err
				}
				arr = append(arr, val)
			}
		}
		return nil, fmt.Errorf("unexpected delimiter %v", t)
	default:
		return tok, nil
	}
}

// Lookup walks a dotted path such as "data.items" through decoded JSON.
func Lookup(root any, path string) (any, bool) {
	if path == "" {
		return root, true
	}
	current := root
	for _, segment := range strings.Split(path, ".") {
		obj, ok := current.(*Object)
		if !ok {
			return nil, false
		}
		current, ok = obj.Get(segment)
		if !ok {
			return nil, false
		}
	}
	return current, true
}
