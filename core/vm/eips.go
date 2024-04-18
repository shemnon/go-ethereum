// Copyright 2019 The go-ethereum Authors
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

package vm

import (
	"errors"
	"fmt"
	"sort"

	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var activators = map[int]func(*JumpTable){
	5656: enable5656,
	6780: enable6780,
	3855: enable3855,
	3860: enable3860,
	3529: enable3529,
	3198: enable3198,
	2929: enable2929,
	2200: enable2200,
	1884: enable1884,
	1344: enable1344,
	1153: enable1153,
}

// EnableEIP enables the given EIP on the config.
// This operation writes in-place, and callers need to ensure that the globally
// defined jump tables are not polluted.
func EnableEIP(eipNum int, jt *JumpTable) error {
	enablerFn, ok := activators[eipNum]
	if !ok {
		return fmt.Errorf("undefined eip %d", eipNum)
	}
	enablerFn(jt)
	return nil
}

func ValidEip(eipNum int) bool {
	_, ok := activators[eipNum]
	return ok
}
func ActivateableEips() []string {
	var nums []string
	for k := range activators {
		nums = append(nums, fmt.Sprintf("%d", k))
	}
	sort.Strings(nums)
	return nums
}

// enable1884 applies EIP-1884 to the given jump table:
// - Increase cost of BALANCE to 700
// - Increase cost of EXTCODEHASH to 700
// - Increase cost of SLOAD to 800
// - Define SELFBALANCE, with cost GasFastStep (5)
func enable1884(jt *JumpTable) {
	// Gas cost changes
	jt[SLOAD].constantGas = params.SloadGasEIP1884
	jt[BALANCE].constantGas = params.BalanceGasEIP1884
	jt[EXTCODEHASH].constantGas = params.ExtcodeHashGasEIP1884

	// New opcode
	jt[SELFBALANCE] = &operation{
		execute:     opSelfBalance,
		constantGas: GasFastStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
}

func opSelfBalance(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	balance := interpreter.evm.StateDB.GetBalance(scope.Contract.Address())
	scope.Stack.push(balance)
	return nil, nil
}

// enable1344 applies EIP-1344 (ChainID Opcode)
// - Adds an opcode that returns the current chain’s EIP-155 unique identifier
func enable1344(jt *JumpTable) {
	// New opcode
	jt[CHAINID] = &operation{
		execute:     opChainID,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
}

// opChainID implements CHAINID opcode
func opChainID(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	chainId, _ := uint256.FromBig(interpreter.evm.chainConfig.ChainID)
	scope.Stack.push(chainId)
	return nil, nil
}

// enable2200 applies EIP-2200 (Rebalance net-metered SSTORE)
func enable2200(jt *JumpTable) {
	jt[SLOAD].constantGas = params.SloadGasEIP2200
	jt[SSTORE].dynamicGas = gasSStoreEIP2200
}

// enable2929 enables "EIP-2929: Gas cost increases for state access opcodes"
// https://eips.ethereum.org/EIPS/eip-2929
func enable2929(jt *JumpTable) {
	jt[SSTORE].dynamicGas = gasSStoreEIP2929

	jt[SLOAD].constantGas = 0
	jt[SLOAD].dynamicGas = gasSLoadEIP2929

	jt[EXTCODECOPY].constantGas = params.WarmStorageReadCostEIP2929
	jt[EXTCODECOPY].dynamicGas = gasExtCodeCopyEIP2929

	jt[EXTCODESIZE].constantGas = params.WarmStorageReadCostEIP2929
	jt[EXTCODESIZE].dynamicGas = gasEip2929AccountCheck

	jt[EXTCODEHASH].constantGas = params.WarmStorageReadCostEIP2929
	jt[EXTCODEHASH].dynamicGas = gasEip2929AccountCheck

	jt[BALANCE].constantGas = params.WarmStorageReadCostEIP2929
	jt[BALANCE].dynamicGas = gasEip2929AccountCheck

	jt[CALL].constantGas = params.WarmStorageReadCostEIP2929
	jt[CALL].dynamicGas = gasCallEIP2929

	jt[CALLCODE].constantGas = params.WarmStorageReadCostEIP2929
	jt[CALLCODE].dynamicGas = gasCallCodeEIP2929

	jt[STATICCALL].constantGas = params.WarmStorageReadCostEIP2929
	jt[STATICCALL].dynamicGas = gasStaticCallEIP2929

	jt[DELEGATECALL].constantGas = params.WarmStorageReadCostEIP2929
	jt[DELEGATECALL].dynamicGas = gasDelegateCallEIP2929

	// This was previously part of the dynamic cost, but we're using it as a constantGas
	// factor here
	jt[SELFDESTRUCT].constantGas = params.SelfdestructGasEIP150
	jt[SELFDESTRUCT].dynamicGas = gasSelfdestructEIP2929
}

// enable3529 enabled "EIP-3529: Reduction in refunds":
// - Removes refunds for selfdestructs
// - Reduces refunds for SSTORE
// - Reduces max refunds to 20% gas
func enable3529(jt *JumpTable) {
	jt[SSTORE].dynamicGas = gasSStoreEIP3529
	jt[SELFDESTRUCT].dynamicGas = gasSelfdestructEIP3529
}

// enable3198 applies EIP-3198 (BASEFEE Opcode)
// - Adds an opcode that returns the current block's base fee.
func enable3198(jt *JumpTable) {
	// New opcode
	jt[BASEFEE] = &operation{
		execute:     opBaseFee,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
}

// enable1153 applies EIP-1153 "Transient Storage"
// - Adds TLOAD that reads from transient storage
// - Adds TSTORE that writes to transient storage
func enable1153(jt *JumpTable) {
	jt[TLOAD] = &operation{
		execute:     opTload,
		constantGas: params.WarmStorageReadCostEIP2929,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}

	jt[TSTORE] = &operation{
		execute:     opTstore,
		constantGas: params.WarmStorageReadCostEIP2929,
		minStack:    minStack(2, 0),
		maxStack:    maxStack(2, 0),
	}
}

// opTload implements TLOAD opcode
func opTload(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	loc := scope.Stack.peek()
	hash := common.Hash(loc.Bytes32())
	val := interpreter.evm.StateDB.GetTransientState(scope.Contract.Address(), hash)
	loc.SetBytes(val.Bytes())
	return nil, nil
}

// opTstore implements TSTORE opcode
func opTstore(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	if interpreter.readOnly {
		return nil, ErrWriteProtection
	}
	loc := scope.Stack.pop()
	val := scope.Stack.pop()
	interpreter.evm.StateDB.SetTransientState(scope.Contract.Address(), loc.Bytes32(), val.Bytes32())
	return nil, nil
}

// opBaseFee implements BASEFEE opcode
func opBaseFee(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	baseFee, _ := uint256.FromBig(interpreter.evm.Context.BaseFee)
	scope.Stack.push(baseFee)
	return nil, nil
}

// enable3855 applies EIP-3855 (PUSH0 opcode)
func enable3855(jt *JumpTable) {
	// New opcode
	jt[PUSH0] = &operation{
		execute:     opPush0,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
}

// opPush0 implements the PUSH0 opcode
func opPush0(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	scope.Stack.push(new(uint256.Int))
	return nil, nil
}

// enable3860 enables "EIP-3860: Limit and meter initcode"
// https://eips.ethereum.org/EIPS/eip-3860
func enable3860(jt *JumpTable) {
	jt[CREATE].dynamicGas = gasCreateEip3860
	jt[CREATE2].dynamicGas = gasCreate2Eip3860
}

// enable5656 enables EIP-5656 (MCOPY opcode)
// https://eips.ethereum.org/EIPS/eip-5656
func enable5656(jt *JumpTable) {
	jt[MCOPY] = &operation{
		execute:     opMcopy,
		constantGas: GasFastestStep,
		dynamicGas:  gasMcopy,
		minStack:    minStack(3, 0),
		maxStack:    maxStack(3, 0),
		memorySize:  memoryMcopy,
	}
}

// opMcopy implements the MCOPY opcode (https://eips.ethereum.org/EIPS/eip-5656)
func opMcopy(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		dst    = scope.Stack.pop()
		src    = scope.Stack.pop()
		length = scope.Stack.pop()
	)
	// These values are checked for overflow during memory expansion calculation
	// (the memorySize function on the opcode).
	scope.Memory.Copy(dst.Uint64(), src.Uint64(), length.Uint64())
	return nil, nil
}

// opBlobHash implements the BLOBHASH opcode
func opBlobHash(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	index := scope.Stack.peek()
	if index.LtUint64(uint64(len(interpreter.evm.TxContext.BlobHashes))) {
		blobHash := interpreter.evm.TxContext.BlobHashes[index.Uint64()]
		index.SetBytes32(blobHash[:])
	} else {
		index.Clear()
	}
	return nil, nil
}

// opBlobBaseFee implements BLOBBASEFEE opcode
func opBlobBaseFee(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	blobBaseFee, _ := uint256.FromBig(interpreter.evm.Context.BlobBaseFee)
	scope.Stack.push(blobBaseFee)
	return nil, nil
}

// enable4844 applies EIP-4844 (BLOBHASH opcode)
func enable4844(jt *JumpTable) {
	jt[BLOBHASH] = &operation{
		execute:     opBlobHash,
		constantGas: GasFastestStep,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}
}

// enable7516 applies EIP-7516 (BLOBBASEFEE opcode)
func enable7516(jt *JumpTable) {
	jt[BLOBBASEFEE] = &operation{
		execute:     opBlobBaseFee,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
}

// enable6780 applies EIP-6780 (deactivate SELFDESTRUCT)
func enable6780(jt *JumpTable) {
	jt[SELFDESTRUCT] = &operation{
		execute:     opSelfdestruct6780,
		dynamicGas:  gasSelfdestructEIP3529,
		constantGas: params.SelfdestructGasEIP150,
		minStack:    minStack(1, 0),
		maxStack:    maxStack(1, 0),
	}
}

// enableEOF applies the EOF changes.
func enableEOF(jt *JumpTable) {
	// Deprecate opcodes
	undefined := &operation{
		execute:     opUndefined,
		constantGas: 0,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
		undefined:   true,
	}
	jt[CALL] = undefined
	jt[CALLCODE] = undefined
	jt[DELEGATECALL] = undefined
	jt[STATICCALL] = undefined
	jt[SELFDESTRUCT] = undefined
	jt[JUMP] = undefined
	jt[JUMPI] = undefined
	jt[PC] = undefined
	jt[CREATE] = undefined
	jt[CREATE2] = undefined
	jt[CODESIZE] = undefined
	jt[CODECOPY] = undefined
	jt[EXTCODESIZE] = undefined
	jt[EXTCODECOPY] = undefined
	jt[EXTCODEHASH] = undefined
	jt[GAS] = undefined

	// New opcodes
	jt[RJUMP] = &operation{
		execute:     opRjump,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
		terminal:    true,
		immediate:   2,
	}
	jt[RJUMPI] = &operation{
		execute:     opRjumpi,
		constantGas: GasFastishStep,
		minStack:    minStack(1, 0),
		maxStack:    maxStack(1, 0),
		immediate:   2,
	}
	jt[RJUMPV] = &operation{
		execute:     opRjumpv,
		constantGas: GasFastishStep,
		minStack:    minStack(1, 0),
		maxStack:    maxStack(1, 0),
		immediate:   1, // at least 1, maybe more
	}
	jt[CALLF] = &operation{
		execute:     opCallf,
		constantGas: GasFastStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
		immediate:   2,
	}
	jt[RETF] = &operation{
		execute:     opRetf,
		constantGas: GasFastishStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
		terminal:    true,
	}
	jt[JUMPF] = &operation{
		execute:     opJumpf,
		constantGas: GasFastStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
		immediate:   2,
	}
	jt[EOFCREATE] = &operation{
		execute:     opEOFCreate,
		constantGas: params.Create2Gas,
		dynamicGas:  gasEOFCreate,
		minStack:    minStack(4, 1),
		maxStack:    maxStack(4, 1),
		memorySize:  memoryEOFCreate,
		immediate:   1,
	}
	jt[TXCREATE] = &operation{
		execute:     opTXCreate,
		constantGas: params.Create2Gas,
		dynamicGas:  gasEOFCreate,
		minStack:    minStack(5, 1),
		maxStack:    maxStack(5, 1),
		memorySize:  memoryEOFCreate,
	}
	jt[RETURNCONTRACT] = &operation{
		execute:     opReturnContract,
		constantGas: GasZeroStep,
		dynamicGas:  memoryCopierGas(2),
		minStack:    minStack(2, 0),
		maxStack:    maxStack(2, 0),
		memorySize:  memoryEOFCreate,
		immediate:   1,
	}
	jt[DATALOAD] = &operation{
		execute:     opDataLoad,
		constantGas: GasFastishStep,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}
	jt[DATALOADN] = &operation{
		execute:     opDataLoadN,
		constantGas: GasFastestStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
		immediate:   2,
	}
	jt[DATASIZE] = &operation{
		execute:     opDataSize,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
	jt[DATACOPY] = &operation{
		execute:     opDataCopy,
		constantGas: GasFastestStep,
		dynamicGas:  memoryCopierGas(3),
		minStack:    minStack(3, 0),
		maxStack:    maxStack(3, 0),
		memorySize:  memoryMcopy,
	}
	jt[DUPN] = &operation{
		execute:     opDupN,
		constantGas: GasFastestStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
		immediate:   1,
	}
	jt[SWAPN] = &operation{
		execute:     opSwapN,
		constantGas: GasFastestStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
		immediate:   1,
	}
	jt[EXCHANGE] = &operation{
		execute:     opExchange,
		constantGas: GasFastestStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
		immediate:   1,
	}
	jt[RETURNDATALOAD] = &operation{
		execute:     opExchange,
		constantGas: GasFastestStep,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}
	jt[EXTCALL] = &operation{
		execute:     opExtCall,
		constantGas: params.CallGasEIP150,
		dynamicGas:  gasCall,
		minStack:    minStack(6, 1),
		maxStack:    maxStack(6, 1),
		memorySize:  memoryCall,
	}
	jt[EXTDELEGATECALL] = &operation{
		execute:     opExtDelegateCall,
		dynamicGas:  gasDelegateCall,
		constantGas: params.CallGasEIP150,
		minStack:    minStack(5, 1),
		maxStack:    maxStack(5, 1),
		memorySize:  memoryDelegateCall,
	}
	jt[STATICCALL] = &operation{
		execute:     opExtStaticCall,
		constantGas: params.CallGasEIP150,
		dynamicGas:  gasStaticCall,
		minStack:    minStack(5, 1),
		maxStack:    maxStack(5, 1),
		memorySize:  memoryStaticCall,
	}
}

func opExtCodeCopyEOF(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		stack      = scope.Stack
		a          = stack.pop()
		memOffset  = stack.pop()
		codeOffset = stack.pop()
		length     = stack.pop()
	)
	uint64CodeOffset, overflow := codeOffset.Uint64WithOverflow()
	if overflow {
		uint64CodeOffset = math.MaxUint64
	}
	addr := common.Address(a.Bytes20())
	code := interpreter.evm.StateDB.GetCode(addr)
	// Check if we're copying an EOF contract
	if len(code) >= 2 && code[0] == 0xEF && code[1] == 0x00 {
		code = []byte{0xEF, 0x00}
	}
	codeCopy := getData(code, uint64CodeOffset, length.Uint64())
	scope.Memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy)

	return nil, nil
}

// opRjump implements the rjump opcode.
func opRjump(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code   = scope.Contract.CodeAt(scope.CodeSection)
		offset = parseInt16(code[*pc+1:])
	)
	// move pc past op and operand (+3), add relative offset, subtract 1 to
	// account for interpreter loop.
	*pc = uint64(int64(*pc+3) + int64(offset) - 1)
	return nil, nil
}

// opRjumpi implements the RJUMPI opcode
func opRjumpi(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	condition := scope.Stack.pop()
	if condition.BitLen() == 0 {
		// Not branching, just skip over immediate argument.
		*pc += 2
		return nil, nil
	}
	return opRjump(pc, interpreter, scope)
}

// opRjumpv implements the RJUMPV opcode
func opRjumpv(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code  = scope.Contract.CodeAt(scope.CodeSection)
		count = uint64(code[*pc+1])
		idx   = scope.Stack.pop()
	)
	if idx, overflow := idx.Uint64WithOverflow(); overflow || idx >= count {
		// Index out-of-bounds, don't branch, just skip over immediate
		// argument.
		*pc += 1 + count*2
		return nil, nil
	}
	offset := parseInt16(code[*pc+2+2*idx.Uint64():])
	*pc = uint64(int64(*pc+2+count*2) + int64(offset) - 1)
	return nil, nil
}

// opCallf implements the CALLF opcode
func opCallf(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code = scope.Contract.CodeAt(scope.CodeSection)
		idx  = binary.BigEndian.Uint16(code[*pc+1:])
		typ  = scope.Contract.Container.Types[scope.CodeSection]
	)
	if scope.Stack.len()+int(typ.MaxStackHeight) >= 1024 {
		return nil, fmt.Errorf("stack overflow")
	}
	retCtx := &ReturnContext{
		Section:     scope.CodeSection,
		Pc:          *pc + 3,
		StackHeight: scope.Stack.len() - int(typ.Input),
	}
	scope.ReturnStack = append(scope.ReturnStack, retCtx)
	scope.CodeSection = uint64(idx)
	*pc = 0
	*pc -= 1 // hacks xD
	return nil, nil
}

// opRetf implements the RETF opcode
func opRetf(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		last   = len(scope.ReturnStack) - 1
		retCtx = scope.ReturnStack[last]
	)
	scope.ReturnStack = scope.ReturnStack[:last]
	scope.CodeSection = retCtx.Section
	*pc = retCtx.Pc - 1

	// If returning from top frame, exit cleanly.
	if len(scope.ReturnStack) == 0 {
		return nil, errStopToken
	}
	return nil, nil
}

