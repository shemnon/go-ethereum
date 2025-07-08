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

package live

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	tracers.LiveDirectory.Register("blockMetrics", newBlockMetricsTracer)
}

// blockMetricsConfig defines the configuration for the block metrics tracer
type blockMetricsConfig struct {
	Path       string `json:"path"`       // Output directory path
	DetailedTx bool   `json:"detailedTx"` // Whether to collect detailed per-transaction metrics
	MaxSize    int    `json:"maxSize"`    // Maximum size of log file in MB
	MaxBackups int    `json:"maxBackups"` // Maximum number of backup files
	MaxAge     int    `json:"maxAge"`     // Maximum age of backup files in days
	Compress   bool   `json:"compress"`   // Whether to compress backup files
}

// jsonDuration wraps time.Duration to provide custom JSON marshaling as a string
type jsonDuration time.Duration

func (d jsonDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *jsonDuration) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	duration, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	*d = jsonDuration(duration)
	return nil
}

// blockMetrics represents the metrics collected for a single block
type blockMetrics struct {
	// Block identification
	BlockNumber uint64      `json:"block_number"`
	BlockHash   common.Hash `json:"block_hash"`
	Timestamp   uint64      `json:"-"`

	// Timing metrics
	ProcessingTime jsonDuration `json:"processing_time"`
	StartTime      time.Time    `json:"-"`
	EndTime        time.Time    `json:"-"`

	// Storage metrics
	StorageReads     uint64 `json:"storage_reads"`
	StorageWrites    uint64 `json:"storage_writes"`
	NetStorageGrowth int64  `json:"net_storage_growth"`

	// Transaction metrics
	TransactionCount uint64      `json:"transaction_count"`
	TxMetrics        []txMetrics `json:"tx_metrics,omitempty"`

	// Block size metrics
	BlockSize blockSizeMetrics `json:"block_size"`

	// Log/bloom metrics
	LogCount             uint64                  `json:"log_count"`
	LogTopicCount        uint64                  `json:"log_topic_count"`
	DistinctLogAddresses map[common.Address]bool `json:"-"` // Not serialized
	DistinctLogTopics    map[common.Hash]bool    `json:"-"` // Not serialized
	UniqueLogAddresses   uint64                  `json:"unique_log_addresses"`
	UniqueLogTopics      uint64                  `json:"unique_log_topics"`

	// Memory expansion metrics
	TotalMemoryExpansion uint64 `json:"total_memory_expansion"`

	// Distinct node metrics
	DistinctReads     uint64 `json:"distinct_trie_node_reads"`
	DistinctWrites    uint64 `json:"distinct_trie_node_writes"`
	NetDistinctGrowth int64  `json:"net_distinct_trie_node_growth"`

	// Cached block event for size calculation (not serialized)
	cachedBlockEvent *tracing.BlockEvent `json:"-"`
}

// txMetrics represents per-transaction metrics
type txMetrics struct {
	MemoryExpansion    uint64 `json:"memory_expansion"`
	LogTopicCount      uint64 `json:"log_topic_count"`
	UniqueLogTopics    uint64 `json:"unique_log_topics"`
	UniqueLogAddresses uint64 `json:"unique_log_addresses"`
	StorageReads       uint64 `json:"storage_reads"`
	StorageWrites      uint64 `json:"storage_writes"`
	NetStorageGrowth   int64  `json:"net_storage_growth"`

	// Internal tracking maps (not serialized)
	distinctLogTopics    map[common.Hash]bool    `json:"-"`
	distinctLogAddresses map[common.Address]bool `json:"-"`
}

// blockSizeMetrics represents block size breakdown
type blockSizeMetrics struct {
	Total        uint64 `json:"total_bytes"`
	Header       uint64 `json:"header_bytes"`
	Transactions uint64 `json:"transactions_bytes"`
	Receipts     uint64 `json:"receipts_bytes"`
}

// blockMetricsTracer implements the block metrics collection tracer
type blockMetricsTracer struct {
	config     blockMetricsConfig
	logger     *lumberjack.Logger
	useConsole bool // true if outputting to console, false if outputting to file

	// Current block being processed
	currentBlock       *types.Block
	currentMetrics     *blockMetrics
	currentTxIndex     int
	currentTxHash      common.Hash
	storageChangeCount uint64

	// Memory expansion tracking
	currentCallDepth     int
	memoryUsageByDepth   []uint64 // indexed by call depth
	txMemoryAggregate    uint64   // per transaction aggregate
	blockMemoryAggregate uint64   // per block aggregate
}

