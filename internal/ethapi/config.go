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
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// ============================================================================
// EIP-7910 Types and Constants
// ============================================================================

// ForkConfig represents the configuration of a specific fork as defined by EIP-7910.
type ForkConfig struct {
	ActivationTime  uint64                    `json:"activationTime"`
	BlobSchedule    BlobScheduleParams        `json:"blobSchedule"`
	ChainID         string                    `json:"chainId"`
	ForkID          string                    `json:"forkId"`
	Precompiles     map[string]string         `json:"precompiles"`
	SystemContracts map[string]common.Address `json:"systemContracts"`
}

// BlobScheduleParams represents the blob configuration parameters as defined by EIP-7910.
// These correspond to the blob parameters for the specific fork.
type BlobScheduleParams struct {
	BaseFeeUpdateFraction uint64 `json:"baseFeeUpdateFraction"`
	Max                   int    `json:"max"`
	Target                int    `json:"target"`
}

// SystemContractName constants for the known system contracts
const (
	BeaconRootsAddressName                   = "BEACON_ROOTS_ADDRESS"
	HistoryStorageAddressName                = "HISTORY_STORAGE_ADDRESS"
	WithdrawalRequestPredeployAddressName    = "WITHDRAWAL_REQUEST_PREDEPLOY_ADDRESS"
	ConsolidationRequestPredeployAddressName = "CONSOLIDATION_REQUEST_PREDEPLOY_ADDRESS"
	DepositContractAddressName               = "DEPOSIT_CONTRACT_ADDRESS"
)

// precompileNames maps precompile addresses to their canonical names as defined by EIP-7910
var precompileNames = map[common.Address]string{
	common.BytesToAddress([]byte{0x1}):       "ECREC",
	common.BytesToAddress([]byte{0x2}):       "SHA256",
	common.BytesToAddress([]byte{0x3}):       "RIPEMD160",
	common.BytesToAddress([]byte{0x4}):       "ID",
	common.BytesToAddress([]byte{0x5}):       "MODEXP",
	common.BytesToAddress([]byte{0x6}):       "BN254_ADD",
	common.BytesToAddress([]byte{0x7}):       "BN254_MUL",
	common.BytesToAddress([]byte{0x8}):       "BN254_PAIRING",
	common.BytesToAddress([]byte{0x9}):       "BLAKE2F",
	common.BytesToAddress([]byte{0xa}):       "KZG_POINT_EVALUATION",
	common.BytesToAddress([]byte{0xb}):       "BLS12_G1ADD",
	common.BytesToAddress([]byte{0xc}):       "BLS12_G1MSM",
	common.BytesToAddress([]byte{0xd}):       "BLS12_G2ADD",
	common.BytesToAddress([]byte{0xe}):       "BLS12_G2MSM",
	common.BytesToAddress([]byte{0xf}):       "BLS12_PAIRING_CHECK",
	common.BytesToAddress([]byte{0x10}):      "BLS12_MAP_FP_TO_G1",
	common.BytesToAddress([]byte{0x11}):      "BLS12_MAP_FP2_TO_G2",
	common.BytesToAddress([]byte{0x1, 0x00}): "P256VERIFY", // EIP-7212 precompile
}

// / When asking for a future fork time this signals there is no future configured fork
var ErrNoFutureFork = errors.New("no future fork scheduled")

// ============================================================================
// Fork Configuration Building
// ============================================================================

// BuildForkConfig constructs a complete fork configuration for the given block number and time
func BuildForkConfig(chainConfig *params.ChainConfig, genesis *types.Block, blockNumber uint64, blockTime uint64) (*ForkConfig, error) {
	// Calculate activation time for this fork
	activationTime, err := calculateActivationTimeForFork(chainConfig, blockTime)
	if err == ErrNoFutureFork {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to calculate activation time: %w", err)
	}

	// Get chain rules for this block
	rules := chainConfig.Rules(new(big.Int).SetUint64(blockNumber), true, blockTime)

	// Calculate fork ID
	forkId := forkid.NewID(chainConfig, genesis, blockNumber, activationTime)
	forkIdStr := "0x" + hex.EncodeToString(forkId.Hash[:])

	// Build configuration
	config := &ForkConfig{
		ActivationTime:  activationTime,
		BlobSchedule:    extractBlobSchedule(chainConfig, blockNumber, blockTime),
		ChainID:         fmt.Sprintf("0x%x", chainConfig.ChainID.Uint64()),
		ForkID:          forkIdStr,
		Precompiles:     getActivePrecompiles(rules),
		SystemContracts: getSystemContracts(rules, chainConfig),
	}

	return config, nil
}