func opJumpf(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code = scope.Contract.CodeAt(scope.CodeSection)
		idx  = binary.BigEndian.Uint16(code[*pc+1:])
		typ  = scope.Contract.Container.Types[idx]
	)
	if scope.Stack.len()+int(typ.MaxStackHeight)-int(typ.Input) > 1024 {
		return nil, fmt.Errorf("stack overflow")
	}
	scope.CodeSection = uint64(idx)
	*pc = 0
	return nil, nil
}

func opEOFCreate(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	if interpreter.readOnly {
		return nil, ErrWriteProtection
	}
	var (
		code         = scope.Contract.CodeAt(scope.CodeSection)
		idx          = binary.BigEndian.Uint16(code[*pc+1:])
		subcontainer = scope.Contract.Container.ContainerSections[idx]
		value        = scope.Stack.pop()
		salt         = scope.Stack.pop()
		offset, size = scope.Stack.pop(), scope.Stack.pop()
		input        = scope.Memory.GetCopy(int64(offset.Uint64()), int64(size.Uint64()))
		gas          = scope.Contract.Gas
	)
	// Reuse last popped value from stack
	stackvalue := size
	// Apply EIP150
	gas -= gas / 64
	scope.Contract.UseGas(gas, interpreter.evm.Config.Tracer, tracing.GasChangeCallContractCreation2)

	res, addr, returnGas, suberr := interpreter.evm.EOFCreate(scope.Contract, input, subcontainer, gas, &value, &salt)
	if suberr != nil {
		stackvalue.Clear()
	} else {
		stackvalue.SetBytes(addr.Bytes())
	}
	scope.Stack.push(&stackvalue)

	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer, tracing.GasChangeCallLeftOverRefunded)

	if suberr == ErrExecutionReverted {
		interpreter.returnData = res // set REVERT data to return data buffer
		return res, nil
	}
	interpreter.returnData = nil // clear dirty return data buffer
	return nil, nil
}