// newBlockMetricsTracer creates a new block metrics tracer
func newBlockMetricsTracer(cfg json.RawMessage) (*tracing.Hooks, error) {
	var config blockMetricsConfig
	if err := json.Unmarshal(cfg, &config); err != nil {
		return nil, fmt.Errorf("failed to parse block metrics config: %v", err)
	}

	t := &blockMetricsTracer{
		config: config,
	}

	// If no path is specified, use console output
	if config.Path == "" {
		t.useConsole = true
		log.Info("Block metrics tracer initialized with console output")
	} else {
		// Set up file output
		t.useConsole = false

		// Create output directory if it doesn't exist
		if err := os.MkdirAll(config.Path, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %v", err)
		}

		// Set up rotating file logger
		logger := &lumberjack.Logger{
			Filename:   filepath.Join(config.Path, "block_metrics.jsonl"),
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}

		// Set defaults
		if logger.MaxSize == 0 {
			logger.MaxSize = 100 // 100MB default
		}
		if logger.MaxBackups == 0 {
			logger.MaxBackups = 5
		}
		if logger.MaxAge == 0 {
			logger.MaxAge = 30 // 30 days default
		}

		t.logger = logger
		log.Info("Block metrics tracer initialized with file output", "path", config.Path)
	}

	return &tracing.Hooks{
		OnBlockStart:    t.onBlockStart,
		OnBlockEnd:      t.onBlockEnd,
		OnTxStart:       t.onTxStart,
		OnTxEnd:         t.onTxEnd,
		OnStorageChange: t.onStorageChange,
		OnOpcode:        t.onOpcode,
		OnLog:           t.onLog,
		OnClose:         t.onClose,
		OnTrieUpdate:    t.onTrieUpdate,
	}, nil
}

// onBlockStart is called when a new block starts processing
func (t *blockMetricsTracer) onBlockStart(event tracing.BlockEvent) {
	t.currentBlock = event.Block
	t.currentMetrics = &blockMetrics{
		BlockNumber:          event.Block.Number().Uint64(),
		BlockHash:            event.Block.Hash(),
		Timestamp:            event.Block.Time(),
		StartTime:            time.Now(),
		DistinctLogAddresses: make(map[common.Address]bool),
		DistinctLogTopics:    make(map[common.Hash]bool),
		TransactionCount:     uint64(len(event.Block.Transactions())),
		cachedBlockEvent:     &event, // Cache the block event for size calculation later
	}

	// Initialize per-transaction metrics if detailed tracking is enabled
	if t.config.DetailedTx {
		t.currentMetrics.TxMetrics = make([]txMetrics, len(event.Block.Transactions()))
		for i := range event.Block.Transactions() {
			t.currentMetrics.TxMetrics[i] = txMetrics{
				distinctLogTopics:    make(map[common.Hash]bool),
				distinctLogAddresses: make(map[common.Address]bool),
			}
		}
	}

	t.currentTxIndex = -1
	t.storageChangeCount = 0
}

// onBlockEnd is called when block processing completes
func (t *blockMetricsTracer) onBlockEnd(err error) {
	if t.currentMetrics == nil {
		return
	}

	// Finalize metrics
	t.currentMetrics.EndTime = time.Now()
	t.currentMetrics.ProcessingTime = jsonDuration(t.currentMetrics.EndTime.Sub(t.currentMetrics.StartTime))
	t.currentMetrics.UniqueLogAddresses = uint64(len(t.currentMetrics.DistinctLogAddresses))
	t.currentMetrics.UniqueLogTopics = uint64(len(t.currentMetrics.DistinctLogTopics))
	t.currentMetrics.TotalMemoryExpansion = t.blockMemoryAggregate

	// Calculate block size metrics using cached block event
	if t.currentMetrics.cachedBlockEvent != nil {
		t.currentMetrics.BlockSize = t.calculateBlockSize(t.currentMetrics.cachedBlockEvent.Block)
	}

	// Write metrics to file
	if err := t.writeMetrics(t.currentMetrics); err != nil {
		log.Error("Failed to write block metrics", "block", t.currentMetrics.BlockNumber, "err", err)
	}

	// Reset for next block
	t.currentMetrics.cachedBlockEvent = nil
	t.currentMetrics = nil
	t.currentBlock = nil
	t.blockMemoryAggregate = 0
}

