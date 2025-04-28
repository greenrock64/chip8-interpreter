package internal

type Stack struct {
	stack []uint16
}

func (s *Stack) Push(value uint16) {
	s.stack = append(s.stack, value)
}

func (s *Stack) Pop() uint16 {
	if len(s.stack) == 0 {
		return 0
	}
	value := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	return value
}
