package godb

//BufferPool provides methods to cache pages that have been read from disk.
//It has a fixed capacity to limit the total amount of memory used by GoDB.
//It is also the primary way in which transactions are enforced, by using page
//level locking (you will not need to worry about this until lab3).

import (
	"errors"
	"sync"
	"time"
)

// Permissions used to when reading / locking pages
type RWPerm int

const (
	ReadPerm  RWPerm = iota
	WritePerm RWPerm = iota
)

type BufferPool struct {
	pages    map[any]Page
	maxPages int
	logFile  *LogFile

	// the transactions that are currently running. This is a set, so the value
	// is not important

	// TODO: some code goes here
	mu DebugMutex
	// mu      sync.Mutex
	locks   map[any]*PageLock
	running map[TransactionID]struct{}
	wfg     map[TransactionID]map[TransactionID]struct{}
}

type DebugMutex struct {
	mu sync.Mutex
}

func (dm *DebugMutex) Lock(tid TransactionID, from string) {
	// log.Printf("Locking, tid %d\n", tid)
	dm.mu.Lock()
	// log.Printf("Locked from %s,\ttid %d\n", from, tid)
}

func (dm *DebugMutex) Unlock(tid TransactionID, from string) {
	// log.Printf("Unlocked from %s,\ttid %d\n", from, tid)
	dm.mu.Unlock()
	// log.Println("Unlocked, tid %d\n", tid)
}

type PageLock struct {
	shared    map[TransactionID]struct{}
	exclusive *TransactionID
	// waiting   map[TransactionID]struct{}
}

// Create a new BufferPool with the specified number of pages
func NewBufferPool(numPages int) (*BufferPool, error) {
	// return &BufferPool{}, fmt.Errorf("NewBufferPool not implemented") //replace it
	// TODO: some code goes here
	return &BufferPool{make(map[any]Page), numPages, nil, DebugMutex{sync.Mutex{}},
		make(map[any]*PageLock),
		make(map[TransactionID]struct{}),
		make(map[TransactionID]map[TransactionID]struct{})}, nil
}

// Testing method -- iterate through all pages in the buffer pool and flush them
// using [DBFile.flushPage]. Does not need to be thread/transaction safe
func (bp *BufferPool) FlushAllPages() {
	for _, page := range bp.pages {
		page.getFile().flushPage(page)
	}
}

// Testing method -- flush all dirty pages in the buffer pool and set them to
// clean. Does not need to be thread/transaction safe.
// TODO: some code goes here : func (bp *BufferPool) flushDirtyPages(tid TransactionID) error

// Returns true if the transaction is runing.
//
// Caller must hold the bufferpool lock.
// TODO: some code goes here : func (bp *BufferPool) tidIsRunning(tid TransactionID) bool

// Abort the transaction, releasing locks. Because GoDB is FORCE/NO STEAL, none
// of the pages tid has dirtied will be on disk so it is sufficient to just
// release locks to abort. You do not need to implement this for lab 1.
// TODO: some code goes here : func (bp *BufferPool) AbortTransaction(tid TransactionID)
func (bp *BufferPool) AbortTransaction(tid TransactionID) error {
	bp.mu.Lock(tid, "AbortTransaction")
	// fmt.Printf("lock from abortTransaction, tid: %d\n", tid)
	if _, ok := bp.running[tid]; !ok {
		bp.mu.Unlock(tid, "AbortTransaction")
		return errors.New("txn is not running")
	}
	// fmt.Println("aborting")
	for k := range bp.pages {
		if bp.locks[k] == nil {
			continue
		}
		if bp.locks[k].exclusive == nil {
			continue
		}
		if *bp.locks[k].exclusive == tid {
			delete(bp.pages, k)
		}
	}
	// bp.mu.Unlock(tid)
	bp.releaseLocks(tid)
	// bp.mu.Lock(tid)
	delete(bp.running, tid)
	bp.mu.Unlock(tid, "AbortTransaction")
	return nil
}

