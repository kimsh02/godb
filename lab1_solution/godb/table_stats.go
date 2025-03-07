package godb

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
}

// The default cost to read a page from disk. This value can be adjusted to
// accommodate different storage devices.
const CostPerPage = 1000

// Number of bins for histograms. Feel free to increase this value over 100,
// though our tests assume that you have at least 100 bins in your histograms.
const NumHistBins = 100

// Create a new TableStats object, that keeps track of statistics on each column of a table.
func ComputeTableStats(bp *BufferPool, dbFile DBFile) (*TableStats, error) {
	tid := NewTID()

	bp.BeginTransaction(tid)
	defer bp.CommitTransaction(tid)

	return &TableStats{}, nil
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
	return 0.0
}

// This method returns the number of tuples in the relation, given that a
// predicate with selectivity is applied.
func (t *TableStats) EstimateCardinality(selectivity float64) int {

	return 0
}

// Given a field name, boolean predicate, and a constant, look up the relevant
// histogram and estimate the selectivity of the filter.
func (t *TableStats) EstimateSelectivity(field string, op BoolOp, value DBValue) (float64, error) {

	return 0.0, nil
}
