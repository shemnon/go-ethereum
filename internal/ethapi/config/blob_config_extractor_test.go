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
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

func TestExtractBlobSchedule(t *testing.T) {
	tests := []struct {
		name        string
		chainConfig *params.ChainConfig
		blockNumber uint64
		blockTime   uint64
		expected    BlobScheduleParams
	}{
		{
			name: "pre-cancun (no blobs)",
			chainConfig: &params.ChainConfig{
				ChainID: big.NewInt(1),
				// No CancunTime set
			},
			blockNumber: 1000,
			blockTime:   1000,
			expected: BlobScheduleParams{
				BaseFeeUpdateFraction: 0,
				Max:                   0,
				Target:                0,
			},
		},
		{
			name: "cancun with default config",
			chainConfig: &params.ChainConfig{
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
				ShanghaiTime:        &[]uint64{500}[0],
				CancunTime:          &[]uint64{1000}[0],
			},
			blockNumber: 1500,
			blockTime:   1500,
			expected: BlobScheduleParams{
				BaseFeeUpdateFraction: params.DefaultCancunBlobConfig.UpdateFraction,
				Max:                   params.DefaultCancunBlobConfig.Max,
				Target:                params.DefaultCancunBlobConfig.Target,
			},
		},
		{
			name: "cancun with custom blob schedule",
			chainConfig: &params.ChainConfig{
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
				ShanghaiTime:        &[]uint64{500}[0],
				CancunTime:          &[]uint64{1000}[0],
				BlobScheduleConfig: &params.BlobScheduleConfig{
					Cancun: &params.BlobConfig{
						UpdateFraction: 1234567,
						Max:            8,
						Target:         4,
					},
				},
			},
			blockNumber: 1500,
			blockTime:   1500,
			expected: BlobScheduleParams{
				BaseFeeUpdateFraction: 1234567,
				Max:                   8,
				Target:                4,
			},
		},
		{
			name: "prague with custom blob schedule",
			chainConfig: &params.ChainConfig{
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
				ShanghaiTime:        &[]uint64{500}[0],
				CancunTime:          &[]uint64{1000}[0],
				PragueTime:          &[]uint64{2000}[0],
				BlobScheduleConfig: &params.BlobScheduleConfig{
					Cancun: &params.BlobConfig{
						UpdateFraction: 1111111,
						Max:            6,
						Target:         3,
					},
					Prague: &params.BlobConfig{
						UpdateFraction: 2222222,
						Max:            12,
						Target:         6,
					},
				},
			},
			blockNumber: 2500,
			blockTime:   2500,
			expected: BlobScheduleParams{
				BaseFeeUpdateFraction: 2222222,
				Max:                   12,
				Target:                6,
			},
		},
		{
			name: "osaka with custom blob schedule",
			chainConfig: &params.ChainConfig{
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
				ShanghaiTime:        &[]uint64{500}[0],
				CancunTime:          &[]uint64{1000}[0],
				PragueTime:          &[]uint64{2000}[0],
				OsakaTime:           &[]uint64{3000}[0],
				BlobScheduleConfig: &params.BlobScheduleConfig{
					Cancun: &params.BlobConfig{
						UpdateFraction: 1111111,
						Max:            6,
						Target:         3,
					},
					Prague: &params.BlobConfig{
						UpdateFraction: 2222222,
						Max:            12,
						Target:         6,
					},
					Osaka: &params.BlobConfig{
						UpdateFraction: 3333333,
						Max:            16,
						Target:         8,
					},
				},
			},
			blockNumber: 3500,
			blockTime:   3500,
			expected: BlobScheduleParams{
				BaseFeeUpdateFraction: 3333333,
				Max:                   16,
				Target:                8,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractBlobSchedule(tt.chainConfig, tt.blockNumber, tt.blockTime)

			if result.BaseFeeUpdateFraction != tt.expected.BaseFeeUpdateFraction {
				t.Errorf("BaseFeeUpdateFraction = %d, want %d", result.BaseFeeUpdateFraction, tt.expected.BaseFeeUpdateFraction)
			}
			if result.Max != tt.expected.Max {
				t.Errorf("Max = %d, want %d", result.Max, tt.expected.Max)
			}
			if result.Target != tt.expected.Target {
				t.Errorf("Target = %d, want %d", result.Target, tt.expected.Target)
			}
		})
	}
}
