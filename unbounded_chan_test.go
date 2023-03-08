package chanx

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMakeUnboundedChan(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := NewUnboundedChan(ctx, 100)

	for i := 1; i < 200; i++ {
		ch.In <- int64(i)
	}

	var count int64
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for v := range ch.Out {
			count += v.(int64)
		}
	}()

	for i := 200; i <= 1000; i++ {
		ch.In <- int64(i)
	}
	close(ch.In)

	wg.Wait()

	if count != 500500 {
		t.Fatalf("expected 500500 but got %d", count)
	}
}

func TestMakeUnboundedChanSize(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := NewUnboundedChanSize(ctx, 10, 50, 100)

	for i := 1; i < 200; i++ {
		ch.In <- int64(i)
	}

	var count int64
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for v := range ch.Out {
			count += v.(int64)
		}
	}()

	for i := 200; i <= 1000; i++ {
		ch.In <- int64(i)
	}
	close(ch.In)

	wg.Wait()

	if count != 500500 {
		t.Fatalf("expected 500500 but got %d", count)
	}
}

func TestUnboundedChan_DataRace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := NewUnboundedChan(ctx, 1)
	stop := make(chan bool)
	wg := &sync.WaitGroup{}
	for i := 0; i < 100; i++ { // may tweak the number of iterations
		wg.Add(1)
		go func(ctx context.Context) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-stop:
					return
				default:
					ch.In <- 42
					<-ch.Out
				}
			}
		}(ctx)
	}

	for i := 0; i < 10000; i++ { // may tweak the number of iterations
		ch.Len()
	}
	close(stop)
	wg.Wait()
}

func TestUnbounded_GetDataWithGoleak(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := NewUnboundedChanSize(ctx, 10, 50, 100)
	for i := 1; i < 200; i++ {
		ch.In <- int64(i)
	}
}

func TestUnboundedChanLen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := NewUnboundedChanSize(ctx, 10, 50, 100)

	for i := 1; i < 200; i++ {
		ch.In <- int64(i)
	}

	// wait ch processing in normal case
	time.Sleep(time.Second)
	assert.Equal(t, 0, len(ch.In))
	assert.Equal(t, 50, len(ch.Out))
	assert.Equal(t, 199, ch.Len())
	assert.Equal(t, 149, ch.BufLen())

	for i := 0; i < 50; i++ {
		<-ch.Out
	}

	time.Sleep(time.Second)
	assert.Equal(t, 0, len(ch.In))
	assert.Equal(t, 50, len(ch.Out))
	assert.Equal(t, 149, ch.Len())
	assert.Equal(t, 99, ch.BufLen())

	for i := 0; i < 149; i++ {
		<-ch.Out
	}

	time.Sleep(time.Second)
	assert.Equal(t, 0, len(ch.In))
	assert.Equal(t, 0, len(ch.Out))
	assert.Equal(t, 0, ch.Len())
	assert.Equal(t, 0, ch.BufLen())
}

func TestMakeUnboundedChanSizeMaxBuf(t *testing.T) {
	initInCapacity := 10
	initOutCapacity := 50
	initBufCapacity := 100
	maxBufCapacity := 110
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := NewUnboundedChanSize(ctx, initInCapacity, initOutCapacity, initBufCapacity, maxBufCapacity)

	for i := 0; i < 200; i++ {
		ch.In <- uint64(i)
		time.Sleep(time.Millisecond)
	}

	maxBufferSize := ch.MaxBufferSize()
	maxDiscards := uint64(200 - maxBufferSize - initOutCapacity)
	discards := ch.Discards()

	if maxBufferSize != maxBufCapacity {
		t.Fatalf("expected maxBufCapacity == 110 but got %d", maxBufferSize)
	}

	if discards != maxDiscards {
		t.Fatalf("expected discards == %d but got %d", maxDiscards, discards)
	}

	// Restore unlimited
	ch.SetMaxBufferSize(0)

	for i := 0; i < 200; i++ {
		ch.In <- uint64(i)
		time.Sleep(time.Millisecond)
	}

	newMaxBufCap := ch.MaxBufferSize()
	newDiscards := ch.Discards()
	bufCapacity := ch.BufCapacity()

	if newMaxBufCap != 0 {
		t.Fatalf("expected maxBufCapacity == 0 but got %d", maxBufferSize)
	}

	if newDiscards != discards {
		t.Fatalf("expected discards == %d but got %d", newDiscards, discards)
	}

	if bufCapacity != 400 {
		t.Fatalf("expected BufCapacity == 400 but got %d", bufCapacity)
	}
}

func TestMakeUnboundedChanSizeMaxBufCount(t *testing.T) {
	var (
		initInCapacity        = 10
		initOutCapacity       = 50
		initBufCapacity       = 100
		maxBufCapacity        = 110
		count                 uint64
		callbackDiscardsCount uint64
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := NewUnboundedChanSize(ctx, initInCapacity, initOutCapacity, initBufCapacity, maxBufCapacity)
	ch.SetOnDiscards(func(v interface{}) {
		callbackDiscardsCount++
	})

	for i := 1; i < 200; i++ {
		ch.In <- uint64(i)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for range ch.Out {
			count += 1
		}
	}()

	for i := 200; i <= 1000; i++ {
		ch.In <- uint64(i)
	}
	close(ch.In)
	wg.Wait()

	maxBufferSize := ch.MaxBufferSize()
	maxDiscards := uint64(1000 - maxBufferSize - initOutCapacity)
	discards := ch.Discards()
	allCount := count + discards

	if maxBufferSize != maxBufCapacity {
		t.Fatalf("expected maxBufCapacity == 110 but got %d", maxBufferSize)
	}

	if discards < 1 {
		t.Fatalf("expected discards > 0 but got %d", discards)
	}

	if discards > maxDiscards {
		t.Fatalf("expected discards <= %d but got %d", maxDiscards, discards)
	}

	if count == 1000 {
		t.Fatalf("expected count < 1000 but got %d", count)
	}

	if allCount != 1000 {
		t.Fatalf("expected 1000 but got %d", allCount)
	}
}
