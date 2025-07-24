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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

// BuildForkConfig constructs a complete fork configuration for the given block number and time
func BuildForkConfig(chainConfig *params.ChainConfig, blockNumber uint64, blockTime uint64) (*ForkConfig, error) {
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
