package godb

import (
	"github.com/tylertreat/BoomFilters"
)

/*
 Represents a fixed-width histogram over a single string field.
*/

type StringHistogram struct {
	cms *boom.CountMinSketch
}

// Create a new StringHistogram with a specified number of buckets.
//
// Our implementation is written in terms of an IntHistogram by converting each
// string to an integer.
func NewStringHistogram() (*StringHistogram, error) {
	cms := boom.NewCountMinSketch(0.001, 0.999)
	return &StringHistogram{cms}, nil
}

func (h *StringHistogram) AddValue(s string) {
	h.cms.Add([]byte(s))
}

func (h *StringHistogram) EstimateSelectivity(op BoolOp, s string) float64 {
	return float64(h.cms.Count([]byte(s))) / float64(h.cms.TotalCount())
}
