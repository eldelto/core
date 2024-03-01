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

	s.cursor++
	s.data[s.cursor] = value
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

func (vm *VM) Execute() error {
	for {
		instruction := vm.memory[vm.programCounter]

		switch instruction {
		case EXIT:
			return nil
		case DROP:
			if _, err := vm.dataStack.Pop(); err != nil {
				return err
			}
		case RET:
			addr, err := vm.returnStack.Pop()
			if err != nil {
				return err
			}
			vm.programCounter = addr
			continue
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
