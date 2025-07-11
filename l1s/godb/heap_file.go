package godb

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// A HeapFile is an unordered collection of tuples.
//
// HeapFile is a public class because external callers may wish to instantiate
// database tables using the method [LoadFromCSV]
type HeapFile struct {
	td            *TupleDesc
	numPages      int
	backingFile   string
	lastEmptyPage int
	// HeapFile should include the fields below;  you may want to add
	// additional fields
	bufPool *BufferPool
	sync.Mutex
}

// Hint: heap_page and heap_file need function there:  type heapFileRid struct
type heapFileRid struct {
	pageNo int
	slotNo int
}

// Create a HeapFile.
// Parameters
// - fromFile: backing file for the HeapFile.  May be empty or a previously created heap file.
// - td: the TupleDesc for the HeapFile.
// - bp: the BufferPool that is used to store pages read from the HeapFile
// May return an error if the file cannot be opened or created.
func NewHeapFile(fromFile string, td *TupleDesc, bp *BufferPool) (*HeapFile, error) {
	// TODO: some code goes here
	f, err := os.OpenFile(fromFile, os.O_CREATE|os.O_WRONLY, 0644)
	// f, err := os.Open(fromFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	numPages := fi.Size() / int64(PageSize)
	return &HeapFile{td, int(numPages), fromFile, -1, bp, sync.Mutex{}}, nil

}

// Return the name of the backing file
func (f *HeapFile) BackingFile() string {
	// TODO: some code goes here
	return f.backingFile

}

// Return the number of pages in the heap file
func (f *HeapFile) NumPages() int {
	// TODO: some code goes here
	return f.numPages

}

// Load the contents of a heap file from a specified CSV file.  Parameters are as follows:
// - hasHeader:  whether or not the CSV file has a header
// - sep: the character to use to separate fields
// - skipLastField: if true, the final field is skipped (some TPC datasets include a trailing separator on each line)
// Returns an error if the field cannot be opened or if a line is malformed
// We provide the implementation of this method, but it won't work until
// [HeapFile.insertTuple] is implemented
func (f *HeapFile) LoadFromCSV(file *os.File, hasHeader bool, sep string, skipLastField bool) error {
	// 使用 bufio.Scanner 逐行读取文件内容
	scanner := bufio.NewScanner(file)
	cnt := 0 // 行计数器，用于跟踪当前处理到的行号
	for scanner.Scan() {
		line := scanner.Text() // 获取当前行文本
		// 将行内容按指定的分隔符 sep 分割成字段数组
		fields := strings.Split(line, sep)
		if skipLastField {
			// 如果 skipLastField 为 true，跳过最后一个字段
			fields = fields[0 : len(fields)-1]
		}
		numFields := len(fields) // 获取当前行的字段数
		cnt++                    // 行数增加

		// 获取文件的描述符，包含字段类型等元信息
		desc := f.Descriptor()
		if desc == nil || desc.Fields == nil {
			// 如果描述符为 nil，说明数据格式不正确，返回错误
			return GoDBError{MalformedDataError, "Descriptor was nil"}
		}
		// 检查当前行的字段数是否与描述符中的字段数匹配
		if numFields != len(desc.Fields) {
			return GoDBError{MalformedDataError, fmt.Sprintf("LoadFromCSV:  line %d (%s) does not have expected number of fields (expected %d, got %d)", cnt, line, len(f.Descriptor().Fields), numFields)}
		}
		// 如果第一行是标题行并且 hasHeader 为 true，跳过该行
		if cnt == 1 && hasHeader {
			continue
		}

		var newFields []DBValue // 用于存储处理后的字段值
		// 遍历当前行的每个字段，并根据字段类型进行处理
		for fno, field := range fields {
			switch f.Descriptor().Fields[fno].Ftype {
			case IntType:
				// 对于整数类型字段，先移除空白字符
				field = strings.TrimSpace(field)
				// 将字段转换为浮点数，然后再转换为整数
				floatVal, err := strconv.ParseFloat(field, 64)
				if err != nil {
					// 如果转换失败，返回类型不匹配错误
					return GoDBError{TypeMismatchError, fmt.Sprintf("LoadFromCSV: couldn't convert value %s to int, tuple %d", field, cnt)}
				}
				intValue := int(floatVal)                                // 将浮点数转为整数
				newFields = append(newFields, IntField{int64(intValue)}) // 将转换后的整数添加到字段列表中
			case StringType:
				// 对于字符串类型字段，截取最大允许长度的字符串
				if len(field) > StringLength {
					field = field[0:StringLength]
				}
				newFields = append(newFields, StringField{field}) // 将字符串字段添加到字段列表中
			}
		}

		// 创建一个新的元组
		newT := Tuple{*f.Descriptor(), newFields, nil}
		// 为新元组生成一个唯一的TID（事务ID）
		tid := NewTID()

		// 获取缓冲池（BufferPool）并开始一个新的事务
		bp := f.bufPool
		bp.BeginTransaction(tid)

		// 将元组插入到 HeapFile 中
		f.insertTuple(&newT, tid)

		// 将脏页（dirty pages）强制写入磁盘，可能是由于事务提交还未实现，所以手动调用
		bp.FlushAllPages()

		// 频繁提交事务，以避免缓冲池中的所有页都被占满
		bp.CommitTransaction(tid)
	}
	return nil // 所有数据加载完成后，返回nil表示成功
}

