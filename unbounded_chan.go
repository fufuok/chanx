package chanx

import (
	"context"
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
// It is not accurate and only for your evaluating approximate number of elements in this chan,
// see https://github.com/smallnest/chanx/issues/7.
func (c UnboundedChan) Len() int {
	return len(c.In) + c.BufLen() + len(c.Out)
}

// BufLen returns len of the buffer.
// It is not accurate and only for your evaluating approximate number of elements in this chan,
// see https://github.com/smallnest/chanx/issues/7.
func (c UnboundedChan) BufLen() int {
	return int(atomic.LoadInt64(&c.bufCount))
}

// BufCapacity returns capacity of the buffer.
func (c UnboundedChan) BufCapacity() int {
	return c.buffer.Capacity()
}

// MaxBufferSize returns maximum capacity of the buffer.
func (c UnboundedChan) MaxBufferSize() int {
	return c.buffer.MaxSize()
}

// Discards returns the number of discards.
func (c UnboundedChan) Discards() uint64 {
	return c.buffer.Discards()
}

// SetMaxBufferSize reset the maximum capacity of buffer
func (c UnboundedChan) SetMaxBufferSize(n int) int {
	return c.buffer.SetMaxSize(n)
}

// SetOnDiscards set the callback function when data is discarded
func (c UnboundedChan) SetOnDiscards(fn func(interface{})) {
	c.buffer.SetOnDiscards(fn)
}

// NewUnboundedChan creates the unbounded chan.
// in is used to write without blocking, which supports multiple writers.
// and out is used to read, which supports multiple readers.
// You can close the in channel if you want.
func NewUnboundedChan(ctx context.Context, initCapacity int, maxBufferSize ...int) *UnboundedChan {
	return NewUnboundedChanSize(ctx, initCapacity, initCapacity, initCapacity, maxBufferSize...)
}

// NewUnboundedChanSize is like NewUnboundedChan but you can set initial capacity for In, Out, Buffer.
// and max buffer capactiy.
func NewUnboundedChanSize(ctx context.Context, initInCapacity, initOutCapacity, initBufCapacity int, maxBufferSize ...int) *UnboundedChan {
	in := make(chan T, initInCapacity)
	out := make(chan T, initOutCapacity)
	ch := UnboundedChan{In: in, Out: out, buffer: NewRingBuffer(initBufCapacity, maxBufferSize...)}

	go process(ctx, in, out, &ch)

	return &ch
}

func process(ctx context.Context, in, out chan T, ch *UnboundedChan) {
	defer close(out)
	drain := func() {
		for !ch.buffer.IsEmpty() {
			select {
			case out <- ch.buffer.Pop():
				atomic.AddInt64(&ch.bufCount, -1)
			case <-ctx.Done():
				return
			}
		}

		ch.buffer.Reset()
		atomic.StoreInt64(&ch.bufCount, 0)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case val, ok := <-in:
			if !ok { // in is closed
				drain()
				return
			}

			// make sure values' order
			// buffer has some values
			if atomic.LoadInt64(&ch.bufCount) > 0 {
				ch.buffer.Write(val)
				atomic.AddInt64(&ch.bufCount, 1)
			} else {
				// out is not full
				select {
				case out <- val:
					continue
				default:
				}

				// out is full
				ch.buffer.Write(val)
				atomic.AddInt64(&ch.bufCount, 1)
			}

			for !ch.buffer.IsEmpty() {
				select {
				case <-ctx.Done():
					return
				case val, ok := <-in:
					if !ok { // in is closed
						drain()
						return
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
	}
}
