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

package vm

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// opRjump implements the RJUMP opcode.
func opRjump(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		offset = parseInt16(scope.Contract.Code[*pc+1:])
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
		maxIndex = uint64(scope.Contract.Code[*pc+1]) + 1
		idx      = scope.Stack.pop()
	)
	if idx, overflow := idx.Uint64WithOverflow(); overflow || idx >= maxIndex {
		// Index out-of-bounds, don't branch, just skip over immediate
		// argument.
		*pc += 1 + maxIndex*2
		return nil, nil
	}
	offset := parseInt16(scope.Contract.Code[*pc+2+2*idx.Uint64():])
	// move pc past op and count byte (2), move past count number of 16bit offsets (count*2), add relative offset, subtract 1 to
	// account for interpreter loop.
	*pc = uint64(int64(*pc+2+maxIndex*2) + int64(offset) - 1)
	return nil, nil
}

// opCallf implements the CALLF opcode
func opCallf(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		idx = binary.BigEndian.Uint16(scope.Contract.Code[*pc+1:])
		typ = scope.Contract.Container.types[idx]
	)
	if scope.Stack.len()+int(typ.maxStackHeight)-int(typ.inputs) > 1024 {
		return nil, fmt.Errorf("stack overflow")
	}
	if scope.ReturnStack.Len() >= 1024 {
		return nil, fmt.Errorf("return stack overflow")
	}
	retCtx := &ReturnContext{
		Section:     scope.CodeSection,
		Pc:          *pc + 3,
		StackHeight: scope.Stack.len() - int(typ.inputs),
	}
	scope.ReturnStack = append(scope.ReturnStack, retCtx)
	scope.CodeSection = uint64(idx)
	*pc = uint64(scope.Contract.Container.codeSectionOffsets[idx]) - 1
	return nil, nil
}

// opRetf implements the RETF opcode
func opRetf(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	retCtx := scope.ReturnStack.Pop()
	scope.CodeSection = retCtx.Section
	*pc = retCtx.Pc - 1

	return nil, nil
}

// opJumpf implements the JUMPF opcode
func opJumpf(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		idx = binary.BigEndian.Uint16(scope.Contract.Code[*pc+1:])
		typ = scope.Contract.Container.types[idx]
	)
	if scope.Stack.len()+int(typ.maxStackHeight)-int(typ.inputs) > 1024 {
		return nil, fmt.Errorf("stack overflow")
	}
	scope.CodeSection = uint64(idx)
	*pc = uint64(scope.Contract.Container.codeSectionOffsets[idx]) - 1
	return nil, nil
}

