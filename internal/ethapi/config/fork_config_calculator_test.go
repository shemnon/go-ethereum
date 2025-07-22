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
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

// mockBackend implements Backend interface for testing
type mockBackend struct {
	chainConfig   *params.ChainConfig
	currentHeader *types.Header
	genesisError  bool // If true, simulate pruned node by failing genesis block requests
}

func (m *mockBackend) ChainConfig() *params.ChainConfig {
	return m.chainConfig
}

func (m *mockBackend) CurrentHeader() *types.Header {
	return m.currentHeader
}

func (m *mockBackend) BlockByNumber(_ context.Context, number rpc.BlockNumber) (*types.Block, error) {
	if number == 0 && m.genesisError {
		return nil, errors.New("pruned history unavailable")
	}

	// For testing, return a mock block
	header := &types.Header{
		Number: big.NewInt(int64(number)),
		Time:   1234567890,
	}
	return types.NewBlock(header, &types.Body{}, nil, nil), nil
}

func TestForkConfigCalculator_PrunedNode(t *testing.T) {
	// Test configuration for Cancun fork
	chainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		MergeNetsplitBlock:  big.NewInt(0),
		ShanghaiTime:        &[]uint64{0}[0],
		CancunTime:          &[]uint64{1000}[0],
	}

	currentHeader := &types.Header{
		Number: big.NewInt(2000),
		Time:   2000, // After Cancun activation
	}

	// Test with pruned node (genesis block unavailable)
	backend := &mockBackend{
		chainConfig:   chainConfig,
		currentHeader: currentHeader,
		genesisError:  true, // Simulate pruned node
	}

	calc := NewForkConfigCalculator(backend)

	// Test GetCurrentConfig
	t.Run("current_config_pruned", func(t *testing.T) {
		config, hash, forkID, err := calc.GetCurrentConfig()
		if err != nil {
			t.Fatalf("GetCurrentConfig failed on pruned node: %v", err)
		}

		if config == nil {
			t.Error("Config should not be nil")
		}

		if hash == "" {
			t.Error("Hash should not be empty")
		}

		// Fork ID should be nil for pruned nodes
		if forkID != nil {
			t.Error("Fork ID should be nil for pruned nodes")
		}

		// Verify config has expected values
		if config.ChainID != "0x1" {
			t.Errorf("Expected chain ID 0x1, got %s", config.ChainID)
		}

		// Should have Cancun precompiles
		if len(config.Precompiles) < 10 {
			t.Errorf("Expected at least 10 precompiles for Cancun, got %d", len(config.Precompiles))
		}

		// Should have blob schedule for Cancun
		if config.BlobSchedule.Max != 6 {
			t.Errorf("Expected blob max 6 for Cancun, got %d", config.BlobSchedule.Max)
		}
	})

	// Test GetNextConfig
	t.Run("next_config_pruned", func(t *testing.T) {
		config, hash, forkID, err := calc.GetNextConfig()
		if err != nil {
			t.Fatalf("GetNextConfig failed on pruned node: %v", err)
		}

		// Next config might be nil if no future forks
		if config == nil {
			return // No future forks configured, this is fine
		}

		if hash == "" {
			t.Error("Hash should not be empty when config exists")
		}

		// Fork ID should be nil for pruned nodes
		if forkID != nil {
			t.Error("Fork ID should be nil for pruned nodes")
		}
	})

	// Test GetLastConfig
	t.Run("last_config_pruned", func(t *testing.T) {
		config, hash, forkID, err := calc.GetLastConfig()
		if err != nil {
			t.Fatalf("GetLastConfig failed on pruned node: %v", err)
		}

		if config == nil {
			t.Error("Config should not be nil")
		}

		if hash == "" {
			t.Error("Hash should not be empty")
		}

		// Fork ID should be nil for pruned nodes
		if forkID != nil {
			t.Error("Fork ID should be nil for pruned nodes")
		}
	})
}

func TestForkConfigCalculator_FullNode(t *testing.T) {
	// Test with full node (genesis block available)
	chainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		MergeNetsplitBlock:  big.NewInt(0),
		ShanghaiTime:        &[]uint64{0}[0],
		CancunTime:          &[]uint64{1000}[0],
	}

	currentHeader := &types.Header{
		Number: big.NewInt(2000),
		Time:   2000,
	}

	backend := &mockBackend{
		chainConfig:   chainConfig,
		currentHeader: currentHeader,
		genesisError:  false, // Genesis block available
	}

	calc := NewForkConfigCalculator(backend)

	// Test GetCurrentConfig with fork ID
	t.Run("current_config_full", func(t *testing.T) {
		config, hash, forkID, err := calc.GetCurrentConfig()
		if err != nil {
			t.Fatalf("GetCurrentConfig failed on full node: %v", err)
		}

		if config == nil {
			t.Error("Config should not be nil")
		}

		if hash == "" {
			t.Error("Hash should not be empty")
		}

		// Fork ID should be available for full nodes
		if forkID == nil {
			t.Error("Fork ID should be available for full nodes")
		}
	})
}
