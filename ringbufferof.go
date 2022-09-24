//go:build go1.18
// +build go1.18

package chanx

// RingBufferOf is a ring buffer for common types.
// It never is full and always grows if it will be full.
// It is not thread-safe(goroutine-safe) so you must use the lock-like synchronization primitive
// to use it in multiple writers and multiple readers.
// Beyond maxBufSize, data is discarded.
type RingBufferOf[T any] struct {
	buf         []T
	initialSize int
	size        int
	maxBufSize  int
	discards    uint64
	r           int // read pointer
	w           int // write pointer
	onDiscards  func(T)
}

func NewRingBufferOf[T any](initialSize int, maxBufCapacity ...int) *RingBufferOf[T] {
	if initialSize <= 0 {
		panic("initial size must be great than zero")
	}
	// initial size must >= 2
	if initialSize == 1 {
		initialSize = 2
	}

	maxBufSize := 0
	if len(maxBufCapacity) > 0 {
		if maxBufCapacity[0] > 0 {
			maxBufSize = maxBufCapacity[0] + initialSize
		}
	}

	return &RingBufferOf[T]{
		buf:         make([]T, initialSize),
		initialSize: initialSize,
		size:        initialSize,
		maxBufSize:  maxBufSize,
	}
}

func (r *RingBufferOf[T]) Read() (T, error) {
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

func (r *RingBufferOf[T]) Pop() T {
	v, err := r.Read()
	if err == ErrIsEmpty { // Empty
		panic(ErrIsEmpty.Error())
	}

	return v
}

func (r *RingBufferOf[T]) Peek() T {
	if r.r == r.w { // Empty
		panic(ErrIsEmpty.Error())
	}

	v := r.buf[r.r]
	return v
}

func (r *RingBufferOf[T]) Write(v T) {
	if r.maxBufSize > 0 && r.Len() >= r.maxBufSize {
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

func (r *RingBufferOf[T]) grow() {
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

func (r *RingBufferOf[T]) IsEmpty() bool {
	return r.r == r.w
}

// Capacity returns the size of the underlying buffer.
func (r *RingBufferOf[T]) Capacity() int {
	return r.size
}

func (r *RingBufferOf[T]) MaxBufSize() int {
	return r.maxBufSize
}

func (r *RingBufferOf[T]) Discards() uint64 {
	return r.discards
}

func (r *RingBufferOf[T]) Len() int {
	if r.r == r.w {
		return 0
	}

	if r.w > r.r {
		return r.w - r.r
	}

	return r.size - r.r + r.w
}

func (r *RingBufferOf[T]) Reset() {
	r.r = 0
	r.w = 0
	r.size = r.initialSize
	r.buf = make([]T, r.initialSize)
}

func (r *RingBufferOf[T]) SetMaxCapacity(n int) int {
	if n == 0 {
		// Unbounded
		r.maxBufSize = 0
	} else if n > 0 {
		// Reset maximum limit
		r.maxBufSize = r.initialSize + n
	}

	return r.MaxBufSize()
}

func (r *RingBufferOf[T]) SetOnDiscards(fn func(T)) {
	if fn != nil {
		r.onDiscards = fn
	}
}
