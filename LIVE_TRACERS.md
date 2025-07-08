# Live Tracers Documentation

This document describes the live tracers available in go-ethereum for real-time blockchain analysis and data collection.

## Overview

Live tracers are components that collect metrics during block processing and transaction execution. They output structured data that can be consumed by data pipelines, monitoring systems, and analytics tools.

**Important**: Only one live tracer can be active at a time. You cannot run multiple tracers simultaneously.

## Block Metrics Tracer

The Block Metrics tracer provides comprehensive metrics about block processing, storage operations, memory usage, and transaction patterns.

### Command Line Flags

| Flag | Type | Description |
|------|------|-------------|
| `--trace.blockmetrics` | boolean | Enable the block metrics live tracer |
| `--trace.blockmetrics.path` | string | Directory path for output files (optional, defaults to console) |
| `--trace.blockmetrics.detailed-tx` | boolean | Enable detailed per-transaction metrics (increases overhead) |

### Configuration

When using file output, the tracer supports log rotation:
- **Default file**: `block_metrics.jsonl` (JSON Lines format)
- **Max file size**: 100MB (configurable)
- **Max backups**: 5 files (configurable)
- **Max age**: 30 days (configurable)
- **Compression**: Optional gzip compression of rotated files

### Output Format

Each line contains a JSON object representing one block's metrics:

```json
{
  "block_number": 18500000,
  "block_hash": "0x1234...",
  "processing_time": "45.2ms",
  "storage_reads": 1250,
  "storage_writes": 89,
  "net_storage_growth": 12,
  "transaction_count": 150,
  "log_count": 450,
  "log_topic_count": 1200,
  "unique_log_addresses": 95,
  "unique_log_topics": 180,
  "total_memory_expansion": 2097152,
  "distinct_trie_node_reads": 3400,
  "distinct_trie_node_writes": 210,
  "net_distinct_trie_node_growth": 45,
  "block_size": {
    "total_bytes": 125000,
    "header_bytes": 508,
    "transactions_bytes": 118000,
    "receipts_bytes": 6492
  },
  "tx_metrics": [ /* optional per-transaction details */ ]
}
```

### Field Definitions

#### Block Identification
- **`block_number`**: Sequential block number in the chain
- **`block_hash`**: Unique block hash (32 bytes, hex-encoded)

#### Timing Metrics
- **`processing_time`**: Total time spent processing this block (from start to completion)

#### Storage Operations
- **`storage_reads`**: Number of SLOAD operations across all transactions
- **`storage_writes`**: Number of SSTORE operations across all transactions  
- **`net_storage_growth`**: Net change in storage slots (-1 for deleted, +1 for new, 0 for updates)

#### Transaction Metrics
- **`transaction_count`**: Number of transactions in the block
- **`tx_metrics`**: Array of per-transaction metrics (only when `--trace.blockmetrics.detailed-tx` is enabled)

#### Event Log Metrics
- **`log_count`**: Total number of event logs emitted
- **`log_topic_count`**: Total number of indexed topics across all logs
- **`unique_log_addresses`**: Number of distinct contract addresses that emitted logs
- **`unique_log_topics`**: Number of distinct topic0 values (event signatures)

#### Memory Metrics  
- **`total_memory_expansion`**: Total bytes of EVM memory expansion across all transactions

#### Trie Node Metrics
- **`distinct_trie_node_reads`**: Number of unique trie nodes read from disk
- **`distinct_trie_node_writes`**: Number of unique trie nodes written to disk
- **`net_distinct_trie_node_growth`**: Net change in distinct trie nodes stored

#### Block Size Metrics
- **`block_size.total_bytes`**: Total serialized block size
- **`block_size.header_bytes`**: Block header size
- **`block_size.transactions_bytes`**: Combined size of all transactions
- **`block_size.receipts_bytes`**: Combined size of all transaction receipts

#### Per-Transaction Metrics (when `detailed-tx` enabled)
```json
{
  "memory_expansion": 65536,
  "log_topic_count": 8,
  "unique_log_topics": 3,
  "unique_log_addresses": 2,
  "storage_reads": 15,
  "storage_writes": 3,
  "net_storage_growth": 1
}
```

## Opcode Timer Tracer

The Opcode Timer tracer provides precise timing and execution count metrics for each EVM opcode at the block level.

### Command Line Flags

| Flag | Type | Description |
|------|------|-------------|
| `--trace.opcodetimer` | boolean | Enable the opcode timer live tracer |
| `--trace.opcodetimer.path` | string | Directory path for output files (optional, defaults to console) |

