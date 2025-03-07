package godb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
)

/*
log_file.go implements the recovery subsystem of GoDb. The functions in this
file assist with reading and writing log records to a log file. The log file
is used to recover the database in the event of a crash.

It is the responsibility of the user of this module to ensure that write
ahead logging and two-phase locking discipline are followed.

The log file is formatted as a sequence of log records. Log records are
variable-length, and have the following high-level structure:

+--------------------------------------------------------+
| Record type (1 byte)                                   |
+--------------------------------------------------------+
| Transaction ID (4 bytes)                               |
+--------------------------------------------------------+
| Record body (variable length)                          |
|                                                        |
+--------------------------------------------------------+
| Offset (8 bytes)                                       |
+--------------------------------------------------------+

Records start with a type, which will be one of the following: AbortRecord,
CommitRecord, UpdateRecord, BeginRecord. The type is followed by the ID of
the transaction that created the record.

The contents of the body depends on the type. Abort, Commit, and Begin
records are empty. Update records consist of the before and after pages. A
page has the following format:

+--------------------------------------------------------+
| File num (4 bytes)                                     |
+--------------------------------------------------------+
| Page num (4 bytes)                                     |
+--------------------------------------------------------+
| Page contents (PageSize bytes)                         |
|                                                        |
+--------------------------------------------------------+

The file number of a page is an internal identifier for the page's file that
is tracked by the catalog.
*/

type LogFile struct {
	file       *os.File
	buf        bytes.Buffer
	offset     int64
	bufferPool *BufferPool
	catalog    *Catalog
}

type LogRecordType int8

const (
	AbortRecord  LogRecordType = iota
	CommitRecord LogRecordType = iota
	UpdateRecord LogRecordType = iota
	BeginRecord  LogRecordType = iota
)

func (t LogRecordType) String() string {
	switch t {
	case AbortRecord:
		return "abort"
	case CommitRecord:
		return "commit"
	case UpdateRecord:
		return "update"
	case BeginRecord:
		return "begin"
	default:
		return "unknown"
	}
}

// Initialize and back the log file with the specified file.
func NewLogFile(fileName string, bufferPool *BufferPool, catalog *Catalog) (*LogFile, error) {
	if bufferPool == nil || catalog == nil {
		return nil, fmt.Errorf("bufferPool and catalog must be non-nil")
	}
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	return &LogFile{file, buf, 0, bufferPool, catalog}, nil
}

func (w *LogFile) write(data any) {
	binary.Write(&w.buf, binary.LittleEndian, data)
	size := int64(binary.Size(data))
	w.offset += size
}

func (w *LogFile) Force() error {
	if w.buf.Len() == 0 {
		return nil
	}

	_, err := w.file.Write(w.buf.Bytes())
	if err != nil {
		return err
	}

	off, _ := w.file.Seek(0, io.SeekCurrent)
	if off != w.offset {
		log.Printf("offset mismatch: %d != %d", off, w.offset)
	}

	w.buf.Reset()
	return w.file.Sync()
}

func (f *LogFile) seek(offset int64, whence int) error {
	if err := f.Force(); err != nil {
		return err
	}

	new_offset, err := f.file.Seek(offset, whence)
	if err != nil {
		return fmt.Errorf("invalid seek (%d, %d): %w", offset, whence, err)
	}
	f.offset = new_offset

	return nil
}

func (f *LogFile) read(data any) error {
	var err error

	if err = f.Force(); err != nil {
		return err
	}

	if err = binary.Read(f.file, binary.LittleEndian, data); err != nil {
		return err
	}
	// log.Printf("read @%d", f.offset)
	f.offset += int64(binary.Size(data))
	return nil
}

func (w *LogFile) readTransactionID(tid *TransactionID) error {
	var v int32
	if err := w.read(&v); err != nil {
		return err
	}
	*tid = TransactionID(v)
	return nil
}

func (w *LogFile) writeHeader(typ LogRecordType, tid TransactionID) {
	w.write(int8(typ))
	w.write(int32(tid))
}

