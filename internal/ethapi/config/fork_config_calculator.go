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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/params"
)

// ForkConfigCalculator orchestrates the calculation of fork configurations
type ForkConfigCalculator struct {
	chainConfig *params.ChainConfig
	backend     Backend
}

// NewForkConfigCalculator creates a new fork configuration calculator
func NewForkConfigCalculator(backend Backend) *ForkConfigCalculator {
	return &ForkConfigCalculator{
		chainConfig: backend.ChainConfig(),
		backend:     backend,
	}
}

// GetCurrentConfig returns the configuration for the current fork
func (calc *ForkConfigCalculator) GetCurrentConfig() (*ForkConfig, string, *forkid.ID, error) {
	currentHeader := calc.backend.CurrentHeader()
	if currentHeader == nil {
		return nil, "", nil, fmt.Errorf("cannot get current header")
	}

	blockNumber := currentHeader.Number.Uint64()
	blockTime := currentHeader.Time

	// Calculate current configuration
	config, err := calc.buildForkConfig(blockNumber, blockTime)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to build current fork config: %w", err)
	}

	// Hash the configuration
	hash, err := HashConfig(config)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to hash current config: %w", err)
	}

	// Use geth's existing fork ID calculation with adapter
	blockchainAdapter := NewBlockchainAdapter(calc.backend)
	forkID := forkid.NewIDWithChain(blockchainAdapter)

	return config, hash, &forkID, nil
}

// GetNextConfig returns the configuration for the next scheduled fork
func (calc *ForkConfigCalculator) GetNextConfig() (*ForkConfig, string, *forkid.ID, error) {
	currentHeader := calc.backend.CurrentHeader()
	if currentHeader == nil {
		return nil, "", nil, fmt.Errorf("cannot get current header")
	}

	currentBlockTime := currentHeader.Time

	// Find the next fork activation time
	nextActivationTime, err := GetNextForkActivationTime(calc.chainConfig, currentBlockTime)
	if err != nil {
		// No next fork configured
		return nil, "", nil, nil
	}

	// Build config for next fork - use a future block number (current + some estimated blocks)
	// This is an approximation since we don't know the exact block number for the future timestamp
	estimatedNextBlockNumber := currentHeader.Number.Uint64() + ((nextActivationTime - currentBlockTime) / 12) // Assume 12s block time

	nextConfig, err := calc.buildForkConfig(estimatedNextBlockNumber, nextActivationTime)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to build next fork config: %w", err)
	}

	// Hash the configuration
	hash, err := HashConfig(nextConfig)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to hash next config: %w", err)
	}

	// Calculate fork ID for next configuration using estimated block/time
	genesis, err := calc.backend.BlockByNumber(context.Background(), 0)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to get genesis block: %w", err)
	}
	nextForkID := forkid.NewID(calc.chainConfig, genesis, estimatedNextBlockNumber, nextActivationTime)

	return nextConfig, hash, &nextForkID, nil
}

// GetLastConfig returns the configuration for the last known fork
func (calc *ForkConfigCalculator) GetLastConfig() (*ForkConfig, string, *forkid.ID, error) {
	// Find the last known fork activation time
	lastActivationTime, err := GetLastKnownForkActivationTime(calc.chainConfig)
	if err != nil {
		// No known forks - return current config as last config
		return calc.GetCurrentConfig()
	}

	// For the last config, we use a very high block number to ensure all features are enabled
	// This represents the "final" state of the chain with all configured forks activated
	maxBlockNumber := uint64(999999999) // Arbitrarily high number

	lastConfig, err := calc.buildForkConfig(maxBlockNumber, lastActivationTime)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to build last fork config: %w", err)
	}

	// Hash the configuration
	hash, err := HashConfig(lastConfig)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to hash last config: %w", err)
	}

	// Calculate fork ID for last configuration using high future block/time
	genesis, err := calc.backend.BlockByNumber(context.Background(), 0)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to get genesis block: %w", err)
	}
	lastForkID := forkid.NewID(calc.chainConfig, genesis, maxBlockNumber, lastActivationTime)

	return lastConfig, hash, &lastForkID, nil
}

// buildForkConfig constructs a complete fork configuration for the given block number and time
func (calc *ForkConfigCalculator) buildForkConfig(blockNumber uint64, blockTime uint64) (*ForkConfig, error) {
	// Calculate activation time for this fork
	activationTime, err := CalculateActivationTimeForFork(calc.chainConfig, blockNumber, blockTime)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate activation time: %w", err)
	}

	// Get chain rules for this block
	rules := calc.chainConfig.Rules(new(big.Int).SetUint64(blockNumber), true, blockTime)

	// Build configuration
	config := &ForkConfig{
		ActivationTime:  activationTime,
		BlobSchedule:    ExtractBlobSchedule(calc.chainConfig, blockNumber, blockTime),
		ChainID:         fmt.Sprintf("0x%x", calc.chainConfig.ChainID.Uint64()),
		Precompiles:     GetActivePrecompiles(rules),
		SystemContracts: GetSystemContracts(rules, calc.chainConfig),
	}

	return config, nil
}

// GetAll returns current, next, and last configurations in one call
func (calc *ForkConfigCalculator) GetAll() (*ForkConfig, string, *forkid.ID, *ForkConfig, string, *forkid.ID, *ForkConfig, string, *forkid.ID, error) {
	// Get current config
	currentConfig, currentHash, currentForkID, err := calc.GetCurrentConfig()
	if err != nil {
		return nil, "", nil, nil, "", nil, nil, "", nil, fmt.Errorf("failed to get current config: %w", err)
	}

	// Get next config (may be nil)
	nextConfig, nextHash, nextForkID, err := calc.GetNextConfig()
	if err != nil {
		return nil, "", nil, nil, "", nil, nil, "", nil, fmt.Errorf("failed to get next config: %w", err)
	}

	// Get last config
	lastConfig, lastHash, lastForkID, err := calc.GetLastConfig()
	if err != nil {
		return nil, "", nil, nil, "", nil, nil, "", nil, fmt.Errorf("failed to get last config: %w", err)
	}

	return currentConfig, currentHash, currentForkID,
		nextConfig, nextHash, nextForkID,
		lastConfig, lastHash, lastForkID, nil
}
