//go:build go1.18
// +build go1.18

package chanx

import (
	"context"
	"sync/atomic"
)

// UnboundedChanOf is an unbounded chan.
// In is used to write without blocking, which supports multiple writers.
// and Out is used to read, which supports multiple readers.
// You can close the in channel if you want.
type UnboundedChanOf[T any] struct {
	bufCount int64
	In       chan<- T         // channel for write
	Out      <-chan T         // channel for read
	buffer   *RingBufferOf[T] // buffer
}

// Len returns len of In plus len of Out plus len of buffer.
// It is not accurate and only for your evaluating approximate number of elements in this chan,
// see https://github.com/smallnest/chanx/issues/7.
func (c *UnboundedChanOf[T]) Len() int {
	return len(c.In) + c.BufLen() + len(c.Out)
}

// BufLen returns len of the buffer.
// It is not accurate and only for your evaluating approximate number of elements in this chan,
// see https://github.com/smallnest/chanx/issues/7.
func (c *UnboundedChanOf[T]) BufLen() int {
	return int(atomic.LoadInt64(&c.bufCount))
}

// BufCapacity returns capacity of the buffer.
func (c *UnboundedChanOf[T]) BufCapacity() int {
	return c.buffer.Capacity()
}

// MaxBufferSize returns maximum capacity of the buffer.
func (c *UnboundedChanOf[T]) MaxBufferSize() int {
	return c.buffer.MaxSize()
}

// Discards returns the number of discards.
func (c *UnboundedChanOf[T]) Discards() uint64 {
	return c.buffer.Discards()
}

// SetMaxBufferSize reset the maximum capacity of buffer
func (c *UnboundedChanOf[T]) SetMaxBufferSize(n int) int {
	return c.buffer.SetMaxSize(n)
}

// SetOnDiscards set the callback function when data is discarded
func (c *UnboundedChanOf[T]) SetOnDiscards(fn func(T)) {
	c.buffer.SetOnDiscards(fn)
}

// NewUnboundedChanOf creates the unbounded chan.
// in is used to write without blocking, which supports multiple writers.
// and out is used to read, which supports multiple readers.
// You can close the in channel if you want.
func NewUnboundedChanOf[T any](ctx context.Context, initCapacity int, maxBufferSize ...int) *UnboundedChanOf[T] {
	return NewUnboundedChanSizeOf[T](ctx, initCapacity, initCapacity, initCapacity, maxBufferSize...)
}

// NewUnboundedChanSizeOf is like NewUnboundedChanOf but you can set initial capacity for In, Out, Buffer.
// and max buffer capactiy.
func NewUnboundedChanSizeOf[T any](ctx context.Context, initInCapacity, initOutCapacity, initBufCapacity int, maxBufferSize ...int) *UnboundedChanOf[T] {
	in := make(chan T, initInCapacity)
	out := make(chan T, initOutCapacity)
	ch := UnboundedChanOf[T]{In: in, Out: out, buffer: NewRingBufferOf[T](initBufCapacity, maxBufferSize...)}

	go processOf(ctx, in, out, &ch)

	return &ch
}

func processOf[T any](ctx context.Context, in, out chan T, ch *UnboundedChanOf[T]) {
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
