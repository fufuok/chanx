package chanx

import (
	"errors"
)

const minBufferSize = 2

var ErrIsEmpty = errors.New("ringbuffer is empty")

// RingBuffer is a ring buffer for common types.
// It is never full and always grows if it will be full.
// It is not thread-safe(goroutine-safe) so you must use the lock-like synchronization primitive
// to use it in multiple writers and multiple readers.
// Exceeding maxSize, data will be discarded.
type RingBuffer[T any] struct {
	buf         []T
	initialSize int
	size        int
	maxSize     int
	discards    uint64
	r           int // read pointer
	w           int // write pointer
	onDiscards  func(T)
}

func NewRingBuffer[T any](initialSize int, maxBufferSize ...int) *RingBuffer[T] {
	if initialSize < minBufferSize {
		initialSize = minBufferSize
	}

	maxSize := 0
	if len(maxBufferSize) > 0 && maxBufferSize[0] >= minBufferSize {
		maxSize = maxBufferSize[0]
	}

	return &RingBuffer[T]{
		buf:         make([]T, initialSize),
		initialSize: initialSize,
		size:        initialSize,
		maxSize:     maxSize,
	}
}

func (r *RingBuffer[T]) Read() (T, error) {
	var t T
	if r.r == r.w {
		return t, ErrIsEmpty
	}

	v := r.buf[r.r]
	r.r++
	if r.r == r.size {
		r.r = 0
	}

	return v, nil
}

func (r *RingBuffer[T]) Pop() T {
	v, err := r.Read()
	if errors.Is(err, ErrIsEmpty) { // Empty
		panic(ErrIsEmpty.Error())
	}

	return v
}

func (r *RingBuffer[T]) Peek() T {
	if r.r == r.w { // Empty
		panic(ErrIsEmpty.Error())
	}

	v := r.buf[r.r]
	return v
}

func (r *RingBuffer[T]) Write(v T) {
	if r.maxSize > 0 && r.Len() >= r.maxSize {
		r.discards++
		if r.onDiscards != nil {
			r.onDiscards(v)
		}
		return
	}

	r.buf[r.w] = v
	r.w++

	if r.w == r.size {
		r.w = 0
	}

	if r.w == r.r { // full
		r.grow()
	}
}

func (r *RingBuffer[T]) grow() {
	var size int
	if r.size < 1024 {
		size = r.size * 2
	} else {
		size = r.size + r.size/4
	}

	buf := make([]T, size)

	copy(buf[0:], r.buf[r.r:])
	copy(buf[r.size-r.r:], r.buf[0:r.r])

	r.r = 0
	r.w = r.size
	r.size = size
	r.buf = buf
}

func (r *RingBuffer[T]) IsEmpty() bool {
	return r.r == r.w
}

// Capacity returns the size of the underlying buffer.
func (r *RingBuffer[T]) Capacity() int {
	return r.size
}

func (r *RingBuffer[T]) MaxSize() int {
	return r.maxSize
}

func (r *RingBuffer[T]) Discards() uint64 {
	return r.discards
}

func (r *RingBuffer[T]) Len() int {
	if r.r == r.w {
		return 0
	}

	if r.w > r.r {
		return r.w - r.r
	}

	return r.size - r.r + r.w
}

func (r *RingBuffer[T]) Reset() {
	r.r = 0
	r.w = 0
	r.size = r.initialSize
	r.buf = make([]T, r.initialSize)
}

func (r *RingBuffer[T]) SetMaxSize(n int) int {
	if n == 0 {
		// Unbounded
		r.maxSize = 0
	} else if n >= minBufferSize {
		// Reset maximum limit
		r.maxSize = n
	}

	return r.maxSize
}

func (r *RingBuffer[T]) SetOnDiscards(fn func(T)) {
	if fn != nil {
		r.onDiscards = fn
	}
}
