package slices

import (
	"testing"
)

func TestIndex(t *testing.T) {
	tests := []struct {
		s    []int
		v    int
		want int
	}{
		{[]int{1, 1, 2, 3, 5}, 0, -1},
		{[]int{1, 1, 2, 3, 5}, 1, 0},
		{[]int{1, 1, 2, 3, 5}, 2, 2},
		{[]int{1, 1, 2, 3, 5}, 3, 3},
		{[]int{1, 1, 2, 3, 5}, 4, -1},
		{[]int{1, 1, 2, 3, 5}, 5, 4},
		{[]int{}, 123, -1},
	}

	for _, tt := range tests {
		actual := Index(tt.s, tt.v)
		if actual != tt.want {
			t.Errorf("Index(%#v, %#v) = %#v, want %#v",
				tt.s, tt.s, actual, tt.want)
		}
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s    []int
		v    int
		want bool
	}{
		{[]int{1, 1, 2, 3, 5}, 0, false},
		{[]int{1, 1, 2, 3, 5}, 1, true},
		{[]int{1, 1, 2, 3, 5}, 2, true},
		{[]int{1, 1, 2, 3, 5}, 3, true},
		{[]int{1, 1, 2, 3, 5}, 4, false},
		{[]int{1, 1, 2, 3, 5}, 5, true},
		{[]int{}, 123, false},
	}

	for _, tt := range tests {
		actual := Contains(tt.s, tt.v)
		if actual != tt.want {
			t.Errorf("Contains(%#v, %#v) = %#v, want %#v",
				tt.s, tt.s, actual, tt.want)
		}
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		a, b []int
		want bool
	}{
		{[]int{1, 2, 3}, []int{1, 2, 3}, true},
		{[]int{1, 2, 3}, []int{1, 2, 4}, false},
		{[]int{1, 2, 3, 4}, []int{1, 2, 3}, false},
		{[]int{}, []int{1, 2, 3}, false},
		{nil, []int{1, 2, 3}, false},
		{nil, nil, true},
		{nil, []int{}, true},
		{[]int{}, []int{}, true},
	}

	for _, tt := range tests {
		actual := Equal(tt.a, tt.b)
		if actual != tt.want {
			t.Errorf("Equal(%#v, %#v) = %#v, want %#v",
				tt.a, tt.b, actual, tt.want)
		}

		// Equality is symetric.
		actual = Equal(tt.b, tt.a)
		if actual != tt.want {
			t.Errorf("Equal(%#v, %#v) = %#v, want %#v",
				tt.b, tt.a, actual, tt.want)
		}
	}
}