func opTXCreate(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	if interpreter.readOnly {
		return nil, ErrWriteProtection
	}
	var (
		value          = scope.Stack.pop()
		salt           = scope.Stack.pop()
		offset, size   = scope.Stack.pop(), scope.Stack.pop()
		txInitCodeHash = scope.Stack.pop()
		input          = scope.Memory.GetCopy(int64(offset.Uint64()), int64(size.Uint64()))
		gas            = scope.Contract.Gas
	)
	// Reuse last popped value from stack
	stackvalue := txInitCodeHash

	var initCode []byte
	for _, code := range interpreter.evm.InitCodes {
		if interpreter.hasher == nil {
			interpreter.hasher = crypto.NewKeccakState()
		} else {
			interpreter.hasher.Reset()
		}
		interpreter.hasher.Write(code)
		interpreter.hasher.Read(interpreter.hasherBuf[:])
		if interpreter.hasherBuf.Cmp(txInitCodeHash.Bytes32()) == 0 {
			initCode = code
		}
	}
	if len(initCode) == 0 {
		stackvalue.Clear()
		scope.Stack.push(&stackvalue)
	}

	// Additional hashing charge
	hashingCharge := params.InitCodeWordGas * ((uint64(len(initCode)) + 31) / 32)
	scope.Contract.UseGas(hashingCharge, interpreter.evm.Config.Tracer, tracing.GasChangeCallContractCreation2)
	// Apply EIP150
	gas -= gas / 64
	scope.Contract.UseGas(gas, interpreter.evm.Config.Tracer, tracing.GasChangeCallContractCreation2)
	scope.InitCodeMode = true
	res, addr, returnGas, suberr := interpreter.evm.EOFCreate(scope.Contract, input, initCode, gas, &value, &salt)
	if suberr != nil {
		stackvalue.Clear()
	} else {
		stackvalue.SetBytes(addr.Bytes())
	}
	scope.Stack.push(&stackvalue)

	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer, tracing.GasChangeCallLeftOverRefunded)

	if suberr == ErrExecutionReverted {
		interpreter.returnData = res // set REVERT data to return data buffer
		return res, nil
	}
	interpreter.returnData = nil // clear dirty return data buffer
	return nil, nil
}

