package godb

import "fmt"

type DeleteOp struct {
}

// Construct a delete operator. The delete operator deletes the records in the
// child Operator from the specified DBFile.
func NewDeleteOp(deleteFile DBFile, child Operator) *DeleteOp {
	return nil
}

// The delete TupleDesc is a one column descriptor with an integer field named
// "count".
func (i *DeleteOp) Descriptor() *TupleDesc {
	return nil

}

// Return an iterator that deletes all of the tuples from the child iterator
// from the DBFile passed to the constructor and then returns a one-field tuple
// with a "count" field indicating the number of tuples that were deleted.
// Tuples should be deleted using the [DBFile.deleteTuple] method.
func (dop *DeleteOp) Iterator(tid TransactionID) (func() (*Tuple, error), error) {
	return nil, fmt.Errorf("delete_op.Iterator not implemented")
}