### Configuration

When using file output, the tracer supports log rotation:
- **Default file**: `opcode_timer.json` (JSON format)
- **Max file size**: 100MB (configurable)
- **Max backups**: 5 files (configurable)
- **Max age**: 30 days (configurable)
- **Compression**: Optional gzip compression of rotated files

### Output Format

Each line contains a JSON object representing one block's opcode metrics:

```json
{
  "block_number": 18500000,
  "opcode_ns": [1250000, 0, 890000, 0, 450000, ...],
  "opcode_count": [42, 0, 15, 0, 8, ...],
  "gas_used": 2975432,
  "total_time_ns": 45200000,
  "non_evm_time_ns": 12800000
}
```

### Field Definitions

#### Block Identification
- **`block_number`**: Sequential block number in the chain

#### Opcode Metrics
- **`opcode_ns`**: Array of 256 int64 values representing total nanoseconds spent executing each opcode
  - Index corresponds to opcode value (0=STOP, 1=ADD, 2=MUL, etc.)
  - Aggregated across all transactions in the block
  - Measured from `OnOpcode` call to next `OnOpcode`/`OnExit`/`OnFault`
- **`opcode_count`**: Array of 256 int64 values representing execution count for each opcode
  - Index corresponds to opcode value (0=STOP, 1=ADD, 2=MUL, etc.)
  - Incremented once per `OnOpcode` call
  - Aggregated across all transactions in the block

#### Gas and Timing
- **`gas_used`**: Total gas consumed by all transactions in the block
- **`total_time_ns`**: Total block processing time in nanoseconds (from `OnBlockStart` to `OnBlockEnd`)
- **`non_evm_time_ns`**: Time spent outside EVM execution (total_time_ns - sum of EVM execution time)

### EVM Execution Timing

The tracer tracks EVM execution boundaries:
- **EVM execution starts**: First `OnOpcode` call after `OnEnter`
- **EVM execution ends**: `OnExit` or `OnFault` call
- **Non-EVM time**: Block processing time excluding EVM execution

### Opcode Index Reference

The arrays use standard EVM opcode values as indices. Complete list as of Pectra fork:

#### 0x00-0x0f: Arithmetic Operations
| Int | Hex | Opcode | Description |
|-----|-----|--------|-------------|
| 0 | 0x00 | STOP | Halt execution |
| 1 | 0x01 | ADD | Addition |
| 2 | 0x02 | MUL | Multiplication |
| 3 | 0x03 | SUB | Subtraction |
| 4 | 0x04 | DIV | Integer division |
| 5 | 0x05 | SDIV | Signed integer division |
| 6 | 0x06 | MOD | Modulo |
| 7 | 0x07 | SMOD | Signed modulo |
| 8 | 0x08 | ADDMOD | Addition with modulo |
| 9 | 0x09 | MULMOD | Multiplication with modulo |
| 10 | 0x0a | EXP | Exponentiation |
| 11 | 0x0b | SIGNEXTEND | Sign extension |

#### 0x10-0x1f: Comparison and Bitwise Operations
| Int | Hex | Opcode | Description |
|-----|-----|--------|-------------|
| 16 | 0x10 | LT | Less than |
| 17 | 0x11 | GT | Greater than |
| 18 | 0x12 | SLT | Signed less than |
| 19 | 0x13 | SGT | Signed greater than |
| 20 | 0x14 | EQ | Equality |
| 21 | 0x15 | ISZERO | Is zero |
| 22 | 0x16 | AND | Bitwise AND |
| 23 | 0x17 | OR | Bitwise OR |
| 24 | 0x18 | XOR | Bitwise XOR |
| 25 | 0x19 | NOT | Bitwise NOT |
| 26 | 0x1a | BYTE | Byte extraction |
| 27 | 0x1b | SHL | Shift left (Constantinople) |
| 28 | 0x1c | SHR | Shift right (Constantinople) |
| 29 | 0x1d | SAR | Arithmetic shift right (Constantinople) |

#### 0x20-0x2f: Cryptographic Operations
| Int | Hex | Opcode | Description |
|-----|-----|--------|-------------|
| 32 | 0x20 | KECCAK256 | Keccak-256 hash |

