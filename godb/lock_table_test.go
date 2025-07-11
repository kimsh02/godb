package godb

import (
	"testing"
)

func TestLockTableReadRead(t *testing.T) {
	lt := NewLockTable()
	tid1 := NewTID()
	tid2 := NewTID()
	f := &MemFile{0, nil, nil}
	if lt.TryLock(f, 0, tid1, ReadPerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
	if lt.TryLock(f, 0, tid2, ReadPerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
}

func TestLockTableReadWrite(t *testing.T) {
	lt := NewLockTable()
	tid1 := NewTID()
	tid2 := NewTID()
	f1 := &MemFile{0, nil, nil}
	f2 := &MemFile{1, nil, nil}
	if lt.TryLock(f1, 0, tid1, ReadPerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
	if lt.TryLock(f2, 0, tid2, WritePerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
}

func TestLockTableUpgrade(t *testing.T) {
	lt := NewLockTable()
	tid1 := NewTID()
	f1 := &MemFile{0, nil, nil}
	if lt.TryLock(f1, 0, tid1, ReadPerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
	if lt.TryLock(f1, 0, tid1, WritePerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
}

func TestLockTableReadRelease(t *testing.T) {
	lt := NewLockTable()
	tid1 := NewTID()
	tid2 := NewTID()
	f1 := &MemFile{0, nil, nil}
	if lt.TryLock(f1, 0, tid1, ReadPerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
	if lt.TryLock(f1, 0, tid2, WritePerm) != Wait {
		t.Errorf("Expected to wait")
	}
	lt.ReleaseLocks(tid1)
	if lt.TryLock(f1, 0, tid2, WritePerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
}

func TestLockTableWriteRelease(t *testing.T) {
	lt := NewLockTable()
	tid1 := NewTID()
	tid2 := NewTID()
	f1 := &MemFile{0, nil, nil}
	if lt.TryLock(f1, 0, tid1, WritePerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
	if lt.TryLock(f1, 0, tid2, WritePerm) != Wait {
		t.Errorf("Expected to wait")
	}
	lt.ReleaseLocks(tid1)
	if lt.TryLock(f1, 0, tid2, WritePerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
}

func TestLockTableDeadlock(t *testing.T) {
	lt := NewLockTable()
	tid1 := NewTID()
	tid2 := NewTID()
	f1 := &MemFile{0, nil, nil}
	f2 := &MemFile{1, nil, nil}
	if lt.TryLock(f1, 0, tid1, WritePerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
	if lt.TryLock(f2, 0, tid2, WritePerm) != Grant {
		t.Errorf("Expected lock to be granted")
	}
	if lt.TryLock(f2, 0, tid1, ReadPerm) != Wait {
		t.Errorf("Expected to wait")
	}
	if lt.TryLock(f1, 0, tid2, ReadPerm) != Abort {
		t.Errorf("Expected to abort")
	}
}
