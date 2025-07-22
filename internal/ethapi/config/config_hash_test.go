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
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestHashConfig(t *testing.T) {
	config := &ForkConfig{
		ActivationTime: 1234567890,
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 3338477,
			Max:                   6,
			Target:                3,
		},
		ChainID: "0x1",
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000001": "ECREC",
			"0x0000000000000000000000000000000000000002": "SHA256",
		},
		SystemContracts: map[string]common.Address{
			"BEACON_ROOTS_ADDRESS": common.HexToAddress("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02"),
		},
	}

	hash, err := HashConfig(config)
	if err != nil {
		t.Fatalf("HashConfig failed: %v", err)
	}

	// Verify hash format
	if len(hash) != 10 { // "0x" + 8 hex characters
		t.Errorf("Expected hash length 10, got %d", len(hash))
	}

	if hash[:2] != "0x" {
		t.Errorf("Expected hash to start with '0x', got %s", hash[:2])
	}
}

func TestHashConfigConsistency(t *testing.T) {
	config := &ForkConfig{
		ActivationTime: 1234567890,
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 3338477,
			Max:                   6,
			Target:                3,
		},
		ChainID: "0x1",
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000001": "ECREC",
		},
		SystemContracts: map[string]common.Address{},
	}

	hash1, err := HashConfig(config)
	if err != nil {
		t.Fatalf("First HashConfig failed: %v", err)
	}

	hash2, err := HashConfig(config)
	if err != nil {
		t.Fatalf("Second HashConfig failed: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Hash consistency failed: %s != %s", hash1, hash2)
	}
}

func TestHashConfigDifferentConfigs(t *testing.T) {
	config1 := &ForkConfig{
		ActivationTime: 1234567890,
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 3338477,
			Max:                   6,
			Target:                3,
		},
		ChainID: "0x1",
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000001": "ECREC",
		},
		SystemContracts: map[string]common.Address{},
	}

	config2 := &ForkConfig{
		ActivationTime: 1234567890,
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 3338477,
			Max:                   6,
			Target:                3,
		},
		ChainID: "0x1",
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000001": "ECREC",
			"0x0000000000000000000000000000000000000002": "SHA256", // Additional precompile
		},
		SystemContracts: map[string]common.Address{},
	}

	hash1, err := HashConfig(config1)
	if err != nil {
		t.Fatalf("First HashConfig failed: %v", err)
	}

	hash2, err := HashConfig(config2)
	if err != nil {
		t.Fatalf("Second HashConfig failed: %v", err)
	}

	if hash1 == hash2 {
		t.Errorf("Different configs produced same hash: %s", hash1)
	}
}

func TestHashConfigNilConfig(t *testing.T) {
	_, err := HashConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config, got nil")
	}
}

// Test cases for different fork configurations that would be typical for EIP-7910

func TestHashConfig_HomesteadFork(t *testing.T) {
	// Typical Homestead fork configuration
	config := &ForkConfig{
		ActivationTime: 0, // Genesis
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 0,
			Max:                   0,
			Target:                0,
		},
		ChainID: "0x1",
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000001": "ECREC",
			"0x0000000000000000000000000000000000000002": "SHA256",
			"0x0000000000000000000000000000000000000003": "RIPEMD160",
			"0x0000000000000000000000000000000000000004": "ID",
		},
		SystemContracts: map[string]common.Address{},
	}

	hash, err := HashConfig(config)
	if err != nil {
		t.Fatalf("HashConfig failed: %v", err)
	}

	// Expected hash should be deterministic
	expectedHash := "0x4b620b96"
	if hash != expectedHash {
		t.Errorf("Homestead config hash = %s, want %s", hash, expectedHash)
	}
}

