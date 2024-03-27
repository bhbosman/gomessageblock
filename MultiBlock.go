package gomessageblock

import (
	"bytes"
)

type MultiBlock struct {
	bytes bytes.Buffer
	next  *MultiBlock
}

func (self *MultiBlock) clearNext() {
	self.next = nil
}

func (self *MultiBlock) nextMultiBlock() *MultiBlock {
	return self.next
}

func (self *MultiBlock) Read(p []byte) (n int, err error) {
	return self.bytes.Read(p)
}

func (self *MultiBlock) Write(p []byte) (n int, err error) {
	return self.bytes.Write(p)
}

func NewMultiBlockSize(n int) *MultiBlock {
	result := &MultiBlock{
		bytes: bytes.Buffer{},
	}
	result.bytes.Grow(n)
	return result
}

func NewMultiBlockWithBlock(b []byte) *MultiBlock {
	return &MultiBlock{
		bytes: *bytes.NewBuffer(b),
	}
}

func NewMultiBlock() *MultiBlock {
	result := &MultiBlock{
		bytes: bytes.Buffer{},
	}
	return result
}
