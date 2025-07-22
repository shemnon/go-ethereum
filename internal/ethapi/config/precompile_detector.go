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
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// GetActivePrecompiles returns a map of precompile addresses to their names
// for the given chain rules, formatted for EIP-7910 eth_config response.
func GetActivePrecompiles(rules params.Rules) map[string]string {
	addresses := vm.ActivePrecompiles(rules)
	precompiles := make(map[string]string, len(addresses))

	for _, addr := range addresses {
		name := getPrecompileName(addr)
		if name != "" {
			precompiles[strings.ToLower(addr.Hex())] = name
		}
	}

	return precompiles
}

// GetActivePrecompilesForFork returns the active precompiles for a specific fork
// based on chain configuration and activation time or block number.
func GetActivePrecompilesForFork(chainConfig *params.ChainConfig, blockNumber uint64, blockTime uint64) map[string]string {
	rules := chainConfig.Rules(new(big.Int).SetUint64(blockNumber), true, blockTime)
	return GetActivePrecompiles(rules)
}

// getPrecompileName returns the canonical name for a precompile address as defined by EIP-7910
func getPrecompileName(addr common.Address) string {
	// Convert address to byte slice for comparison
	addrBytes := addr.Bytes()

	// Check for known precompile addresses based on the last significant bytes
	switch {
	case isAddressEqual(addrBytes, []byte{0x1}):
		return PrecompileECREC
	case isAddressEqual(addrBytes, []byte{0x2}):
		return PrecompileSHA256
	case isAddressEqual(addrBytes, []byte{0x3}):
		return PrecompileRIPEMD160
	case isAddressEqual(addrBytes, []byte{0x4}):
		return PrecompileID
	case isAddressEqual(addrBytes, []byte{0x5}):
		return PrecompileMODEXP
	case isAddressEqual(addrBytes, []byte{0x6}):
		return PrecompileBN254_ADD
	case isAddressEqual(addrBytes, []byte{0x7}):
		return PrecompileBN254_MUL
	case isAddressEqual(addrBytes, []byte{0x8}):
		return PrecompileBN254_PAIRING
	case isAddressEqual(addrBytes, []byte{0x9}):
		return PrecompileBLAKE2F
	case isAddressEqual(addrBytes, []byte{0xa}):
		return PrecompileKZG_POINT_EVALUATION
	case isAddressEqual(addrBytes, []byte{0xb}):
		return PrecompileBLS12_G1ADD
	case isAddressEqual(addrBytes, []byte{0xc}):
		return PrecompileBLS12_G1MSM
	case isAddressEqual(addrBytes, []byte{0xd}):
		return PrecompileBLS12_G2ADD
	case isAddressEqual(addrBytes, []byte{0xe}):
		return PrecompileBLS12_G2MSM
	case isAddressEqual(addrBytes, []byte{0xf}):
		return PrecompileBLS12_PAIRING_CHECK
	case isAddressEqual(addrBytes, []byte{0x10}):
		return PrecompileBLS12_MAP_FP_TO_G1
	case isAddressEqual(addrBytes, []byte{0x11}):
		return PrecompileBLS12_MAP_FP2_TO_G2
	case isAddressEqual(addrBytes, []byte{0x1, 0x00}):
		return "P256VERIFY" // EIP-7212 precompile
	default:
		// Unknown precompile - this shouldn't happen with well-known addresses
		// but we return empty string to filter it out
		return ""
	}
}

// isAddressEqual checks if an address (20 bytes) ends with the given suffix bytes
func isAddressEqual(fullAddr []byte, suffix []byte) bool {
	if len(fullAddr) != 20 || len(suffix) == 0 || len(suffix) > 20 {
		return false
	}

	// Check if the suffix matches the end of the address
	start := 20 - len(suffix)
	for i := 0; i < len(suffix); i++ {
		if fullAddr[start+i] != suffix[i] {
			return false
		}
	}

	// Check that all preceding bytes are zero
	for i := 0; i < start; i++ {
		if fullAddr[i] != 0 {
			return false
		}
	}

	return true
}
