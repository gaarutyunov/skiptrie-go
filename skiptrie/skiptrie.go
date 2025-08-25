package skiptrie

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	MaxKey  = (1 << 32) - 1 // u = 2^32 - 1 (max uint32)
	LogLogU = 5             // log log u = 5 for u = 2^32
)

// Node represents a skiplist node
type Node struct {
	key        uint32
	next       []*atomic.Pointer[Node] // next pointers for each level
	marked     atomic.Bool             // logical deletion flag
	origHeight int                     // original height of the node
}

// Key returns the key of the node
func (n *Node) Key() uint32 {
	return n.key
}

// SkipTrie is a simplified implementation for demonstration
type SkipTrie struct {
	head *Node      // sentinel head of skiplist
	tail *Node      // sentinel tail of skiplist
	rng  *rand.Rand // random number generator
	mu   sync.Mutex // mutex for RNG
}

// NewSkipTrie creates a new SkipTrie instance
func NewSkipTrie() *SkipTrie {
	st := &SkipTrie{
		rng: rand.New(rand.NewSource(rand.Int63())),
	}

	// Initialize sentinel nodes
	st.head = &Node{
		key:        0,
		next:       make([]*atomic.Pointer[Node], LogLogU),
		origHeight: LogLogU,
	}
	st.tail = &Node{
		key:        MaxKey,
		next:       make([]*atomic.Pointer[Node], LogLogU),
		origHeight: LogLogU,
	}

	// Initialize all levels to point from head to tail
	for i := 0; i < LogLogU; i++ {
		st.head.next[i] = &atomic.Pointer[Node]{}
		st.head.next[i].Store(st.tail)
		st.tail.next[i] = &atomic.Pointer[Node]{}
	}

	return st
}

// randomHeight generates a random height for a new node
func (st *SkipTrie) randomHeight() int {
	st.mu.Lock()
	defer st.mu.Unlock()

	height := 1
	for height < LogLogU && st.rng.Float32() < 0.5 {
		height++
	}
	return height
}

// Insert inserts a key into the SkipTrie
func (st *SkipTrie) Insert(key uint32) bool {
	// Find predecessors at each level
	preds := make([]*Node, LogLogU)
	succs := make([]*Node, LogLogU)

	// Search from top to bottom
	curr := st.head
	for level := LogLogU - 1; level >= 0; level-- {
		for {
			next := curr.next[level].Load()
			if next == nil || next.key >= key || next.marked.Load() {
				break
			}
			curr = next
		}

		preds[level] = curr
		succs[level] = curr.next[level].Load()

		// If key already exists, return false
		if succs[level] != nil && succs[level].key == key && !succs[level].marked.Load() {
			return false
		}
	}

	// Create new node with random height
	height := st.randomHeight()
	newNode := &Node{
		key:        key,
		next:       make([]*atomic.Pointer[Node], height),
		origHeight: height,
	}

	// Initialize atomic pointers
	for i := 0; i < height; i++ {
		newNode.next[i] = &atomic.Pointer[Node]{}
	}

	// Insert from bottom to top
	for level := 0; level < height; level++ {
		newNode.next[level].Store(succs[level])
		if !preds[level].next[level].CompareAndSwap(succs[level], newNode) {
			// If CAS failed, the structure changed, need to restart
			// For simplicity, we'll just fail the insertion
			return false
		}
	}

	return true
}

// Contains checks if a key exists in the SkipTrie
func (st *SkipTrie) Contains(key uint32) bool {
	curr := st.head

	// Search from top to bottom
	for level := LogLogU - 1; level >= 0; level-- {
		for {
			next := curr.next[level].Load()
			if next == nil || next.key > key || next.marked.Load() {
				break
			}
			if next.key == key {
				return true
			}
			curr = next
		}
	}

	return false
}

// Predecessor finds the predecessor of a key
func (st *SkipTrie) Predecessor(key uint32) *Node {
	curr := st.head
	var result *Node

	// Search from top to bottom
	for level := LogLogU - 1; level >= 0; level-- {
		for {
			next := curr.next[level].Load()
			if next == nil || next.key >= key || next.marked.Load() {
				break
			}
			curr = next
			if curr != st.head && curr.key < key {
				result = curr
			}
		}
	}

	return result
}

// Delete deletes a key from the SkipTrie
func (st *SkipTrie) Delete(key uint32) bool {
	// Find the node at level 0
	curr := st.head
	for {
		next := curr.next[0].Load()
		if next == nil || next.key > key {
			return false // Key not found
		}
		if next.key == key {
			// Mark the node as deleted
			if next.marked.CompareAndSwap(false, true) {
				return true
			}
			return false // Already deleted
		}
		curr = next
	}
}

// Helper functions for compatibility (even though not used in simplified version)
func cas(ptr **Node, old, new *Node) bool {
	return atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(ptr)),
		unsafe.Pointer(old),
		unsafe.Pointer(new),
	)
}

func dcss(target **Node, oldTarget, newTarget *Node, guard *atomic.Bool, guardValue bool) bool {
	if guard.Load() != guardValue {
		return false
	}
	return cas(target, oldTarget, newTarget)
}
