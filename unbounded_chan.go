package chanx

import (
	"sync/atomic"
)

// T defines interface{}, and will be used for generic type after go 1.18 is released.
type T interface{}

// UnboundedChan is an unbounded chan.
// In is used to write without blocking, which supports multiple writers.
// and Out is used to read, which supports multiple readers.
// You can close the in channel if you want.
type UnboundedChan struct {
	bufCount int64
	In       chan<- T    // channel for write
	Out      <-chan T    // channel for read
	buffer   *RingBuffer // buffer
}

// Len returns len of In plus len of Out plus len of buffer.
func (c UnboundedChan) Len() int {
	return len(c.In) + c.BufLen() + len(c.Out)
}

// BufLen returns len of the buffer.
func (c UnboundedChan) BufLen() int {
	return int(atomic.LoadInt64(&c.bufCount))
}

// BufCapacity returns capacity of the buffer.
func (c UnboundedChan) BufCapacity() int {
	return c.buffer.Capacity()
}

// MaxBufSize returns maximum capacity of the buffer.
func (c UnboundedChan) MaxBufSize() int {
	return c.buffer.MaxBufSize()
}

// Discards returns the number of discards.
func (c UnboundedChan) Discards() uint64 {
	return c.buffer.Discards()
}

// SetMaxCapacity reset the maximum capacity of buffer
func (c UnboundedChan) SetMaxCapacity(n int) int {
	return c.buffer.SetMaxCapacity(n)
}

// NewUnboundedChan creates the unbounded chan.
// in is used to write without blocking, which supports multiple writers.
// and out is used to read, which supports multiple readers.
// You can close the in channel if you want.
func NewUnboundedChan(initCapacity int, maxBufCapacity ...int) *UnboundedChan {
	return NewUnboundedChanSize(initCapacity, initCapacity, initCapacity, maxBufCapacity...)
}

// NewUnboundedChanSize is like NewUnboundedChan but you can set initial capacity for In, Out, Buffer.
// and max buffer capactiy.
func NewUnboundedChanSize(initInCapacity, initOutCapacity, initBufCapacity int, maxBufCapacity ...int) *UnboundedChan {
	in := make(chan T, initInCapacity)
	out := make(chan T, initOutCapacity)
	ch := UnboundedChan{In: in, Out: out, buffer: NewRingBuffer(initBufCapacity, maxBufCapacity...)}

	go process(in, out, &ch)

	return &ch
}

func process(in, out chan T, ch *UnboundedChan) {
	defer close(out)
loop:
	for {
		val, ok := <-in
		if !ok { // in is closed
			break loop
		}

		// out is not full
		select {
		case out <- val:
			continue
		default:
		}

		// out is full
		ch.buffer.Write(val)
		atomic.AddInt64(&ch.bufCount, 1)
		for !ch.buffer.IsEmpty() {
			select {
			case val, ok := <-in:
				if !ok { // in is closed
					break loop
				}
				ch.buffer.Write(val)
				atomic.AddInt64(&ch.bufCount, 1)

			case out <- ch.buffer.Peek():
				ch.buffer.Pop()
				atomic.AddInt64(&ch.bufCount, -1)
				if ch.buffer.IsEmpty() && ch.buffer.size > ch.buffer.initialSize { // after burst
					ch.buffer.Reset()
					atomic.StoreInt64(&ch.bufCount, 0)
				}
			}
		}
	}

	// drain
	for !ch.buffer.IsEmpty() {
		out <- ch.buffer.Pop()
		atomic.AddInt64(&ch.bufCount, -1)
	}

	ch.buffer.Reset()
	atomic.StoreInt64(&ch.bufCount, 0)
}
