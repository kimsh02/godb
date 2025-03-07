package godb

import (
	"testing"
)

// Tests the graph tid1 <-> tid2
func TestWaitsForPairCycle(t *testing.T) {
	tid1 := NewTID()
	tid2 := NewTID()
	wf := make(WaitFor)
	wf.AddEdges(tid1, []TransactionID{tid2})
	wf.AddEdges(tid2, []TransactionID{tid1})

	if !wf.DetectDeadlock(tid1) {
		t.Errorf("Expected tid1 to be part of a deadlock")
	}
	if !wf.DetectDeadlock(tid2) {
		t.Errorf("Expected tid2 to be part of a deadlock")
	}
}

// Tests the graph tid1 -> tid2
func TestWaitsForPair(t *testing.T) {
	tid1 := NewTID()
	tid2 := NewTID()
	wf := make(WaitFor)
	wf.AddEdges(tid1, []TransactionID{tid2})

	if wf.DetectDeadlock(tid1) || wf.DetectDeadlock(tid2) {
		t.Errorf("Unexpected deadlock")
	}
}

// Tests the graph tid1 <-> tid2, tid3 -> tid2
func TestWaitsForPairExtra(t *testing.T) {
	tid1 := NewTID()
	tid2 := NewTID()
	tid3 := NewTID()
	wf := make(WaitFor)
	wf.AddEdges(tid1, []TransactionID{tid2})
	wf.AddEdges(tid2, []TransactionID{tid1})
	wf.AddEdges(tid3, []TransactionID{tid2})

	if !wf.DetectDeadlock(tid1) {
		t.Errorf("Expected tid1 to be part of a deadlock")
	}
	if !wf.DetectDeadlock(tid2) {
		t.Errorf("Expected tid2 to be part of a deadlock")
	}
	if wf.DetectDeadlock(tid3) {
		t.Errorf("Did not expect tid3 to be part of a deadlock")
	}
}

// Tests the graph tid1 <-> tid2, tid3 -> tid2
func TestWaitsForRing(t *testing.T) {
	tid1 := NewTID()
	tid2 := NewTID()
	tid3 := NewTID()
	wf := make(WaitFor)
	wf.AddEdges(tid1, []TransactionID{tid2})
	wf.AddEdges(tid2, []TransactionID{tid3})
	wf.AddEdges(tid3, []TransactionID{tid1})

	if !wf.DetectDeadlock(tid1) {
		t.Errorf("Expected tid1 to be part of a deadlock")
	}
	if !wf.DetectDeadlock(tid2) {
		t.Errorf("Expected tid2 to be part of a deadlock")
	}
	if !wf.DetectDeadlock(tid3) {
		t.Errorf("Expected tid3 to be part of a deadlock")
	}
}
