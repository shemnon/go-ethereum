// Copyright 2025 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// CanonicalJSON produces canonical JSON as per RFC-8785 for deterministic hashing.
// This implementation ensures:
// - No whitespace except inside strings
// - Object keys sorted lexicographically
// - Numeric values in simplest form
// - No trailing zeros after decimal point
func CanonicalJSON(v interface{}) ([]byte, error) {
	return canonicalMarshal(reflect.ValueOf(v))
}

// canonicalMarshal recursively marshals a value to canonical JSON
func canonicalMarshal(v reflect.Value) ([]byte, error) {
	// Handle invalid values first
	if !v.IsValid() {
		return []byte("null"), nil
	}

	// Special handling for common.Address type
	if v.Type() == reflect.TypeOf(common.Address{}) {
		addr := v.Interface().(common.Address)
		return json.Marshal(addr.Hex())
	}

	switch v.Kind() {
	case reflect.Invalid:
		return []byte("null"), nil
	case reflect.Bool:
		if v.Bool() {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []byte(strconv.FormatInt(v.Int(), 10)), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []byte(strconv.FormatUint(v.Uint(), 10)), nil
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		// Use the simplest form - remove trailing zeros
		s := strconv.FormatFloat(f, 'f', -1, 64)
		return []byte(s), nil
	case reflect.String:
		return json.Marshal(v.String())
	case reflect.Array, reflect.Slice:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return []byte("null"), nil
		}
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				buf.WriteByte(',')
			}
			elemData, err := canonicalMarshal(v.Index(i))
			if err != nil {
				return nil, err
			}
			buf.Write(elemData)
		}
		buf.WriteByte(']')
		return buf.Bytes(), nil
	case reflect.Map:
		if v.IsNil() {
			return []byte("null"), nil
		}
		return canonicalMarshalMap(v)
	case reflect.Struct:
		return canonicalMarshalStruct(v)
	case reflect.Ptr:
		if v.IsNil() {
			return []byte("null"), nil
		}
		return canonicalMarshal(v.Elem())
	case reflect.Interface:
		if v.IsNil() {
			return []byte("null"), nil
		}
		return canonicalMarshal(v.Elem())
	default:
		return nil, fmt.Errorf("unsupported type: %v", v.Type())
	}
}

// canonicalMarshalMap marshals a map with keys sorted lexicographically
func canonicalMarshalMap(v reflect.Value) ([]byte, error) {
	keys := v.MapKeys()
	if len(keys) == 0 {
		return []byte("{}"), nil
	}

	// Convert keys to strings and sort them
	keyStrings := make([]string, len(keys))
	keyMap := make(map[string]reflect.Value)

	for i, key := range keys {
		var keyStr string
		switch key.Kind() {
		case reflect.String:
			keyStr = key.String()
		default:
			keyData, err := canonicalMarshal(key)
			if err != nil {
				return nil, err
			}
			keyStr = string(keyData)
			// Remove quotes for non-string keys when using as map key
			if strings.HasPrefix(keyStr, `"`) && strings.HasSuffix(keyStr, `"`) {
				keyStr = keyStr[1 : len(keyStr)-1]
			}
		}
		keyStrings[i] = keyStr
		keyMap[keyStr] = key
	}

	sort.Strings(keyStrings)

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, keyStr := range keyStrings {
		if i > 0 {
			buf.WriteByte(',')
		}

		// Marshal the key as a JSON string
		keyJSON, err := json.Marshal(keyStr)
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		buf.WriteByte(':')

		// Marshal the value
		value := v.MapIndex(keyMap[keyStr])
		valueData, err := canonicalMarshal(value)
		if err != nil {
			return nil, err
		}
		buf.Write(valueData)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// canonicalMarshalStruct marshals a struct with JSON tags, sorted by field names
func canonicalMarshalStruct(v reflect.Value) ([]byte, error) {
	t := v.Type()

	type field struct {
		name      string
		value     reflect.Value
		omitEmpty bool
	}

	var fields []field

	for i := 0; i < v.NumField(); i++ {
		fieldValue := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		// Get JSON tag
		tag := fieldType.Tag.Get("json")
		if tag == "-" {
			continue
		}

		name := fieldType.Name
		omitEmpty := false

		if tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" {
				name = parts[0]
			}
			for _, part := range parts[1:] {
				if part == "omitempty" {
					omitEmpty = true
					break
				}
			}
		}

		// Skip if omitempty and value is empty
		if omitEmpty && isEmptyValue(fieldValue) {
			continue
		}

		fields = append(fields, field{
			name:      name,
			value:     fieldValue,
			omitEmpty: omitEmpty,
		})
	}

	// Sort fields by name
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].name < fields[j].name
	})

	if len(fields) == 0 {
		return []byte("{}"), nil
	}

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, field := range fields {
		if i > 0 {
			buf.WriteByte(',')
		}

		// Marshal field name
		nameJSON, err := json.Marshal(field.name)
		if err != nil {
			return nil, err
		}
		buf.Write(nameJSON)
		buf.WriteByte(':')

		// Marshal field value
		valueData, err := canonicalMarshal(field.value)
		if err != nil {
			return nil, err
		}
		buf.Write(valueData)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// isEmptyValue checks if a value is considered empty for omitempty
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	default:
		return true // if we don't know what it is, don't jsonify it
	}
}