func opReturnContract(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	if !scope.InitCodeMode {
		return nil, errors.New("returncontract in non-initcode mode")
	}
	// TODO (MariusVanDerWijden) implement
	return nil, nil
}

func opDataLoad(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		stackItem        = scope.Stack.pop()
		offset, overflow = stackItem.Uint64WithOverflow()
		length           = uint64(len(scope.Contract.Container.Data))
		end              = min(length, offset+32)
	)

	if overflow {
		stackItem.Clear()
		scope.Stack.push(&stackItem)
	} else {
		scope.Stack.push(stackItem.SetBytes(
			common.RightPadBytes(
				scope.Contract.Code[offset:end],
				32,
			)),
		)
	}
	return nil, nil
}

func opDataLoadN(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code   = scope.Contract.CodeAt(scope.CodeSection)
		offset = int(binary.BigEndian.Uint16(code[*pc+1:]))
		end    = min(len(code), offset+32)
	)
	scope.Stack.push(new(uint256.Int).SetBytes(
		common.RightPadBytes(
			scope.Contract.Code[offset:end],
			32,
		)),
	)
	return nil, nil
}

func opDataSize(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	length := len(scope.Contract.Container.Data)
	item := uint256.NewInt(uint64(length))
	scope.Stack.push(item)
	return nil, nil
}

