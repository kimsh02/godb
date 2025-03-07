package godb

import "sort"

type OrderBy struct {
	orderBy []Expr // OrderBy should include these two fields (used by parser)
	child   Operator
	// TODO: some code goes here
	//add additional fields here
	ascending []bool
}

// Construct an order by operator. Saves the list of field, child, and ascending
// values for use in the Iterator() method. Here, orderByFields is a list of
// expressions that can be extracted from the child operator's tuples, and the
// ascending bitmap indicates whether the ith field in the orderByFields list
// should be in ascending (true) or descending (false) order.
func NewOrderBy(orderByFields []Expr, child Operator, ascending []bool) (*OrderBy, error) {
	// TODO: some code goes here
	return &OrderBy{orderByFields, child, ascending}, nil

	// return nil, nil //replace me

}

// Return the tuple descriptor.
//
// Note that the order by just changes the order of the child tuples, not the
// fields that are emitted.
func (o *OrderBy) Descriptor() *TupleDesc {
	// TODO: some code goes here
	return o.child.Descriptor()

	// return nil
}

type TupSortState struct {
	op       *OrderBy
	tupArray []*Tuple
}

func (ts TupSortState) Len() int {
	return len(ts.tupArray)
}

func (ts TupSortState) Swap(i, j int) {
	ts.tupArray[i], ts.tupArray[j] = ts.tupArray[j], ts.tupArray[i]
}
func (ts TupSortState) Less(i, j int) bool {
	tup1 := ts.tupArray[i]
	tup2 := ts.tupArray[j]
	for i, ft := range ts.op.orderBy {
		res, _ := tup1.compareField(tup2, ft)
		switch res {
		case OrderedLessThan:
			return ts.op.ascending[i]
		case OrderedGreaterThan:
			return !ts.op.ascending[i]
		}
		//if equal, move on to next field
	}
	return false
}

// Return a function that iterates through the results of the child iterator in
// ascending/descending order, as specified in the constructor.  This sort is
// "blocking" -- it should first construct an in-memory sorted list of results
// to return, and then iterate through them one by one on each subsequent
// invocation of the iterator function.
//
// Although you are free to implement your own sorting logic, you may wish to
// leverage the go sort package and the [sort.Sort] method for this purpose. To
// use this you will need to implement three methods: Len, Swap, and Less that
// the sort algorithm will invoke to produce a sorted list. See the first
// example, example of SortMultiKeys, and documentation at:
// https://pkg.go.dev/sort
func (o *OrderBy) Iterator(tid TransactionID) (func() (*Tuple, error), error) {
	// TODO: some code goes here
	var tups []*Tuple
	childIter, err := o.child.Iterator(tid)
	if err != nil {
		return nil, err
	}
	for {
		t, err := childIter()
		if err != nil {
			return nil, err
		}
		if t == nil {
			break
		}
		tups = append(tups, t)
	}
	tstate := TupSortState{o, tups}
	sort.Sort(tstate)
	curTup := 0
	return func() (*Tuple, error) {
		if curTup < len(tups) {
			t := tups[curTup]
			curTup++
			return t, nil
		} else {
			return nil, nil
		}
	}, nil
	// return nil, fmt.Errorf("order_by_op.Iterator not implemented") //replace me
}
