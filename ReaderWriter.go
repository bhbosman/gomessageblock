package gomessageblock

import (
	"encoding/binary"
	"io"
	"strings"
	"sync"
	"sync/atomic"
)

type iMultiBlockBase interface {
	nextMultiBlock() *MultiBlock
	clearNext()
}

type iMultiBlock interface {
	iMultiBlockBase
	io.Reader
	io.Writer
	Size() int
	AddReaders(readWriters ...io.Reader) error
}

type ReaderWriter struct {
	uniqueNumber uint64
	blockSize    int
	mutex        sync.Mutex
	next         *MultiBlock
	last         *MultiBlock
	typeCode     uint32
	typeCodeRead bool
}

func (self *ReaderWriter) clearNext() {
	self.next = nil
}

func (self *ReaderWriter) nextMultiBlock() *MultiBlock {
	return self.next
}

func (self *ReaderWriter) Read(p []byte) (n int, err error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return self.InternalRead(p)
}

func (self *ReaderWriter) addByteBlock(p []byte) (int, error) {
	block := NewMultiBlockSize(self.blockSize)
	_, err := block.Write(p)
	if err != nil {
		return 0, err
	}
	self.addBlock(block)
	return len(p), nil
}

func (self *ReaderWriter) addBlockToFront(block *MultiBlock) {
	if self.last == nil {
		self.last = block
	}
	block.next = self.next
	self.next = block
}

func (self *ReaderWriter) addBlock(block *MultiBlock) {
	if self.last != nil {
		self.last.next = block
	}
	self.last = block
	if self.next == nil {
		self.next = block
	}
}

func (self *ReaderWriter) Write(p []byte) (int, error) {
	b, i, err := func() (bool, int, error) {
		self.mutex.Lock()
		defer self.mutex.Unlock()
		if self.last != nil {
			leftOver := self.last.bytes.Cap() - self.last.bytes.Len()
			if leftOver > len(p) {
				n, err := self.last.Write(p)
				return true, n, err
			}
		}
		return false, 0, nil
	}()

	if b {
		return i, err
	}

	return func() (int, error) {
		self.mutex.Lock()
		defer self.mutex.Unlock()

		if len(p) <= self.blockSize {
			return self.addByteBlock(p)
		} else {
			outstanding := len(p)
			index := 0
			nn := 0
			for outstanding > 0 {
				bi := index
				ei := index + self.blockSize
				if ei > len(p) {
					ei = len(p)
				}
				n, err := self.addByteBlock(p[bi:ei])
				if err != nil {
					return 0, err
				}
				nn += n
				index += n
				outstanding -= n
			}
			return nn, nil
		}
	}()
}

func (self *ReaderWriter) InternalRead(p []byte) (n int, err error) {
	moveNext := func(next *MultiBlock) {
		self.next = next.next
		if self.next == nil {
			self.last = nil
		}
	}
	next := self.next
	if next != nil {
		n, err := next.Read(p)
		if err == io.EOF {
			moveNext(next)
			return self.InternalRead(p)
		}
		if n < len(p) {

			var nn int
			nn, err = self.InternalRead(p[n:])
			if nn == 0 && err == io.EOF {
				if n > 0 {
					err = nil
				}
			}
			return n + nn, err
		}
		l := self.next.bytes.Len()
		if l == 0 {
			moveNext(next)
		}
		return n, nil
	}
	return 0, io.EOF
}

func (self *ReaderWriter) Flatten() ([]byte, error) {
	if self.next == nil {
		return nil, nil
	}
	if self.next.next == nil {
		return self.next.bytes.Bytes(), nil
	}
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return self.internalFlatten()
}

func (self *ReaderWriter) Size() int {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return self.InternalSize()
}

func (self *ReaderWriter) BlockCount() int {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return self.InternalBlockCount()
}

func (self *ReaderWriter) Waste() int {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return self.InternalWaste()
}

func (self *ReaderWriter) InternalSize() int {
	size := 0
	for node := self.next; node != nil; node = node.next {
		size += node.bytes.Len()
	}
	return size
}

func (self *ReaderWriter) internalFlatten() ([]byte, error) {
	size := self.InternalSize()
	if self.next.bytes.Cap() > size {
		var err error
		prev := self.next
		for node := prev.next; node != nil; prev, node = node, node.next {
			_, err = self.next.Write(node.bytes.Bytes())
			prev.next = nil
		}
		if err != nil {
			return nil, err
		}
		self.last = self.next
		return self.next.bytes.Bytes(), nil
	} else {
		if size < self.blockSize {
			size = self.blockSize
		}
		blockSize := NewMultiBlockSize(size)
		var err error
		var prev iMultiBlockBase = self
		for node := prev.nextMultiBlock(); node != nil; prev, node = node, node.nextMultiBlock() {
			_, err = blockSize.Write(node.bytes.Bytes())
			prev.clearNext()
		}
		if err != nil {
			return nil, err
		}
		self.next = blockSize
		self.last = blockSize
		return blockSize.bytes.Bytes(), nil
	}
}

