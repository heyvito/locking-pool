package locking_pool

import "sync"

// LockingPool provides a Pool implementation with support to wait for a
// resource to be returned.
type LockingPool[T comparable] interface {
	// Get attempts to obtain a resource T from the pool, blocking the caller
	// until one is returned, if the pool is empty.
	Get() T

	// TryGet attempts to obtain a resource T from the pool, returning
	// immediately in case the pool is empty. Returns a boolean value indicating
	// if the resource has been successfully acquired, and, if true, the
	// acquired resource itself.
	TryGet() (bool, T)

	// Return returns an obtained resource to the pool. Panics if a resource
	// that was not part of the pool is provided as v.
	Return(v T)
}

type lockingPoolImpl[T comparable] struct {
	mu      *sync.Mutex
	cond    *sync.Cond
	condMu  *sync.Mutex
	objects map[T]bool
	zeroT   T
}

func (l *lockingPoolImpl[T]) TryGet() (bool, T) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for obj, inUse := range l.objects {
		if !inUse {
			l.objects[obj] = true
			return true, obj
		}
	}

	return false, l.zeroT
}

func (l *lockingPoolImpl[T]) Get() T {
	for {
		ok, obj := l.TryGet()
		if ok {
			return obj
		}

		// No object is available. Wait for one to become available.
		l.condMu.Lock()
		l.cond.Wait()
		l.condMu.Unlock()
	}
}

func (l *lockingPoolImpl[T]) Return(v T) {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, hasObj := l.objects[v]
	if !hasObj {
		panic("attempt to return unowned object")
	}

	l.objects[v] = false
	l.cond.Signal()
}

// New returns a new LockingPool using the provided `items` as its content.
// Panics if items is nil or empty.
func New[T comparable](items []T) LockingPool[T] {
	if len(items) == 0 {
		panic("items cannot be empty or nil")
	}

	condMu := &sync.Mutex{}
	pool := &lockingPoolImpl[T]{
		mu:      &sync.Mutex{},
		cond:    sync.NewCond(condMu),
		condMu:  condMu,
		objects: make(map[T]bool),
	}
	for _, v := range items {
		pool.objects[v] = false
	}

	return pool
}
