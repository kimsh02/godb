package godb

import (
	"testing"
)

func TestLockTableReadersWriter(t *testing.T) {
	lt := NewLockTable()
	tid1 := NewTID()
	tid2 := NewTID()
	tid3 := NewTID()
	f1 := &MemFile{0, nil, nil}

	if lt.TryLock(f1, 0, tid1, ReadPerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
	if lt.TryLock(f1, 0, tid2, ReadPerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
	if lt.TryLock(f1, 0, tid1, WritePerm) != Wait {
		t.Errorf("Expected to wait")
	}
	if lt.TryLock(f1, 0, tid3, ReadPerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
	if lt.TryLock(f1, 0, tid1, WritePerm) != Wait {
		t.Errorf("Expected to wait")
	}
	lt.ReleaseLocks(tid2)
	lt.ReleaseLocks(tid3)
	if lt.TryLock(f1, 0, tid1, WritePerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
}