func (self *ReaderWriter) AddReaders(readWriters ...io.Reader) error {
	readerWriterFunc := func(v *ReaderWriter) error {
		v.mutex.Lock()
		defer v.mutex.Unlock()
		if v.next != nil {
			if self.next == nil {
				self.next = v.next
				self.last = v.last
			} else {
				self.last.next = v.next
				self.last = v.last
			}
			v.next = nil
			v.last = nil
		}
		return nil
	}

	self.mutex.Lock()
	defer self.mutex.Unlock()
	for _, rw := range readWriters {
		switch v := rw.(type) {
		case *ReaderWriter:
			err := readerWriterFunc(v)
			if err != nil {
				return err
			}
		default:
			instance := *NewReaderWriter()
			_, err := io.Copy(&instance, v)
			if err != nil {
				return err
			}
			err = readerWriterFunc(&instance)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *ReaderWriter) Add(rws ...*ReaderWriter) error {
	l := make([]io.Reader, len(rws))
	for i, v := range rws {
		l[i] = v
	}
	return self.AddReaders(l...)
}

func (self *ReaderWriter) InternalWaste() int {
	waste := 0
	for node := self.next; node != nil; node = node.next {
		leftOver := node.bytes.Cap() - node.bytes.Len()
		waste += leftOver
	}
	return waste
}

func (self *ReaderWriter) InternalBlockCount() int {
	count := 0
	for node := self.next; node != nil; node = node.next {
		count++
	}
	return count
}

func (self *ReaderWriter) Dump(writer io.Writer) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	for node := self.next; node != nil; node = node.next {
		_, err := writer.Write(node.bytes.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *ReaderWriter) SetNext(next io.ReadWriter) error {
	sn := func(next *MultiBlock) {
		self.mutex.Lock()
		defer self.mutex.Unlock()
		self.addBlock(next)
	}
	switch v := next.(type) {
	case *ReaderWriter:
		sn(v.next)
		return nil
	default:
		dst := NewMultiBlock()
		_, err := io.Copy(dst, next)
		if err != nil {
			return err
		}
		sn(dst)
		return nil
	}
}

func (self *ReaderWriter) ReadTypeCode() (uint32, error) {
	if self.typeCodeRead {
		return self.typeCode, nil
	}
	if self.Size() >= 4 {
		self.mutex.Lock()
		defer self.mutex.Unlock()
		b := [4]byte{}
		_, err := self.InternalRead(b[:])
		if err != nil {
			return 0, err
		}
		self.typeCode = binary.LittleEndian.Uint32(b[:])
		self.typeCodeRead = true

		self.addBlockToFront(NewMultiBlockWithBlock(b[:]))
	}
	return self.typeCode, nil
}

func (self *ReaderWriter) ToString() string {
	sb := strings.Builder{}
	_, _ = io.Copy(&sb, self)
	return sb.String()

}

type nopWriter struct {
}

func (self *nopWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (self *ReaderWriter) Skip(toSkip int) error {
	w := &nopWriter{}
	_, err := io.CopyN(w, self, int64(toSkip))
	return err
}

//func (self *ReaderWriter) CopyFrom(reader io.Reader) (int, error) {
//
//	nn := 0
//	for {
//		block := make([]byte, maxBlockSize)
//		n, err := reader.Read(block)
//		if err != nil {
//			return 0, err
//		}
//		fmt.Println(block[:n])
//		time.Sleep(time.Second)
//		nn += n
//		if n == 0 {
//			break
//		}
//	}
//	return nn, nil
//}

func NewReaderWriter() *ReaderWriter {
	return NewReaderWriterSize(1024)
}
func NewReaderWriterString(s string) (*ReaderWriter, error) {
	rws := NewReaderWriterSize(1024)
	_, err := rws.Write([]byte(s))
	if err != nil {
		return nil, err
	}
	return rws, nil
}

var readerWriterUniqueNumber uint64

const maxBlockSize int = 64 * 1024

func NewReaderWriterSize(blockSize int) *ReaderWriter {
	if blockSize > maxBlockSize {
		blockSize = maxBlockSize
	}
	n := atomic.AddUint64(&readerWriterUniqueNumber, 1)
	return &ReaderWriter{
		uniqueNumber: n,
		blockSize:    blockSize,
		mutex:        sync.Mutex{},
		next:         nil,
		last:         nil,
	}
}

func NewReaderWriterBlock(block []byte) *ReaderWriter {
	return NewReaderWriterWithBlocks(block)
}

func NewReaderWriterWithBlocks(data ...[]byte) *ReaderWriter {
	result := NewReaderWriter()
	for _, item := range data {
		block := NewMultiBlockWithBlock(item)
		result.addBlock(block)
	}
	return result
}
