package tmux

import "testing"

func Test_splitTargetPaneIndex(t *testing.T) {
	type expected struct {
		index     []int
		direction string
	}

	expectation := []expected{
		{[]int{0}, "-h"},
		{[]int{0, 2}, "-v"},
		{[]int{0, 2, 4, 6}, "-h"},
		{[]int{0, 2, 4, 6, 8, 10, 12, 14}, "-v"},
	}

	numPane := 1
	for _, e := range expectation {
		for _, i := range e.index {
			dir, index := splitTargetPaneIndex(numPane)
			if dir != e.direction || index != i {
				t.Fatalf("wrong split target: want(%q) != got(%q) and/or want(%d) != got(%d)", e.direction, dir, i, index)
			}
			numPane++
		}
	}
}