func TestHashConfig_CancunFork(t *testing.T) {
	// Typical Cancun fork configuration with blobs
	config := &ForkConfig{
		ActivationTime: 1710338135, // March 13, 2024 - Mainnet Dencun activation
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 3338477,
			Max:                   6,
			Target:                3,
		},
		ChainID: "0x1",
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000001": "ECREC",
			"0x0000000000000000000000000000000000000002": "SHA256",
			"0x0000000000000000000000000000000000000003": "RIPEMD160",
			"0x0000000000000000000000000000000000000004": "ID",
			"0x0000000000000000000000000000000000000005": "MODEXP",
			"0x0000000000000000000000000000000000000006": "BN254_ADD",
			"0x0000000000000000000000000000000000000007": "BN254_MUL",
			"0x0000000000000000000000000000000000000008": "BN254_PAIRING",
			"0x0000000000000000000000000000000000000009": "BLAKE2F",
			"0x000000000000000000000000000000000000000a": "KZG_POINT_EVALUATION",
		},
		SystemContracts: map[string]common.Address{
			"BEACON_ROOTS_ADDRESS": common.HexToAddress("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02"),
		},
	}

	hash, err := HashConfig(config)
	if err != nil {
		t.Fatalf("HashConfig failed: %v", err)
	}

	// Expected hash should be deterministic
	expectedHash := "0xff060729"
	if hash != expectedHash {
		t.Errorf("Cancun config hash = %s, want %s", hash, expectedHash)
	}
}

func TestHashConfig_PragueFork(t *testing.T) {
	// Future Prague fork configuration with more system contracts
	config := &ForkConfig{
		ActivationTime: 1730000000, // Future timestamp
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 5007716, // Updated in EIP-7691
			Max:                   9,
			Target:                6,
		},
		ChainID: "0x1",
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000001": "ECREC",
			"0x0000000000000000000000000000000000000002": "SHA256",
			"0x0000000000000000000000000000000000000003": "RIPEMD160",
			"0x0000000000000000000000000000000000000004": "ID",
			"0x0000000000000000000000000000000000000005": "MODEXP",
			"0x0000000000000000000000000000000000000006": "BN254_ADD",
			"0x0000000000000000000000000000000000000007": "BN254_MUL",
			"0x0000000000000000000000000000000000000008": "BN254_PAIRING",
			"0x0000000000000000000000000000000000000009": "BLAKE2F",
			"0x000000000000000000000000000000000000000a": "KZG_POINT_EVALUATION",
			"0x000000000000000000000000000000000000000b": "BLS12_G1ADD",
			"0x000000000000000000000000000000000000000c": "BLS12_G1MSM",
			"0x000000000000000000000000000000000000000d": "BLS12_G2ADD",
			"0x000000000000000000000000000000000000000e": "BLS12_G2MSM",
			"0x000000000000000000000000000000000000000f": "BLS12_PAIRING_CHECK",
			"0x0000000000000000000000000000000000000010": "BLS12_MAP_FP_TO_G1",
			"0x0000000000000000000000000000000000000011": "BLS12_MAP_FP2_TO_G2",
		},
		SystemContracts: map[string]common.Address{
			"BEACON_ROOTS_ADDRESS":                    common.HexToAddress("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02"),
			"HISTORY_STORAGE_ADDRESS":                 common.HexToAddress("0x0AAE40965E6800CD9b1F4B05FF21581047E3f91e"),
			"WITHDRAWAL_REQUEST_PREDEPLOY_ADDRESS":    common.HexToAddress("0x00A3ca265EBcb825B45F985A16CEFB49958cE017"),
			"CONSOLIDATION_REQUEST_PREDEPLOY_ADDRESS": common.HexToAddress("0x00b42dbF2194e931E80326D950320f7d9Dbeac02"),
			"DEPOSIT_CONTRACT_ADDRESS":                common.HexToAddress("0x00000000219ab540356cBB839Cbe05303d7705Fa"),
		},
	}

	hash, err := HashConfig(config)
	if err != nil {
		t.Fatalf("HashConfig failed: %v", err)
	}

	// Expected hash should be deterministic
	expectedHash := "0x604c3d19"
	if hash != expectedHash {
		t.Errorf("Prague config hash = %s, want %s", hash, expectedHash)
	}
}