func opDataCopy(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		memOffset = scope.Stack.pop()
		offset    = scope.Stack.pop()
		size      = scope.Stack.pop()
		data      = scope.Contract.Container.Data
		end       = min(uint64(len(data)), offset.Uint64()+size.Uint64())
	)
	// These values are checked for overflow during memory expansion calculation
	// (the memorySize function on the opcode).
	result := make([]byte, size.Uint64())
	copy(result, data[offset.Uint64():end])
	scope.Memory.Set(memOffset.Uint64(), size.Uint64(), data)
	return nil, nil
}

func opDupN(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code  = scope.Contract.CodeAt(scope.CodeSection)
		index = int(code[*pc+1]) + 1
	)
	scope.Stack.dup(index)
	return nil, nil
}

func opSwapN(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code  = scope.Contract.CodeAt(scope.CodeSection)
		index = int(code[*pc+1]) + 1
	)
	scope.Stack.swap(index)
	return nil, nil
}

func opExchange(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		code  = scope.Contract.CodeAt(scope.CodeSection)
		index = int(code[*pc+1]) + 1
		n     = index>>4 + 1
		m     = index%0x0F + 1
	)
	scope.Stack.swapN(n, m)
	return nil, nil
}

func opReturnDataLoad(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		offset = scope.Stack.pop()
	)
	if offset.Uint64()+32 > uint64(len(interpreter.returnData)) {
		return nil, errors.New("return buffer overflow")
	}
	scope.Stack.push(offset.SetBytes(interpreter.returnData[offset.Uint64() : offset.Uint64()+32]))
	return nil, nil
}

