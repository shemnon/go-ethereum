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

	"github.com/ethereum/go-ethereum/params"
)

// ExtractBlobSchedule extracts blob configuration parameters for a specific fork
// based on chain configuration and activation rules.
func ExtractBlobSchedule(chainConfig *params.ChainConfig, blockNumber uint64, blockTime uint64) BlobScheduleParams {
	rules := chainConfig.Rules(new(big.Int).SetUint64(blockNumber), true, blockTime)

	// Default blob schedule (pre-EIP-4844)
	if !rules.IsCancun {
		return BlobScheduleParams{
			BaseFeeUpdateFraction: 0,
			Max:                   0,
			Target:                0,
		}
	}

	// Get the blob schedule configuration
	blobScheduleConfig := chainConfig.BlobScheduleConfig
	if blobScheduleConfig == nil {
		// Fallback to default Cancun blob configuration
		return BlobScheduleParams{
			BaseFeeUpdateFraction: params.DefaultCancunBlobConfig.UpdateFraction,
			Max:                   params.DefaultCancunBlobConfig.Max,
			Target:                params.DefaultCancunBlobConfig.Target,
		}
	}

	// Determine which blob config to use based on fork
	var blobConfig *params.BlobConfig

	switch {
	case rules.IsOsaka && blobScheduleConfig.Osaka != nil:
		blobConfig = blobScheduleConfig.Osaka
	case rules.IsPrague && blobScheduleConfig.Prague != nil:
		blobConfig = blobScheduleConfig.Prague
	case rules.IsCancun && blobScheduleConfig.Cancun != nil:
		blobConfig = blobScheduleConfig.Cancun
	default:
		// Fallback to default Cancun configuration
		blobConfig = params.DefaultCancunBlobConfig
	}

	return BlobScheduleParams{
		BaseFeeUpdateFraction: blobConfig.UpdateFraction,
		Max:                   blobConfig.Max,
		Target:                blobConfig.Target,
	}
}
