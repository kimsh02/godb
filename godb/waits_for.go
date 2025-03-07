package godb

// WaitFor is a wait-for graph. It maps waiting transactions to the transactions
// that they are waiting for.
type WaitFor map[TransactionID][]TransactionID

// Extend the graph so that [tid] waits for each of [tids].
func (w WaitFor) AddEdges(tid TransactionID, tids []TransactionID) {
	// TODO: some code goes here
}

// Remove the transaction [tid] from the graph. After this method runs, the
// graph will not contain any references to [tid].
func (w WaitFor) RemoveTransaction(tid TransactionID) {
	// TODO: some code goes here
}

// Returns true if [start] is part of a cycle and false otherwise.
func (w WaitFor) DetectDeadlock(start TransactionID) bool {
	return false // TODO implement me
}
