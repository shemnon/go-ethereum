// Copyright 2022 The go-ethereum Authors
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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func TestValidateCode(t *testing.T) {
	for i, test := range []struct {
		code     []byte
		section  int
		metadata []*FunctionMetadata
		err      error
	}{
		{
			code: []byte{
				byte(CALLER),
				byte(POP),
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
		},
		{
			code: []byte{
				byte(CALLF), 0x00, 0x00,
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 0}},
		},
		{
			code: []byte{
				byte(ADDRESS),
				byte(CALLF), 0x00, 0x00,
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
		},
		{
			code: []byte{
				byte(CALLER),
				byte(POP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
			err:      ErrInvalidCodeTermination,
		},
		{
			code: []byte{
				byte(RJUMP),
				byte(0x00),
				byte(0x01),
				byte(CALLER),
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 0}},
			err:      ErrUnreachableCode,
		},
		{
			code: []byte{
				byte(PUSH1),
				byte(0x42),
				byte(ADD),
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
			err:      ErrStackUnderflow{stackLen: 1, required: 2},
		},
		{
			code: []byte{
				byte(PUSH1),
				byte(0x42),
				byte(POP),
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 2}},
			err:      ErrInvalidMaxStackHeight,
		},
		{
			code: []byte{
				byte(PUSH0),
				byte(RJUMPI),
				byte(0x00),
				byte(0x01),
				byte(PUSH1),
				byte(0x42), // jumps to here
				byte(POP),
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
			err:      ErrInvalidJumpDest,
		},
		{
			code: []byte{
				byte(PUSH0),
				byte(RJUMPV),
				byte(0x01),
				byte(0x00),
				byte(0x01),
				byte(0x00),
				byte(0x02),
				byte(PUSH1),
				byte(0x42), // jumps to here
				byte(POP),  // and here
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
			err:      ErrInvalidJumpDest,
		},
		{
			code: []byte{
				byte(PUSH0),
				byte(RJUMPV),
				byte(0x00),
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
			err:      ErrTruncatedImmediate,
		},
		{
			code: []byte{
				byte(RJUMP), 0x00, 0x03,
				byte(JUMPDEST), // this code is unreachable to forward jumps alone
				byte(JUMPDEST),
				byte(RETURN),
				byte(PUSH1), 20,
				byte(PUSH1), 39,
				byte(PUSH1), 0x00,
				byte(DATACOPY),
				byte(PUSH1), 20,
				byte(PUSH1), 0x00,
				byte(RJUMP), 0xff, 0xef,
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 3}},
			err:      ErrUnreachableCode,
		},
		{
			code: []byte{
				byte(PUSH1), 1,
				byte(RJUMPI), 0x00, 0x03,
				byte(JUMPDEST),
				byte(JUMPDEST),
				byte(STOP),
				byte(PUSH1), 20,
				byte(PUSH1), 39,
				byte(PUSH1), 0x00,
				byte(DATACOPY),
				byte(PUSH1), 20,
				byte(PUSH1), 0x00,
				byte(RETURN),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 3}},
		},
		{
			code: []byte{
				byte(PUSH1), 1,
				byte(RJUMPV), 0x01, 0x00, 0x03, 0xff, 0xf8,
				byte(JUMPDEST),
				byte(JUMPDEST),
				byte(STOP),
				byte(PUSH1), 20,
				byte(PUSH1), 39,
				byte(PUSH1), 0x00,
				byte(DATACOPY),
				byte(PUSH1), 20,
				byte(PUSH1), 0x00,
				byte(RETURN),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 3}},
		},
		{
			code: []byte{
				byte(STOP),
				byte(STOP),
				byte(INVALID),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 0}},
			err:      ErrUnreachableCode,
		},
		{
			code: []byte{
				byte(RETF),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 1, MaxStackHeight: 0}},
			err:      ErrInvalidOutputs,
		},
		{
			code: []byte{
				byte(RETF),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 3, Output: 3, MaxStackHeight: 3}},
		},
		{
			code: []byte{
				byte(CALLF), 0x00, 0x01,
				byte(POP),
				byte(STOP),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}, {Input: 0, Output: 1, MaxStackHeight: 0}},
		},
		{
			code: []byte{
				byte(ORIGIN),
				byte(ORIGIN),
				byte(CALLF), 0x00, 0x01,
				byte(POP),
				byte(RETF),
			},
			section:  0,
			metadata: []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 2}, {Input: 2, Output: 1, MaxStackHeight: 2}},
		},
	} {
		container := &Container{
			Types:             test.metadata,
			Data:              make([]byte, 0),
			ContainerSections: make([]*Container, 0),
		}
		_, err := validateCode(test.code, test.section, container, &pragueEOFInstructionSet)
		if !errors.Is(err, test.err) {
			t.Errorf("test %d (%s): unexpected error (want: %v, got: %v)", i, common.Bytes2Hex(test.code), test.err, err)
		}
	}
}

func BenchmarkRJUMPI(b *testing.B) {
	snippet := []byte{
		byte(PUSH0),
		byte(RJUMPI), 0xFF, 0xFC,
	}
	code := []byte{}
	for i := 0; i < params.MaxCodeSize/len(snippet)-1; i++ {
		code = append(code, snippet...)
	}
	code = append(code, byte(STOP))
	container := &Container{
		Types:             []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
		Data:              make([]byte, 0),
		ContainerSections: make([]*Container, 0),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validateCode(code, 0, container, &pragueEOFInstructionSet)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRJUMPV(b *testing.B) {
	snippet := []byte{
		byte(PUSH0),
		byte(RJUMPV),
		0xff, // count
		0x00, 0x00,
	}
	for i := 0; i < 255; i++ {
		snippet = append(snippet, []byte{0x00, 0x00}...)
	}
	code := []byte{}
	for i := 0; i < 24576/len(snippet)-1; i++ {
		code = append(code, snippet...)
	}
	code = append(code, byte(PUSH0))
	code = append(code, byte(STOP))
	container := &Container{
		Types:             []*FunctionMetadata{{Input: 0, Output: 0, MaxStackHeight: 1}},
		Data:              make([]byte, 0),
		ContainerSections: make([]*Container, 0),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validateCode(code, 0, container, &pragueEOFInstructionSet)
		if err != nil {
			b.Fatal(err)
		}
	}
}
