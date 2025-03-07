package godb

import (
	"bytes"
	"encoding/binary"
	"sync"
)

/* HeapPage implements the Page interface for pages of HeapFiles. We have
provided our interface to HeapPage below for you to fill in, but you are not
required to implement these methods except for the three methods that the Page
interface requires.  You will want to use an interface like what we provide to
implement the methods of [HeapFile] that insert, delete, and iterate through
tuples.

In GoDB all tuples are fixed length, which means that given a TupleDesc it is
possible to figure out how many tuple "slots" fit on a given page.

In addition, all pages are PageSize bytes.  They begin with a header with a 32
bit integer with the number of slots (tuples), and a second 32 bit integer with
the number of used slots.

Each tuple occupies the same number of bytes.  You can use the go function
unsafe.Sizeof() to determine the size in bytes of an object.  So, a GoDB integer
(represented as an int64) requires unsafe.Sizeof(int64(0)) bytes.  For strings,
we encode them as byte arrays of StringLength, so they are size
((int)(unsafe.Sizeof(byte('a')))) * StringLength bytes.  The size in bytes  of a
tuple is just the sum of the size in bytes of its fields.

Once you have figured out how big a record is, you can determine the number of
slots on on the page as:

remPageSize = PageSize - 8 // bytes after header
numSlots = remPageSize / bytesPerTuple //integer division will round down

To serialize a page to a buffer, you can then:

write the number of slots as an int32
write the number of used slots as an int32
write the tuples themselves to the buffer

You will follow the inverse process to read pages from a buffer.

Note that to process deletions you will likely delete tuples at a specific
position (slot) in the heap page.  This means that after a page is read from
disk, tuples should retain the same slot number. Because GoDB will never evict a
dirty page, it's OK if tuples are renumbered when they are written back to disk.

*/

type heapPage struct {
	// TODO: some code goes here
	desc     TupleDesc
	numSlots int32
	numUsed  int32
	dirty    bool
	tuples   []*Tuple
	pageNo   int
	file     *HeapFile
	sync.Mutex
}

// Construct a new heap page
func newHeapPage(desc *TupleDesc, pageNo int, f *HeapFile) (*heapPage, error) {
	// TODO: some code goes here
	var pg heapPage
	pg.desc = *desc
	pg.numSlots = int32((PageSize - 8) / desc.bytesPerTuple())
	pg.numUsed = 0
	pg.dirty = false
	pg.tuples = make([]*Tuple, pg.numSlots)
	pg.pageNo = pageNo
	pg.file = f
	//pg.SetBeforeImage()
	return &pg, nil
}

// Hint: heapfile/insertTuple needs function there:  func (h *heapPage) getNumEmptySlots() int
func (h *heapPage) getNumEmptySlots() int {
	return int(h.numSlots - h.numUsed)
}

func (h *heapPage) getNumSlots() int {
	// TODO: some code goes here
	return int(h.numSlots)

}

var ErrPageFull = GoDBError{PageFullError, "page is full"}

// Insert the tuple into a free slot on the page, or return an error if there are
// no free slots.  Set the tuples rid and return it.
func (h *heapPage) insertTuple(t *Tuple) (recordID, error) {
	// TODO: some code goes here
	for i := 0; i < int(h.numSlots); i++ {
		if h.tuples[i] == nil {
			h.tuples[i] = t
			h.numUsed++
			t.Rid = heapFileRid{h.pageNo, i}
			return t.Rid, nil
		}
	}
	return 0, ErrPageFull
}

// Delete the tuple at the specified record ID, or return an error if the ID is
// invalid.
func (h *heapPage) deleteTuple(rid recordID) error {
	// TODO: some code goes here
	heapRid, ok := rid.(heapFileRid)
	if !ok {
		return GoDBError{TupleNotFoundError, "supplied rid is not a heapFileRid"}
	}
	slot := heapRid.slotNo
	if slot < 0 || slot >= int(h.numSlots) {
		return GoDBError{TupleNotFoundError, "slot does not exist on delete"}
	}
	if h.tuples[slot] == nil {
		return GoDBError{TupleNotFoundError, "element already deleted"}
	}
	h.numUsed--
	h.tuples[slot] = nil
	return nil
}

// Page method - return whether or not the page is dirty
func (h *heapPage) isDirty() bool {
	// TODO: some code goes here
	return h.dirty
}

// Page method - mark the page as dirty
func (h *heapPage) setDirty(tid TransactionID, dirty bool) {
	// TODO: some code goes here
	h.dirty = dirty

}

// Page method - return the corresponding HeapFile
// for this page.
func (p *heapPage) getFile() DBFile {
	// TODO: some code goes here
	return p.file

}

// Allocate a new bytes.Buffer and write the heap page to it. Returns an error
// if the write to the the buffer fails. You will likely want to call this from
// your [HeapFile.flushPage] method.  You should write the page header, using
// the binary.Write method in LittleEndian order, followed by the tuples of the
// page, written using the Tuple.writeTo method.
func (h *heapPage) toBuffer() (*bytes.Buffer, error) {
	// TODO: some code goes here
	b := new(bytes.Buffer)

	err := binary.Write(b, binary.LittleEndian, (int32)(h.numSlots))
	if err != nil {
		return nil, err
	}
	err = binary.Write(b, binary.LittleEndian, (int32)(h.numUsed))
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(h.tuples); i++ {
		t := h.tuples[i]
		if t != nil {
			err = t.writeTo(b)
			if err != nil {
				return nil, err
			}
		}
	}
	if b.Len() > PageSize {
		return nil, GoDBError{MalformedDataError, "buffer is greater than page size"}
	}
	b.Write(make([]byte, PageSize-b.Len())) // pad to page size

	return b, nil
}

// Read the contents of the HeapPage from the supplied buffer.
func (h *heapPage) initFromBuffer(buf *bytes.Buffer) error {
	// TODO: some code goes here
	var numSlotsHeader, numUsedHeader int32
	err := binary.Read(buf, binary.LittleEndian, &numSlotsHeader)
	if err != nil {
		return err
	}
	err = binary.Read(buf, binary.LittleEndian, &numUsedHeader)
	if err != nil {
		return err
	}
	tups := make([]*Tuple, numSlotsHeader)
	for i := 0; i < int(numUsedHeader); i++ {
		t, err := readTupleFrom(buf, &h.desc)
		t.Rid = heapFileRid{h.pageNo, i}
		if err != nil {
			return err
		}
		tups[i] = t
	}
	h.numSlots = numSlotsHeader
	h.numUsed = numUsedHeader
	h.dirty = false
	h.tuples = tups
	//h.SetBeforeImage()
	return nil
}

// Return a function that iterates through the tuples of the heap page.  Be sure
// to set the rid of the tuple to the rid struct of your choosing beforing
// return it. Return nil, nil when the last tuple is reached.
func (p *heapPage) tupleIter() func() (*Tuple, error) {
	// TODO: some code goes here
	i := 0
	return func() (*Tuple, error) {
		for {
			if i >= len(p.tuples) {
				return nil, nil
			}
			t := p.tuples[i]
			i++
			if t != nil {
				return t, nil
			}
		}
	}
}