// Commit the transaction, releasing locks. Because GoDB is FORCE/NO STEAL, none
// of the pages tid has dirtied will be on disk, so prior to releasing locks you
// should iterate through pages and write them to disk.  In GoDB lab3 we assume
// that the system will not crash while doing this, allowing us to avoid using a
// WAL. You do not need to implement this for lab 1.
// TODO: some code goes here : func (bp *BufferPool) CommitTransaction(tid TransactionID)
func (bp *BufferPool) CommitTransaction(tid TransactionID) error {
	bp.mu.Lock(tid, "CommitTransaction")
	// fmt.Printf("lock from commitTransaction, tid: %d\n", tid)
	if _, ok := bp.running[tid]; !ok {
		bp.mu.Unlock(tid, "CommitTransaction")
		return errors.New("txn is not running")
	}
	// fmt.Println("committing")
	// for k := range bp.locks {
	// 	fmt.Println(bp.locks[k])
	// }

	for k, v := range bp.pages {
		if bp.locks[k] == nil {
			continue
		}
		if bp.locks[k].exclusive == nil {
			continue
		}
		if *bp.locks[k].exclusive == tid && v.isDirty() {
			v.getFile().flushPage(v)
			v.setDirty(tid, false)
		}
	}
	// bp.mu.Unlock(tid)
	bp.releaseLocks(tid)
	// bp.mu.Lock(tid)
	delete(bp.running, tid)
	bp.mu.Unlock(tid, "CommitTransaction")
	return nil
}

func (bp *BufferPool) releaseLocks(tid TransactionID) {
	// bp.mu.Lock(tid)
	for k := range bp.locks {
		if bp.locks[k].exclusive != nil {
			bp.locks[k].exclusive = nil
		}
		if _, ok := bp.locks[k].shared[tid]; ok {
			delete(bp.locks[k].shared, tid)
		}
		if len(bp.locks[k].shared) == 0 && bp.locks[k].exclusive == nil {
			delete(bp.locks, k)
		}
		delete(bp.wfg, tid)
		for k_ := range bp.wfg {
			delete(bp.wfg[k_], tid)
		}
	}

	// bp.mu.Unlock(tid)
}

// Begin a new transaction. You do not need to implement this for lab 1.
//
// Returns an error if the transaction is already running.
// TODO: some code goes here: func (bp *BufferPool) BeginTransaction(tid TransactionID) error
func (bp *BufferPool) BeginTransaction(tid TransactionID) error {
	bp.mu.Lock(tid, "BeginTransaction")
	// fmt.Printf("lock from beginTransaction, tid: %d\n", tid)
	if _, ok := bp.running[tid]; ok {
		bp.mu.Unlock(tid, "BeginTransaction")
		return errors.New("txn already running")
	}
	bp.running[tid] = struct{}{}
	bp.mu.Unlock(tid, "BeginTransaction")
	return nil
}

// If necessary, evict clean page from the buffer pool. If all pages are dirty,
// return an error.
func (bp *BufferPool) evictPage() error {
	if len(bp.pages) < bp.maxPages {
		return nil
	}

	// evict first clean page
	for key, page := range bp.pages {
		if !page.isDirty() {
			delete(bp.pages, key)
			return nil
		}
	}

	return GoDBError{BufferPoolFullError, "all pages in buffer pool are dirty"}
}

// Returns true if the transaction is runing.
// TODO: some code goes here :func (bp *BufferPool) IsRunning(tid TransactionID) bool

// Loads the specified page from the specified DBFile, but does not lock it.
// TODO: some code goes here : func (bp *BufferPool) loadPage(file DBFile,
// pageNo int) (Page, error)

// func (bp *BufferPool) loadPage(file DBFile, pageNo int) (Page, error) {

// }

