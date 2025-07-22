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

// CalculateActivationTimeForFork calculates the activation time for a specific fork
func CalculateActivationTimeForFork(chainConfig *params.ChainConfig, targetBlockNumber uint64, targetBlockTime uint64) (uint64, error) {
	return determineActivationTime(chainConfig, targetBlockNumber, targetBlockTime)
}

// determineActivationTime determines the activation time for the fork that would be active
// at the given block number and time
func determineActivationTime(chainConfig *params.ChainConfig, blockNumber uint64, blockTime uint64) (uint64, error) {
	// Check time-based forks first (more recent)
	if chainConfig.VerkleTime != nil && blockTime >= *chainConfig.VerkleTime {
		return *chainConfig.VerkleTime, nil
	}
	if chainConfig.OsakaTime != nil && blockTime >= *chainConfig.OsakaTime {
		return *chainConfig.OsakaTime, nil
	}
	if chainConfig.PragueTime != nil && blockTime >= *chainConfig.PragueTime {
		return *chainConfig.PragueTime, nil
	}
	if chainConfig.CancunTime != nil && blockTime >= *chainConfig.CancunTime {
		return *chainConfig.CancunTime, nil
	}
	if chainConfig.ShanghaiTime != nil && blockTime >= *chainConfig.ShanghaiTime {
		return *chainConfig.ShanghaiTime, nil
	}

	// Check block-based forks (older forks)
	// For block-based forks, we use 0 as activation time since they were activated at genesis
	// or we would need to estimate the timestamp from block number

	blockNumberBig := new(big.Int).SetUint64(blockNumber)

	if chainConfig.GrayGlacierBlock != nil && blockNumberBig.Cmp(chainConfig.GrayGlacierBlock) >= 0 {
		return 0, nil // Block-based fork, use 0 for activation time
	}
	if chainConfig.ArrowGlacierBlock != nil && blockNumberBig.Cmp(chainConfig.ArrowGlacierBlock) >= 0 {
		return 0, nil
	}
	if chainConfig.LondonBlock != nil && blockNumberBig.Cmp(chainConfig.LondonBlock) >= 0 {
		return 0, nil
	}
	if chainConfig.BerlinBlock != nil && blockNumberBig.Cmp(chainConfig.BerlinBlock) >= 0 {
		return 0, nil
	}
	if chainConfig.MuirGlacierBlock != nil && blockNumberBig.Cmp(chainConfig.MuirGlacierBlock) >= 0 {
		return 0, nil
	}
	if chainConfig.IstanbulBlock != nil && blockNumberBig.Cmp(chainConfig.IstanbulBlock) >= 0 {
		return 0, nil
	}
	if chainConfig.PetersburgBlock != nil && blockNumberBig.Cmp(chainConfig.PetersburgBlock) >= 0 {
		return 0, nil
	}
	if chainConfig.ConstantinopleBlock != nil && blockNumberBig.Cmp(chainConfig.ConstantinopleBlock) >= 0 {
		return 0, nil
	}
	if chainConfig.ByzantiumBlock != nil && blockNumberBig.Cmp(chainConfig.ByzantiumBlock) >= 0 {
		return 0, nil
	}
	if chainConfig.EIP158Block != nil && blockNumberBig.Cmp(chainConfig.EIP158Block) >= 0 {
		return 0, nil
	}
	if chainConfig.EIP155Block != nil && blockNumberBig.Cmp(chainConfig.EIP155Block) >= 0 {
		return 0, nil
	}
	if chainConfig.EIP150Block != nil && blockNumberBig.Cmp(chainConfig.EIP150Block) >= 0 {
		return 0, nil
	}
	if chainConfig.HomesteadBlock != nil && blockNumberBig.Cmp(chainConfig.HomesteadBlock) >= 0 {
		return 0, nil
	}

	// Default to genesis (block 0)
	return 0, nil
}

// GetNextForkActivationTime returns the activation time of the next scheduled fork
func GetNextForkActivationTime(chainConfig *params.ChainConfig, currentBlockTime uint64) (uint64, error) {
	// Look for the next time-based fork that hasn't activated yet
	forks := []struct {
		time *uint64
		name string
	}{
		{chainConfig.ShanghaiTime, "shanghai"},
		{chainConfig.CancunTime, "cancun"},
		{chainConfig.PragueTime, "prague"},
		{chainConfig.OsakaTime, "osaka"},
		{chainConfig.VerkleTime, "verkle"},
	}

	for _, fork := range forks {
		if fork.time != nil && *fork.time > currentBlockTime {
			return *fork.time, nil
		}
	}

	// No future fork scheduled
	return 0, fmt.Errorf("no future fork scheduled")
}

// GetLastKnownForkActivationTime returns the activation time of the last known fork
func GetLastKnownForkActivationTime(chainConfig *params.ChainConfig) (uint64, error) {
	// Look for the latest configured fork (reverse order)
	if chainConfig.VerkleTime != nil {
		return *chainConfig.VerkleTime, nil
	}
	if chainConfig.OsakaTime != nil {
		return *chainConfig.OsakaTime, nil
	}
	if chainConfig.PragueTime != nil {
		return *chainConfig.PragueTime, nil
	}
	if chainConfig.CancunTime != nil {
		return *chainConfig.CancunTime, nil
	}
	if chainConfig.ShanghaiTime != nil {
		return *chainConfig.ShanghaiTime, nil
	}

	// No time-based forks configured - this shouldn't happen in practice
	return 0, fmt.Errorf("no time-based forks configured")
}