// Read the specified page number from the HeapFile on disk. This method is
// called by the [BufferPool.GetPage] method when it cannot find the page in its
// cache.
//
// This method will need to open the file supplied to the constructor, seek to
// the appropriate offset, read the bytes in, and construct a [heapPage] object,
// using the [heapPage.initFromBuffer] method.
func (f *HeapFile) readPage(pageNo int) (Page, error) {
	// TODO: some code goes here
	file, err := os.OpenFile(f.backingFile, os.O_CREATE|os.O_RDONLY, 0644)
	// file, err := os.Open(f.backingFile)

	if err != nil {
		return nil, err
	}
	defer file.Close()
	b := make([]byte, PageSize)
	n, err := file.ReadAt(b, int64(pageNo*PageSize))
	if err != nil {
		return nil, err
	}
	if n != PageSize {
		return nil, GoDBError{MalformedDataError, "not enough bytes read in ReadPage"}
	}
	pg, err := newHeapPage(f.Descriptor(), pageNo, f)
	if err != nil {
		return nil, err
	}
	pg.initFromBuffer(bytes.NewBuffer(b))
	return pg, nil
}

// Add the tuple to the HeapFile. This method should search through pages in the
// heap file, looking for empty slots and adding the tuple in the first empty
// slot if finds.
//
// If none are found, it should create a new [heapPage] and insert the tuple
// there, and write the heapPage to the end of the HeapFile (e.g., using the
// [flushPage] method.)
//
// To iterate through pages, it should use the [BufferPool.GetPage method]
// rather than directly reading pages itself. For lab 1, you do not need to
// worry about concurrent transactions modifying the Page or HeapFile. We will
// add support for concurrent modifications in lab 3.
//
// The page the tuple is inserted into should be marked as dirty.
func (f *HeapFile) insertTuple(t *Tuple, tid TransactionID) error {
	// TODO: some code goes here
	var start int

	if f.lastEmptyPage == -1 {
		start = 0
	} else {
		start = f.lastEmptyPage
	}
	endPage := f.numPages

	for p := start; p < endPage; p++ {
		pg, err := f.bufPool.GetPage(f, p, tid, ReadPerm)
		if err != nil {
			return err
		}
		if pg.(*heapPage).getNumEmptySlots() == 0 {
			continue
		}

		pg, err = f.bufPool.GetPage(f, p, tid, WritePerm)
		if err != nil {
			return err
		}
		heapp := pg.(*heapPage)
		_, err = heapp.insertTuple(t)
		if err != nil && err != ErrPageFull {
			return err
		}
		if err == nil {
			heapp.setDirty(tid, true)
			f.lastEmptyPage = p // this is fine because lastEmptyPage is a hint, not forcing
			return nil
		}
	}

	//no free slots, create new page
	heapp, err := newHeapPage(f.td, f.numPages, f)
	err = f.flushPage(heapp) // flush an empty page to later add to buffer pool, helps maintain dirtiness
	if err != nil {
		return err
	}
	f.lastEmptyPage = f.numPages
	p := f.numPages
	f.numPages++

	pg, err := f.bufPool.GetPage(f, p, tid, WritePerm)
	if err != nil {
		return err
	}
	heapp = pg.(*heapPage)
	_, err = heapp.insertTuple(t)
	if err != nil {
		return err
	}
	heapp.setDirty(tid, true)

	f.lastEmptyPage = p

	return nil
}

