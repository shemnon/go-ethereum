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
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// GetCurrentConfig returns the configuration for the current fork
func GetCurrentConfig(chainConfig *params.ChainConfig, currentHeader *types.Header, genesis *types.Block) (ForkConfig, string, string, error) {
	if currentHeader == nil {
		return ForkConfig{}, "", "", fmt.Errorf("cannot get current header")
	}

	blockNumber := currentHeader.Number.Uint64()
	blockTime := currentHeader.Time

	// Calculate current configuration
	config, err := buildForkConfig(chainConfig, blockNumber, blockTime)
	if err != nil {
		return ForkConfig{}, "", "", fmt.Errorf("failed to build current fork config: %w", err)
	}

	// Hash the configuration
	hash, err := HashConfig(config)
	if err != nil {
		return ForkConfig{}, "", "", fmt.Errorf("failed to hash current config: %w", err)
	}

	// Use geth's existing fork ID calculation
	forkID := forkid.NewID(chainConfig, genesis, blockNumber, blockTime)
	currentForkId := "0x" + hex.EncodeToString(forkID.Hash[:])

	return *config, hash, currentForkId, nil
}

// GetNextConfig returns the configuration for the next scheduled fork
func GetNextConfig(chainConfig *params.ChainConfig, currentHeader *types.Header, genesis *types.Block) (ForkConfig, string, string, error) {
	if currentHeader == nil {
		return ForkConfig{}, "", "", fmt.Errorf("cannot get current header")
	}

	currentBlockTime := currentHeader.Time

	// Find the next fork activation time
	nextActivationTime, err := GetNextForkActivationTime(chainConfig, currentBlockTime)
	if err != nil {
		// No next fork configured - return empty values
		return ForkConfig{}, "", "", nil
	}

	// Build config for next fork - use a future block number (current + some estimated blocks)
	// This is an approximation since we don't know the exact block number for the future timestamp
	estimatedNextBlockNumber := currentHeader.Number.Uint64() + ((nextActivationTime - currentBlockTime) / 12) // Assume 12s block time

	nextConfig, err := buildForkConfig(chainConfig, estimatedNextBlockNumber, nextActivationTime)
	if err != nil {
		return ForkConfig{}, "", "", fmt.Errorf("failed to build next fork config: %w", err)
	}

	// Hash the configuration
	hash, err := HashConfig(nextConfig)
	if err != nil {
		return ForkConfig{}, "", "", fmt.Errorf("failed to hash next config: %w", err)
	}

	// Calculate fork ID for next configuration using estimated block/time
	nextForkID := forkid.NewID(chainConfig, genesis, estimatedNextBlockNumber, nextActivationTime)
	nextForkIdHash := "0x" + hex.EncodeToString(nextForkID.Hash[:])

	return *nextConfig, hash, nextForkIdHash, nil
}

// GetLastConfig returns the configuration for the last known fork
func GetLastConfig(chainConfig *params.ChainConfig, currentHeader *types.Header, genesis *types.Block) (ForkConfig, string, string, error) {
	// Find the last known fork activation time
	lastActivationTime, err := GetLastKnownForkActivationTime(chainConfig)
	if err != nil {
		// No known forks - return current config as last config but with empty fork ID
		currentConfig, currentHash, _, currentErr := GetCurrentConfig(chainConfig, currentHeader, genesis)
		if currentErr != nil {
			return ForkConfig{}, "", "", currentErr
		}
		return currentConfig, currentHash, "", nil
	}

	// For the last config, we use a very high block number to ensure all features are enabled
	// This represents the "final" state of the chain with all configured forks activated
	maxBlockNumber := uint64(999999999) // Arbitrarily high number

	lastConfig, err := buildForkConfig(chainConfig, maxBlockNumber, lastActivationTime)
	if err != nil {
		return ForkConfig{}, "", "", fmt.Errorf("failed to build last fork config: %w", err)
	}

	// Hash the configuration
	hash, err := HashConfig(lastConfig)
	if err != nil {
		return ForkConfig{}, "", "", fmt.Errorf("failed to hash last config: %w", err)
	}

	// Calculate fork ID for last configuration using high future block/time
	lastForkID := forkid.NewID(chainConfig, genesis, maxBlockNumber, lastActivationTime)
	lastForkIdHash := "0x" + hex.EncodeToString(lastForkID.Hash[:])

	return *lastConfig, hash, lastForkIdHash, nil
}

// buildForkConfig constructs a complete fork configuration for the given block number and time
func buildForkConfig(chainConfig *params.ChainConfig, blockNumber uint64, blockTime uint64) (*ForkConfig, error) {
	// Calculate activation time for this fork
	activationTime, err := CalculateActivationTimeForFork(chainConfig, blockNumber, blockTime)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate activation time: %w", err)
	}

	// Get chain rules for this block
	rules := chainConfig.Rules(new(big.Int).SetUint64(blockNumber), true, blockTime)

	// Build configuration
	config := &ForkConfig{
		ActivationTime:  activationTime,
		BlobSchedule:    ExtractBlobSchedule(chainConfig, blockNumber, blockTime),
		ChainID:         fmt.Sprintf("0x%x", chainConfig.ChainID.Uint64()),
		Precompiles:     GetActivePrecompiles(rules),
		SystemContracts: GetSystemContracts(rules, chainConfig),
	}

	return config, nil
}

// GetAll returns current, next, and last configurations in one call
func GetAll(chainConfig *params.ChainConfig, currentHeader *types.Header, genesis *types.Block) (ForkConfig, string, string, ForkConfig, string, string, ForkConfig, string, string, error) {
	// Get current config
	currentConfig, currentHash, currentForkID, err := GetCurrentConfig(chainConfig, currentHeader, genesis)
	if err != nil {
		return ForkConfig{}, "", "", ForkConfig{}, "", "", ForkConfig{}, "", "", fmt.Errorf("failed to get current config: %w", err)
	}

	// Get next config (may be empty)
	nextConfig, nextHash, nextForkID, err := GetNextConfig(chainConfig, currentHeader, genesis)
	if err != nil {
		return ForkConfig{}, "", "", ForkConfig{}, "", "", ForkConfig{}, "", "", fmt.Errorf("failed to get next config: %w", err)
	}

	// Get last config
	lastConfig, lastHash, lastForkID, err := GetLastConfig(chainConfig, currentHeader, genesis)
	if err != nil {
		return ForkConfig{}, "", "", ForkConfig{}, "", "", ForkConfig{}, "", "", fmt.Errorf("failed to get last config: %w", err)
	}

	return currentConfig, currentHash, currentForkID,
		nextConfig, nextHash, nextForkID,
		lastConfig, lastHash, lastForkID, nil
}
