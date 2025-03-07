package godb

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// Run threads transactions, each each of which reads a single tuple from a
// page, deletes the tuple, and re-inserts it with an incremented value. There
// will be deadlocks, so your deadlock handling will have to be correct to allow
// all transactions to be committed and the value to be incremented threads
// times.
func validateTransactionsBarrier(t *testing.T, threads int) {
	bp, hf, _, _, _, t2 := transactionTestSetUpVarLen(t, 1, 1)

	var startWg sync.WaitGroup

	// sleep for an increasingly long time after deadlocks. this backoff helps avoid starvation
	printErr := func(thrId int, err error) {
		t.Logf("thread %d operation failed: %v", thrId, err)
	}

	waitCond := NewBarrier(threads)

	incrementer := func(thrId int) {
		for tid := TransactionID(0); ; bp.AbortTransaction(tid) {
			waitCond.Wait()

			tid = NewTID()
			bp.BeginTransaction(tid)
			iter1, err := hf.Iterator(tid)
			if err != nil {
				continue
			}

			readTup, err := iter1()
			if err != nil {
				printErr(thrId, err)
				continue
			}

			time.Sleep(10 * time.Millisecond)

			var writeTup = Tuple{
				Desc: readTup.Desc,
				Fields: []DBValue{
					readTup.Fields[0],
					IntField{readTup.Fields[1].(IntField).Value + 1},
				}}

			fmt.Println(writeTup)

			dop := NewDeleteOp(hf, hf)
			iterDel, err := dop.Iterator(tid)
			if err != nil {
				continue
			}
			delCnt, err := iterDel()
			if err != nil {
				printErr(thrId, err)
				continue
			}

			if delCnt.Fields[0].(IntField).Value != 1 {
				t.Errorf("Delete Op should return 1")
				waitCond.Done()
				break
			}

			iop := NewInsertOp(hf, &Singleton{writeTup, false})
			fmt.Println(writeTup)
			iterIns, err := iop.Iterator(tid)
			if err != nil {
				continue
			}
			insCnt, err := iterIns()
			if err != nil {
				printErr(thrId, err)
				continue
			}

			if insCnt.Fields[0].(IntField).Value != 1 {
				t.Errorf("Insert Op should return 1")
				waitCond.Done()
				break
			}

			bp.CommitTransaction(tid)
			waitCond.Done()
			break //exit on success, so we don't do terminal abort
		}

		startWg.Done()
	}

	// Prepare goroutines
	startWg.Add(threads)
	for i := 0; i < threads; i++ {
		go incrementer(i)
	}

	// Wait for all goroutines to finish
	startWg.Wait()

	tid := NewTID()
	bp.BeginTransaction(tid)
	iter, _ := hf.Iterator(tid)
	tup, _ := iter()

	diff := tup.Fields[1].(IntField).Value - t2.Fields[1].(IntField).Value
	if diff != int64(threads) {
		t.Errorf("Expected #increments = %d, found %d", threads, diff)
	}
}

func TestTransactionFiveThreadsBarrier(t *testing.T) {
	nRuns := 1
	fmt.Println("test")
	for i := 0; i < nRuns; i++ {
		validateTransactionsBarrier(t, 25)
		t.Logf("Test %d/%d completed", i+1, nRuns)
	}
}
