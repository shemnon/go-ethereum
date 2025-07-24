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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/forkid"
)

// EthConfigResponse represents the response structure for the eth_config JSON-RPC method
// as specified by EIP-7910.
type EthConfigResponse struct {
	Current       *ForkConfig `json:"current"`
	CurrentHash   string      `json:"currentHash"`
	CurrentForkId *forkid.ID  `json:"currentForkId"`
	Next          *ForkConfig `json:"next,omitempty"`
	NextHash      string      `json:"nextHash,omitempty"`
	NextForkId    *forkid.ID  `json:"nextForkId,omitempty"`
	Last          *ForkConfig `json:"last,omitempty"`
	LastHash      string      `json:"lastHash,omitempty"`
	LastForkId    *forkid.ID  `json:"lastForkId,omitempty"`
}

// ForkConfig represents the configuration of a specific fork as defined by EIP-7910.
// All fields are required and must be present in canonical order for proper hashing.
type ForkConfig struct {
	ActivationTime  uint64                    `json:"activationTime"`
	BlobSchedule    BlobScheduleParams        `json:"blobSchedule"`
	ChainID         string                    `json:"chainId"`
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

// PrecompileName constants for the known precompiles as defined by EIP-7910
const (
	PrecompileECREC                = "ECREC"
	PrecompileSHA256               = "SHA256"
	PrecompileRIPEMD160            = "RIPEMD160"
	PrecompileID                   = "ID"
	PrecompileMODEXP               = "MODEXP"
	PrecompileBN254_ADD            = "BN254_ADD"
	PrecompileBN254_MUL            = "BN254_MUL"
	PrecompileBN254_PAIRING        = "BN254_PAIRING"
	PrecompileBLAKE2F              = "BLAKE2F"
	PrecompileKZG_POINT_EVALUATION = "KZG_POINT_EVALUATION"
	PrecompileBLS12_G1ADD          = "BLS12_G1ADD"
	PrecompileBLS12_G1MSM          = "BLS12_G1MSM"
	PrecompileBLS12_G2ADD          = "BLS12_G2ADD"
	PrecompileBLS12_G2MSM          = "BLS12_G2MSM"
	PrecompileBLS12_PAIRING_CHECK  = "BLS12_PAIRING_CHECK"
	PrecompileBLS12_MAP_FP_TO_G1   = "BLS12_MAP_FP_TO_G1"
	PrecompileBLS12_MAP_FP2_TO_G2  = "BLS12_MAP_FP2_TO_G2"
)
