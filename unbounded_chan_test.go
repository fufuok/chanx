package chanx

import (
	"sync"
	"testing"
	"time"
)

func TestMakeUnboundedChan(t *testing.T) {
	ch := NewUnboundedChan(100)

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
	ch := NewUnboundedChanSize(10, 50, 100)

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

func TestMakeUnboundedChanSizeMaxBuf(t *testing.T) {
	initInCapacity := 10
	initOutCapacity := 50
	initBufCapacity := 100
	maxBufCapacity := 10
	ch := NewUnboundedChanSize(initInCapacity, initOutCapacity, initBufCapacity, maxBufCapacity)

	for i := 0; i < 200; i++ {
		ch.In <- uint64(i)
		time.Sleep(time.Millisecond)
	}

	maxBufCap := ch.MaxBufSize()
	maxDiscards := uint64(200 - maxBufCap - initOutCapacity)
	discards := ch.Discards()

	if maxBufCap != (initBufCapacity + maxBufCapacity) {
		t.Fatalf("expected maxBufCapacity == 110 but got %d", maxBufCap)
	}

	if discards != maxDiscards {
		t.Fatalf("expected discards == %d but got %d", maxDiscards, discards)
	}

	// Restore unlimited
	ch.SetMaxCapacity(0)

	for i := 0; i < 200; i++ {
		ch.In <- uint64(i)
		time.Sleep(time.Millisecond)
	}

	newMaxBufCap := ch.MaxBufSize()
	newDiscards := ch.Discards()
	bufCapacity := ch.BufCapacity()

	if newMaxBufCap != 0 {
		t.Fatalf("expected maxBufCapacity == 0 but got %d", maxBufCap)
	}

	if newDiscards != discards {
		t.Fatalf("expected discards == %d but got %d", newDiscards, discards)
	}

	if bufCapacity != 400 {
		t.Fatalf("expected BufCapacity == 400 but got %d", bufCapacity)
	}
}

func TestMakeUnboundedChanSizeMaxBufCount(t *testing.T) {
	initInCapacity := 10
	initOutCapacity := 50
	initBufCapacity := 100
	maxBufCapacity := 10
	ch := NewUnboundedChanSize(initInCapacity, initOutCapacity, initBufCapacity, maxBufCapacity)

	var count uint64

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

	maxBufCap := ch.MaxBufSize()
	maxDiscards := uint64(1000 - maxBufCap - initOutCapacity)
	discards := ch.Discards()
	allCount := count + discards

	if maxBufCap != (initBufCapacity + maxBufCapacity) {
		t.Fatalf("expected maxBufCapacity == 110 but got %d", maxBufCap)
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
