package godb

import (
	//<silentstrip lab1|lab2|lab3>
	"fmt"
	"log"
	"math"
	//</silentstrip>
)

/*
 TableStats represents statistics (e.g., histograms) about base tables in a
 query.

 Not used for labs 1-3.
*/

// Interface for statistics that are maintained for a table.
type Stats interface {
	EstimateScanCost() float64
	EstimateCardinality(selectivity float64) int
	EstimateSelectivity(field string, op BoolOp, value DBValue) (float64, error)
}

type TableStats struct {
	//<strip lab3>
	basePages  int
	baseTups   int
	histograms map[string]any
	tupleDesc  *TupleDesc
	//</strip>
}

// The default cost to read a page from disk. This value can be adjusted to
// accommodate different storage devices.
const CostPerPage = 1000

// Number of bins for histograms. Feel free to increase this value over 100,
// though our tests assume that you have at least 100 bins in your histograms.
const NumHistBins = 100

// <silentstrip lab3>
func tableMinMax(tid TransactionID, dbFile DBFile) ([]int64, []int64, error) {
	td := dbFile.Descriptor()
	mins := make([]int64, len(td.Fields))
	maxs := make([]int64, len(td.Fields))
	for i := range mins {
		mins[i] = math.MaxInt32
		maxs[i] = math.MinInt32
	}

	iter, err := dbFile.Iterator(tid)
	if err != nil {
		return nil, nil, err
	}
	for tup, err := iter(); tup != nil; tup, err = iter() {
		if err != nil {
			return nil, nil, err
		}

		for i, f := range td.Fields {
			if f.Ftype == IntType {
				v := tup.Fields[i].(IntField).Value
				mins[i] = min(mins[i], v)
				maxs[i] = max(maxs[i], v)
			}
		}
	}
	for i := range mins {
		if mins[i] > maxs[i] {
			mins[i] = 0
			maxs[i] = 0
		}
	}
	return mins, maxs, nil
}

// </silentstrip>
// Create a new TableStats object, that keeps track of statistics on each column of a table.
func ComputeTableStats(bp *BufferPool, dbFile DBFile) (*TableStats, error) {
	tid := NewTID()

	bp.BeginTransaction(tid)
	defer bp.CommitTransaction(tid)

	//<strip lab3>
	td := dbFile.Descriptor()

	// Compute min/max for table fields
	mins, maxs, err := tableMinMax(tid, dbFile)
	if err != nil {
		return nil, err
	}

	// Create histograms using field min/max
	hists := make(map[string]any, len(td.Fields))
	for i, f := range td.Fields {
		switch f.Ftype {
		case IntType:
			h, err := NewIntHistogram(NumHistBins, mins[i], maxs[i])
			if err != nil {
				return nil, err
			}
			hists[f.Fname] = h
		case StringType:
			h, err := NewStringHistogram()
			if err != nil {
				return nil, err
			}
			hists[f.Fname] = h
		case UnknownType:
			return nil, fmt.Errorf("unexpected unknown type")
		}
	}

	iter, err := dbFile.Iterator(tid)
	if err != nil {
		return nil, err
	}

	baseTups := 0
	for tup, err := iter(); tup != nil; tup, err = iter() {
		if err != nil {
			return nil, err
		}

		for i, f := range td.Fields {
			switch f.Ftype {
			case IntType:
				v := tup.Fields[i].(IntField).Value
				hists[f.Fname].(*IntHistogram).AddValue(v)
			case StringType:
				v := tup.Fields[i].(StringField).Value
				hists[f.Fname].(*StringHistogram).AddValue(v)
			case UnknownType:
				return nil, fmt.Errorf("unexpected unknown type")
			}
		}
		baseTups++
	}

	return &TableStats{dbFile.NumPages(), baseTups, hists, td}, nil
	//</strip>
}

// Estimates the cost of sequentially scanning the file, given that the cost to
// read a page is costPerPageIO. You can assume that there are no seeks and that
// no pages are in the buffer pool.
//
// Also, assume that your hard drive can only read entire pages at once, so if
// the last page of the table only has one tuple on it, it's just as expensive
// to read as a full page. (Most real hard drives can't efficiently address
// regions smaller than a page at a time.)
func (t *TableStats) EstimateScanCost() float64 {
	//<strip lab3>
	return float64(t.basePages * CostPerPage)
	//</strip>
}

// This method returns the number of tuples in the relation, given that a
// predicate with selectivity is applied.
func (t *TableStats) EstimateCardinality(selectivity float64) int {
	//<strip lab3>
	return int(float64(t.baseTups) * selectivity)
	//</strip>
}

// Given a field name, boolean predicate, and a constant, look up the relevant
// histogram and estimate the selectivity of the filter.
func (t *TableStats) EstimateSelectivity(field string, op BoolOp, value DBValue) (float64, error) {
	//<strip lab3>
	hist, ok := t.histograms[field]
	if !ok {
		log.Printf("WARNING: no histogram found for field %s", field)
		return 1.0, nil
	}

	switch h := hist.(type) {
	case *IntHistogram:
		value, ok := value.(IntField)
		if !ok {
			return 1.0, fmt.Errorf("field '%s' is int, but value %v is not an IntField", field, value)
		}
		return h.EstimateSelectivity(op, value.Value), nil

	case *StringHistogram:
		value, ok := value.(StringField)
		if !ok {
			return 1.0, fmt.Errorf("field is string, but value is not a StringField")
		}
		return h.EstimateSelectivity(op, value.Value), nil
	}

	return 1.0, fmt.Errorf("unexpected histogram type")
	//</strip>
}