// Remove the provided tuple from the HeapFile.
//
// This method should use the [Tuple.Rid] field of t to determine which tuple to
// remove. The Rid field should be set when the tuple is read using the
// [Iterator] method, or is otherwise created (as in tests). Note that Rid is an
// empty interface, so you can supply any object you wish. You will likely want
// to identify the heap page and slot within the page that the tuple came from.
//
// The page the tuple is deleted from should be marked as dirty.
func (f *HeapFile) deleteTuple(t *Tuple, tid TransactionID) error {
	// TODO: some code goes here
	if t.Rid == nil {
		return GoDBError{TupleNotFoundError, "provided tuple has null rid, cannot delete"}
	}

	rid, ok := t.Rid.(heapFileRid)
	if !ok {
		return GoDBError{TupleNotFoundError, "provided tuple is not a heap file tuple, based on rid"}
	}

	if rid.pageNo < 0 || rid.pageNo >= f.NumPages() {
		return GoDBError{TupleNotFoundError, "provided tuple references a page that does not exists"}
	}

	pg, err := f.bufPool.GetPage(f, rid.pageNo, tid, WritePerm)
	if err != nil {
		return err
	}
	hp, ok := pg.(*heapPage)
	if !ok {
		return GoDBError{IncompatibleTypesError, "buffer pool returned non-heap page when heap page expected"}
	}
	hp.setDirty(tid, true)
	err = hp.deleteTuple(rid)
	if err != nil {
		return err
	}

	if rid.pageNo < f.lastEmptyPage {
		f.lastEmptyPage = rid.pageNo
	}

	return nil
}

// Method to force the specified page back to the backing file at the
// appropriate location. This will be called by BufferPool when it wants to
// evict a page. The Page object should store information about its offset on
// disk (e.g., that it is the ith page in the heap file), so you can determine
// where to write it back.
func (f *HeapFile) flushPage(p Page) error {
	// TODO: some code goes here
	file, err := os.OpenFile(f.backingFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	hp := p.(*heapPage)

	buf, err := hp.toBuffer()
	if err != nil {
		return err
	}
	_, err = file.WriteAt(buf.Bytes(), int64(hp.pageNo*PageSize))
	return err
}

// [Operator] descriptor method -- return the TupleDesc for this HeapFile
// Supplied as argument to NewHeapFile.
func (f *HeapFile) Descriptor() *TupleDesc {
	// TODO: some code goes here
	return f.td

}

// [Operator] iterator method
// Return a function that iterates through the records in the heap file
// Note that this method should read pages from the HeapFile using the
// BufferPool method GetPage, rather than reading pages directly,
// since the BufferPool caches pages and manages page-level locking state for
// transactions
// You should esnure that Tuples returned by this method have their Rid object
// set appropriate so that [deleteTuple] will work (see additional comments there).
func (f *HeapFile) Iterator(tid TransactionID) (func() (*Tuple, error), error) {
	// TODO: some code goes here
	nPages := f.NumPages()
	pgNo := 0
	var pgIter func() (*Tuple, error)
	return func() (*Tuple, error) {
		for {
			if pgIter == nil {
				if pgNo == nPages {
					return nil, nil
				}
				p, err := f.bufPool.GetPage(f, pgNo, tid, ReadPerm)
				if err != nil {
					return nil, err
				}
				pgIter = p.(*heapPage).tupleIter() //assume this is a heapPage object
				pgNo++
			}
			next, err := pgIter()
			if err != nil {
				return nil, err
			}
			if next == nil {
				pgIter = nil
			} else {
				return &Tuple{*f.td, next.Fields, next.Rid}, nil
			}
		}
	}, nil
}

// internal strucuture to use as key for a heap page
type heapHash struct {
	FileName string
	PageNo   int
}

// This method returns a key for a page to use in a map object, used by
// BufferPool to determine if a page is cached or not.  We recommend using a
// heapHash struct as the key for a page, although you can use any struct that
// does not contain a slice or a map that uniquely identifies the page.
func (f *HeapFile) pageKey(pgNo int) any {
	// TODO: some code goes here
	return heapHash{f.backingFile, pgNo}

}
