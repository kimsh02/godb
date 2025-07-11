package godb

import (
	"math/rand"
	"os"
	"testing"
	"time"
)

func TestDeadlockReadWriteDifferentFiles(t *testing.T) {
	bp, hf1, tid1, tid2 := lockingTestSetUp(t)

	os.Remove("test2.dat")
	td, _, _ := makeTupleTestVars()
	hf2, err := NewHeapFile("test2.dat", &td, bp)
	if err != nil {
		t.Fatalf(err.Error())
	}

	csvFile, err := os.Open("txn_test_300_3.csv")
	if err != nil {
		t.Fatalf("error opening test file")
	}
	hf2.LoadFromCSV(csvFile, false, ",", false)

	lg1Read := startGrabber(bp, tid1, hf1, 0, ReadPerm)
	lg2Read := startGrabber(bp, tid2, hf2, 0, ReadPerm)

	time.Sleep(POLL_INTERVAL)

	lg1Write := startGrabber(bp, tid1, hf2, 0, WritePerm)
	lg2Write := startGrabber(bp, tid2, hf1, 0, WritePerm)

	for {
		time.Sleep(POLL_INTERVAL)

		if lg1Write.acquired() && lg2Write.acquired() {
			t.Errorf("Should not both get write lock")
		}
		if lg1Write.acquired() != lg2Write.acquired() {
			break
		}

		if lg1Write.getError() != nil {
			bp.AbortTransaction(tid1) // at most abort twice; should be able to abort twice
			time.Sleep(time.Duration((float64(WAIT_INTERVAL) * rand.Float64())))

			tid1 = NewTID()
			lg1Read = startGrabber(bp, tid1, hf1, 0, ReadPerm)
			time.Sleep(POLL_INTERVAL)
			lg1Write = startGrabber(bp, tid1, hf2, 0, WritePerm)
		}

		if lg2Write.getError() != nil {
			bp.AbortTransaction(tid2) // at most abort twice; should be able to abort twice
			time.Sleep(time.Duration((float64(WAIT_INTERVAL) * rand.Float64())))

			tid2 = NewTID()
			lg2Read = startGrabber(bp, tid2, hf2, 0, ReadPerm)
			time.Sleep(POLL_INTERVAL)
			lg2Write = startGrabber(bp, tid2, hf1, 0, WritePerm)
		}
	}

	if lg1Read == nil || lg2Read == nil {
		t.Errorf("should not be nil")
	}
}