func (w *LogFile) writeFooter(offset int64) {
	w.write(offset)
}

func (w *LogFile) readPage() (Page, error) {
	var fileId int32
	if err := w.read(&fileId); err != nil {
		return nil, err
	}
	var pageNo int32
	if err := w.read(&pageNo); err != nil {
		return nil, err
	}
	f, err := w.catalog.GetTableInfoId(int(fileId))
	if err != nil {
		return nil, err
	}
	pg, err := newHeapPage(f.file.Descriptor(), int(pageNo), f.file.(*HeapFile))
	if err != nil {
		return nil, err
	}
	buf := make([]byte, PageSize)
	if err := w.read(buf); err != nil {
		return nil, err
	}
	if err := pg.initFromBuffer(bytes.NewBuffer(buf)); err != nil {
		return nil, err
	}
	return pg, nil
}

func (w *LogFile) writePage(page Page) error {
	switch p := page.(type) {
	case *heapPage:
		// if w.catalog == nil {
		// 	return fmt.Errorf("catalog must be non-nil")
		// }
		f, err := w.catalog.GetTableInfoDBFile(page.getFile())
		if err != nil {
			return err
		}
		w.write(int32(f.id))
		w.write(int32(p.PageNo()))
		buf, err := p.toBuffer()
		if err != nil {
			return err
		}
		w.write(buf.Bytes())
	default:
		return fmt.Errorf("unsupported page type: %T", page)
	}
	return nil
}

func (w *LogFile) LogAbort(tid TransactionID) {
	offset := w.offset
	// log.Printf("LogAbort@%d: %v", offset, tid)
	w.writeHeader(AbortRecord, tid)
	w.write(offset)
}

func (w *LogFile) LogCommit(tid TransactionID) {
	offset := w.offset
	// log.Printf("LogCommit@%d: %v", offset, tid)
	w.writeHeader(CommitRecord, tid)
	w.write(offset)
}

// Write an Update record that records the transaction ID and the before and
// after states of a page.
//
// Note: does not force the log to disk.
func (w *LogFile) LogUpdate(tid TransactionID, before Page, after Page) error {
	if before == nil || after == nil {
		return fmt.Errorf("before and after images must be non-nil")
	}
	offset := w.offset
	// log.Printf("LogUpdate@%d for %v: page %v", offset, tid, before.(*heapPage).pageNo)
	w.writeHeader(UpdateRecord, tid)
	w.writePage(before)
	w.writePage(after)
	w.write(offset)
	return nil
}

// Write a Begin record that records the transaction ID.
func (w *LogFile) LogBegin(tid TransactionID) {
	offset := w.offset
	// log.Printf("LogBegin@%d: %v", offset, tid)
	w.writeHeader(BeginRecord, tid)
	w.writeFooter(offset)
}

func (f *LogFile) writeString(s string) {
	f.write(int32(len(s)))
	f.write([]byte(s))
}

