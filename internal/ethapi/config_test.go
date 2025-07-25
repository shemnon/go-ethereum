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
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// ============================================================================
// Configuration Hashing Tests
// ============================================================================

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

	hash, err := hashConfig(config)
	if err != nil {
		t.Fatalf("hashConfig failed: %v", err)
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

	hash1, err := hashConfig(config)
	if err != nil {
		t.Fatalf("First hashConfig failed: %v", err)
	}

	hash2, err := hashConfig(config)
	if err != nil {
		t.Fatalf("Second hashConfig failed: %v", err)
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

	hash1, err := hashConfig(config1)
	if err != nil {
		t.Fatalf("First hashConfig failed: %v", err)
	}

	hash2, err := hashConfig(config2)
	if err != nil {
		t.Fatalf("Second hashConfig failed: %v", err)
	}

	if hash1 == hash2 {
		t.Errorf("Different configs produced same hash: %s", hash1)
	}
}

func TestHashConfigNilConfig(t *testing.T) {
	_, err := hashConfig(nil)
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

	hash, err := hashConfig(config)
	if err != nil {
		t.Fatalf("hashConfig failed: %v", err)
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

	hash, err := hashConfig(config)
	if err != nil {
		t.Fatalf("hashConfig failed: %v", err)
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

	hash, err := hashConfig(config)
	if err != nil {
		t.Fatalf("hashConfig failed: %v", err)
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

	hash, err := hashConfig(config)
	if err != nil {
		t.Fatalf("hashConfig failed: %v", err)
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

	hash, err := hashConfig(config)
	if err != nil {
		t.Fatalf("hashConfig failed: %v", err)
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

	hash1, err := hashConfig(config1)
	if err != nil {
		t.Fatalf("First hashConfig failed: %v", err)
	}

	hash2, err := hashConfig(config2)
	if err != nil {
		t.Fatalf("Second hashConfig failed: %v", err)
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

	hash, err := hashConfig(config)
	if err != nil {
		t.Fatalf("hashConfig failed: %v", err)
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

	hash, err := hashConfig(config)
	if err != nil {
		t.Fatalf("hashConfig failed: %v", err)
	}

	expectedHash := "0xd80b437d"
	if hash != expectedHash {
		t.Errorf("EIP-7910 Sample 2 hash = %s, want %s", hash, expectedHash)
	}
}

// ============================================================================
// Blob Configuration Tests
// ============================================================================

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
			result := extractBlobSchedule(tt.chainConfig, tt.blockNumber, tt.blockTime)

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

// ============================================================================
// Precompile Detection Tests
// ============================================================================

func TestGetActivePrecompiles(t *testing.T) {
	// Test Homestead rules (basic precompiles)
	homesteadRules := params.Rules{
		ChainID:  big.NewInt(1),
		IsLondon: false,
		IsBerlin: false,
	}

	homesteadPrecompiles := getActivePrecompiles(homesteadRules)
	expected := map[string]string{
		"0x0000000000000000000000000000000000000001": "ECREC",
		"0x0000000000000000000000000000000000000002": "SHA256",
		"0x0000000000000000000000000000000000000003": "RIPEMD160",
		"0x0000000000000000000000000000000000000004": "ID",
	}

	if !reflect.DeepEqual(homesteadPrecompiles, expected) {
		t.Errorf("Homestead precompiles mismatch.\nGot: %v\nWant: %v", homesteadPrecompiles, expected)
	}
}

func TestGetActivePrecompilesForFork(t *testing.T) {
	chainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(1000),
		ConstantinopleBlock: big.NewInt(1000),
		PetersburgBlock:     big.NewInt(1000),
		IstanbulBlock:       big.NewInt(2000),
		BerlinBlock:         big.NewInt(3000),
		LondonBlock:         big.NewInt(3000),
		MergeNetsplitBlock:  big.NewInt(3000),
		ShanghaiTime:        &[]uint64{3500}[0],
		CancunTime:          &[]uint64{4000}[0],
		PragueTime:          &[]uint64{5000}[0],
	}

	tests := []struct {
		name        string
		blockNumber uint64
		blockTime   uint64
		expectedLen int
		mustHave    []string
	}{
		{
			name:        "homestead",
			blockNumber: 500,
			blockTime:   500,
			expectedLen: 4,
			mustHave:    []string{"ECREC", "SHA256", "RIPEMD160", "ID"},
		},
		{
			name:        "byzantium",
			blockNumber: 1500,
			blockTime:   1500,
			expectedLen: 8,
			mustHave:    []string{"ECREC", "SHA256", "RIPEMD160", "ID", "MODEXP", "BN254_ADD", "BN254_MUL", "BN254_PAIRING"},
		},
		{
			name:        "istanbul",
			blockNumber: 2500,
			blockTime:   2500,
			expectedLen: 9,
			mustHave:    []string{"ECREC", "SHA256", "RIPEMD160", "ID", "MODEXP", "BN254_ADD", "BN254_MUL", "BN254_PAIRING", "BLAKE2F"},
		},
		{
			name:        "berlin",
			blockNumber: 3500,
			blockTime:   3500,
			expectedLen: 9, // Berlin doesn't add new precompiles
			mustHave:    []string{"ECREC", "SHA256", "RIPEMD160", "ID", "MODEXP", "BN254_ADD", "BN254_MUL", "BN254_PAIRING", "BLAKE2F"},
		},
		{
			name:        "cancun",
			blockNumber: 4500,
			blockTime:   4500,
			expectedLen: 10,
			mustHave:    []string{"ECREC", "SHA256", "RIPEMD160", "ID", "MODEXP", "BN254_ADD", "BN254_MUL", "BN254_PAIRING", "BLAKE2F", "KZG_POINT_EVALUATION"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := chainConfig.Rules(new(big.Int).SetUint64(tt.blockNumber), true, tt.blockTime)
			precompiles := getActivePrecompiles(rules)

			if len(precompiles) != tt.expectedLen {
				t.Errorf("Expected %d precompiles, got %d", tt.expectedLen, len(precompiles))
			}

			// Check that all required precompiles are present
			found := make(map[string]bool)
			for _, name := range precompiles {
				found[name] = true
			}

			for _, required := range tt.mustHave {
				if !found[required] {
					t.Errorf("Missing required precompile: %s", required)
				}
			}
		})
	}
}

func TestGetPrecompileName(t *testing.T) {
	tests := []struct {
		name     string
		address  common.Address
		expected string
	}{
		{
			name:     "ECREC",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000001"),
			expected: "ECREC",
		},
		{
			name:     "SHA256",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000002"),
			expected: "SHA256",
		},
		{
			name:     "RIPEMD160",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000003"),
			expected: "RIPEMD160",
		},
		{
			name:     "ID",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000004"),
			expected: "ID",
		},
		{
			name:     "MODEXP",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000005"),
			expected: "MODEXP",
		},
		{
			name:     "BN254_ADD",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000006"),
			expected: "BN254_ADD",
		},
		{
			name:     "BN254_MUL",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000007"),
			expected: "BN254_MUL",
		},
		{
			name:     "BN254_PAIRING",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000008"),
			expected: "BN254_PAIRING",
		},
		{
			name:     "BLAKE2F",
			address:  common.HexToAddress("0x0000000000000000000000000000000000000009"),
			expected: "BLAKE2F",
		},
		{
			name:     "KZG_POINT_EVALUATION",
			address:  common.HexToAddress("0x000000000000000000000000000000000000000a"),
			expected: "KZG_POINT_EVALUATION",
		},
		{
			name:     "unknown address",
			address:  common.HexToAddress("0x1000000000000000000000000000000000000000"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := precompileNames[tt.address]
			if result != tt.expected {
				t.Errorf("getPrecompileName(%s) = %q, want %q", tt.address.Hex(), result, tt.expected)
			}
		})
	}
}

// ============================================================================
// Canonical JSON Tests
// ============================================================================

func TestCanonicalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: `"hello"`,
		},
		{
			name:     "integer",
			input:    42,
			expected: `42`,
		},
		{
			name:     "boolean true",
			input:    true,
			expected: `true`,
		},
		{
			name:     "boolean false",
			input:    false,
			expected: `false`,
		},
		{
			name:     "null",
			input:    nil,
			expected: `null`,
		},
		{
			name:     "empty object",
			input:    map[string]interface{}{},
			expected: `{}`,
		},
		{
			name: "simple object",
			input: map[string]interface{}{
				"b": 2,
				"a": 1,
			},
			expected: `{"a":1,"b":2}`,
		},
		{
			name: "nested object",
			input: map[string]interface{}{
				"z": map[string]interface{}{
					"y": 2,
					"x": 1,
				},
				"a": 3,
			},
			expected: `{"a":3,"z":{"x":1,"y":2}}`,
		},
		{
			name:     "array",
			input:    []interface{}{3, 1, 2},
			expected: `[3,1,2]`,
		},
		{
			name: "complex structure",
			input: map[string]interface{}{
				"precompiles": map[string]string{
					"0x0000000000000000000000000000000000000002": "SHA256",
					"0x0000000000000000000000000000000000000001": "ECREC",
				},
				"chainId":        "0x1",
				"activationTime": 0,
			},
			expected: `{"activationTime":0,"chainId":"0x1","precompiles":{"0x0000000000000000000000000000000000000001":"ECREC","0x0000000000000000000000000000000000000002":"SHA256"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := canonicalJSON(tt.input)
			if err != nil {
				t.Fatalf("canonicalJSON failed: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("canonicalJSON() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

func TestCanonicalJSONStruct(t *testing.T) {
	type TestStruct struct {
		Z string `json:"z"`
		A int    `json:"a"`
		B string `json:"b,omitempty"`
	}

	tests := []struct {
		name     string
		input    TestStruct
		expected string
	}{
		{
			name:     "struct with all fields",
			input:    TestStruct{Z: "last", A: 1, B: "middle"},
			expected: `{"a":1,"b":"middle","z":"last"}`,
		},
		{
			name:     "struct with omitempty field empty",
			input:    TestStruct{Z: "last", A: 1, B: ""},
			expected: `{"a":1,"z":"last"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := canonicalJSON(tt.input)
			if err != nil {
				t.Fatalf("canonicalJSON failed: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("canonicalJSON() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

func TestCanonicalJSONForkConfig(t *testing.T) {
	config := &ForkConfig{
		ActivationTime: 1234567890,
		BlobSchedule: BlobScheduleParams{
			BaseFeeUpdateFraction: 3338477,
			Max:                   6,
			Target:                3,
		},
		ChainID: "0x1",
		Precompiles: map[string]string{
			"0x0000000000000000000000000000000000000002": "SHA256",
			"0x0000000000000000000000000000000000000001": "ECREC",
		},
		SystemContracts: map[string]common.Address{
			"BEACON_ROOTS_ADDRESS": common.HexToAddress("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02"),
		},
	}

	result, err := canonicalJSON(config)
	if err != nil {
		t.Fatalf("canonicalJSON failed: %v", err)
	}

	// Verify that the JSON is properly ordered and formatted
	expected := `{"activationTime":1234567890,"blobSchedule":{"baseFeeUpdateFraction":3338477,"max":6,"target":3},"chainId":"0x1","precompiles":{"0x0000000000000000000000000000000000000001":"ECREC","0x0000000000000000000000000000000000000002":"SHA256"},"systemContracts":{"BEACON_ROOTS_ADDRESS":"0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02"}}`

	if string(result) != expected {
		t.Errorf("canonicalJSON() for ForkConfig:\ngot:  %s\nwant: %s", string(result), expected)
	}
}

func TestCanonicalJSONConsistency(t *testing.T) {
	// Test that the same input always produces the same output
	config := map[string]interface{}{
		"c": 3,
		"a": 1,
		"b": 2,
	}

	result1, err := canonicalJSON(config)
	if err != nil {
		t.Fatalf("First canonicalJSON failed: %v", err)
	}

	result2, err := canonicalJSON(config)
	if err != nil {
		t.Fatalf("Second canonicalJSON failed: %v", err)
	}

	if string(result1) != string(result2) {
		t.Errorf("canonicalJSON results differ: %s != %s", string(result1), string(result2))
	}
}