// ============================================================================
// System Contracts
// ============================================================================

// getSystemContracts returns the system contract addresses for the given chain rules
// formatted for EIP-7910 eth_config response.
func getSystemContracts(rules params.Rules, chainConfig *params.ChainConfig) map[string]common.Address {
	contracts := make(map[string]common.Address)

	// Deposit contract (Pre-merge, since this is post-merge only always load it)
	if chainConfig.DepositContractAddress != (common.Address{}) {
		contracts[DepositContractAddressName] = chainConfig.DepositContractAddress
	}

	if rules.IsCancun {
		// EIP-4788: Beacon block root in the EVM (activated in Cancun)
		contracts[BeaconRootsAddressName] = params.BeaconRootsAddress
	}

	if rules.IsPrague {
		// EIP-2935: Serve historical block hashes from state (activated in Prague)
		contracts[HistoryStorageAddressName] = params.HistoryStorageAddress
		// EIP-7002: Execution layer triggerable withdrawals (activated in Prague)
		contracts[WithdrawalRequestPredeployAddressName] = params.WithdrawalQueueAddress
		// EIP-7251: Increase the MAX_EFFECTIVE_BALANCE (activated in Prague)
		contracts[ConsolidationRequestPredeployAddressName] = params.ConsolidationQueueAddress
	}

	return contracts
}

// ============================================================================
// Precompiles
// ============================================================================

// getActivePrecompiles returns a map of precompile names to their addresses
// for the given chain rules, formatted for EIP-7910 eth_config response.
func getActivePrecompiles(rules params.Rules) map[string]string {
	addresses := vm.ActivePrecompiles(rules)
	precompiles := make(map[string]string, len(addresses))

	for _, addr := range addresses {
		if name, ok := precompileNames[addr]; ok {
			precompiles[name] = strings.ToLower(addr.Hex())
		}
	}

	return precompiles
}

// ============================================================================
// Activation Time Calculation
// ============================================================================

// calculateActivationTimeForFork calculates the activation time for a specific fork
func calculateActivationTimeForFork(chainConfig *params.ChainConfig, targetBlockTime uint64) (uint64, error) {
	fork := chainConfig.LatestFork(targetBlockTime)
	time := chainConfig.Timestamp(fork)
	if time == nil {
		return 0, fmt.Errorf("failed to calculate fork time, only post-merge forks expected")
	} else {
		return *time, nil
	}
}

// GetNextForkActivationTime returns the activation time of the next scheduled fork
func GetNextForkActivationTime(chainConfig *params.ChainConfig, currentBlockTime uint64) (uint64, error) {
	// Look for the next time-based fork that hasn't activated yet
	// no handy methods, we have to hand enumerate
	forks := []*uint64{
		chainConfig.ShanghaiTime,
		chainConfig.CancunTime,
		chainConfig.PragueTime,
		chainConfig.OsakaTime,
		chainConfig.BPO1Time,
		chainConfig.BPO2Time,
		chainConfig.BPO3Time,
		chainConfig.BPO4Time,
		chainConfig.BPO5Time,
	}
	//TODO should we hedge and sort? for testnets are BPOs guaranteed to keep number/named fork order?

	for _, fork := range forks {
		if fork != nil && *fork > currentBlockTime {
			return *fork, nil
		}
	}

	// No future fork scheduled
	return 0, ErrNoFutureFork
}

// GetLastKnownForkActivationTime returns the activation time of the last known fork
func GetLastKnownForkActivationTime(chainConfig *params.ChainConfig) (uint64, error) {
	time := chainConfig.Timestamp(chainConfig.LatestFork(math.MaxUint64))
	if time == nil {
		// No time-based forks configured - this shouldn't happen in practice
		return 0, fmt.Errorf("no time-based forks configured")
	} else {
		return *time, nil
	}
}

// ============================================================================
// Blob Configuration
// ============================================================================

// extractBlobSchedule extracts blob configuration parameters for a specific fork
// based on chain configuration and activation rules.
func extractBlobSchedule(chainConfig *params.ChainConfig, blockNumber uint64, blockTime uint64) BlobScheduleParams {
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
