package godb

import "fmt"

type InsertOp struct {
	// TODO: some code goes here

	child      Operator
	insertFile DBFile
}

// Construct an insert operator that inserts the records in the child Operator
// into the specified DBFile.
func NewInsertOp(insertFile DBFile, child Operator) *InsertOp {
	// TODO: some code goes here
	return &InsertOp{child, insertFile}

	// return nil
}

// The insert TupleDesc is a one column descriptor with an integer field named "count"
func (i *InsertOp) Descriptor() *TupleDesc {
	// TODO: some code goes here
	return &TupleDesc{[]FieldType{{"count", "", IntType}}}

	// return nil
}

// Return an iterator function that inserts all of the tuples from the child
// iterator into the DBFile passed to the constuctor and then returns a
// one-field tuple with a "count" field indicating the number of tuples that
// were inserted.  Tuples should be inserted using the [DBFile.insertTuple]
// method.
func (iop *InsertOp) Iterator(tid TransactionID) (func() (*Tuple, error), error) {
	// TODO: some code goes here
	iter, err := iop.child.Iterator(tid)
	if err != nil {
		return nil, err
	}
	didIterate := false
	return func() (*Tuple, error) {
		if didIterate {
			return nil, nil
		}
		cnt := 0
		td := iop.insertFile.Descriptor()
		for {
			t, err := iter()
			if err != nil {
				return nil, err
			}
			if t == nil {
				break
			}
			if len(td.Fields) != len(t.Fields) {
				return nil, GoDBError{TypeMismatchError, "inserted tuple doesn't have same number of fields as table."}
			}
			for i, f := range t.Desc.Fields {
				if f.Ftype != td.Fields[i].Ftype {
					return nil, GoDBError{TypeMismatchError, fmt.Sprintf("expected type %s in %dth inserted field, got %s", td.Fields[i].Ftype.String(), i, f.Ftype.String())}
				}
			}
			err = iop.insertFile.insertTuple(t, tid)
			if err != nil {
				return nil, err
			}
			cnt = cnt + 1
		}
		didIterate = true
		return &Tuple{*iop.Descriptor(), []DBValue{IntField{int64(cnt)}}, nil}, nil
	}, nil
	// return nil, fmt.Errorf("insert_op.Iterator not implemented")
}