#### 0x30-0x3f: Environmental Information
| Int | Hex | Opcode | Description |
|-----|-----|--------|-------------|
| 48 | 0x30 | ADDRESS | Address of executing account |
| 49 | 0x31 | BALANCE | Balance of given account |
| 50 | 0x32 | ORIGIN | Transaction origin |
| 51 | 0x33 | CALLER | Message caller |
| 52 | 0x34 | CALLVALUE | Value sent with message |
| 53 | 0x35 | CALLDATALOAD | Load call data |
| 54 | 0x36 | CALLDATASIZE | Size of call data |
| 55 | 0x37 | CALLDATACOPY | Copy call data |
| 56 | 0x38 | CODESIZE | Size of executing code |
| 57 | 0x39 | CODECOPY | Copy executing code |
| 58 | 0x3a | GASPRICE | Gas price |
| 59 | 0x3b | EXTCODESIZE | Size of account code |
| 60 | 0x3c | EXTCODECOPY | Copy account code |
| 61 | 0x3d | RETURNDATASIZE | Size of return data |
| 62 | 0x3e | RETURNDATACOPY | Copy return data |
| 63 | 0x3f | EXTCODEHASH | Code hash of account (Constantinople) |

#### 0x40-0x4f: Block Information
| Int | Hex | Opcode | Description |
|-----|-----|--------|-------------|
| 64 | 0x40 | BLOCKHASH | Block hash |
| 65 | 0x41 | COINBASE | Block beneficiary |
| 66 | 0x42 | TIMESTAMP | Block timestamp |
| 67 | 0x43 | NUMBER | Block number |
| 68 | 0x44 | PREVRANDAO | Previous block randomness (post-merge) |
| 69 | 0x45 | GASLIMIT | Block gas limit |
| 70 | 0x46 | CHAINID | Chain ID (Istanbul - EIP-1344) |
| 71 | 0x47 | SELFBALANCE | Self balance (Istanbul - EIP-1884) |
| 72 | 0x48 | BASEFEE | Base fee (London - EIP-3198) |
| 73 | 0x49 | BLOBHASH | Blob hash (Cancun - EIP-4844) |
| 74 | 0x4a | BLOBBASEFEE | Blob base fee (Cancun - EIP-7516) |

#### 0x50-0x5f: Stack, Memory, Storage and Flow Operations
| Int | Hex | Opcode | Description |
|-----|-----|--------|-------------|
| 80 | 0x50 | POP | Remove item from stack |
| 81 | 0x51 | MLOAD | Load from memory |
| 82 | 0x52 | MSTORE | Store to memory |
| 83 | 0x53 | MSTORE8 | Store single byte to memory |
| 84 | 0x54 | SLOAD | Load from storage |
| 85 | 0x55 | SSTORE | Store to storage |
| 86 | 0x56 | JUMP | Jump to location |
| 87 | 0x57 | JUMPI | Conditional jump |
| 88 | 0x58 | PC | Program counter |
| 89 | 0x59 | MSIZE | Memory size |
| 90 | 0x5a | GAS | Available gas |
| 91 | 0x5b | JUMPDEST | Jump destination |
| 92 | 0x5c | TLOAD | Load from transient storage (Cancun - EIP-1153) |
| 93 | 0x5d | TSTORE | Store to transient storage (Cancun - EIP-1153) |
| 94 | 0x5e | MCOPY | Memory copy (Cancun - EIP-5656) |
| 95 | 0x5f | PUSH0 | Push zero onto stack (Shanghai - EIP-3855) |

