package history

import (
	"slices"
	"testing"
)

func TestRingKeepsInsertionOrder(t *testing.T) {
	ring := NewRing(4)
	ring.Add(1)
	ring.Add(2)

	if got, want := ring.Values(), []float64{1, 2}; !slices.Equal(got, want) {
		t.Fatalf("Values() = %v, want %v", got, want)
	}
	if ring.Len() != 2 {
		t.Errorf("Len() = %d, want 2", ring.Len())
	}
}

func TestRingDropsOldestWhenFull(t *testing.T) {
	ring := NewRing(3)
	for _, value := range []float64{1, 2, 3, 4, 5} {
		ring.Add(value)
	}

	if got, want := ring.Values(), []float64{3, 4, 5}; !slices.Equal(got, want) {
		t.Fatalf("Values() = %v, want %v", got, want)
	}
	if ring.Len() != 3 {
		t.Errorf("Len() = %d, want 3", ring.Len())
	}
}

func TestRingValuesIsACopy(t *testing.T) {
	ring := NewRing(2)
	ring.Add(1)

	values := ring.Values()
	values[0] = 99

	if ring.Values()[0] != 1 {
		t.Fatal("mutating the returned slice must not affect the ring")
	}
}

func TestRingRejectsZeroSize(t *testing.T) {
	ring := NewRing(0)
	ring.Add(7)

	if got := ring.Values(); len(got) != 1 || got[0] != 7 {
		t.Fatalf("Values() = %v, want [7]", got)
	}
}
