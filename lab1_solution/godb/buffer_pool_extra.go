package godb

import (
	"fmt"
	"io"
	"log"
)

// Rolls back a transaction by reading the log and undoing the changes made by
// the transaction.
func (bp *BufferPool) Rollback(tid TransactionID) error {
	if bp.logFile == nil {
		return fmt.Errorf("log file not initialized")
	}

	iter, err := bp.logFile.ReverseIterator()
	if err != nil {
		return err
	}

	for record, err := iter(); record != nil && err == nil; record, err = iter() {
		if record.Tid() != tid {
			continue
		}

		if record.Type() == BeginRecord {
			break
		}

		if record.Type() == UpdateRecord {
			switch b := record.(*UpdateLogRecord).Before.(type) {
			case *heapPage:
				delete(bp.pages, b.getFile().pageKey(b.PageNo()))
				b.getFile().flushPage(b)
			default:
				return fmt.Errorf("unexpected page type")
			}
		}
	}

	return bp.logFile.seek(0, io.SeekEnd)

}

// Returns the log file associated with the buffer pool.
func (bp *BufferPool) LogFile() *LogFile {

	return bp.logFile

}

// Recover the buffer pool from a log file. This should be called when the
// database is started, even if the log file is empty.
func (bp *BufferPool) Recover(logFile *LogFile) error {

	bp.logFile = logFile

	if err := bp.logFile.seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to start of file: %w", err)
	}

	// replay updates from the log and record losers
	losers := make(map[TransactionID]int64)
	iter := bp.logFile.ForwardIterator()
	record, err := iter()
	for record != nil && err == nil {
		log.Printf("Recovering record %+v\n", record)
		switch record.Type() {
		case BeginRecord:
			// record that a transaction has started
			losers[record.Tid()] = record.Offset()
		case AbortRecord:
		case CommitRecord:
			// if the transaction has committed or aborted, it is no longer a loser
			delete(losers, record.Tid())
		case UpdateRecord:
			updateRecord := record.(*UpdateLogRecord)

			// apply updates as we see them
			after := updateRecord.After.(*heapPage)
			pageKey := after.getFile().pageKey(after.PageNo())
			log.Printf("REDO %v", pageKey)
			delete(bp.pages, pageKey)
			if err := after.getFile().flushPage(after); err != nil {
				return err
			}
		}
		record, err = iter()
	}
	if err != nil {
		return err
	}

	// losers now contains the transactions that did not commit before the crash
	iter, err = bp.logFile.ReverseIterator()
	if err != nil {
		return fmt.Errorf("failed to create rev iterator: %w", err)
	}
	record, err = iter()
	for len(losers) > 0 && record != nil && err == nil {
		tid := record.Tid()
		_, is_loser := losers[tid]
		if is_loser {
			switch record.Type() {
			case UpdateRecord:
				updateRecord := record.(*UpdateLogRecord)
				page := updateRecord.Before.(*heapPage)
				pageKey := page.getFile().pageKey(page.PageNo())
				log.Printf("UNDO %v", pageKey)
				delete(bp.pages, pageKey)
				if err := page.getFile().flushPage(page); err != nil {
					return err
				}
			case BeginRecord:
				// seek to end of log, write an abort record, seek back
				offset := bp.logFile.offset
				if err := bp.logFile.seek(0, io.SeekEnd); err != nil {
					return err
				}
				bp.logFile.LogAbort(tid)
				if err := bp.logFile.Force(); err != nil {
					return err
				}
				if err := bp.logFile.seek(offset, io.SeekStart); err != nil {
					return err
				}
				delete(losers, tid)
			}
		}
		record, err = iter()
	}
	if err != nil {
		return fmt.Errorf("failed to read from reversed iterator: %w", err)
	}

	// reset to end of log
	return bp.logFile.seek(0, io.SeekEnd)
}