// opEOFCreate implements the EOFCREATE opcode
func opEOFCreate(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	if interpreter.readOnly {
		return nil, ErrWriteProtection
	}
	var (
		idx          = scope.Contract.Code[*pc+1]
		value        = scope.Stack.pop()
		salt         = scope.Stack.pop()
		offset, size = scope.Stack.pop(), scope.Stack.pop()
		input        = scope.Memory.GetCopy(offset.Uint64(), size.Uint64())
	)
	if int(idx) >= len(scope.Contract.Container.subContainerOffsets) {
		return nil, fmt.Errorf("invalid subcontainer")
	}

	// Deduct hashing charge
	// Since size <= params.MaxInitCodeSize, these multiplication cannot overflow
	hashingCharge := (params.Keccak256WordGas) * ((uint64(scope.Contract.Container.subContainerSize(int(idx))) + 31) / 32)
	if ok := scope.Contract.UseGas(hashingCharge, interpreter.evm.Config.Tracer, tracing.GasChangeUnspecified); !ok {
		return nil, ErrGasUintOverflow
	}
	if interpreter.evm.Config.Tracer != nil {
		if interpreter.evm.Config.Tracer != nil {
			interpreter.evm.Config.Tracer.OnOpcode(*pc, scope.CodeSectionNum(), byte(EOFCREATE), 0, hashingCharge, scope, interpreter.returnData, interpreter.evm.depth, scope.ReturnStackDepth(), nil)
		}
	}
	gas := scope.Contract.Gas
	// Reuse last popped value from stack
	stackvalue := size
	// Apply EIP150
	gas -= gas / 64
	scope.Contract.UseGas(gas, interpreter.evm.Config.Tracer, tracing.GasChangeCallContractCreation2)
	// Skip the immediate
	*pc += 1
	res, addr, returnGas, suberr := interpreter.evm.EOFCreate(scope.Contract, input, scope.Contract.Container.subContainerBytes(int(idx)), gas, &value, &salt)
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

// opReturnContract implements the RETURNCONTRACT opcode
func opReturnContract(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	if !scope.InitCodeMode {
		return nil, errors.New("returncontract in non-initcode mode")
	}
	var (
		idx    = scope.Contract.Code[*pc+1]
		offset = scope.Stack.pop()
		size   = scope.Stack.pop()
	)
	if int(idx) >= len(scope.Contract.Container.subContainerOffsets) {
		return nil, fmt.Errorf("invalid subcontainer")
	}
	ret := scope.Memory.GetPtr(offset.Uint64(), size.Uint64())
	containerCode := scope.Contract.Container.subContainerBytes(int(idx))
	if len(containerCode) == 0 {
		return nil, errors.New("nonexistant subcontainer")
	}
	// Validate the subcontainer
	var c Container
	if err := c.UnmarshalSubContainer(containerCode, false); err != nil {
		return nil, err
	}

	// append the auxdata
	c.rawContainer = append(c.rawContainer, ret...)
	newDataSize := c.dataLen()
	if newDataSize < c.dataSize {
		return nil, errors.New("incomplete aux data")
	}
	c.dataSize = newDataSize

	// probably unneeded as subcontainers are deeply validated
	if err := c.ValidateCode(interpreter.tableEOF, false); err != nil {
		return nil, err
	}

	return c.MarshalBinary(), errStopToken
}

// opDataLoad implements the DATALOAD opcode
func opDataLoad(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		stackItem        = scope.Stack.pop()
		offset, overflow = stackItem.Uint64WithOverflow()
	)
	if overflow {
		stackItem.Clear()
		scope.Stack.push(&stackItem)
	} else {
		data := getData(scope.Contract.Container.rawContainer, uint64(scope.Contract.Container.dataOffest)+offset, 32)
		scope.Stack.push(stackItem.SetBytes(data))
	}
	return nil, nil
}

// opDataLoadN implements the DATALOADN opcode
func opDataLoadN(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		offset = uint64(binary.BigEndian.Uint16(scope.Contract.Code[*pc+1:]))
	)
	data := getData(scope.Contract.Container.rawContainer, uint64(scope.Contract.Container.dataOffest)+offset, 32)
	scope.Stack.push(new(uint256.Int).SetBytes(data))
	*pc += 2 // move past 2 byte immediate
	return nil, nil
}

// opDataSize implements the DATASIZE opcode
func opDataSize(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	item := uint256.NewInt(uint64(scope.Contract.Container.dataLen()))
	scope.Stack.push(item)
	return nil, nil
}

// opDataCopy implements the DATACOPY opcode
func opDataCopy(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		memOffset = scope.Stack.pop()
		offset    = scope.Stack.pop()
		size      = scope.Stack.pop()
	)
	// These values are checked for overflow during memory expansion calculation
	// (the memorySize function on the opcode).
	data := getData(scope.Contract.Container.rawContainer, uint64(scope.Contract.Container.dataOffest)+offset.Uint64(), size.Uint64())
	scope.Memory.Set(memOffset.Uint64(), size.Uint64(), data)
	return nil, nil
}

// opDupN implements the DUPN opcode
func opDupN(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		index = int(scope.Contract.Code[*pc+1]) + 1
	)
	scope.Stack.dup(index)
	*pc += 1 // move past immediate
	return nil, nil
}

// opSwapN implements the SWAPN opcode
func opSwapN(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		index = int(scope.Contract.Code[*pc+1]) + 1
	)
	scope.Stack.swap(index + 1)
	*pc += 1 // move past immediate
	return nil, nil
}

// opExchange implements the EXCHANGE opcode
func opExchange(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		index = int(scope.Contract.Code[*pc+1])
		n     = (index >> 4) + 1
		m     = (index & 0x0F) + 1
	)
	scope.Stack.swapN(n+1, n+m+1)
	*pc += 1 // move past immediate
	return nil, nil
}

// opReturnDataLoad implements the RETURNDATALOAD opcode
func opReturnDataLoad(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		offset = scope.Stack.pop()
	)
	offset64, overflow := offset.Uint64WithOverflow()
	if overflow {
		offset64 = math.MaxUint64
	}
	scope.Stack.push(offset.SetBytes(getData(interpreter.returnData, offset64, 32)))
	return nil, nil
}

