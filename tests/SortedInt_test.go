package tests

import (
	"DSM-project/Benchmarks"
	"testing"
)

func TestSortedInt(t *testing.T) {
	Benchmarks.SortedIntBenchmark(1, true, 10, 1000, 10)
}
