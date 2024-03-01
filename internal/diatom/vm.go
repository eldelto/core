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

func (i *Input) NextChar() (byte, error) {
	if i.cursor >= i.len {
		read, err := io.ReadAtLeast(os.Stdin, i.buffer[:], 1)
		if read == 0 || err != nil {
			return 0, fmt.Errorf("failed to read from stdin: %w", err)
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
	memory         [MemorySize]byte
}

func NewVM(program []byte) (*VM, error) {
	programLen := len(program)
	if programLen > MemorySize {
		return nil, fmt.Errorf("program length (%d bytes) exceeds available memory (%d bytes)",
			programLen, MemorySize)
	}

	vm := VM{}
	copy(vm.memory[:], program)

	return &vm, nil
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

//func (vm *VM) storeByte(addr Word, b byte) error {
//	if err := vm.validateMemoryAccess(addr); err != nil {
//		return err
//	}
//
//	vm.memory[addr] = b
//	return nil
//}

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
		case DROP:
			if _, err := vm.dataStack.Pop(); err != nil {
				return err
			}
		case KEY:
			b, err := vm.inputBuffer.NextChar()
			if err != nil {
				return err
			}
			if err := vm.dataStack.Push(Word(b)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown instruction '%d' at memory address '%d' - terminating",
				instruction, vm.programCounter)
		}

		vm.programCounter++
	}
}
