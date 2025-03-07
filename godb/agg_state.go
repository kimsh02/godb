package godb

// interface for an aggregation state
type AggState interface {
	// Initializes an aggregation state. Is supplied with an alias, an expr to
	// evaluate an input tuple into a DBValue, and a getter to extract from the
	// DBValue its int or string field's value.
	Init(alias string, expr Expr) error

	// Makes an copy of the aggregation state.
	Copy() AggState

	// Adds an tuple to the aggregation state.
	AddTuple(*Tuple)

	// Returns the final result of the aggregation as a tuple.
	Finalize() *Tuple

	// Gets the tuple description of the tuple that Finalize() returns.
	GetTupleDesc() *TupleDesc
}

// Implements the aggregation state for COUNT
type CountAggState struct {
	alias string
	expr  Expr
	count int
}

func (a *CountAggState) Copy() AggState {
	return &CountAggState{a.alias, a.expr, a.count}
}

func (a *CountAggState) Init(alias string, expr Expr) error {
	a.count = 0
	a.expr = expr
	a.alias = alias
	return nil
}

func (a *CountAggState) AddTuple(t *Tuple) {
	a.count++
}

func (a *CountAggState) Finalize() *Tuple {
	td := a.GetTupleDesc()
	f := IntField{int64(a.count)}
	fs := []DBValue{f}
	t := Tuple{*td, fs, nil}
	return &t
}

func (a *CountAggState) GetTupleDesc() *TupleDesc {
	ft := FieldType{a.alias, "", IntType}
	fts := []FieldType{ft}
	td := TupleDesc{}
	td.Fields = fts
	return &td
}

// Implements the aggregation state for SUM
type SumAggState struct {
	alias string
	expr  Expr
	sum   int64
}

func (a *SumAggState) Copy() AggState {
	return &SumAggState{a.alias, a.expr, a.sum}
}

func intAggGetter(v DBValue) any {
	intV := v.(IntField)
	return intV.Value
}

func stringAggGetter(v DBValue) any {
	stringV := v.(StringField)
	return stringV.Value
}

func (a *SumAggState) Init(alias string, expr Expr) error {
	a.sum = 0
	a.expr = expr
	a.alias = alias
	return nil
}

func (a *SumAggState) AddTuple(t *Tuple) {
	v, err := a.expr.EvalExpr(t)
	if err != nil {
		return
	}
	switch v.(type) {
	case IntField:
		a.sum += v.(IntField).Value
	}
}

func (a *SumAggState) GetTupleDesc() *TupleDesc {
	return &TupleDesc{[]FieldType{{a.alias, "", IntType}}}
}

func (a *SumAggState) Finalize() *Tuple {
	return &Tuple{*a.GetTupleDesc(), []DBValue{IntField{a.sum}}, nil}
}

// Implements the aggregation state for AVG
// Note that we always AddTuple() at least once before Finalize()
// so no worries for divide-by-zero
type AvgAggState struct {
	alias string
	expr  Expr
	sum   int64
	count int64
}

func (a *AvgAggState) Copy() AggState {
	return &AvgAggState{a.alias, a.expr, a.sum, a.count}
}

func (a *AvgAggState) Init(alias string, expr Expr) error {
	a.sum = 0
	a.count = 0
	a.expr = expr
	a.alias = alias
	return nil
}

func (a *AvgAggState) AddTuple(t *Tuple) {
	v, err := a.expr.EvalExpr(t)
	if err != nil {
		return
	}
	switch v.(type) {
	case IntField:
		a.sum += v.(IntField).Value
	}
	a.count++
}

func (a *AvgAggState) GetTupleDesc() *TupleDesc {
	return &TupleDesc{[]FieldType{{a.alias, "", IntType}}}
}

func (a *AvgAggState) Finalize() *Tuple {
	return &Tuple{*a.GetTupleDesc(), []DBValue{IntField{a.sum / a.count}}, nil}
}

// Implements the aggregation state for MAX
// Note that we always AddTuple() at least once before Finalize()
// so no worries for NaN max
type MaxAggState struct {
	alias string
	expr  Expr
	val   DBValue
	null  bool // whether the agg state have not seen any tuple inputted yet
}

func (a *MaxAggState) Copy() AggState {
	// Minor point: I see that you set to a.null true rather than a.null because later you used
	// copy() as a way to create new aggstate, but semantically, shouldn't the copy() function
	// copy the exact state of a?
	return &MaxAggState{a.alias, a.expr, a.val, true}
}

func (a *MaxAggState) Init(alias string, expr Expr) error {
	a.expr = expr
	a.alias = alias
	return nil
}

func (a *MaxAggState) AddTuple(t *Tuple) {
	v, err := a.expr.EvalExpr(t)
	if err != nil {
		return
	}

	if a.null {
		a.val = v
		a.null = false
	} else if a.val.EvalPred(v, OpLt) {
		a.val = v
	}
}

func (a *MaxAggState) GetTupleDesc() *TupleDesc {
	return &TupleDesc{[]FieldType{{a.alias, "", a.expr.GetExprType().Ftype}}}
}

func (a *MaxAggState) Finalize() *Tuple {
	return &Tuple{*a.GetTupleDesc(), []DBValue{a.val}, nil}
}

// Implements the aggregation state for MIN
// Note that we always AddTuple() at least once before Finalize()
// so no worries for NaN min
type MinAggState struct {
	MaxAggState
}

func (a *MinAggState) Copy() AggState {
	// Minor point: I see that you set to a.null true rather than a.null because later you used
	// copy() as a way to create new aggstate, but semantically, shouldn't the copy() function
	// copy the exact state of a?
	return &MinAggState{MaxAggState{a.alias, a.expr, a.val, true}}
}

func (a *MinAggState) Init(alias string, expr Expr) error {
	a.expr = expr
	a.alias = alias
	return nil
}

func (a *MinAggState) AddTuple(t *Tuple) {
	v, err := a.expr.EvalExpr(t)
	if err != nil {
		return
	}
	if a.null {
		a.val = v
		a.null = false
	} else if a.val.EvalPred(v, OpGt) {
		a.val = v
	}
}

func (a *MinAggState) GetTupleDesc() *TupleDesc {
	return &TupleDesc{[]FieldType{{a.alias, "", a.expr.GetExprType().Ftype}}}
}

func (a *MinAggState) Finalize() *Tuple {
	return &Tuple{*a.GetTupleDesc(), []DBValue{a.val}, nil}
}