// opExtCall implements the EOFCREATE opcode
func opExtCall(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	stack := scope.Stack
	// Use all available gas
	gas := interpreter.evm.callGasTemp
	// Pop other call parameters.
	addr, inOffset, inSize, value := stack.pop(), stack.pop(), stack.pop(), stack.pop()
	toAddr := common.Address(addr.Bytes20())
	if addr.ByteLen() > 20 {
		return nil, errors.New("address space extension")
	}
	// safe a memory alloc
	temp := addr
	// Get the arguments from the memory.
	args := scope.Memory.GetPtr(inOffset.Uint64(), inSize.Uint64())

	if interpreter.readOnly && !value.IsZero() {
		return nil, ErrWriteProtection
	}

	var (
		ret       []byte
		returnGas uint64
		err       error
	)
	if interpreter.evm.callGasTemp == 0 {
		// zero temp call gas indicates a min retained gas error
		ret, returnGas, err = nil, 0, ErrExecutionReverted
	} else {
		ret, returnGas, err = interpreter.evm.Call(scope.Contract, toAddr, args, gas, &value)
	}

	if errors.Is(err, ErrExecutionReverted) || errors.Is(err, ErrInsufficientBalance) || errors.Is(err, ErrDepth) {
		temp.SetOne()
	} else if err != nil {
		temp.SetUint64(2)
	} else {
		temp.Clear()
	}
	stack.push(&temp)
	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer, tracing.GasChangeCallLeftOverRefunded)

	interpreter.returnData = ret
	return ret, nil
}

// opExtDelegateCall implements the EXTDELEGATECALL opcode
func opExtDelegateCall(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	stack := scope.Stack
	// Use all available gas
	gas := interpreter.evm.callGasTemp
	// Pop other call parameters.
	addr, inOffset, inSize := stack.pop(), stack.pop(), stack.pop()
	toAddr := common.Address(addr.Bytes20())
	if addr.ByteLen() > 20 {
		return nil, errors.New("address space extension")
	}
	// safe a memory alloc
	temp := addr
	// Get arguments from the memory.
	args := scope.Memory.GetPtr(inOffset.Uint64(), inSize.Uint64())

	// Check that we're only calling non-legacy contracts
	var (
		err       error
		ret       []byte
		returnGas uint64
	)
	code := interpreter.evm.StateDB.GetCode(toAddr)
	if !hasEOFMagic(code) {
		// Delegate-calling a non-eof contract should return 1
		err = ErrExecutionReverted
		ret = nil
		returnGas = gas
	} else if interpreter.evm.callGasTemp == 0 {
		// zero temp call gas indicates a min retained gas error
		ret, returnGas, err = nil, 0, ErrExecutionReverted
	} else {
		ret, returnGas, err = interpreter.evm.DelegateCall(scope.Contract, toAddr, args, gas, true)
	}

	if err == ErrExecutionReverted || err == ErrDepth {
		temp.SetOne()
	} else if err != nil {
		temp.SetUint64(2)
	} else {
		temp.Clear()
	}
	stack.push(&temp)

	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer, tracing.GasChangeCallLeftOverRefunded)

	interpreter.returnData = ret
	return ret, nil
}

// opExtStaticCall implements the EXTSTATICCALL opcode
func opExtStaticCall(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	stack := scope.Stack
	// Use all available gas
	gas := interpreter.evm.callGasTemp
	// Pop other call parameters.
	addr, inOffset, inSize := stack.pop(), stack.pop(), stack.pop()
	toAddr := common.Address(addr.Bytes20())
	if addr.ByteLen() > 20 {
		return nil, errors.New("address space extension")
	}
	// safe a memory alloc
	temp := addr
	// Get arguments from the memory.
	args := scope.Memory.GetPtr(inOffset.Uint64(), inSize.Uint64())

	var (
		ret       []byte
		returnGas uint64
		err       error
	)
	if interpreter.evm.callGasTemp == 0 {
		// zero temp call gas indicates a min retained gas error
		ret, returnGas, err = nil, 0, ErrExecutionReverted
	} else {
		ret, returnGas, err = interpreter.evm.StaticCall(scope.Contract, toAddr, args, gas)
	}

	if err == ErrExecutionReverted || err == ErrDepth {
		temp.SetOne()
	} else if err != nil {
		temp.SetUint64(2)
	} else {
		temp.Clear()
	}
	stack.push(&temp)

	scope.Contract.RefundGas(returnGas, interpreter.evm.Config.Tracer, tracing.GasChangeCallLeftOverRefunded)

	interpreter.returnData = ret
	return ret, nil
}
