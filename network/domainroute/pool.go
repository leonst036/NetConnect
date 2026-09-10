package domainroute

import "sync"

const BufferSize = 32 * 1024

var bufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, BufferSize)
		return &b
	},
}

// GetBuffer gets a 32KB buffer from the pool.
func GetBuffer() *[]byte {
	return bufferPool.Get().(*[]byte)
}

// PutBuffer returns a buffer to the pool.
func PutBuffer(b *[]byte) {
	if b != nil && len(*b) == BufferSize {
		bufferPool.Put(b)
	}
}
