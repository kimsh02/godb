package godb

// The result of a page lock request
type LockResponse int

const (
	Grant LockResponse = iota
	Wait  LockResponse = iota
	Abort LockResponse = iota
)

// PageLocks represents the locks held on a page.
//
// A page can have multiple read locks, but at most one write lock.
// TODO: some code goes here: type PageLocks struct

// LockTable is a table that keeps track of the locks held on each page, the
// pages that each transaction has locks on, and the wait-for graph.
type LockTable struct {
	// TODO: some code goes here
}

// Create a new LockTable.
func NewLockTable() *LockTable {
	return nil // replace it
	// TODO: some code goes here:
}

// Release all locks held by the transaction. This is called when a transaction
// is aborted or committed.
func (t *LockTable) ReleaseLocks(tid TransactionID) {
	// TODO: some code goes here
	// TODO implement me
}

// Return the page key for each page that the transaction has taken a write lock on.
//
// These are the pages that need to be written to disk when the transaction
// commits or dropped from the buffer pool when the transaction aborts.
func (t *LockTable) WriteLockedPages(tid TransactionID) []any {
	// TODO: some code goes here
	return nil // replace it, implement me
}

func (bp *LockTable) addTidPage(tid TransactionID, hashCode any) {
	// TODO: some code goes here
	// TODO implement me
}

// Try to lock a page with the given permissions. If the lock is granted, return
// true. If the lock is not granted, return false and potentially return a
// transaction to abort in order to break a deadlock.
//
// If the lock is granted, return Grant. If the lock is not granted, return
// either Wait or Abort. Upon receiving Wait, the caller should wait and then
// try again. Upon receiving Abort, the caller should abort the transaction by
// calling AbortTransaction.
func (t *LockTable) TryLock(file DBFile, pageNo int, tid TransactionID, perm RWPerm) LockResponse {
	// TODO: some code goes here

	return Wait // replace it, implement me
}
