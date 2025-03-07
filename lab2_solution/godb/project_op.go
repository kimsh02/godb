package godb

type Project struct {
	selectFields []Expr // required fields for parser
	outputNames  []string
	child        Operator
	//add additional fields here
	// TODO: some code goes here

	outDesc  TupleDesc
	distinct bool
}

// Construct a projection operator. It saves the list of selected field, child,
// and the child op. Here, selectFields is a list of expressions that represents
// the fields to be selected, outputNames are names by which the selected fields
// are named (should be same length as selectFields; throws error if not),
// distinct is for noting whether the projection reports only distinct results,
// and child is the child operator.
func NewProjectOp(selectFields []Expr, outputNames []string, distinct bool, child Operator) (Operator, error) {
	// TODO: some code goes here
	if len(selectFields) != len(outputNames) {
		return nil, GoDBError{IncompatibleTypesError, "output name list must have same number of fields as input list"}
	}
	outFields := make([]FieldType, len(selectFields))
	//copy(outFields, selectFields)
	for i, expr := range selectFields {
		outFields[i].Ftype = expr.GetExprType().Ftype
		outFields[i].Fname = outputNames[i]
	}
	return &Project{selectFields, outputNames, child, TupleDesc{outFields}, distinct}, nil

	// return nil, nil
}

// Return a TupleDescriptor for this projection. The returned descriptor should
// contain fields for each field in the constructor selectFields list with
// outputNames as specified in the constructor.
//
// HINT: you can use expr.GetExprType() to get the field type
func (p *Project) Descriptor() *TupleDesc {
	// TODO: some code goes here
	return &p.outDesc

	// return nil

}

// Project operator implementation. This function should iterate over the
// results of the child iterator, projecting out the fields from each tuple. In
// the case of distinct projection, duplicate tuples should be removed. To
// implement this you will need to record in some data structure with the
// distinct tuples seen so far. Note that support for the distinct keyword is
// optional as specified in the lab 2 assignment.
func (p *Project) Iterator(tid TransactionID) (func() (*Tuple, error), error) {
	// TODO: some code goes here
	childIter, err := p.child.Iterator(tid)
	if err != nil {
		return nil, err
	}
	distinctState := make(map[any]*Tuple)
	var distinctTups []*Tuple
	didDistinct := false
	curDistinct := 0
	return func() (*Tuple, error) {
		if (p.distinct && !didDistinct) || (!p.distinct) {
			for {
				tup, err := childIter()
				if err != nil {
					return nil, err
				}
				if tup == nil {
					if p.distinct {
						didDistinct = true
						break
					}
					return nil, nil
				}

				outVals := make([]DBValue, len(p.selectFields))
				for i := 0; i < len(p.selectFields); i++ {
					outVals[i], err = p.selectFields[i].EvalExpr(tup)
					if err != nil {
						return nil, err
					}
				}
				//outTup, err := tup.project(p.selectFields)
				outTup := Tuple{p.outDesc, outVals, tup.Rid}

				if p.distinct {
					key := outTup.tupleKey()
					distinctTup := (distinctState[key])
					if distinctTup == nil {
						distinctState[key] = &outTup
						distinctTups = append(distinctTups, &outTup)
					}
				} else { //not distinct
					return &outTup, nil
				}
			}
		}
		//distinct, iterating results
		if curDistinct >= len(distinctTups) {
			return nil, nil
		}

		tup := distinctTups[curDistinct]
		curDistinct++
		return tup, nil
	}, nil
	// return nil, fmt.Errorf("project_op.Iterator not implemented")
}
