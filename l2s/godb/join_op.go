package godb

type EqualityJoin struct {
	// Expressions that when applied to tuples from the left or right operators,
	// respectively, return the value of the left or right side of the join
	leftField, rightField Expr

	left, right *Operator // Operators for the two inputs of the join

	// The maximum number of records of intermediate state that the join should
	// use (only required for optional exercise).
	maxBufferSize int
}

// Constructor for a join of integer expressions.
//
// Returns an error if either the left or right expression is not an integer.
func NewJoin(left Operator, leftField Expr, right Operator, rightField Expr, maxBufferSize int) (*EqualityJoin, error) {
	return &EqualityJoin{leftField, rightField, &left, &right, maxBufferSize}, nil
}

// Return a TupleDesc for this join. The returned descriptor should contain the
// union of the fields in the descriptors of the left and right operators.
//
// HINT: use [TupleDesc.merge].
func (hj *EqualityJoin) Descriptor() *TupleDesc {
	// TODO: some code goes here
	return (*hj.left).Descriptor().merge((*hj.right).Descriptor())

	// return nil
}

func (joinOp *EqualityJoin) loadOuterBatch(n int, iter func() (*Tuple, error)) (map[DBValue]([]*Tuple), bool, error) {
	hashmap := make(map[DBValue]([]*Tuple))
	for {
		if n == 0 {
			return hashmap, false, nil
		}
		t, err := iter()
		if err != nil {
			return nil, false, err
		}
		if t == nil { //finished iterating - 2nd bool indicates we have exhausted the iterator
			return hashmap, true, nil
		}

		v, err := joinOp.leftField.EvalExpr(t)
		if err != nil {
			return nil, false, err
		}

		hashmap[v] = append(hashmap[v], t)
		n--
	}
}

// Join operator implementation. This function should iterate over the results
// of the join. The join should be the result of joining joinOp.left and
// joinOp.right, applying the joinOp.leftField and joinOp.rightField expressions
// to the tuples of the left and right iterators respectively, and joining them
// using an equality predicate.
//
// HINT: When implementing the simple nested loop join, you should keep in mind
// that you only iterate through the left iterator once (outer loop) but iterate
// through the right iterator once for every tuple in the left iterator (inner
// loop).
//
// HINT: You can use [Tuple.joinTuples] to join two tuples.
//
// OPTIONAL EXERCISE: the operator implementation should not use more than
// maxBufferSize records, and should pass the testBigJoin test without timing
// out. To pass this test, you will need to use something other than a nested
// loops join.
func (joinOp *EqualityJoin) Iterator(tid TransactionID) (func() (*Tuple, error), error) {
	// TODO: some code goes here

	//build map on the left
	var hashmap map[DBValue]([]*Tuple)
	var rightIter func() (*Tuple, error)
	build_it, err := (*joinOp.left).Iterator(tid)
	if err != nil {
		return nil, err
	}
	var matches []*Tuple
	var curT *Tuple
	exhausted := false
	curMatch := 0
	needLoad := true

	return func() (*Tuple, error) {
		for {
			if needLoad && exhausted {
				return nil, nil
			}
			if needLoad {
				hashmap, exhausted, err = joinOp.loadOuterBatch(joinOp.maxBufferSize, build_it)
				if err != nil {
					return nil, err
				}
				rightIter, err = (*joinOp.right).Iterator(tid)
				if err != nil {
					return nil, err
				}
				needLoad = false
			}

			for {
				if curT == nil {
					var err error
					curT, err = rightIter()
					if err != nil {
						return nil, err
					}
					if curT == nil {
						needLoad = true
						break
					}
					v, err := joinOp.rightField.EvalExpr(curT)
					if err != nil {
						return nil, err
					}
					matches = hashmap[v]
					curMatch = 0
				}
				if matches != nil && curMatch < len(matches) {
					retT := joinTuples(matches[curMatch], curT)
					curMatch++
					return retT, nil
				} else {
					curT = nil
				}
			}
		}
	}, err
	// return nil, fmt.Errorf("join_op.Iterator not implemented")
}
