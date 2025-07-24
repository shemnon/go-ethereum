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
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// ============================================================================
// EIP-7910 Types and Constants
// ============================================================================

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

// ============================================================================
// Fork Configuration Building
// ============================================================================

// BuildForkConfig constructs a complete fork configuration for the given block number and time
func BuildForkConfig(chainConfig *params.ChainConfig, blockNumber uint64, blockTime uint64) (*ForkConfig, error) {
	// Calculate activation time for this fork
	activationTime, err := calculateActivationTimeForFork(chainConfig, blockNumber, blockTime)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate activation time: %w", err)
	}

	// Get chain rules for this block
	rules := chainConfig.Rules(new(big.Int).SetUint64(blockNumber), true, blockTime)

	// Build configuration
	config := &ForkConfig{
		ActivationTime:  activationTime,
		BlobSchedule:    extractBlobSchedule(chainConfig, blockNumber, blockTime),
		ChainID:         fmt.Sprintf("0x%x", chainConfig.ChainID.Uint64()),
		Precompiles:     getActivePrecompiles(rules),
		SystemContracts: getSystemContracts(rules, chainConfig),
	}

	return config, nil
}

// ============================================================================
// Configuration Hashing
// ============================================================================

// hashConfig computes the CRC-32 hash of a fork configuration as specified by EIP-7910.
// The configuration is first serialized to canonical JSON (RFC-8785) and then hashed.
func hashConfig(config *ForkConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("config cannot be nil")
	}

	// Serialize to canonical JSON
	canonical, err := canonicalJSON(config)
	if err != nil {
		return "", fmt.Errorf("failed to serialize config to canonical JSON: %w", err)
	}

	// Compute CRC-32 hash
	hash := crc32.ChecksumIEEE(canonical)

	// Return as hex string with 0x prefix
	return fmt.Sprintf("0x%08x", hash), nil
}

// ============================================================================
// Canonical JSON Implementation (RFC-8785)
// ============================================================================

// canonicalJSON produces canonical JSON as per RFC-8785 for deterministic hashing.
// This implementation ensures:
// - No whitespace except inside strings
// - Object keys sorted lexicographically
// - Numeric values in simplest form
// - No trailing zeros after decimal point
func canonicalJSON(v interface{}) ([]byte, error) {
	return canonicalMarshal(reflect.ValueOf(v))
}

// canonicalMarshal recursively marshals a value to canonical JSON
//
//nolint:exhaustive // Unsupported types handled in default case
func canonicalMarshal(v reflect.Value) ([]byte, error) {
	// Handle invalid values first
	if !v.IsValid() {
		return []byte("null"), nil
	}

	// Special handling for common.Address type
	if v.Type() == reflect.TypeOf(common.Address{}) {
		addr := v.Interface().(common.Address)
		return json.Marshal(addr.Hex())
	}

	switch v.Kind() {
	case reflect.Invalid:
		return []byte("null"), nil
	case reflect.Bool:
		if v.Bool() {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []byte(strconv.FormatInt(v.Int(), 10)), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []byte(strconv.FormatUint(v.Uint(), 10)), nil
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		// Use the simplest form - remove trailing zeros
		s := strconv.FormatFloat(f, 'f', -1, 64)
		return []byte(s), nil
	case reflect.String:
		return json.Marshal(v.String())
	case reflect.Array, reflect.Slice:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return []byte("null"), nil
		}
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				buf.WriteByte(',')
			}
			elemData, err := canonicalMarshal(v.Index(i))
			if err != nil {
				return nil, err
			}
			buf.Write(elemData)
		}
		buf.WriteByte(']')
		return buf.Bytes(), nil
	case reflect.Map:
		if v.IsNil() {
			return []byte("null"), nil
		}
		return canonicalMarshalMap(v)
	case reflect.Struct:
		return canonicalMarshalStruct(v)
	case reflect.Ptr:
		if v.IsNil() {
			return []byte("null"), nil
		}
		return canonicalMarshal(v.Elem())
	case reflect.Interface:
		if v.IsNil() {
			return []byte("null"), nil
		}
		return canonicalMarshal(v.Elem())
	default:
		return nil, fmt.Errorf("unsupported type: %v", v.Type())
	}
}

