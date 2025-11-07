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
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/ethereum/go-ethereum/rpc"
)

// ForkConfig represents the configuration for a specific fork as defined in EIP-7910.
type ForkConfig struct {
	ActivationTime  uint64            `json:"activationTime"`
	BlobSchedule    *BlobScheduleJSON `json:"blobSchedule"`
	ChainID         string            `json:"chainId"`
	ForkID          string            `json:"forkId"`
	Precompiles     map[string]string `json:"precompiles"`
	SystemContracts map[string]string `json:"systemContracts,omitempty"`
}

// BlobScheduleJSON represents the blob schedule configuration in JSON format.
type BlobScheduleJSON struct {
	BaseFeeUpdateFraction uint64 `json:"baseFeeUpdateFraction"`
	Max                   int    `json:"max"`
	Target                int    `json:"target"`
}

// ConfigResponse represents the response for eth_config as defined in EIP-7910.
type ConfigResponse struct {
	Current *ForkConfig `json:"current"`
	Next    *ForkConfig `json:"next,omitempty"`
	Last    *ForkConfig `json:"last,omitempty"`
}

// Config returns the current, next, and last fork configurations (EIP-7910).
func (api *EthereumAPI) Config(ctx context.Context) (*ConfigResponse, error) {
	// Get chain config and current header
	chainConfig := api.b.ChainConfig()
	currentHeader := api.b.CurrentHeader()

	// Get genesis block
	genesisBlock, err := api.b.BlockByNumber(ctx, rpc.BlockNumber(0))
	if err != nil {
		return nil, fmt.Errorf("failed to get genesis block: %w", err)
	}

	// Determine current, next, and last forks
	currentFork := determineFork(chainConfig, currentHeader.Time)
	nextFork := determineNextFork(chainConfig, currentHeader.Time)
	var lastFork forks.Fork
	if nextFork != nil {
		lastFork = *nextFork
	}

	// Build current configuration
	currentConfig, err := buildForkConfig(chainConfig, genesisBlock, currentHeader, currentFork)
	if err != nil {
		return nil, fmt.Errorf("failed to build current config: %w", err)
	}

	// Build next configuration if applicable
	var nextConfig *ForkConfig
	if nextFork != nil {
		nextConfig, err = buildForkConfig(chainConfig, genesisBlock, currentHeader, *nextFork)
		if err != nil {
			return nil, fmt.Errorf("failed to build next config: %w", err)
		}
	}

	// Build last configuration
	var lastConfig *ForkConfig
	if nextFork != nil {
		lastConfig, err = buildForkConfig(chainConfig, genesisBlock, currentHeader, lastFork)
		if err != nil {
			return nil, fmt.Errorf("failed to build last config: %w", err)
		}
	}

	return &ConfigResponse{
		Current: currentConfig,
		Next:    nextConfig,
		Last:    lastConfig,
	}, nil
}

// determineFork determines the current fork based on the timestamp.
func determineFork(config *params.ChainConfig, timestamp uint64) forks.Fork {
	// Check forks in reverse chronological order
	if config.IsOsaka(nil, timestamp) {
		return forks.Osaka
	}
	if config.IsPrague(nil, timestamp) {
		return forks.Prague
	}
	if config.IsCancun(nil, timestamp) {
		return forks.Cancun
	}
	if config.IsShanghai(nil, timestamp) {
		return forks.Shanghai
	}
	// For pre-Shanghai, we're always on Paris (post-merge)
	return forks.Paris
}

// determineNextFork determines the next scheduled fork after the current timestamp.
func determineNextFork(config *params.ChainConfig, currentTime uint64) *forks.Fork {
	// Check each fork in chronological order
	if config.OsakaTime != nil && *config.OsakaTime > currentTime {
		fork := forks.Osaka
		return &fork
	}
	if config.PragueTime != nil && *config.PragueTime > currentTime {
		fork := forks.Prague
		return &fork
	}
	if config.CancunTime != nil && *config.CancunTime > currentTime {
		fork := forks.Cancun
		return &fork
	}
	if config.ShanghaiTime != nil && *config.ShanghaiTime > currentTime {
		fork := forks.Shanghai
		return &fork
	}
	return nil
}

