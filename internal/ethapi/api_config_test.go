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

package ethapi

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/forks"
)

func TestGetPrecompileName(t *testing.T) {
	tests := []struct {
		addr string
		fork forks.Fork
		want string
	}{
		{"0x0000000000000000000000000000000000000001", forks.Cancun, "ECREC"},
		{"0x0000000000000000000000000000000000000002", forks.Cancun, "SHA256"},
		{"0x000000000000000000000000000000000000000a", forks.Cancun, "KZG_POINT_EVALUATION"},
		{"0x000000000000000000000000000000000000000b", forks.Prague, "BLS12_G1ADD"},
		{"0x0000000000000000000000000000000000000100", forks.Osaka, "P256VERIFY"},
	}

	for _, tt := range tests {
		addr := common.HexToAddress(tt.addr)
		got := getPrecompileName(addr, tt.fork)
		if got != tt.want {
			t.Errorf("getPrecompileName(%s, %v) = %s, want %s", tt.addr, tt.fork, got, tt.want)
		}
	}
}

func TestDetermineFork(t *testing.T) {
	// Test with a simple config
	config := params.HoodiChainConfig

	tests := []struct {
		timestamp uint64
		wantFork  forks.Fork
	}{
		{0, forks.Cancun},                          // Genesis
		{1742999831, forks.Cancun},                 // Just before Prague
		{1742999832, forks.Prague},                 // At Prague activation
		{1742999833, forks.Prague},                 // After Prague activation
		{uint64(0xFFFFFFFFFFFFFFFF), forks.Prague}, // Far future
	}

	for _, tt := range tests {
		got := determineFork(config, tt.timestamp)
		if got != tt.wantFork {
			t.Errorf("determineFork(config, %d) = %v, want %v", tt.timestamp, got, tt.wantFork)
		}
	}
}
