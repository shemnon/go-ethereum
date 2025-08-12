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

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// ============================================================================
// EIP-7910 Configuration Tests
// ============================================================================

func TestBuildForkConfig(t *testing.T) {
	chainConfig := params.MainnetChainConfig

	// Create a proper genesis block with header
	header := &types.Header{
		Time: 0,
	}
	genesis := types.NewBlockWithHeader(header)

	// Use a post-merge timestamp (after Shanghai)
	config, err := BuildForkConfig(chainConfig, genesis, 20000000, 1681338479) // Post-Shanghai
	if err != nil {
		t.Fatalf("BuildForkConfig failed: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config, got nil")
	}

	// Verify basic structure
	if config.ChainID != "0x1" {
		t.Errorf("Expected chainId '0x1', got %s", config.ChainID)
	}

	if config.ActivationTime == 0 {
		t.Error("ActivationTime should be set")
	}

	if config.ForkID == "" {
		t.Error("ForkID should be set")
	}

	if len(config.ForkID) < 10 { // "0x" + at least 8 hex chars
		t.Errorf("ForkID seems too short: %s", config.ForkID)
	}
}

func TestPrecompileFormat(t *testing.T) {
	chainConfig := params.MainnetChainConfig
	header := &types.Header{Time: 0}
	genesis := types.NewBlockWithHeader(header)

	config, err := BuildForkConfig(chainConfig, genesis, 20000000, 1681338479) // Post-Shanghai
	if err != nil {
		t.Fatalf("BuildForkConfig failed: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config, got nil")
	}

	// Verify precompiles are in name -> address format (not address -> name)
	for name, address := range config.Precompiles {
		// Name should not be a hex address
		if len(name) == 42 && name[:2] == "0x" {
			t.Errorf("Precompile key should be name, not address. Got: %s -> %s", name, address)
		}

		// Address should be a valid hex address
		if len(address) != 42 || address[:2] != "0x" {
			t.Errorf("Precompile value should be hex address. Got: %s -> %s", name, address)
		}

		// Check some known precompiles
		switch name {
		case "ECREC":
			if address != "0x0000000000000000000000000000000000000001" {
				t.Errorf("Expected ECREC at 0x01, got %s", address)
			}
		case "SHA256":
			if address != "0x0000000000000000000000000000000000000002" {
				t.Errorf("Expected SHA256 at 0x02, got %s", address)
			}
		}
	}
}

func TestSystemContracts(t *testing.T) {
	chainConfig := params.MainnetChainConfig
	header := &types.Header{Time: 0}
	genesis := types.NewBlockWithHeader(header)

	// Test with a post-Cancun timestamp to get beacon roots address
	config, err := BuildForkConfig(chainConfig, genesis, 20000000, 1710338479) // Post-Cancun
	if err != nil {
		t.Fatalf("BuildForkConfig failed: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config, got nil")
	}

	// Should have beacon roots address for Cancun
	if _, exists := config.SystemContracts["BEACON_ROOTS_ADDRESS"]; !exists {
		t.Error("Expected BEACON_ROOTS_ADDRESS in system contracts for Cancun fork")
	}
}

func TestBlobSchedule(t *testing.T) {
	chainConfig := params.MainnetChainConfig
	header := &types.Header{Time: 0}
	genesis := types.NewBlockWithHeader(header)

	// Test pre-Cancun (no blobs)
	configPreCancun, err := BuildForkConfig(chainConfig, genesis, 18000000, 1700000000) // Post-Shanghai, Pre-Cancun
	if err != nil {
		t.Fatalf("BuildForkConfig failed: %v", err)
	}

	if configPreCancun != nil {
		if configPreCancun.BlobSchedule.Max != 0 {
			t.Errorf("Expected blob max 0 pre-Cancun, got %d", configPreCancun.BlobSchedule.Max)
		}
	}

	// Test post-Cancun (with blobs)
	configPostCancun, err := BuildForkConfig(chainConfig, genesis, 20000000, 1720000000) // Post-Cancun
	if err != nil {
		t.Fatalf("BuildForkConfig failed: %v", err)
	}

	if configPostCancun == nil {
		t.Fatal("Expected config, got nil")
	}

	if configPostCancun.BlobSchedule.Max == 0 {
		t.Error("Expected non-zero blob max post-Cancun")
	}

	if configPostCancun.BlobSchedule.Target == 0 {
		t.Error("Expected non-zero blob target post-Cancun")
	}
}

func TestActivationTimeCalculation(t *testing.T) {
	tests := []struct {
		name        string
		chainConfig *params.ChainConfig
		targetTime  uint64
		expectError bool
	}{
		{
			name:        "mainnet_cancun",
			chainConfig: params.MainnetChainConfig,
			targetTime:  1720000000, // Post-Cancun
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activationTime, err := calculateActivationTimeForFork(tt.chainConfig, tt.targetTime)
			if tt.expectError && err == nil {
				t.Error("Expected error, got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && activationTime == 0 {
				t.Error("Expected non-zero activation time")
			}
		})
	}
}