func TestHashConfig_TestnetConfig(t *testing.T) {
	// Typical testnet (Sepolia) configuration
	config := &ForkConfig{
		ActivationTime: 1706655072, // Sepolia Dencun activation
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 3338477,
			Max:                   6,
			Target:                3,
		},
		ChainID: "0xaa36a7", // Sepolia chain ID
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000001": "ECREC",
			"0x0000000000000000000000000000000000000002": "SHA256",
			"0x0000000000000000000000000000000000000003": "RIPEMD160",
			"0x0000000000000000000000000000000000000004": "ID",
			"0x0000000000000000000000000000000000000005": "MODEXP",
			"0x0000000000000000000000000000000000000006": "BN254_ADD",
			"0x0000000000000000000000000000000000000007": "BN254_MUL",
			"0x0000000000000000000000000000000000000008": "BN254_PAIRING",
			"0x0000000000000000000000000000000000000009": "BLAKE2F",
			"0x000000000000000000000000000000000000000a": "KZG_POINT_EVALUATION",
		},
		SystemContracts: map[string]common.Address{
			"BEACON_ROOTS_ADDRESS": common.HexToAddress("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02"),
		},
	}

	hash, err := HashConfig(config)
	if err != nil {
		t.Fatalf("HashConfig failed: %v", err)
	}

	// Expected hash should be deterministic
	expectedHash := "0xb006dc70"
	if hash != expectedHash {
		t.Errorf("Testnet config hash = %s, want %s", hash, expectedHash)
	}
}

func TestHashConfig_EmptyConfig(t *testing.T) {
	// Minimal configuration
	config := &ForkConfig{
		ActivationTime:  0,
		BlobSchedule:    BlobScheduleParams{},
		ChainID:         "0x1",
		Precompiles:     map[string]string{},
		SystemContracts: map[string]common.Address{},
	}

	hash, err := HashConfig(config)
	if err != nil {
		t.Fatalf("HashConfig failed: %v", err)
	}

	// Expected hash should be deterministic
	expectedHash := "0x2879b169"
	if hash != expectedHash {
		t.Errorf("Empty config hash = %s, want %s", hash, expectedHash)
	}
}

func TestHashConfig_FieldOrderIndependence(t *testing.T) {
	// Test that hash is independent of map iteration order
	// Create two identical configs
	config1 := &ForkConfig{
		ActivationTime: 1234567890,
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 3338477,
			Max:                   6,
			Target:                3,
		},
		ChainID: "0x1",
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000001": "ECREC",
			"0x0000000000000000000000000000000000000002": "SHA256",
			"0x0000000000000000000000000000000000000003": "RIPEMD160",
		},
		SystemContracts: map[string]common.Address{
			"BEACON_ROOTS_ADDRESS":    common.HexToAddress("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02"),
			"HISTORY_STORAGE_ADDRESS": common.HexToAddress("0x0AAE40965E6800CD9b1F4B05FF21581047E3f91e"),
		},
	}

	config2 := &ForkConfig{
		ActivationTime: 1234567890,
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 3338477,
			Max:                   6,
			Target:                3,
		},
		ChainID: "0x1",
		// Same precompiles in different order
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000003": "RIPEMD160",
			"0x0000000000000000000000000000000000000001": "ECREC",
			"0x0000000000000000000000000000000000000002": "SHA256",
		},
		// Same system contracts in different order
		SystemContracts: map[string]common.Address{
			"HISTORY_STORAGE_ADDRESS": common.HexToAddress("0x0AAE40965E6800CD9b1F4B05FF21581047E3f91e"),
			"BEACON_ROOTS_ADDRESS":    common.HexToAddress("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02"),
		},
	}

	hash1, err := HashConfig(config1)
	if err != nil {
		t.Fatalf("First HashConfig failed: %v", err)
	}

	hash2, err := HashConfig(config2)
	if err != nil {
		t.Fatalf("Second HashConfig failed: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Field order affected hash: %s != %s", hash1, hash2)
	}
}

// EIP-7910 specific test cases with expected hashes

