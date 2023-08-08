package locking_pool

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetReturn(t *testing.T) {
	p := New([]int{1})
	obj := p.Get()
	assert.True(t, p.(*lockingPoolImpl[int]).objects[1])
	assert.Equal(t, 1, obj)
	p.Return(obj)
	assert.False(t, p.(*lockingPoolImpl[int]).objects[1])
}

func TestTryGet(t *testing.T) {
	p := New([]int{1})
	release := make(chan bool)
	released := make(chan bool)
	obtained := make(chan bool)
	go func() {
		obj := p.Get()
		close(obtained)
		<-release
		p.Return(obj)
		close(released)
	}()
	<-obtained
	// Try to get object. It is currently held by the goroutine
	ok, obj := p.TryGet()
	require.False(t, ok)
	// Release it from the goroutine
	close(release)
	<-released
	ok, obj = p.TryGet()
	assert.True(t, ok)
	assert.Equal(t, 1, obj)
	p.Return(obj)
}

func withTimeout(t *testing.T, timeout time.Duration, fn func(*testing.T, func())) {
	timer := time.NewTimer(timeout)
	done := make(chan bool)
	go fn(t, func() { close(done) })

	select {
	case <-timer.C:
		t.Fatalf("function did not return under %s", timeout)
	case <-done:
		return
	}
}

func TestConcurrentLocking(t *testing.T) {
	withTimeout(t, 5*time.Second, func(t *testing.T, done func()) {
		p := New([]int{1})
		obtained := make(chan bool)
		finished := make(chan bool)

		go func() {
			obj := p.Get()
			close(obtained)
			time.Sleep(100 * time.Millisecond)
			p.Return(obj)
		}()

		go func() {
			<-obtained
			// Try to get object. It is currently held by the goroutine
			obj := p.Get()
			assert.Equal(t, 1, obj)
			p.Return(obj)
			close(finished)
		}()

		<-finished
		done()
	})
}
