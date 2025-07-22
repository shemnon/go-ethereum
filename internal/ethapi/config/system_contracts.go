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
	"github.com/ethereum/go-ethereum/params"
)

// GetSystemContracts returns the system contract addresses for the given chain rules
// formatted for EIP-7910 eth_config response.
func GetSystemContracts(rules params.Rules, chainConfig *params.ChainConfig) map[string]common.Address {
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
