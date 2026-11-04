package database

// Same-name spells with no rank subtext take their ranks from the trainer rows' supersede chain, and
// anything that is not one chain through all of them stays ambiguous.

import (
	"reflect"
	"testing"
)

func TestSupersedeOrderFollowsTheChain(t *testing.T) {
	// Shank: 1329549 supersedes 1329283.
	got := supersedeOrder([]int32{1329549, 1329283}, map[int32]int32{1329549: 1329283})
	if want := []int32{1329283, 1329549}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Shank ranks = %v, want %v", got, want)
	}
	got = supersedeOrder([]int32{30, 10, 20}, map[int32]int32{20: 10, 30: 20})
	if want := []int32{10, 20, 30}; !reflect.DeepEqual(got, want) {
		t.Fatalf("three ranks = %v, want %v", got, want)
	}
}

func TestSupersedeOrderLeavesNoChainAmbiguous(t *testing.T) {
	for name, c := range map[string]struct {
		ids        []int32
		supersedes map[int32]int32
	}{
		"no chain":       {[]int32{1, 2}, map[int32]int32{}},
		"two from one":   {[]int32{1, 2, 3}, map[int32]int32{2: 1, 3: 1}},
		"broken chain":   {[]int32{1, 2, 3}, map[int32]int32{2: 1, 3: 99}},
		"each the other": {[]int32{1, 2}, map[int32]int32{1: 2, 2: 1}},
	} {
		if got := supersedeOrder(c.ids, c.supersedes); got != nil {
			t.Errorf("%s: got %v, want no order", name, got)
		}
	}
}
