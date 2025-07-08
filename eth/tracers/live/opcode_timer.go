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
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	tracers.LiveDirectory.Register("opcodeTimer", newOpcodeTimerTracer)
}

// opcodeTimerConfig defines the configuration for the opcode timer tracer
type opcodeTimerConfig struct {
	Path       string `json:"path"`       // Output directory path
	MaxSize    int    `json:"maxSize"`    // Maximum size of log file in MB
	MaxBackups int    `json:"maxBackups"` // Maximum number of backup files
	MaxAge     int    `json:"maxAge"`     // Maximum age of backup files in days
	Compress   bool   `json:"compress"`   // Whether to compress backup files
}

// opcodeTimerOutput represents the JSON output for a single block
type opcodeTimerOutput struct {
	BlockNumber uint64     `json:"block_number"`
	OpcodeNs    [256]int64 `json:"opcode_ns"`    // Array of 256 nanosecond values
	OpcodeCount [256]int64 `json:"opcode_count"` // Array of 256 execution count values
	GasUsed     uint64     `json:"gas_used"`
	TotalTime   int64      `json:"total_time_ns"`   // Total block processing time
	NonEvmTime  int64      `json:"non_evm_time_ns"` // Time outside EVM execution
}

// opcodeTimerTracer implements the opcode timing tracer
type opcodeTimerTracer struct {
	config     opcodeTimerConfig
	logger     *lumberjack.Logger
	useConsole bool // true if outputting to console, false if outputting to file

	// Current block being processed
	currentBlock   *types.Block
	currentOutput  *opcodeTimerOutput
	blockStartTime time.Time

	// EVM execution state
	inEvmExecution     bool
	evmStartTime       time.Time
	currentOpcodeStart time.Time
	currentOpcode      byte
	hasCurrentOpcode   bool

	// Timing accumulators
	totalEvmTime    int64
	totalNonEvmTime int64
}

// newOpcodeTimerTracer creates a new opcode timer tracer
func newOpcodeTimerTracer(cfg json.RawMessage) (*tracing.Hooks, error) {
	var config opcodeTimerConfig
	if err := json.Unmarshal(cfg, &config); err != nil {
		return nil, fmt.Errorf("failed to parse opcode timer config: %v", err)
	}

	t := &opcodeTimerTracer{
		config: config,
	}

	// If no path is specified, use console output
	if config.Path == "" {
		t.useConsole = true
		log.Info("Opcode timer tracer initialized with console output")
	} else {
		// Set up file output
		t.useConsole = false

		// Create output directory if it doesn't exist
		if err := os.MkdirAll(config.Path, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %v", err)
		}

		// Set up rotating file logger
		logger := &lumberjack.Logger{
			Filename:   filepath.Join(config.Path, "opcode_timer.json"),
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
		log.Info("Opcode timer tracer initialized with file output", "path", config.Path)
	}

	return &tracing.Hooks{
		OnBlockStart: t.onBlockStart,
		OnBlockEnd:   t.onBlockEnd,
		OnEnter:      t.onEnter,
		OnExit:       t.onExit,
		OnOpcode:     t.onOpcode,
		OnFault:      t.onFault,
		OnClose:      t.onClose,
	}, nil
}

// onBlockStart is called when a new block starts processing
func (t *opcodeTimerTracer) onBlockStart(event tracing.BlockEvent) {
	t.currentBlock = event.Block
	t.blockStartTime = time.Now()
	t.currentOutput = &opcodeTimerOutput{
		BlockNumber: event.Block.Number().Uint64(),
		OpcodeNs:    [256]int64{}, // Initialize with zeros
		OpcodeCount: [256]int64{}, // Initialize with zeros
		GasUsed:     0,
		TotalTime:   0,
		NonEvmTime:  0,
	}

	// Reset EVM execution state
	t.inEvmExecution = false
	t.hasCurrentOpcode = false
	t.totalEvmTime = 0
	t.totalNonEvmTime = 0
}

// onBlockEnd is called when block processing completes
func (t *opcodeTimerTracer) onBlockEnd(err error) {
	if t.currentOutput == nil {
		return
	}

	// Finalize any remaining opcode timing
	if t.hasCurrentOpcode && t.inEvmExecution {
		now := time.Now()
		duration := now.Sub(t.currentOpcodeStart).Nanoseconds()
		t.currentOutput.OpcodeNs[t.currentOpcode] += duration
		t.totalEvmTime += duration
		t.hasCurrentOpcode = false
	}

	// Calculate final metrics
	blockEndTime := time.Now()
	t.currentOutput.TotalTime = blockEndTime.Sub(t.blockStartTime).Nanoseconds()
	t.currentOutput.NonEvmTime = t.currentOutput.TotalTime - t.totalEvmTime

	// Get gas used from the block
	if t.currentBlock != nil {
		t.currentOutput.GasUsed = t.currentBlock.GasUsed()
	}

	// Write output
	if err := t.writeOutput(t.currentOutput); err != nil {
		log.Error("Failed to write opcode timer output", "block", t.currentOutput.BlockNumber, "err", err)
	}

	// Reset for next block
	t.currentOutput = nil
	t.currentBlock = nil
}

// onEnter is called when entering a new execution context
func (t *opcodeTimerTracer) onEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Track when we enter EVM execution context
	if !t.inEvmExecution {
		t.inEvmExecution = true
		t.evmStartTime = time.Now()
	}
}