#### 0x60-0x7f: Push Operations
| Int | Hex | Opcode | Description |
|-----|-----|--------|-------------|
| 96 | 0x60 | PUSH1 | Push 1 byte onto stack |
| 97 | 0x61 | PUSH2 | Push 2 bytes onto stack |
| 98 | 0x62 | PUSH3 | Push 3 bytes onto stack |
| 99 | 0x63 | PUSH4 | Push 4 bytes onto stack |
| 100 | 0x64 | PUSH5 | Push 5 bytes onto stack |
| 101 | 0x65 | PUSH6 | Push 6 bytes onto stack |
| 102 | 0x66 | PUSH7 | Push 7 bytes onto stack |
| 103 | 0x67 | PUSH8 | Push 8 bytes onto stack |
| 104 | 0x68 | PUSH9 | Push 9 bytes onto stack |
| 105 | 0x69 | PUSH10 | Push 10 bytes onto stack |
| 106 | 0x6a | PUSH11 | Push 11 bytes onto stack |
| 107 | 0x6b | PUSH12 | Push 12 bytes onto stack |
| 108 | 0x6c | PUSH13 | Push 13 bytes onto stack |
| 109 | 0x6d | PUSH14 | Push 14 bytes onto stack |
| 110 | 0x6e | PUSH15 | Push 15 bytes onto stack |
| 111 | 0x6f | PUSH16 | Push 16 bytes onto stack |
| 112 | 0x70 | PUSH17 | Push 17 bytes onto stack |
| 113 | 0x71 | PUSH18 | Push 18 bytes onto stack |
| 114 | 0x72 | PUSH19 | Push 19 bytes onto stack |
| 115 | 0x73 | PUSH20 | Push 20 bytes onto stack |
| 116 | 0x74 | PUSH21 | Push 21 bytes onto stack |
| 117 | 0x75 | PUSH22 | Push 22 bytes onto stack |
| 118 | 0x76 | PUSH23 | Push 23 bytes onto stack |
| 119 | 0x77 | PUSH24 | Push 24 bytes onto stack |
| 120 | 0x78 | PUSH25 | Push 25 bytes onto stack |
| 121 | 0x79 | PUSH26 | Push 26 bytes onto stack |
| 122 | 0x7a | PUSH27 | Push 27 bytes onto stack |
| 123 | 0x7b | PUSH28 | Push 28 bytes onto stack |
| 124 | 0x7c | PUSH29 | Push 29 bytes onto stack |
| 125 | 0x7d | PUSH30 | Push 30 bytes onto stack |
| 126 | 0x7e | PUSH31 | Push 31 bytes onto stack |
| 127 | 0x7f | PUSH32 | Push 32 bytes onto stack |

#### 0x80-0x8f: Duplicate Operations
| Int | Hex | Opcode | Description |
|-----|-----|--------|-------------|
| 128 | 0x80 | DUP1 | Duplicate 1st stack item |
| 129 | 0x81 | DUP2 | Duplicate 2nd stack item |
| 130 | 0x82 | DUP3 | Duplicate 3rd stack item |
| 131 | 0x83 | DUP4 | Duplicate 4th stack item |
| 132 | 0x84 | DUP5 | Duplicate 5th stack item |
| 133 | 0x85 | DUP6 | Duplicate 6th stack item |
| 134 | 0x86 | DUP7 | Duplicate 7th stack item |
| 135 | 0x87 | DUP8 | Duplicate 8th stack item |
| 136 | 0x88 | DUP9 | Duplicate 9th stack item |
| 137 | 0x89 | DUP10 | Duplicate 10th stack item |
| 138 | 0x8a | DUP11 | Duplicate 11th stack item |
| 139 | 0x8b | DUP12 | Duplicate 12th stack item |
| 140 | 0x8c | DUP13 | Duplicate 13th stack item |
| 141 | 0x8d | DUP14 | Duplicate 14th stack item |
| 142 | 0x8e | DUP15 | Duplicate 15th stack item |
| 143 | 0x8f | DUP16 | Duplicate 16th stack item |

#### 0x90-0x9f: Swap Operations
| Int | Hex | Opcode | Description |
|-----|-----|--------|-------------|
| 144 | 0x90 | SWAP1 | Swap 1st and 2nd stack items |
| 145 | 0x91 | SWAP2 | Swap 1st and 3rd stack items |
| 146 | 0x92 | SWAP3 | Swap 1st and 4th stack items |
| 147 | 0x93 | SWAP4 | Swap 1st and 5th stack items |
| 148 | 0x94 | SWAP5 | Swap 1st and 6th stack items |
| 149 | 0x95 | SWAP6 | Swap 1st and 7th stack items |
| 150 | 0x96 | SWAP7 | Swap 1st and 8th stack items |
| 151 | 0x97 | SWAP8 | Swap 1st and 9th stack items |
| 152 | 0x98 | SWAP9 | Swap 1st and 10th stack items |
| 153 | 0x99 | SWAP10 | Swap 1st and 11th stack items |
| 154 | 0x9a | SWAP11 | Swap 1st and 12th stack items |
| 155 | 0x9b | SWAP12 | Swap 1st and 13th stack items |
| 156 | 0x9c | SWAP13 | Swap 1st and 14th stack items |
| 157 | 0x9d | SWAP14 | Swap 1st and 15th stack items |
| 158 | 0x9e | SWAP15 | Swap 1st and 16th stack items |
| 159 | 0x9f | SWAP16 | Swap 1st and 17th stack items |

