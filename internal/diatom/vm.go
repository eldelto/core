package diatom

//import (
//	"fmt"
//)
//
//const (
//	StackSize    = 30
//	IOBufferSize = 4096
//)
//
//type Word int32
//
//type Stack struct {
//	cursor int
//	data   [StackSize]Word
//}
//
//func (s *Stack) Push(value Word) error {
//	if s.cursor+1 >= len(s.data) {
//		return fmt.Errorf("push: stack overflow - cursor: %d, stack size: %d",
//			s.cursor, len(s.data))
//	}
//
//	s.cursor++
//	s.data[s.cursor] = value
//	return nil
//}
//
//func (s *Stack) Pop() (Word, error) {
//	if s.cursor <= 0 {
//		return 0, fmt.Errorf("pop: stack underflow - cursor: %d, stack size: %d",
//			s.cursor, len(s.data))
//	}
//
//	s.cursor--
//	return s.data[s.cursor], nil
//}
//
//func (s *Stack) Peek() (Word, error) {
//	if s.cursor <= 0 {
//		return 0, fmt.Errorf("peek: stack underflow - cursor: %d, stack size: %d",
//			s.cursor, len(s.data))
//	}
//
//	return s.data[s.cursor-1], nil
//}
//
//type Input struct {
//	cursor int
//	len    int
//	buffer [IOBufferSize]byte
//}

//func (i *Input) NextChar() (byte, error) {
//  read, err := io.ReadFull(os.Stdin, i.buffer[:])
//  return 0, err
//}