// canonicalMarshalMap marshals a map with keys sorted lexicographically
func canonicalMarshalMap(v reflect.Value) ([]byte, error) {
	keys := v.MapKeys()
	if len(keys) == 0 {
		return []byte("{}"), nil
	}

	// Convert keys to strings and sort them
	keyStrings := make([]string, len(keys))
	keyMap := make(map[string]reflect.Value)

	for i, key := range keys {
		var keyStr string
		switch key.Kind() {
		case reflect.String:
			keyStr = key.String()
		default:
			keyData, err := canonicalMarshal(key)
			if err != nil {
				return nil, err
			}
			keyStr = string(keyData)
			// Remove quotes for non-string keys when using as map key
			if strings.HasPrefix(keyStr, `"`) && strings.HasSuffix(keyStr, `"`) {
				keyStr = keyStr[1 : len(keyStr)-1]
			}
		}
		keyStrings[i] = keyStr
		keyMap[keyStr] = key
	}

	sort.Strings(keyStrings)

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, keyStr := range keyStrings {
		if i > 0 {
			buf.WriteByte(',')
		}

		// Marshal the key as a JSON string
		keyJSON, err := json.Marshal(keyStr)
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		buf.WriteByte(':')

		// Marshal the value
		value := v.MapIndex(keyMap[keyStr])
		valueData, err := canonicalMarshal(value)
		if err != nil {
			return nil, err
		}
		buf.Write(valueData)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// canonicalMarshalStruct marshals a struct with JSON tags, sorted by field names
func canonicalMarshalStruct(v reflect.Value) ([]byte, error) {
	t := v.Type()

	type field struct {
		name      string
		value     reflect.Value
		omitEmpty bool
	}

	var fields []field

	for i := 0; i < v.NumField(); i++ {
		fieldValue := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		// Get JSON tag
		tag := fieldType.Tag.Get("json")
		if tag == "-" {
			continue
		}

		name := fieldType.Name
		omitEmpty := false

		if tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" {
				name = parts[0]
			}
			for _, part := range parts[1:] {
				if part == "omitempty" {
					omitEmpty = true
					break
				}
			}
		}

		// Skip if omitempty and value is empty
		if omitEmpty && isEmptyValue(fieldValue) {
			continue
		}

		fields = append(fields, field{
			name:      name,
			value:     fieldValue,
			omitEmpty: omitEmpty,
		})
	}

	// Sort fields by name
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].name < fields[j].name
	})

	if len(fields) == 0 {
		return []byte("{}"), nil
	}

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, field := range fields {
		if i > 0 {
			buf.WriteByte(',')
		}

		// Marshal field name
		nameJSON, err := json.Marshal(field.name)
		if err != nil {
			return nil, err
		}
		buf.Write(nameJSON)
		buf.WriteByte(':')

		// Marshal field value
		valueData, err := canonicalMarshal(field.value)
		if err != nil {
			return nil, err
		}
		buf.Write(valueData)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// isEmptyValue checks if a value is considered empty for omitempty
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	default:
		return true // if we don't know what it is, don't jsonify it
	}
}

// ============================================================================
// System Contracts
// ============================================================================

// getSystemContracts returns the system contract addresses for the given chain rules
// formatted for EIP-7910 eth_config response.
func getSystemContracts(rules params.Rules, chainConfig *params.ChainConfig) map[string]common.Address {
	contracts := make(map[string]common.Address)

	// EIP-4788: Beacon block root in the EVM (activated in Cancun)
	if rules.IsCancun {
		contracts[BeaconRootsAddressName] = params.BeaconRootsAddress
	}

	// EIP-2935: Serve historical block hashes from state (activated in Prague)
	if rules.IsPrague {
		contracts[HistoryStorageAddressName] = params.HistoryStorageAddress
	}

	// EIP-7002: Execution layer triggerable withdrawals (activated in Prague)
	if rules.IsPrague {
		contracts[WithdrawalRequestPredeployAddressName] = params.WithdrawalQueueAddress
	}

	// EIP-7251: Increase the MAX_EFFECTIVE_BALANCE (activated in Prague)
	if rules.IsPrague {
		contracts[ConsolidationRequestPredeployAddressName] = params.ConsolidationQueueAddress
	}

	// Deposit contract (available since genesis on mainnet, varies by network)
	if chainConfig.DepositContractAddress != (common.Address{}) {
		contracts[DepositContractAddressName] = chainConfig.DepositContractAddress
	}

	return contracts
}

// ============================================================================
// Precompiles
// ============================================================================

// getActivePrecompiles returns a map of precompile addresses to their names
// for the given chain rules, formatted for EIP-7910 eth_config response.
func getActivePrecompiles(rules params.Rules) map[string]string {
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
func getActivePrecompilesForFork(chainConfig *params.ChainConfig, blockNumber uint64, blockTime uint64) map[string]string {
	rules := chainConfig.Rules(new(big.Int).SetUint64(blockNumber), true, blockTime)
	return getActivePrecompiles(rules)
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

// ============================================================================
// Activation Time Calculation
// ============================================================================

// calculateActivationTimeForFork calculates the activation time for a specific fork
func calculateActivationTimeForFork(chainConfig *params.ChainConfig, targetBlockNumber uint64, targetBlockTime uint64) (uint64, error) {
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
