// Copyright 2024 The go-ethereum Authors
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

package trie

import (
	"sync"
)

// TrieStatsCollector aggregates trie statistics across multiple tries during block processing
type TrieStatsCollector struct {
	mu             sync.Mutex
	totalReads     uint64
	totalWrites    uint64
	totalNetGrowth int64
}

// Global collector instance for the current block being processed
var globalStatsCollector = &TrieStatsCollector{}

// StartCollection resets the global collector for a new block
func StartTrieStatsCollection() {
	globalStatsCollector.mu.Lock()
	defer globalStatsCollector.mu.Unlock()
	globalStatsCollector.totalReads = 0
	globalStatsCollector.totalWrites = 0
	globalStatsCollector.totalNetGrowth = 0
}

// AddTrieStats adds statistics from a trie commit to the global collector
func AddTrieStats(reads, writes uint64, netGrowth int64) {
	globalStatsCollector.mu.Lock()
	defer globalStatsCollector.mu.Unlock()
	globalStatsCollector.totalReads += reads
	globalStatsCollector.totalWrites += writes
	globalStatsCollector.totalNetGrowth += netGrowth
}

// GetCollectedStats returns the aggregated statistics and optionally resets them
func GetCollectedTrieStats(reset bool) (reads, writes uint64, netGrowth int64) {
	globalStatsCollector.mu.Lock()
	defer globalStatsCollector.mu.Unlock()

	reads = globalStatsCollector.totalReads
	writes = globalStatsCollector.totalWrites
	netGrowth = globalStatsCollector.totalNetGrowth

	if reset {
		globalStatsCollector.totalReads = 0
		globalStatsCollector.totalWrites = 0
		globalStatsCollector.totalNetGrowth = 0
	}

	return reads, writes, netGrowth
}
