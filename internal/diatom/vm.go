package diatom

import (
	"fmt"
	"io"
	"os"
)

const (
	StackSize    = 30
	IOBufferSize = 4096
	MemorySize   = 8192
)

type Stack struct {
	cursor int
	data   [StackSize]Word
}

func (s *Stack) Push(value Word) error {
	if s.cursor+1 >= len(s.data) {
		return fmt.Errorf("push: stack overflow - cursor: %d, stack size: %d",
			s.cursor, len(s.data))
	}

	s.data[s.cursor] = value
	s.cursor++
	return nil
}

func (s *Stack) Pop() (Word, error) {
	if s.cursor <= 0 {
		return 0, fmt.Errorf("pop: stack underflow - cursor: %d, stack size: %d",
			s.cursor, len(s.data))
	}

	s.cursor--
	return s.data[s.cursor], nil
}

func (s *Stack) Peek() (Word, error) {
	if s.cursor <= 0 {
		return 0, fmt.Errorf("peek: stack underflow - cursor: %d, stack size: %d",
			s.cursor, len(s.data))
	}

	return s.data[s.cursor-1], nil
}

type Input struct {
	cursor int
	len    int
	buffer [IOBufferSize]byte
}

func (i *Input) nextChar(r io.Reader) (byte, error) {
	if i.cursor >= i.len {
		read, err := io.ReadAtLeast(r, i.buffer[:], 1)
		if read == 0 || err != nil {
			return 0, fmt.Errorf("failed to read from input: %w", err)
		}

		i.len = read
		i.cursor = 0
	}

	b := i.buffer[i.cursor]
	i.cursor++

	return b, nil
}

type VM struct {
	programCounter Word
	dataStack      Stack
	returnStack    Stack
	inputBuffer    Input
	input          io.Reader
	output         io.Writer
	memory         [MemorySize]byte
}

func NewVM(program []byte, input io.Reader, output io.Writer) (*VM, error) {
	programLen := len(program)
	if programLen > MemorySize {
		return nil, fmt.Errorf("program length (%d bytes) exceeds available memory (%d bytes)",
			programLen, MemorySize)
	}

	vm := VM{
		input:  input,
		output: output,
	}
	copy(vm.memory[:], program)

	return &vm, nil
}

func NewDefaultVM(program []byte) (*VM, error) {
	return NewVM(program, os.Stdin, os.Stdout)
}

func (vm *VM) validateMemoryAccess(addr Word) error {
	if addr >= Word(len(vm.memory)) || addr < 0 {
		return fmt.Errorf("out of bound memory access: programCounter=%d address=%d",
			vm.programCounter, addr)
	}

	return nil
}

func (vm *VM) fetchByte(addr Word) (byte, error) {
	if err := vm.validateMemoryAccess(addr); err != nil {
		return 0, err
	}

	return vm.memory[addr], nil
}

func (vm *VM) storeByte(addr Word, b byte) error {
	if err := vm.validateMemoryAccess(addr); err != nil {
		return err
	}

	vm.memory[addr] = b
	return nil
}

func (vm *VM) fetchWord(addr Word) (Word, error) {
	var w Word
	for i := 0; i < WordSize; i++ {
		b, err := vm.fetchByte(addr + Word(i))
		if err != nil {
			return w, err
		}

		shift := (WordSize - (i + 1)) * 8
		w = w | (Word(b) << shift)
	}

	return w, nil
}

func (vm *VM) storeWord(addr Word, w Word) error {
	var b byte
	for i := 0; i < WordSize; i++ {
		shift := (WordSize - (i + 1)) * 8
		b = byte(w & (0xff << shift))

		if err := vm.storeByte(addr+Word(i), b); err != nil {
			return err
		}
	}

	return nil
}

func (vm *VM) key() (byte, error) {
	return vm.inputBuffer.nextChar(vm.input)
}

func (vm *VM) emit(b byte) error {
	if _, err := vm.output.Write([]byte{b}); err != nil {
		return fmt.Errorf("failed to write %q to the VM output: %w", b, err)
	}

	return nil
}

