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
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func TestGetActivePrecompiles(t *testing.T) {
	// Test Homestead rules (basic precompiles)
	homesteadRules := params.Rules{
		ChainID:  big.NewInt(1),
		IsLondon: false,
		IsBerlin: false,
	}

	homesteadPrecompiles := GetActivePrecompiles(homesteadRules)
	expected := map[string]string{
		"0x0000000000000000000000000000000000000001": "ECREC",
		"0x0000000000000000000000000000000000000002": "SHA256",
		"0x0000000000000000000000000000000000000003": "RIPEMD160",
		"0x0000000000000000000000000000000000000004": "ID",
	}

	if !reflect.DeepEqual(homesteadPrecompiles, expected) {
		t.Errorf("Homestead precompiles mismatch.\nGot: %v\nWant: %v", homesteadPrecompiles, expected)
	}
}

func TestGetActivePrecompilesForFork(t *testing.T) {
	chainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(1000),
		ConstantinopleBlock: big.NewInt(1000),
		PetersburgBlock:     big.NewInt(1000),
		IstanbulBlock:       big.NewInt(2000),
		BerlinBlock:         big.NewInt(3000),
		LondonBlock:         big.NewInt(3000),
		MergeNetsplitBlock:  big.NewInt(3000),
		ShanghaiTime:        &[]uint64{3500}[0],
		CancunTime:          &[]uint64{4000}[0],
		PragueTime:          &[]uint64{5000}[0],
	}

	tests := []struct {
		name        string
		blockNumber uint64
		blockTime   uint64
		expectedLen int
		mustHave    []string
	}{
		{
			name:        "homestead",
			blockNumber: 500,
			blockTime:   500,
			expectedLen: 4,
			mustHave:    []string{"ECREC", "SHA256", "RIPEMD160", "ID"},
		},
		{
			name:        "byzantium",
			blockNumber: 1500,
			blockTime:   1500,
			expectedLen: 8,
			mustHave:    []string{"ECREC", "SHA256", "RIPEMD160", "ID", "MODEXP", "BN254_ADD", "BN254_MUL", "BN254_PAIRING"},
		},
		{
			name:        "istanbul",
			blockNumber: 2500,
			blockTime:   2500,
			expectedLen: 9,
			mustHave:    []string{"ECREC", "SHA256", "RIPEMD160", "ID", "MODEXP", "BN254_ADD", "BN254_MUL", "BN254_PAIRING", "BLAKE2F"},
		},
		{
			name:        "berlin",
			blockNumber: 3500,
			blockTime:   3500,
			expectedLen: 9, // Berlin doesn't add new precompiles
			mustHave:    []string{"ECREC", "SHA256", "RIPEMD160", "ID", "MODEXP", "BN254_ADD", "BN254_MUL", "BN254_PAIRING", "BLAKE2F"},
		},
		{
			name:        "cancun",
			blockNumber: 4500,
			blockTime:   4500,
			expectedLen: 10,
			mustHave:    []string{"ECREC", "SHA256", "RIPEMD160", "ID", "MODEXP", "BN254_ADD", "BN254_MUL", "BN254_PAIRING", "BLAKE2F", "KZG_POINT_EVALUATION"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			precompiles := GetActivePrecompilesForFork(chainConfig, tt.blockNumber, tt.blockTime)

			if len(precompiles) != tt.expectedLen {
				t.Errorf("Expected %d precompiles, got %d", tt.expectedLen, len(precompiles))
			}

			// Check that all required precompiles are present
			found := make(map[string]bool)
			for _, name := range precompiles {
				found[name] = true
			}

			for _, required := range tt.mustHave {
				if !found[required] {
					t.Errorf("Missing required precompile: %s", required)
				}
			}
		})
	}
}

func TestGetPrecompileName(t *testing.T) {
	tests := []struct {
		name     string
		address  common.Address
		expected string
	}{
		{
			name:     "ECREC",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000001"),
			expected: "ECREC",
		},
		{
			name:     "SHA256",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000002"),
			expected: "SHA256",
		},
		{
			name:     "RIPEMD160",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000003"),
			expected: "RIPEMD160",
		},
		{
			name:     "ID",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000004"),
			expected: "ID",
		},
		{
			name:     "MODEXP",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000005"),
			expected: "MODEXP",
		},
		{
			name:     "BN254_ADD",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000006"),
			expected: "BN254_ADD",
		},
		{
			name:     "BN254_MUL",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000007"),
			expected: "BN254_MUL",
		},
		{
			name:     "BN254_PAIRING",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000008"),
			expected: "BN254_PAIRING",
		},
		{
			name:     "BLAKE2F",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000009"),
			expected: "BLAKE2F",
		},
		{
			name:     "KZG_POINT_EVALUATION",
			address:  common.HexToAddress("0x000000000000000000000000000000000000000a"),
			expected: "KZG_POINT_EVALUATION",
		},
		{
			name:     "unknown address",
			address:  common.HexToAddress("0x1000000000000000000000000000000000000000"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPrecompileName(tt.address)
			if result != tt.expected {
				t.Errorf("getPrecompileName(%s) = %q, want %q", tt.address.Hex(), result, tt.expected)
			}
		})
	}
}

func TestIsAddressEqual(t *testing.T) {
	tests := []struct {
		name     string
		fullAddr []byte
		suffix   []byte
		expected bool
	}{
		{
			name:     "single byte match",
			fullAddr: common.HexToAddress("0x0000000000000000000000000000000000000001").Bytes(),
			suffix:   []byte{0x1},
			expected: true,
		},
		{
			name:     "two byte match",
			fullAddr: common.HexToAddress("0x0000000000000000000000000000000000000100").Bytes(),
			suffix:   []byte{0x1, 0x00},
			expected: true,
		},
		{
			name:     "no match",
			fullAddr: common.HexToAddress("0x0000000000000000000000000000000000000001").Bytes(),
			suffix:   []byte{0x2},
			expected: false,
		},
		{
			name:     "non-zero prefix",
			fullAddr: common.HexToAddress("0x1000000000000000000000000000000000000001").Bytes(),
			suffix:   []byte{0x1},
			expected: false,
		},
		{
			name:     "empty suffix",
			fullAddr: common.HexToAddress("0x0000000000000000000000000000000000000001").Bytes(),
			suffix:   []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAddressEqual(tt.fullAddr, tt.suffix)
			if result != tt.expected {
				t.Errorf("isAddressEqual(%v, %v) = %v, want %v", tt.fullAddr, tt.suffix, result, tt.expected)
			}
		})
	}
}
