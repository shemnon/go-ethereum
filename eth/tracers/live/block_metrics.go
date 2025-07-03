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

// blockMetrics represents the metrics collected for a single block
type blockMetrics struct {
	// Block identification
	BlockNumber uint64      `json:"block_number"`
	BlockHash   common.Hash `json:"block_hash"`
	Timestamp   uint64      `json:"timestamp"`

	// Timing metrics
	ProcessingTime time.Duration `json:"processing_time_ns"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time"`

	// Storage metrics
	StorageReads     uint64 `json:"storage_reads"`
	StorageWrites    uint64 `json:"storage_writes"`
	NetStorageGrowth int64  `json:"net_storage_growth"`

	// Transaction metrics
	TransactionCount uint64            `json:"transaction_count"`
	TxMemoryUsage    []txMemoryMetrics `json:"tx_memory_usage,omitempty"`

	// Block size metrics
	BlockSize blockSizeMetrics `json:"block_size"`

	// Log/bloom metrics
	LogCount          uint64               `json:"log_count"`
	BloomTopics       uint64               `json:"bloom_topics"`
	DistinctLogTopics map[common.Hash]bool `json:"-"` // Not serialized
	UniqueLogTopics   uint64               `json:"unique_log_topics"`

	// Memory expansion metrics
	TotalMemoryExpansion uint64 `json:"total_memory_expansion"`

	// Cached block event for size calculation (not serialized)
	cachedBlockEvent *tracing.BlockEvent `json:"-"`
}

// txMemoryMetrics represents per-transaction memory usage metrics
type txMemoryMetrics struct {
	TxHash   common.Hash `json:"tx_hash"`
	TxIndex  int         `json:"tx_index"`
	MLoads   uint64      `json:"mloads"`
	MStores  uint64      `json:"mstores"`
	MStore8s uint64      `json:"mstore8s"`
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
	}, nil
}

// onBlockStart is called when a new block starts processing
func (t *blockMetricsTracer) onBlockStart(event tracing.BlockEvent) {
	t.currentBlock = event.Block
	t.currentMetrics = &blockMetrics{
		BlockNumber:       event.Block.Number().Uint64(),
		BlockHash:         event.Block.Hash(),
		Timestamp:         event.Block.Time(),
		StartTime:         time.Now(),
		DistinctLogTopics: make(map[common.Hash]bool),
		TransactionCount:  uint64(len(event.Block.Transactions())),
		cachedBlockEvent:  &event, // Cache the block event for size calculation later
	}

	// Initialize per-transaction metrics if detailed tracking is enabled
	if t.config.DetailedTx {
		t.currentMetrics.TxMemoryUsage = make([]txMemoryMetrics, len(event.Block.Transactions()))
		for i, tx := range event.Block.Transactions() {
			t.currentMetrics.TxMemoryUsage[i] = txMemoryMetrics{
				TxHash:  tx.Hash(),
				TxIndex: i,
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
	t.currentMetrics.ProcessingTime = t.currentMetrics.EndTime.Sub(t.currentMetrics.StartTime)
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
			t.currentMetrics.BloomTopics += uint64(len(log.Topics))
			for _, topic := range log.Topics {
				t.currentMetrics.DistinctLogTopics[topic] = true
			}
		}
	}

	// Finalize memory expansion tracking for this transaction
	// Add any remaining memory usage from all call depths to the aggregate
	// we add all call depths in case we OOGed or something similar not at the firs call
	for depth := 0; depth < len(t.memoryUsageByDepth); depth++ {
		t.txMemoryAggregate += t.memoryUsageByDepth[depth]
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
	if prev == new {
		// not a change
	} else if prev == (common.Hash{}) && new != (common.Hash{}) {
		// New storage slot
		t.currentMetrics.NetStorageGrowth++
	} else if prev != (common.Hash{}) && new == (common.Hash{}) {
		// Deleted storage slot
		t.currentMetrics.NetStorageGrowth--
	}
	// If both prev and new are non-zero, it's just an update (no net change)
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

	// Track memory operations if detailed transaction tracking is enabled
	if t.config.DetailedTx && t.currentTxIndex >= 0 && t.currentTxIndex < len(t.currentMetrics.TxMemoryUsage) {
		switch opcode {
		case vm.MLOAD:
			t.currentMetrics.TxMemoryUsage[t.currentTxIndex].MLoads++
		case vm.MSTORE:
			t.currentMetrics.TxMemoryUsage[t.currentTxIndex].MStores++
		case vm.MSTORE8:
			t.currentMetrics.TxMemoryUsage[t.currentTxIndex].MStore8s++
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
	// Remove the internal map before serialization
	metrics.DistinctLogTopics = nil

	if t.useConsole {
		// Output to console using structured logging
		log.Info("Block metrics",
			"block_number", metrics.BlockNumber,
			"block_hash", metrics.BlockHash.Hex(),
			"timestamp", metrics.Timestamp,
			"processing_time_ms", metrics.ProcessingTime.Milliseconds(),
			"storage_reads", metrics.StorageReads,
			"storage_writes", metrics.StorageWrites,
			"net_storage_growth", metrics.NetStorageGrowth,
			"transaction_count", metrics.TransactionCount,
			"log_count", metrics.LogCount,
			"bloom_topics", metrics.BloomTopics,
			"unique_log_topics", metrics.UniqueLogTopics,
			"total_memory_expansion", metrics.TotalMemoryExpansion,
			"block_size_total", metrics.BlockSize.Total,
			"block_size_header", metrics.BlockSize.Header,
			"block_size_transactions", metrics.BlockSize.Transactions,
			"block_size_receipts", metrics.BlockSize.Receipts,
		)

		// Log per-transaction memory metrics if available
		if t.config.DetailedTx && len(metrics.TxMemoryUsage) > 0 {
			for _, txMetrics := range metrics.TxMemoryUsage {
				if txMetrics.MLoads > 0 || txMetrics.MStores > 0 || txMetrics.MStore8s > 0 {
					log.Info("Transaction memory metrics",
						"block_number", metrics.BlockNumber,
						"tx_hash", txMetrics.TxHash.Hex(),
						"tx_index", txMetrics.TxIndex,
						"mloads", txMetrics.MLoads,
						"mstores", txMetrics.MStores,
						"mstore8s", txMetrics.MStore8s,
					)
				}
			}
		}

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