// onTxStart is called when a new transaction starts processing
func (t *blockMetricsTracer) onTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	t.currentTxIndex++
	t.currentTxHash = tx.Hash()

	// Initialize memory expansion tracking for this transaction
	t.currentCallDepth = 0
	t.memoryUsageByDepth = make([]uint64, 1024) // Support up to 1024 call depth levels
	t.txMemoryAggregate = 0
}

// onTxEnd is called when transaction processing completes
func (t *blockMetricsTracer) onTxEnd(receipt *types.Receipt, err error) {
	if t.currentMetrics == nil {
		return
	}

	// Update log metrics from receipt
	if receipt != nil {
		t.currentMetrics.LogCount += uint64(len(receipt.Logs))
		for _, log := range receipt.Logs {
			t.currentMetrics.DistinctLogAddresses[log.Address] = true
			t.currentMetrics.LogTopicCount += uint64(len(log.Topics))
			for _, topic := range log.Topics {
				t.currentMetrics.DistinctLogTopics[topic] = true
			}
		}

		// Track per-transaction log metrics if detailed tracking is enabled
		if t.config.DetailedTx && t.currentTxIndex >= 0 && t.currentTxIndex < len(t.currentMetrics.TxMetrics) {
			tx := &t.currentMetrics.TxMetrics[t.currentTxIndex]
			tx.LogTopicCount = uint64(len(receipt.Logs))
			for _, log := range receipt.Logs {
				tx.distinctLogAddresses[log.Address] = true
				tx.LogTopicCount += uint64(len(log.Topics))
				for _, topic := range log.Topics {
					tx.distinctLogTopics[topic] = true
				}
			}
			// Finalize unique counts
			tx.UniqueLogAddresses = uint64(len(tx.distinctLogAddresses))
			tx.UniqueLogTopics = uint64(len(tx.distinctLogTopics))
		}
	}

	// Finalize memory expansion tracking for this transaction
	// Add any remaining memory usage from all call depths to the aggregate
	// we add all call depths in case we OOGed or something similar not at the firs call
	for depth := 0; depth < len(t.memoryUsageByDepth); depth++ {
		t.txMemoryAggregate += t.memoryUsageByDepth[depth]
	}

	// Update per-transaction memory expansion if detailed tracking is enabled
	if t.config.DetailedTx && t.currentTxIndex >= 0 && t.currentTxIndex < len(t.currentMetrics.TxMetrics) {
		t.currentMetrics.TxMetrics[t.currentTxIndex].MemoryExpansion = t.txMemoryAggregate
	}

	// Add transaction memory aggregate to block aggregate
	t.blockMemoryAggregate += t.txMemoryAggregate
}

// onStorageChange is called for every storage change (SSTORE)
func (t *blockMetricsTracer) onStorageChange(addr common.Address, slot common.Hash, prev, new common.Hash) {
	if t.currentMetrics == nil {
		return
	}

	t.storageChangeCount++
	t.currentMetrics.StorageWrites++

	// Calculate net storage growth
	var netChange int64
	if prev == new {
		// not a change
	} else if prev == (common.Hash{}) && new != (common.Hash{}) {
		// New storage slot
		netChange = 1
	} else if prev != (common.Hash{}) && new == (common.Hash{}) {
		// Deleted storage slot
		netChange = -1
	}
	// If both prev and new are non-zero, it's just an update (no net change)

	t.currentMetrics.NetStorageGrowth += netChange
	// Track per-transaction storage changes if detailed tracking is enabled
	if t.config.DetailedTx && t.currentTxIndex >= 0 && t.currentTxIndex < len(t.currentMetrics.TxMetrics) {
		t.currentMetrics.TxMetrics[t.currentTxIndex].StorageWrites++
		t.currentMetrics.TxMetrics[t.currentTxIndex].NetStorageGrowth += netChange
	}
}

// onOpcode is called for every opcode execution
func (t *blockMetricsTracer) onOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	if t.currentMetrics == nil {
		return
	}

	opcode := vm.OpCode(op)

	// Track storage reads (SLOAD)
	if opcode == vm.SLOAD {
		t.currentMetrics.StorageReads++
	}

	// Track per-transaction storage operations if detailed transaction tracking is enabled
	if t.config.DetailedTx && t.currentTxIndex >= 0 && t.currentTxIndex < len(t.currentMetrics.TxMetrics) {
		if opcode == vm.SLOAD {
			t.currentMetrics.TxMetrics[t.currentTxIndex].StorageReads++
		}
	}

	// Track memory expansion across call depths
	if depth < len(t.memoryUsageByDepth) {
		// Handle call depth changes
		if depth < t.currentCallDepth {
			// Call depth decreased (call returned), add memory from the returning level to aggregate
			t.txMemoryAggregate += t.memoryUsageByDepth[t.currentCallDepth]
			t.memoryUsageByDepth[t.currentCallDepth] = 0
		}

		// Update current call depth
		t.currentCallDepth = depth
		t.memoryUsageByDepth[depth] = uint64(len(scope.MemoryData()))
	}
}

