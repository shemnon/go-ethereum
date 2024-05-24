package vm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/params"
)

type bounds struct {
	min int
	max int
}

func validateControlFlow2(code []byte, section int, metadata []*FunctionMetadata, jt *JumpTable) (int, error) {
	var (
		stackBounds    = make(map[int]*bounds)
		maxStackHeight = int(metadata[section].Input)
	)

	setBounds := func(pos, min, maxi int) *bounds {
		stackBounds[pos] = &bounds{min, maxi}
		maxStackHeight = max(maxStackHeight, maxi)
		return stackBounds[pos]
	}
	// set the initial stack bounds
	setBounds(0, int(metadata[section].Input), int(metadata[section].Input))

	for pos := 0; pos < len(code); pos++ {
		op := OpCode(code[pos])
		currentBounds := stackBounds[pos]
		if currentBounds == nil {
			fmt.Println("Stack bounds not set")
			return 0, ErrUnreachableCode
		}

		switch op {
		case CALLF:
			arg, _ := parseUint16(code[pos+1:])
			newSection := metadata[arg]
			if want, have := int(newSection.Input), currentBounds.min; want > have {
				return 0, fmt.Errorf("%w: at pos %d", ErrStackUnderflow{stackLen: have, required: want}, pos)
			}
			if have, limit := currentBounds.max+int(newSection.MaxStackHeight)-int(newSection.Input), int(params.StackLimit); have > limit {
				return 0, fmt.Errorf("%w: at pos %d", ErrStackOverflow{stackLen: have, limit: limit}, pos)
			}
			change := int(newSection.Output) - int(newSection.Input)
			currentBounds = setBounds(pos, currentBounds.min+change, currentBounds.max+change)
		case RETF:
			if currentBounds.max != currentBounds.min {
				return 0, fmt.Errorf("%w: max %d, min %d, at pos %d", ErrInvalidNumberOfOutputs, currentBounds.max, currentBounds.min, pos)
			}
			if have, want := int(metadata[section].Output), currentBounds.min; have > want { // TODO as I understand the spec, this should be !=, see 2.I
				return 0, fmt.Errorf("%w: have %d, want %d, at pos %d", ErrInvalidOutputs, have, want, pos)
			}
		case JUMPF:
			arg, _ := parseUint16(code[pos+1:])
			newSection := metadata[arg]
			if have, limit := currentBounds.max+int(newSection.MaxStackHeight)-int(newSection.Input), int(params.StackLimit); have > limit {
				return 0, fmt.Errorf("%w: at pos %d", ErrStackOverflow{stackLen: have, limit: limit}, pos)
			}
			if newSection.Output == 0x80 {
				if want, have := int(newSection.Input), currentBounds.min; want > have {
					return 0, fmt.Errorf("%w: at pos %d", ErrStackUnderflow{stackLen: have, required: want}, pos)
				}
			} else {
				if currentBounds.max != currentBounds.min {
					return 0, fmt.Errorf("%w: max %d, min %d, at pos %d", ErrInvalidNumberOfOutputs, currentBounds.max, currentBounds.min, pos)
				}
				if have, want := currentBounds.max, int(metadata[section].Output)+int(newSection.Input)-int(newSection.Output); have != want {
					return 0, fmt.Errorf("%w: at pos %d", ErrInvalidNumberOfOutputs, pos)
				}
			}
		case DUPN, SWAPN:
			arg := int(code[pos+1]) + 1
			if want, have := arg, currentBounds.min; want > have {
				return 0, fmt.Errorf("%w: at pos %d", ErrStackUnderflow{stackLen: have, required: want}, pos)
			}
			pos += 2
		case EXCHANGE:
			arg := int(code[pos+1])
			n := arg>>4 + 1
			m := arg&0x0f + 1
			if want, have := n+m, currentBounds.min; want > have {
				return 0, fmt.Errorf("%w: at pos %d", ErrStackUnderflow{stackLen: have, required: want}, pos)
			}
		default:
			if want, have := jt[op].minStack, currentBounds.min; want > have {
				return 0, fmt.Errorf("%w: at pos %d", ErrStackUnderflow{stackLen: have, required: want}, pos)
			}
		}

		var (
			currentStackMax = currentBounds.max
			currentStackMin = currentBounds.min
		)

		if !jt[op].terminal && op != CALLF {
			change := int(params.StackLimit) - jt[op].maxStack
			currentStackMax += change
			currentStackMin += change
		}

		var next []int
		switch op {
		case RJUMP:
			arg := parseInt16(code[pos+1:])
			next = append(next, pos+2+arg)
		case RJUMPI:
			arg := parseInt16(code[pos+1:])
			next = append(next, pos+2)
			next = append(next, pos+2+arg)
		case RJUMPV:
			count := int(code[pos+1]) + 1
			next = append(next, pos+1+2*count)
			for i := 0; i < count-1; i++ {
				arg := parseInt16(code[pos+2+2*i:])
				next = append(next, pos+1+2*count+arg)
			}
		default:
			if jt[op].immediate != 0 {
				next = append(next, pos+jt[op].immediate)
			} else {
				// Simple op, no operand.
				next = append(next, pos)
			}
		}

		for _, instr := range next[1:] {
			nextPC := instr + 1
			if nextPC >= len(code) {
				return 0, fmt.Errorf("%w: end with %s, pos %d", ErrInvalidCodeTermination, op, pos)
			}
			nextOP := code[nextPC]
			if nextPC >= pos {
				// target reached via forward jump or seq flow
				nextBounds, ok := stackBounds[nextPC]
				if !ok {
					setBounds(nextPC, currentStackMin, currentStackMax)
				} else {
					setBounds(nextPC, min(nextBounds.min, currentStackMin), max(nextBounds.max, currentStackMax))
				}
			} else {
				// target reached via backwards jump
				nextBounds, ok := stackBounds[nextPC]
				if !ok {
					return 0, ErrInvalidBackwardJump
				}
				change := int(params.StackLimit) - jt[nextOP].maxStack + jt[nextOP].minStack
				if have, want := nextBounds.max+change, currentBounds.max; have != want {
					return 0, fmt.Errorf("%w want %d as max got %d at pos %d,", ErrInvalidBackwardJump, want, have, pos)
				}
				if have, want := nextBounds.min+change, currentBounds.min; have != want {
					return 0, fmt.Errorf("%w want %d as min got %d at pos %d,", ErrInvalidBackwardJump, want, have, pos)
				}
			}
		}

		// next[0] must always be the next operation -1
		if op != RJUMP {
			pos = next[0]
		}
		if !jt[op].terminal {
			setBounds(pos+1, currentStackMin, currentStackMax)
		}
	}
	if maxStackHeight >= int(params.StackLimit) {
		return 0, ErrStackOverflow{maxStackHeight, int(params.StackLimit)}
	}
	if maxStackHeight != int(metadata[section].MaxStackHeight) {
		return 0, fmt.Errorf("%w in code section %d: have %d, want %d", ErrInvalidMaxStackHeight, section, maxStackHeight, metadata[section].MaxStackHeight)
	}
	return len(stackBounds), nil
}