// buildForkConfig builds a ForkConfig for the specified fork.
func buildForkConfig(config *params.ChainConfig, genesis *types.Block, currentHeader *types.Header, fork forks.Fork) (*ForkConfig, error) {
	// Get activation time
	activationTime := getForkActivationTime(config, fork)

	// Get blob schedule
	blobSchedule := getBlobSchedule(config, fork)

	// Get chain ID
	chainID := strings.ToLower(fmt.Sprintf("0x%x", config.ChainID))

	// Calculate fork ID
	forkID := calculateForkID(config, genesis, currentHeader, fork)

	// Get precompiles
	precompiles := getPrecompiles(fork)

	// Get system contracts (Cancun and later)
	var systemContracts map[string]string
	if fork >= forks.Cancun {
		systemContracts = getSystemContracts(config, fork)
	}

	return &ForkConfig{
		ActivationTime:  activationTime,
		BlobSchedule:    blobSchedule,
		ChainID:         chainID,
		ForkID:          forkID,
		Precompiles:     precompiles,
		SystemContracts: systemContracts,
	}, nil
}

// getForkActivationTime returns the activation time for a fork.
func getForkActivationTime(config *params.ChainConfig, fork forks.Fork) uint64 {
	switch fork {
	case forks.Osaka:
		if config.OsakaTime != nil {
			return *config.OsakaTime
		}
	case forks.Prague:
		if config.PragueTime != nil {
			return *config.PragueTime
		}
	case forks.Cancun:
		if config.CancunTime != nil {
			return *config.CancunTime
		}
		return 0 // Genesis
	case forks.Shanghai:
		if config.ShanghaiTime != nil {
			return *config.ShanghaiTime
		}
		return 0 // Genesis
	}
	return 0 // Default to genesis for older forks
}

// getBlobSchedule returns the blob schedule for a fork.
func getBlobSchedule(config *params.ChainConfig, fork forks.Fork) *BlobScheduleJSON {
	if config.BlobScheduleConfig == nil {
		// Return default based on fork
		switch fork {
		case forks.Osaka:
			return blobConfigToJSON(params.DefaultOsakaBlobConfig)
		case forks.Prague:
			return blobConfigToJSON(params.DefaultPragueBlobConfig)
		default:
			return blobConfigToJSON(params.DefaultCancunBlobConfig)
		}
	}

	var blobConfig *params.BlobConfig
	switch fork {
	case forks.Osaka:
		blobConfig = config.BlobScheduleConfig.Osaka
		if blobConfig == nil {
			blobConfig = params.DefaultOsakaBlobConfig
		}
	case forks.Prague:
		blobConfig = config.BlobScheduleConfig.Prague
		if blobConfig == nil {
			blobConfig = params.DefaultPragueBlobConfig
		}
	default:
		blobConfig = config.BlobScheduleConfig.Cancun
		if blobConfig == nil {
			blobConfig = params.DefaultCancunBlobConfig
		}
	}

	return blobConfigToJSON(blobConfig)
}

// blobConfigToJSON converts a BlobConfig to JSON format.
func blobConfigToJSON(config *params.BlobConfig) *BlobScheduleJSON {
	if config == nil {
		return &BlobScheduleJSON{
			BaseFeeUpdateFraction: 3338477,
			Max:                   6,
			Target:                3,
		}
	}
	return &BlobScheduleJSON{
		BaseFeeUpdateFraction: config.UpdateFraction,
		Max:                   config.Max,
		Target:                config.Target,
	}
}

// calculateForkID calculates the fork ID for a specific fork.
func calculateForkID(config *params.ChainConfig, genesis *types.Block, currentHeader *types.Header, fork forks.Fork) string {
	// Calculate the fork ID as it would be after this fork activates.
	// This is done by simulating the state at or just after the fork activation.

	// Get the activation time for this fork
	activationTime := getForkActivationTime(config, fork)

	// For the fork ID calculation, we use a timestamp just after the fork activation
	// to ensure the fork is considered "passed"
	var simulatedTime uint64
	if activationTime == 0 {
		simulatedTime = genesis.Time() + 1
	} else {
		simulatedTime = activationTime + 1
	}

	// Use a high block number to ensure all block-based forks are considered passed
	// Fork IDs only care about which forks have been passed, not the exact block number
	id := forkid.NewID(config, genesis, currentHeader.Number.Uint64(), simulatedTime)

	return strings.ToLower(fmt.Sprintf("0x%08x", id.Hash))
}

