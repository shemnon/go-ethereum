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
	"hash/crc32"
)

// HashConfig computes the CRC-32 hash of a fork configuration as specified by EIP-7910.
// The configuration is first serialized to canonical JSON (RFC-8785) and then hashed.
func HashConfig(config *ForkConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("config cannot be nil")
	}

	// Serialize to canonical JSON
	canonical, err := CanonicalJSON(config)
	if err != nil {
		return "", fmt.Errorf("failed to serialize config to canonical JSON: %w", err)
	}

	// Compute CRC-32 hash
	hash := crc32.ChecksumIEEE(canonical)

	// Return as hex string with 0x prefix
	return fmt.Sprintf("0x%08x", hash), nil
}
