package godb

import (
	"fmt"
)

type IntHistogram struct {
}

// NewIntHistogram creates a new IntHistogram with the specified number of bins.
//
// Min and max specify the range of values that the histogram will cover
// (inclusive).
func NewIntHistogram(nBins int64, vMin int64, vMax int64) (*IntHistogram, error) {
	return nil, fmt.Errorf("NewIntHistogram unimplemented")
}

// Add a value v to the histogram.
func (h *IntHistogram) AddValue(v int64) {
}

// Estimate the selectivity of a predicate and operand on the values represented
// by this histogram.
//
// For example, if op is OpLt and v is 10, return the fraction of values that
// are less than 10.
func (h *IntHistogram) EstimateSelectivity(op BoolOp, v int64) float64 {
	return 0.0
}
