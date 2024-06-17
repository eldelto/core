package diatom

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/eldelto/core/internal/collections"
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

func (s *Stack) String() string {
	b := strings.Builder{}

	for _, w := range s.data[:s.cursor] {
		b.WriteString(strconv.Itoa(int(w)))
		b.WriteString(", ")
	}

	return b.String()
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

type traceEntry struct {
	programCounter Word
	instruction    byte
	dataStack      Stack
	returnStack    Stack
}

type ExtensionFunc func(vm *VM)

type Extension struct {
	Addr      uint16
	Functions []ExtensionFunc
	Name      string
}

type VM struct {
	programCounter Word
	dataStack      Stack
	returnStack    Stack
	inputBuffer    Input
	input          io.Reader
	output         io.Writer
	memory         [MemorySize]byte
	executionTrace collections.RingBuffer[traceEntry]
	extensions     map[Word]ExtensionFunc
}

func NewVM(program []byte, input io.Reader, output io.Writer) (*VM, error) {
	programLen := len(program)
	if programLen > MemorySize {
		return nil, fmt.Errorf("program length (%d bytes) exceeds available memory (%d bytes)",
			programLen, MemorySize)
	}

	vm := VM{
		input:          input,
		output:         output,
		executionTrace: collections.NewRingBuffer[traceEntry](StackSize),
		extensions:     map[Word]ExtensionFunc{},
	}
	copy(vm.memory[:], program)

	return &vm, nil
}

func NewDefaultVM(program []byte) (*VM, error) {
	return NewVM(program, os.Stdin, os.Stdout)
}

func (vm *VM) RegisterExtension(ext Extension) error {
	moduleAddr := Word(ext.Addr) << 16
	for funcAddr, extFunc := range ext.Functions {
		addr := moduleAddr | Word(funcAddr)
		if _, ok := vm.extensions[addr]; ok {
			return fmt.Errorf("extension %q: function already exists at address '%d'",
				ext.Name, addr)
		}

		vm.extensions[addr] = extFunc
	}

	return nil
}

func (vm *VM) appendTraceEntry(instruction byte) {
	entry := traceEntry{
		programCounter: vm.programCounter,
		instruction:    instruction,
		dataStack:      vm.dataStack,
		returnStack:    vm.returnStack,
	}

	vm.executionTrace.Append(entry)
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
	bytes := wordToBytes(w)
	for i := 0; i < WordSize; i++ {
		if err := vm.storeByte(addr+(WordSize-(Word(i)+1)), bytes[i]); err != nil {
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

func (vm *VM) dumpMemory(fileName string, endAddr Word) error {
	if err := os.WriteFile(fileName, vm.memory[:endAddr], 0644); err != nil {
		return fmt.Errorf("failed to dump memory addresses 0 - %d to file %q",
			endAddr, fileName)
	}

	return nil
}

func boolToWord(b bool) Word {
	if b {
		return -1
	}

	return 0
}

func add(a, b Word) Word {
	la, lb := int64(a), int64(b)
	lc := la + lb

	if lc > WordMax {
		return WordMax
	} else if lc < WordMin {
		return WordMin
	} else {
		return Word(lc)
	}
}

func subtract(a, b Word) Word {
	la, lb := int64(a), int64(b)
	lc := la - lb

	if lc > WordMax {
		return WordMax
	} else if lc < WordMin {
		return WordMin
	} else {
		return Word(lc)
	}
}

func multiply(a, b Word) Word {
	la, lb := int64(a), int64(b)
	lc := la * lb

	if lc > WordMax {
		return WordMax
	} else if lc < WordMin {
		return WordMin
	} else {
		return Word(lc)
	}
}

func (vm *VM) execute() error {
	for {
		instruction := vm.memory[vm.programCounter]
		vm.appendTraceEntry(instruction)

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
			if err := vm.dataStack.Push(add(b, a)); err != nil {
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
			if err := vm.dataStack.Push(subtract(b, a)); err != nil {
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
			if err := vm.dataStack.Push(multiply(b, a)); err != nil {
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
				if errors.Is(err, io.EOF) {
					return nil
				}
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
		case ECALL:
			addr, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			extFunc, ok := vm.extensions[addr]
			if !ok {
				return fmt.Errorf("extension function at address '%d' not found",
					addr)
			}
			extFunc(vm)
		case DUMP:
			endAddr, err := vm.dataStack.Pop()
			if err != nil {
				return err
			}
			if err := vm.dumpMemory("dump.dopc", endAddr); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown instruction '%d' at memory address '%d' - terminating",
				instruction, vm.programCounter)
		}

		vm.programCounter++
	}
}

func (vm *VM) StackTrace() string {
	b := strings.Builder{}

	trace := vm.executionTrace.Slice()
	slices.Reverse(trace)

	for _, entry := range trace {
		b.WriteString("Program counter: ")
		b.WriteString(strconv.FormatInt(int64(entry.programCounter), 10))

		b.WriteString("  Instruction: ")
		b.WriteString(instructionFromOpcode(entry.instruction))
		b.WriteRune('\n')

		b.WriteString("Data stack: ")
		b.WriteString(entry.dataStack.String())
		b.WriteRune('\n')

		b.WriteString("Return stack: ")
		b.WriteString(entry.returnStack.String())
		b.WriteRune('\n')

		b.WriteRune('\n')
	}

	return b.String()
}

func (vm *VM) Execute() error {
	if err := vm.execute(); err != nil {
		return fmt.Errorf("%w:\n\n%s", err, vm.StackTrace())
	}

	return nil
}