func TestHashConfig_EIP7910_Sample1(t *testing.T) {
	config := &ForkConfig{
		ActivationTime: 0,
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 3338477,
			Max:                   6,
			Target:                3,
		},
		ChainID: "0x88bb0",
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000001": "ECREC",
			"0x0000000000000000000000000000000000000002": "SHA256",
			"0x0000000000000000000000000000000000000003": "RIPEMD160",
			"0x0000000000000000000000000000000000000004": "ID",
			"0x0000000000000000000000000000000000000005": "MODEXP",
			"0x0000000000000000000000000000000000000006": "BN254_ADD",
			"0x0000000000000000000000000000000000000007": "BN254_MUL",
			"0x0000000000000000000000000000000000000008": "BN254_PAIRING",
			"0x0000000000000000000000000000000000000009": "BLAKE2F",
			"0x000000000000000000000000000000000000000a": "KZG_POINT_EVALUATION",
		},
		SystemContracts: map[string]common.Address{
			"BEACON_ROOTS_ADDRESS": common.HexToAddress("0x000f3df6d732807ef1319fb7b8bb8522d0beac02"),
		},
	}

	hash, err := HashConfig(config)
	if err != nil {
		t.Fatalf("HashConfig failed: %v", err)
	}

	expectedHash := "0x919b73b0"
	if hash != expectedHash {
		t.Errorf("EIP-7910 Sample 1 hash = %s, want %s", hash, expectedHash)
	}
}

func TestHashConfig_EIP7910_Sample2(t *testing.T) {
	config := &ForkConfig{
		ActivationTime: 1742999832,
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 5007716,
			Max:                   9,
			Target:                6,
		},
		ChainID: "0x88bb0",
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000001": "ECREC",
			"0x0000000000000000000000000000000000000002": "SHA256",
			"0x0000000000000000000000000000000000000003": "RIPEMD160",
			"0x0000000000000000000000000000000000000004": "ID",
			"0x0000000000000000000000000000000000000005": "MODEXP",
			"0x0000000000000000000000000000000000000006": "BN254_ADD",
			"0x0000000000000000000000000000000000000007": "BN254_MUL",
			"0x0000000000000000000000000000000000000008": "BN254_PAIRING",
			"0x0000000000000000000000000000000000000009": "BLAKE2F",
			"0x000000000000000000000000000000000000000a": "KZG_POINT_EVALUATION",
			"0x000000000000000000000000000000000000000b": "BLS12_G1ADD",
			"0x000000000000000000000000000000000000000c": "BLS12_G1MSM",
			"0x000000000000000000000000000000000000000d": "BLS12_G2ADD",
			"0x000000000000000000000000000000000000000e": "BLS12_G2MSM",
			"0x000000000000000000000000000000000000000f": "BLS12_PAIRING_CHECK",
			"0x0000000000000000000000000000000000000010": "BLS12_MAP_FP_TO_G1",
			"0x0000000000000000000000000000000000000011": "BLS12_MAP_FP2_TO_G2",
		},
		SystemContracts: map[string]common.Address{
			"BEACON_ROOTS_ADDRESS":                    common.HexToAddress("0x000f3df6d732807ef1319fb7b8bb8522d0beac02"),
			"CONSOLIDATION_REQUEST_PREDEPLOY_ADDRESS": common.HexToAddress("0x0000bbddc7ce488642fb579f8b00f3a590007251"),
			"DEPOSIT_CONTRACT_ADDRESS":                common.HexToAddress("0x00000000219ab540356cbb839cbe05303d7705fa"),
			"HISTORY_STORAGE_ADDRESS":                 common.HexToAddress("0x0000f90827f1c53a10cb7a02335b175320002935"),
			"WITHDRAWAL_REQUEST_PREDEPLOY_ADDRESS":    common.HexToAddress("0x00000961ef480eb55e80d19ad83579a64c007002"),
		},
	}

	hash, err := HashConfig(config)
	if err != nil {
		t.Fatalf("HashConfig failed: %v", err)
	}

	expectedHash := "0xd80b437d"
	if hash != expectedHash {
		t.Errorf("EIP-7910 Sample 2 hash = %s, want %s", hash, expectedHash)
	}
}