// onLog is called when a log is emitted
func (t *blockMetricsTracer) onLog(log *types.Log) {
	// Log metrics are handled in onTxEnd to avoid double counting
}

// onClose is called when the tracer is being shut down
func (t *blockMetricsTracer) onClose() {
	if !t.useConsole && t.logger != nil {
		t.logger.Close()
		log.Info("Block metrics tracer file output closed")
	} else if t.useConsole {
		log.Info("Block metrics tracer console output closed")
	}
}

// onTrieUpdate is called during trie commits to pass trie data incrementally
func (t *blockMetricsTracer) onTrieUpdate(distinctReads, distinctWrites uint64, netDistinctGrowth int64) {
	if t.currentMetrics != nil {
		// Set the stats from the global collector (non-accumulative since it's aggregated)
		t.currentMetrics.DistinctReads = distinctReads
		t.currentMetrics.DistinctWrites = distinctWrites
		t.currentMetrics.NetDistinctGrowth = netDistinctGrowth
	}
}

// calculateBlockSize calculates the size breakdown of a block
func (t *blockMetricsTracer) calculateBlockSize(block *types.Block) blockSizeMetrics {
	// This is a simplified calculation
	// In a real implementation, you'd want to use the actual RLP encoding sizes
	headerSize := uint64(len(block.Header().ParentHash) + len(block.Header().UncleHash) + len(block.Header().Coinbase) +
		len(block.Header().Root) + len(block.Header().TxHash) + len(block.Header().ReceiptHash) +
		len(block.Header().Bloom) + 32 + 8 + 8 + 8 + len(block.Header().Extra) + len(block.Header().MixDigest) + 8)

	var txSize uint64
	for _, tx := range block.Transactions() {
		txSize += uint64(tx.Size())
	}

	// Receipt size estimation (simplified)
	var receiptSize uint64
	for _, tx := range block.Transactions() {
		receiptSize += uint64(100 + len(tx.Data())) // Rough estimate
	}

	return blockSizeMetrics{
		Total:        headerSize + txSize + receiptSize,
		Header:       headerSize,
		Transactions: txSize,
		Receipts:     receiptSize,
	}
}

// writeMetrics writes the metrics to the output file or console
func (t *blockMetricsTracer) writeMetrics(metrics *blockMetrics) error {
	// Remove the internal maps before serialization
	metrics.DistinctLogAddresses = nil
	metrics.DistinctLogTopics = nil

	if t.useConsole {
		// Output to console using structured logging
		log.Info("Block metrics",
			"block_number", metrics.BlockNumber,
			"block_hash", metrics.BlockHash.Hex(),
			"processing_time", time.Duration(metrics.ProcessingTime).String(),
			"storage_reads", metrics.StorageReads,
			"storage_writes", metrics.StorageWrites,
			"net_storage_growth", metrics.NetStorageGrowth,
			"transaction_count", metrics.TransactionCount,
			"log_count", metrics.LogCount,
			"log_topic_count", metrics.LogTopicCount,
			"unique_log_addresses", metrics.UniqueLogAddresses,
			"unique_log_topics", metrics.UniqueLogTopics,
			"total_memory_expansion", metrics.TotalMemoryExpansion,
			"distinct_trie_node_reads", metrics.DistinctReads,
			"distinct_trie_node_writes", metrics.DistinctWrites,
			"net_distinct_trie_node_growth", metrics.NetDistinctGrowth,
			"block_size_total", metrics.BlockSize.Total,
			"block_size_header", metrics.BlockSize.Header,
			"block_size_transactions", metrics.BlockSize.Transactions,
			"block_size_receipts", metrics.BlockSize.Receipts,
		)

		return nil
	} else {
		// Output to file
		data, err := json.Marshal(metrics)
		if err != nil {
			return err
		}

		_, err = t.logger.Write(append(data, '\n'))
		return err
	}
}