// Retrieve the specified page from the specified DBFile (e.g., a HeapFile), on
// behalf of the specified transaction. If a page is not cached in the buffer pool,
// you can read it from disk uing [DBFile.readPage]. If the buffer pool is full (i.e.,
// already stores numPages pages), a page should be evicted.  Should not evict
// pages that are dirty, as this would violate NO STEAL. If the buffer pool is
// full of dirty pages, you should return an error. Before returning the page,
// attempt to lock it with the specified permission.  If the lock is
// unavailable, should block until the lock is free. If a deadlock occurs, abort
// one of the transactions in the deadlock. For lab 1, you do not need to
// implement locking or deadlock detection. You will likely want to store a list
// of pages in the BufferPool in a map keyed by the [DBFile.pageKey].

func (bp *BufferPool) GetPage(file DBFile, pageNo int, tid TransactionID, perm RWPerm) (Page, error) {

	bp.mu.Lock(tid, "GetPage")
	// fmt.Printf("lock from getPage, permission %d, tid: %d\n", perm, tid)
	pageId := file.pageKey(pageNo)
	bp.mu.Unlock(tid, "GetPage")

	bp.acquireLock(pageId, tid, perm)

	bp.mu.Lock(tid, "GetPage")

	// fmt.Printf("exclusive lock in getPage: %d", *bp.locks[tid].exclusive)
	if page, ok := bp.pages[pageId]; ok {
		bp.mu.Unlock(tid, "GetPage")
		return page, nil
	}
	if err := bp.evictPage(); err != nil {
		bp.mu.Unlock(tid, "GetPage")
		return nil, err
	}
	page, _ := file.readPage(pageNo)
	bp.pages[pageId] = page

	// fmt.Printf("getPage returned, tid: %d\n", tid)
	bp.mu.Unlock(tid, "GetPage")
	return page, nil
}

func (bp *BufferPool) acquireLock(pageId any, tid TransactionID, perm RWPerm) error {
	// ctr := 0
	for {
		bp.mu.Lock(tid, "acquireLock")

		if bp.locks[pageId] == nil {
			bp.locks[pageId] = &PageLock{make(map[TransactionID]struct{}), nil}
		}
		lock := bp.locks[pageId]

		if perm == ReadPerm && (lock.exclusive == nil || *lock.exclusive == tid) {
			lock.shared[tid] = struct{}{}

			bp.mu.Unlock(tid, "acquireLock")
			return nil
		}
		if _, ok := lock.shared[tid]; (ok && len(lock.shared) == 1 ||
			len(lock.shared) == 0) && perm == WritePerm && (lock.exclusive ==
			nil || *lock.exclusive == tid) {
			lock.exclusive = &tid

			bp.mu.Unlock(tid, "acquireLock")
			return nil
		}

		// populate wfg
		if lock.exclusive != nil {
			if _, ok := bp.wfg[tid]; !ok {
				bp.wfg[tid] = make(map[TransactionID]struct{})

			}
			bp.wfg[tid][*lock.exclusive] = struct{}{}
		}
		for k := range lock.shared {
			if k == tid {
				continue
			}
			if _, ok := bp.wfg[tid]; !ok {
				bp.wfg[tid] = make(map[TransactionID]struct{})

			}
			bp.wfg[tid][k] = struct{}{}
		}

		if bp.detectDeadlock(tid) {
			for i := range bp.wfg {
				for j := range bp.wfg[i] {
					if i == tid || j == tid {
						delete(bp.wfg, i)
					}
				}
			}

			bp.mu.Unlock(tid, "acquireLock")
			bp.AbortTransaction(tid)
			return errors.New("deadlock")
		}

		bp.mu.Unlock(tid, "acquireLock")
		time.Sleep(100 * time.Millisecond)
	}
}

func (bp *BufferPool) detectDeadlock(tid TransactionID) bool {
	visited := make(map[TransactionID]bool)
	recStack := make(map[TransactionID]bool)
	// Helper function for DFS
	var dfs func(TransactionID) bool
	dfs = func(node TransactionID) bool {
		if recStack[node] {
			return true
		}
		if visited[node] {
			return false
		}
		visited[node] = true
		recStack[node] = true
		// Explore neighbors
		for neighbor := range bp.wfg[node] {
			if dfs(neighbor) {
				return true
			}
		}
		recStack[node] = false // Remove the node from recursion stack
		return false
	}
	return dfs(tid)
}