// getPrecompiles returns the precompile addresses for a fork.
func getPrecompiles(fork forks.Fork) map[string]string {
	precompiles := make(map[string]string)

	var contracts vm.PrecompiledContracts
	switch fork {
	case forks.Osaka:
		contracts = vm.PrecompiledContractsOsaka
	case forks.Prague:
		contracts = vm.PrecompiledContractsPrague
	default:
		contracts = vm.PrecompiledContractsCancun
	}

	// Map addresses to standard names according to EIP-7910
	for addr := range contracts {
		name := getPrecompileName(addr, fork)
		if name != "" {
			precompiles[name] = strings.ToLower(addr.Hex())
		}
	}

	return precompiles
}

// getPrecompileName returns the standard name for a precompile address.
func getPrecompileName(addr common.Address, fork forks.Fork) string {
	switch addr.Hex() {
	case "0x0000000000000000000000000000000000000001":
		return "ECREC"
	case "0x0000000000000000000000000000000000000002":
		return "SHA256"
	case "0x0000000000000000000000000000000000000003":
		return "RIPEMD160"
	case "0x0000000000000000000000000000000000000004":
		return "ID"
	case "0x0000000000000000000000000000000000000005":
		return "MODEXP"
	case "0x0000000000000000000000000000000000000006":
		return "BN254_ADD"
	case "0x0000000000000000000000000000000000000007":
		return "BN254_MUL"
	case "0x0000000000000000000000000000000000000008":
		return "BN254_PAIRING"
	case "0x0000000000000000000000000000000000000009":
		return "BLAKE2F"
	case "0x000000000000000000000000000000000000000a":
		return "KZG_POINT_EVALUATION"
	case "0x000000000000000000000000000000000000000b":
		return "BLS12_G1ADD"
	case "0x000000000000000000000000000000000000000c":
		return "BLS12_G1MSM"
	case "0x000000000000000000000000000000000000000d":
		return "BLS12_G2ADD"
	case "0x000000000000000000000000000000000000000e":
		return "BLS12_G2MSM"
	case "0x000000000000000000000000000000000000000f":
		return "BLS12_PAIRING_CHECK"
	case "0x0000000000000000000000000000000000000010":
		return "BLS12_MAP_FP_TO_G1"
	case "0x0000000000000000000000000000000000000011":
		return "BLS12_MAP_FP2_TO_G2"
	case "0x0000000000000000000000000000000000000100":
		return "P256VERIFY"
	}
	return ""
}

// getSystemContracts returns the system contract addresses for a fork.
func getSystemContracts(config *params.ChainConfig, fork forks.Fork) map[string]string {
	contracts := make(map[string]string)

	// Beacon roots address (Cancun+)
	if fork >= forks.Cancun {
		contracts["BEACON_ROOTS_ADDRESS"] = strings.ToLower(params.BeaconRootsAddress.Hex())
	}

	// Prague system contracts
	if fork >= forks.Prague {
		contracts["CONSOLIDATION_REQUEST_PREDEPLOY_ADDRESS"] = strings.ToLower(params.ConsolidationQueueAddress.Hex())
		if config.DepositContractAddress != (common.Address{}) {
			contracts["DEPOSIT_CONTRACT_ADDRESS"] = strings.ToLower(config.DepositContractAddress.Hex())
		}
		contracts["HISTORY_STORAGE_ADDRESS"] = strings.ToLower(params.HistoryStorageAddress.Hex())
		contracts["WITHDRAWAL_REQUEST_PREDEPLOY_ADDRESS"] = strings.ToLower(params.WithdrawalQueueAddress.Hex())
	}

	return contracts
}