func boolToWord(b bool) Word {
	if b {
		return -1
	}

	return 0
}

func (vm *VM) Execute() error {
	for {
		instruction := vm.memory[vm.programCounter]

		switch instruction {
		case EXIT:
			return nil
		case NOP:
		case RET:
			addr, err := vm.returnStack.Pop()
			if err != nil {
				return err
			}
			vm.programCounter = addr
			continue
		case CONST:
			vm.programCounter++
			w, err := vm.fetchWord(vm.programCounter)
			if err != nil {
				return err
			}

			if err := vm.dataStack.Push(w); err != nil {
				return err
			}

			vm.programCounter += WordSize
			continue
		case FETCH:
			addr, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			w, err := vm.fetchWord(addr)
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(w); err != nil {
				return err
			}
		case STORE:
			addr, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			value, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.storeWord(addr, value); err != nil {
				return err
			}
		case ADD:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			b, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(b + a); err != nil {
				return err
			}
		case SUBTRACT:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			b, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(b - a); err != nil {
				return err
			}
		case MULTIPLY:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			b, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(b * a); err != nil {
				return err
			}
		case DIVIDE:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			b, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(b / a); err != nil {
				return err
			}
		case MODULO:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			b, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(b % a); err != nil {
				return err
			}
		case DUP:
			a, err := vm.dataStack.Peek()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(a); err != nil {
				return err
			}
		case DROP:
			if _, err := vm.dataStack.Pop(); err != nil {
				return err
			}
		case SWAP:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			b, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(a); err != nil {
				return err
			}
			if err := vm.dataStack.Push(b); err != nil {
				return err
			}
		case OVER:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			b, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(b); err != nil {
				return err
			}
			if err := vm.dataStack.Push(a); err != nil {
				return err
			}
			if err := vm.dataStack.Push(b); err != nil {
				return err
			}
		case CJMP:
			vm.programCounter++
			conditional, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}

			if conditional == -1 {
				vm.programCounter, err = vm.fetchWord(vm.programCounter)
				if err != nil {
					return err
				}
			} else {
				vm.programCounter += WordSize
			}
			continue
		case CALL:
			vm.programCounter++
			if err := vm.returnStack.Push(vm.programCounter + WordSize); err != nil {
				return err
			}

			target, err := vm.fetchWord(vm.programCounter)
			if err != nil {
				return err
			}
			vm.programCounter = target
			continue
		case SCALL:
			vm.programCounter++
			if err := vm.returnStack.Push(vm.programCounter); err != nil {
				return err
			}

			target, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			vm.programCounter = target
			continue
		case KEY:
			b, err := vm.key()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(Word(b)); err != nil {
				return err
			}
		case EMIT:
			value, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.emit(byte(value)); err != nil {
				return err
			}
		case EQUALS:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			b, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(boolToWord(b == a)); err != nil {
				return err
			}
		case NOT:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(^a); err != nil {
				return err
			}
		case AND:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			b, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(b & a); err != nil {
				return err
			}
		case OR:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			b, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(b | a); err != nil {
				return err
			}
		case LT:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			b, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(boolToWord(b < a)); err != nil {
				return err
			}
		case GT:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			b, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(boolToWord(a < b)); err != nil {
				return err
			}
		case RPUT:
			a, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.returnStack.Push(a); err != nil {
				return err
			}
		case RPOP:
			a, err := vm.returnStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(a); err != nil {
				return err
			}
		case RPEEK:
			a, err := vm.returnStack.Peek()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(a); err != nil {
				return err
			}
		case BFETCH:
			addr, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			b, err := vm.fetchByte(addr)
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(Word(b)); err != nil {
				return err
			}
		case BSTORE:
			addr, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			value, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.storeByte(addr, byte(value)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown instruction '%d' at memory address '%d' - terminating",
				instruction, vm.programCounter)
		}

		vm.programCounter++
	}
}