#### 0xa0-0xa4: Logging Operations
| Int | Hex | Opcode | Description |
|-----|-----|--------|-------------|
| 160 | 0xa0 | LOG0 | Log with 0 topics |
| 161 | 0xa1 | LOG1 | Log with 1 topic |
| 162 | 0xa2 | LOG2 | Log with 2 topics |
| 163 | 0xa3 | LOG3 | Log with 3 topics |
| 164 | 0xa4 | LOG4 | Log with 4 topics |

#### 0xf0-0xff: System Operations
| Int | Hex | Opcode | Description |
|-----|-----|--------|-------------|
| 240 | 0xf0 | CREATE | Create contract |
| 241 | 0xf1 | CALL | Call another contract |
| 242 | 0xf2 | CALLCODE | Call with different context |
| 243 | 0xf3 | RETURN | Return from call |
| 244 | 0xf4 | DELEGATECALL | Call with delegated context |
| 245 | 0xf5 | CREATE2 | Create contract with deterministic address (Constantinople) |
| 247 | 0xf7 | RETURNDATALOAD | Load return data |
| 250 | 0xfa | STATICCALL | Static call |
| 253 | 0xfd | REVERT | Revert execution |
| 254 | 0xfe | INVALID | Invalid opcode |
| 255 | 0xff | SELFDESTRUCT | Destroy contract |

#### Fork-Specific Additions
- **Cancun**: BLOBHASH, BLOBBASEFEE, TLOAD, TSTORE, MCOPY
- **Shanghai**: PUSH0
- **London**: BASEFEE  
- **Istanbul**: CHAINID, SELFBALANCE
- **Constantinople**: SHL, SHR, SAR, EXTCODEHASH, CREATE2

## Usage Examples

### Basic Usage

```bash
# Enable block metrics with console output
geth --trace.blockmetrics

# Enable opcode timer with file output
geth --trace.opcodetimer --trace.opcodetimer.path=/data/opcodes

# Enable block metrics with detailed transaction tracking
geth --trace.blockmetrics --trace.blockmetrics.path=/data/blocks --trace.blockmetrics.detailed-tx
```

### Import Historical Data

```bash
# Process historical blocks with block metrics
geth import --trace.blockmetrics --trace.blockmetrics.path=/data/blocks blockchain.db

# Process historical blocks with opcode timer
geth import --trace.opcodetimer --trace.opcodetimer.path=/data/opcodes blockchain.db
```

### Data Pipeline Integration

```bash
# Continuous monitoring with block metrics
geth --trace.blockmetrics --trace.blockmetrics.path=/data/blocks --datadir=/ethereum/mainnet

# Continuous monitoring with opcode timer
geth --trace.opcodetimer --trace.opcodetimer.path=/data/opcodes --datadir=/ethereum/mainnet
```

## Performance Considerations

### Block Metrics Tracer
- **Overhead**: Low for basic metrics, medium with `--trace.blockmetrics.detailed-tx`
- **Disk I/O**: Moderate (one JSON line per block)
- **Memory**: Low (metrics reset per block)

### Opcode Timer Tracer  
- **Overhead**: Medium to high (timing measurement on every opcode)
- **Disk I/O**: Low (one JSON object per block)
- **Memory**: Low (fixed 256-element arrays)

## Data Analysis Tips

### Block Metrics Analysis
- Use `processing_time` to identify slow blocks
- Monitor `net_storage_growth` for state growth trends
- Analyze `unique_log_topics` for contract activity patterns
- Track `distinct_trie_node_reads` for node sync performance

### Opcode Timer Analysis
- Sum `opcode_count` for total instructions per block
- Use `opcode_ns` / `opcode_count` for average execution time per opcode
- Monitor expensive opcodes (SLOAD, SSTORE, CALL, CREATE)
- Compare `total_time_ns` vs sum of `opcode_ns` for EVM efficiency

### Tracer Selection
- Use **Block Metrics** for comprehensive block-level analysis, storage patterns, and transaction metrics
- Use **Opcode Timer** for detailed performance analysis and opcode-level optimization
- Switch between tracers for different analysis phases (cannot run simultaneously)

## Troubleshooting

### Common Issues
1. **Missing output**: Ensure blockchain is processing blocks (not just syncing)
2. **Large files**: Configure log rotation or use shorter time windows
3. **High CPU usage**: Consider sampling or reducing trace scope
4. **Disk space**: Monitor output directory size, especially with detailed-tx enabled
5. **Tracer conflicts**: Only one tracer can run at a time - ensure you're not enabling multiple tracers

### Output Validation
- Block numbers should be sequential
- Opcode counts should sum to reasonable values
- Timing values should be positive and consistent
- Gas usage should match block header values