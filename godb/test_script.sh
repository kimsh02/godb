#!/bin/zsh

go test -v -run TestLockingAcquireReadLocksOnSamePage
go test -v -run TestLockingAcquireReadWriteLocksOnSamePage
go test -v -run TestLockingAcquireWriteReadLocksOnSamePage
go test -v -run TestLockingAcquireReadWriteLocksOnTwoPages
go test -v -run TestLockingAcquireWriteLocksOnTwoPages
go test -v -run TestLockingAcquireReadLocksOnTwoPages
go test -v -run TestLockingUpgrade
go test -v -run TestLockingAcquireWriteAndReadLocks