func opExtCall(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	stack := scope.Stack
	// Use all available gas
	gas := (scope.Contract.Gas / 64) * 63
	// Pop other call parameters.
	addr, value, inOffset, inSize := stack.pop(), stack.pop(), stack.pop(), stack.pop()
	toAddr := common.Address(addr.Bytes20())
	// safe a memory alloc
	temp := addr
	// Get the arguments from the memory.
	args := scope.Memory.GetPtr(int64(inOffset.Uint64()), int64(inSize.Uint64()))

	if interpreter.readOnly && !value.IsZero() {
		return nil, ErrWriteProtection
	}
	if !value.IsZero() {
		gas += params.CallStipend
	}
	ret, returnGas, err := interpreter.evm.Call(scope.Contract, toAddr, args, gas, &value)

	if err != nil {
		temp.Clear()
	} else {
		temp.SetOne()
	}
	stack.push(&temp)

	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer, tracing.GasChangeCallLeftOverRefunded)

	interpreter.returnData = ret
	return ret, nil
}

func opExtDelegateCall(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	stack := scope.Stack
	// Use all available gas
	gas := (scope.Contract.Gas / 64) * 63
	// Pop other call parameters.
	addr, inOffset, inSize := stack.pop(), stack.pop(), stack.pop()
	toAddr := common.Address(addr.Bytes20())
	// safe a memory alloc
	temp := addr
	// Get arguments from the memory.
	args := scope.Memory.GetPtr(int64(inOffset.Uint64()), int64(inSize.Uint64()))

	ret, returnGas, err := interpreter.evm.DelegateCall(scope.Contract, toAddr, args, gas, true)
	if err != nil {
		temp.Clear()
	} else {
		temp.SetOne()
	}
	stack.push(&temp)

	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer, tracing.GasChangeCallLeftOverRefunded)

	interpreter.returnData = ret
	return ret, nil
}

func opExtStaticCall(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	stack := scope.Stack
	// Use all available gas
	gas := (scope.Contract.Gas / 64) * 63
	// Pop other call parameters.
	addr, inOffset, inSize := stack.pop(), stack.pop(), stack.pop()
	toAddr := common.Address(addr.Bytes20())
	// safe a memory alloc
	temp := addr
	// Get arguments from the memory.
	args := scope.Memory.GetPtr(int64(inOffset.Uint64()), int64(inSize.Uint64()))

	ret, returnGas, err := interpreter.evm.StaticCall(scope.Contract, toAddr, args, gas)
	if err != nil {
		temp.Clear()
	} else {
		temp.SetOne()
	}
	stack.push(&temp)

	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer, tracing.GasChangeCallLeftOverRefunded)

	interpreter.returnData = ret
	return ret, nil
}