func (f *LogFile) readString() (string, error) {
	var n int32
	if err := f.read(&n); err != nil {
		return "", err
	}
	buf := make([]byte, n)
	if err := f.read(buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func (f *LogFile) writeFieldType(t FieldType) {
	f.writeString(t.Fname)
	f.writeString(t.TableQualifier)
	f.write(int8(t.Ftype))
}

func (f *LogFile) readFieldType(t *FieldType) error {
	var err error

	if t.Fname, err = f.readString(); err != nil {
		return err
	}
	if t.TableQualifier, err = f.readString(); err != nil {
		return err
	}
	var ftype int8
	if err := f.read(&ftype); err != nil {
		return err
	}
	t.Ftype = DBType(ftype)
	return nil
}

func (f *LogFile) writeTupleDesc(td *TupleDesc) {
	f.write(int8(len(td.Fields)))
	for _, fd := range td.Fields {
		f.writeFieldType(fd)
	}
}

func (f *LogFile) readTupleDesc(td *TupleDesc) error {
	var l int8
	if err := f.read(&l); err != nil {
		return err
	}
	td.Fields = make([]FieldType, int(l))
	for i := 0; i < int(l); i++ {
		if err := f.readFieldType(&td.Fields[i]); err != nil {
			return err
		}
	}
	return nil
}

type LogRecord interface {
	Offset() int64
	Type() LogRecordType
	Tid() TransactionID
}

type GenericLogRecord struct {
	offset int64
	typ    LogRecordType
	tid    TransactionID
}

func (r GenericLogRecord) Offset() int64 {
	return r.offset
}

func (r GenericLogRecord) Type() LogRecordType {
	return r.typ
}

func (r GenericLogRecord) Tid() TransactionID {
	return r.tid
}

type UpdateLogRecord struct {
	GenericLogRecord
	Before Page
	After  Page
}

// Returns an iterator over the records in a log file.
//
// If the end of the file is reached, the iterator will return nil, nil. If the
// file ends with a partial record, the iterator will return an error.
func (f *LogFile) ForwardIterator() func() (LogRecord, error) {
	partial := func(msg string, err error) (LogRecord, error) {
		return nil, fmt.Errorf("failed to read %s. partial record at offset %d: %v", msg, f.offset, err)
	}

	return func() (LogRecord, error) {
		var record GenericLogRecord
		var ret LogRecord = &record

		record.offset = f.offset

		err := f.read(&record.typ)
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return partial("record type", err)
		}

		if err := f.readTransactionID(&record.tid); err != nil {
			return partial("transaction id", err)
		}

		if record.Type() == UpdateRecord {
			var update UpdateLogRecord
			var err error
			update.GenericLogRecord = record

			if update.Before, err = f.readPage(); err != nil {
				return partial("before page", err)
			}
			if update.After, err = f.readPage(); err != nil {
				return partial("after page", err)
			}
			ret = &update
		}

		var recordOffset int64
		if err := f.read(&recordOffset); err != nil || recordOffset != record.offset {
			return partial("offset", err)
		}

		return ret, nil
	}
}

func (f *LogFile) ReverseIterator() (func() (LogRecord, error), error) {
	// seek to end of file and check if there are any records
	if err := f.seek(0, io.SeekEnd); err != nil {
		return nil, err
	}

	return func() (LogRecord, error) {
		if f.offset < 8 {
			return nil, nil
		}

		var offset int64
		// move to start of record's offset field
		if err := f.seek(-8, io.SeekCurrent); err != nil {
			return nil, err
		}
		// read offset field
		if err := f.read(&offset); err != nil {
			return nil, err
		}
		// move to start of record and read
		if err := f.seek(offset, io.SeekStart); err != nil {
			return nil, err
		}
		record, err := f.ForwardIterator()()
		if err != nil {
			return nil, err
		}
		// move to end of next record
		if err := f.seek(offset, io.SeekStart); err != nil {
			return nil, err
		}
		return record, nil
	}, nil
}

// Print out a human readable representation of the log.
func (f *LogFile) OutputPrettyLog() error {
	oldPos := f.offset
	defer f.seek(oldPos, io.SeekStart)

	f.seek(0, io.SeekStart)

	iter := f.ForwardIterator()
	for {
		pos := f.offset

		record, err := iter()
		if err != nil {
			return err
		}
		if record == nil {
			break
		}

		if record.Type() == BeginRecord || record.Type() == CommitRecord || record.Type() == AbortRecord {
			log.Printf("%d RECORD %s (%d) offset=%d\n", pos, record.Type().String(), record.Tid(), record.Offset())
		} else if record.Type() == UpdateRecord {
			update := record.(*UpdateLogRecord)
			log.Printf("%d RECORD %s (%d) offset=%d page=%v\n", pos, record.Type().String(), record.Tid(), record.Offset(), update.Before.(*heapPage).getFile().pageKey(update.Before.(*heapPage).pageNo))
		} else {
			log.Printf("unexpected record: %#v", record)
		}
	}
	return nil
}
