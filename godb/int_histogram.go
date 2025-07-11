package godb

import (
	"fmt"
)

type IntHistogram struct {
	//<strip lab3>
	bins     []int64
	min      int64
	max      int64
	n        int64
	binWidth int64
	//</strip>
}

// NewIntHistogram creates a new IntHistogram with the specified number of bins.
//
// Min and max specify the range of values that the histogram will cover
// (inclusive).
func NewIntHistogram(nBins int64, vMin int64, vMax int64) (*IntHistogram, error) {
	//<strip lab3>
	if nBins <= 0 {
		return nil, fmt.Errorf("nBins must be positive")
	}
	if vMin > vMax {
		return nil, fmt.Errorf("min must be less than or equal to max")
	}
	bins := make([]int64, nBins)
	binWidth := max(1, (vMax-vMin+1)/nBins)
	return &IntHistogram{bins, vMin, vMax, 0, binWidth}, nil
	//</strip>
}

// <silentstrip lab3>
func (h *IntHistogram) bin(v int64) int64 {
	bin := (v - h.min) / h.binWidth
	return max(0, min(bin, int64(len(h.bins)-1)))
}

//</silentstrip>
// Add a value v to the histogram.
func (h *IntHistogram) AddValue(v int64) {
	//<strip lab3>
	h.bins[h.bin(v)]++
	h.n++
	//</strip>
}

// Estimate the selectivity of a predicate and operand on the values represented
// by this histogram.
//
// For example, if op is OpLt and v is 10, return the fraction of values that
// are less than 10.
func (h *IntHistogram) EstimateSelectivity(op BoolOp, v int64) float64 {
	//<strip lab3>
	// compute an interval [sumMin, sumMax] using op and v
	// we will estimate the number of values in this interval using the
	// histogram and divide by n to get the selectivity
	sumMin := h.min
	sumMax := h.max
	switch op {
	case OpGt:
		sumMin = v + 1
	case OpLt:
		sumMax = v - 1
	case OpGe:
		sumMin = v
	case OpLe:
		sumMax = v
	case OpEq:
		fallthrough
	case OpLike:
		sumMin = v
		sumMax = v
	case OpNeq:
		return 1 - h.EstimateSelectivity(OpEq, v)
	}

	// If the interval is empty, return 0.0
	if sumMin > sumMax {
		return 0.0
	}
	// If the interval includes all values, return 1.0
	if sumMin <= h.min && sumMax >= h.max {
		return 1.0
	}

	loBin := h.bin(sumMin)
	hiBin := h.bin(sumMax)

	total := 0.0
	for i := loBin; i <= hiBin; i++ {
		// The bin contains values in the interval [binMin, binMax]
		binL := h.min + h.binWidth*i
		binH := h.min + h.binWidth*(i+1) - 1

		// [lo, hi] is the intersection
		lo := max(sumMin, binL)
		hi := min(sumMax, binH)

		// If the intersection is non-empty, add the fraction of the bin
		if lo <= hi {
			total += float64(h.bins[i]) * (float64(hi-lo+1) / float64(h.binWidth))
		}
	}

	return total / float64(h.n)
	//</strip>
}
