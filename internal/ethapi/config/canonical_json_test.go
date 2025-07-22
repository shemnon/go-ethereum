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
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestCanonicalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: `"hello"`,
		},
		{
			name:     "integer",
			input:    42,
			expected: `42`,
		},
		{
			name:     "boolean true",
			input:    true,
			expected: `true`,
		},
		{
			name:     "boolean false",
			input:    false,
			expected: `false`,
		},
		{
			name:     "null",
			input:    nil,
			expected: `null`,
		},
		{
			name:     "empty object",
			input:    map[string]interface{}{},
			expected: `{}`,
		},
		{
			name: "simple object",
			input: map[string]interface{}{
				"b": 2,
				"a": 1,
			},
			expected: `{"a":1,"b":2}`,
		},
		{
			name: "nested object",
			input: map[string]interface{}{
				"z": map[string]interface{}{
					"y": 2,
					"x": 1,
				},
				"a": 3,
			},
			expected: `{"a":3,"z":{"x":1,"y":2}}`,
		},
		{
			name:     "array",
			input:    []interface{}{3, 1, 2},
			expected: `[3,1,2]`,
		},
		{
			name: "complex structure",
			input: map[string]interface{}{
				"precompiles": map[string]string{
					"0x0000000000000000000000000000000000000002": "SHA256",
					"0x0000000000000000000000000000000000000001": "ECREC",
				},
				"chainId":        "0x1",
				"activationTime": 0,
			},
			expected: `{"activationTime":0,"chainId":"0x1","precompiles":{"0x0000000000000000000000000000000000000001":"ECREC","0x0000000000000000000000000000000000000002":"SHA256"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CanonicalJSON(tt.input)
			if err != nil {
				t.Fatalf("CanonicalJSON failed: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("CanonicalJSON() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

func TestCanonicalJSONStruct(t *testing.T) {
	type TestStruct struct {
		Z string `json:"z"`
		A int    `json:"a"`
		B string `json:"b,omitempty"`
	}

	tests := []struct {
		name     string
		input    TestStruct
		expected string
	}{
		{
			name:     "struct with all fields",
			input:    TestStruct{Z: "last", A: 1, B: "middle"},
			expected: `{"a":1,"b":"middle","z":"last"}`,
		},
		{
			name:     "struct with omitempty field empty",
			input:    TestStruct{Z: "last", A: 1, B: ""},
			expected: `{"a":1,"z":"last"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CanonicalJSON(tt.input)
			if err != nil {
				t.Fatalf("CanonicalJSON failed: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("CanonicalJSON() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

func TestCanonicalJSONForkConfig(t *testing.T) {
	config := &ForkConfig{
		ActivationTime: 1234567890,
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 3338477,
			Max:                   6,
			Target:                3,
		},
		ChainID: "0x1",
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000002": "SHA256",
			"0x0000000000000000000000000000000000000001": "ECREC",
		},
		SystemContracts: map[string]common.Address{
			"BEACON_ROOTS_ADDRESS": common.HexToAddress("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02"),
		},
	}

	result, err := CanonicalJSON(config)
	if err != nil {
		t.Fatalf("CanonicalJSON failed: %v", err)
	}

	// Verify that the JSON is properly ordered and formatted
	expected := `{"activationTime":1234567890,"blobSchedule":{"baseFeeUpdateFraction":3338477,"max":6,"target":3},"chainId":"0x1","precompiles":{"0x0000000000000000000000000000000000000001":"ECREC","0x0000000000000000000000000000000000000002":"SHA256"},"systemContracts":{"BEACON_ROOTS_ADDRESS":"0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02"}}`

	if string(result) != expected {
		t.Errorf("CanonicalJSON() for ForkConfig:\ngot:  %s\nwant: %s", string(result), expected)
	}
}

func TestCanonicalJSONConsistency(t *testing.T) {
	// Test that the same input always produces the same output
	config := map[string]interface{}{
		"c": 3,
		"a": 1,
		"b": 2,
	}

	result1, err := CanonicalJSON(config)
	if err != nil {
		t.Fatalf("First CanonicalJSON failed: %v", err)
	}

	result2, err := CanonicalJSON(config)
	if err != nil {
		t.Fatalf("Second CanonicalJSON failed: %v", err)
	}

	if string(result1) != string(result2) {
		t.Errorf("CanonicalJSON results differ: %s != %s", string(result1), string(result2))
	}
}