// onExit is called when exiting an execution context
func (t *opcodeTimerTracer) onExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if !t.inEvmExecution {
		return
	}

	// Finalize any current opcode timing
	if t.hasCurrentOpcode {
		now := time.Now()
		duration := now.Sub(t.currentOpcodeStart).Nanoseconds()
		t.currentOutput.OpcodeNs[t.currentOpcode] += duration
		t.totalEvmTime += duration
		t.hasCurrentOpcode = false
	}

	// Mark end of EVM execution
	t.inEvmExecution = false
}

// onOpcode is called for every opcode execution
func (t *opcodeTimerTracer) onOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	if !t.inEvmExecution || t.currentOutput == nil {
		return
	}

	now := time.Now()

	// Finalize timing for previous opcode
	if t.hasCurrentOpcode {
		duration := now.Sub(t.currentOpcodeStart).Nanoseconds()
		t.currentOutput.OpcodeNs[t.currentOpcode] += duration
		t.totalEvmTime += duration
	}

	// Start timing for current opcode and increment count
	t.currentOpcode = op
	t.currentOpcodeStart = now
	t.hasCurrentOpcode = true
	t.currentOutput.OpcodeCount[op]++
}

// onFault is called when an opcode execution fails
func (t *opcodeTimerTracer) onFault(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
	if !t.inEvmExecution || t.currentOutput == nil {
		return
	}

	// Finalize timing for the failed opcode
	if t.hasCurrentOpcode {
		now := time.Now()
		duration := now.Sub(t.currentOpcodeStart).Nanoseconds()
		t.currentOutput.OpcodeNs[t.currentOpcode] += duration
		t.totalEvmTime += duration
		t.hasCurrentOpcode = false
	}

	// Mark end of EVM execution since fault switches to non-EVM time
	t.inEvmExecution = false
}

// onClose is called when the tracer is being shut down
func (t *opcodeTimerTracer) onClose() {
	if !t.useConsole && t.logger != nil {
		t.logger.Close()
		log.Info("Opcode timer tracer file output closed")
	} else if t.useConsole {
		log.Info("Opcode timer tracer console output closed")
	}
}

// writeOutput writes the opcode timer output to the configured destination
func (t *opcodeTimerTracer) writeOutput(output *opcodeTimerOutput) error {
	if t.useConsole {
		// Output to console using structured logging
		log.Info("Opcode timer",
			"block_number", output.BlockNumber,
			"gas_used", output.GasUsed,
			"total_time_ns", output.TotalTime,
			"non_evm_time_ns", output.NonEvmTime,
		)
		return nil
	} else {
		// Output to file
		data, err := json.Marshal(output)
		if err != nil {
			return err
		}

		_, err = t.logger.Write(append(data, '\n'))
		return err
	}
}
