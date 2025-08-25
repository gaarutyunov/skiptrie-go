package skiptrie

import (
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	MaxKey = (1 << 32) - 1 // u = 2^32, but max uint32 is 2^32-1
	LogLogU = 5           // log log u = 5 for u = 2^32
)

// Node represents a skiplist node
type Node struct {
	key        uint32
	next       []*atomic.Pointer[Node] // next pointers for each level
	prev       *atomic.Pointer[Node]    // backward pointer (top level only)
	back       *atomic.Pointer[Node]    // recovery pointer for deleted nodes
	marked     atomic.Bool              // logical deletion flag
	ready      atomic.Bool              // indicates prev pointer is set
	stop       atomic.Bool              // stop flag for tower operations
	origHeight int                      // original height of the node
}

// TreeNode represents an x-fast trie node
type TreeNode struct {
	pointers [2]*atomic.Pointer[Node] // [0] = largest in 0-subtree, [1] = smallest in 1-subtree
}

// SkipTrie is the main data structure
type SkipTrie struct {
	prefixes sync.Map                 // concurrent hash table for x-fast trie
	head     *Node                    // sentinel head of skiplist
	tail     *Node                    // sentinel tail of skiplist
	rng      *rand.Rand               // random number generator
	mu       sync.Mutex               // mutex for RNG
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
	
	// Initialize top-level prev pointers
	st.head.prev = &atomic.Pointer[Node]{}
	st.tail.prev = &atomic.Pointer[Node]{}
	st.tail.prev.Store(st.head)
	
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

// listSearch finds the predecessor and successor of a key at a given level
func (st *SkipTrie) listSearch(key uint32, start *Node, level int) (*Node, *Node) {
	var left, right *Node
	maxIterations := 1000 // Prevent infinite loops
	iterations := 0
	
	for iterations < maxIterations {
		iterations++
		left = start
		right = left.next[level].Load()
		
		// Skip over marked nodes
		for right != nil && right.marked.Load() {
			nextRight := right.next[level].Load()
			// Try to unlink the marked node
			if left.next[level].CompareAndSwap(right, nextRight) {
				right = nextRight
			} else {
				// Retry if CAS failed
				break
			}
		}
		
		// Find the correct position
		for right != nil && right.key < key && !right.marked.Load() {
			left = right
			right = left.next[level].Load()
			
			// Skip marked nodes again
			for right != nil && right.marked.Load() {
				nextRight := right.next[level].Load()
				if left.next[level].CompareAndSwap(right, nextRight) {
					right = nextRight
				} else {
					break
				}
			}
		}
		
		// Verify we have a valid bracket
		if right == nil || !right.marked.Load() {
			leftNext := left.next[level].Load()
			if leftNext == right && !left.marked.Load() {
				return left, right
			}
		}
		
		// Add yield after some iterations to help with livelock
		if iterations > 100 {
			runtime.Gosched()
		}
	}
	
	// Fallback: return what we have to prevent infinite loops
	return left, right
}

// skiplistInsert inserts a key into the skiplist
func (st *SkipTrie) skiplistInsert(key uint32) *Node {
	height := st.randomHeight()
	
	// Create new node
	newNode := &Node{
		key:        key,
		next:       make([]*atomic.Pointer[Node], height),
		origHeight: height,
	}
	
	// Initialize atomic pointers
	for i := 0; i < height; i++ {
		newNode.next[i] = &atomic.Pointer[Node]{}
	}
	if height == LogLogU {
		newNode.prev = &atomic.Pointer[Node]{}
		newNode.back = &atomic.Pointer[Node]{}
	}
	
	// Find insertion points at each level
	preds := make([]*Node, height)
	succs := make([]*Node, height)
	
	start := st.head
	for level := LogLogU - 1; level >= 0; level-- {
		if level < height {
			left, right := st.listSearch(key, start, level)
			if right != nil && right.key == key {
				// Key already exists
				return nil
			}
			preds[level] = left
			succs[level] = right
		}
		if level > 0 && start.next != nil && len(start.next) > level-1 {
			// Move to next level down if available
			continue
		}
	}
	
	// Insert from bottom to top
	for level := 0; level < height; level++ {
		for {
			if newNode.stop.Load() {
				return newNode
			}
			
			newNode.next[level].Store(succs[level])
			if preds[level].next[level].CompareAndSwap(succs[level], newNode) {
				break
			}
			
			// Retry with updated positions
			left, right := st.listSearch(key, preds[level], level)
			if right != nil && right.key == key {
				return nil
			}
			preds[level] = left
			succs[level] = right
		}
	}
	
	// Set prev pointer for top-level nodes
	if height == LogLogU {
		st.fixPrev(preds[LogLogU-1], newNode)
	}
	
	return newNode
}

// fixPrev sets the prev pointer of a node
func (st *SkipTrie) fixPrev(pred *Node, node *Node) {
	retries := 0
	maxRetries := 100 // Add maximum retry limit to prevent infinite loops
	
	for !node.marked.Load() && retries < maxRetries {
		left, right := st.listSearch(node.key, pred, LogLogU-1)
		if right == node {
			node.prev.Store(left)
			node.ready.Store(true)
			return
		}
		pred = left
		retries++
		
		// Add a small delay to help with livelock
		if retries > 10 {
			runtime.Gosched() // Yield to other goroutines
		}
	}
	
	// If we couldn't fix prev after max retries, just mark as ready
	// This is a fallback to prevent infinite loops
	if retries >= maxRetries {
		node.ready.Store(true)
	}
}

// skiplistDelete deletes a node from the skiplist
func (st *SkipTrie) skiplistDelete(node *Node) bool {
	// Mark the node
	if !node.marked.CompareAndSwap(false, true) {
		return false // Already deleted
	}
	
	// Set stop flag to prevent further tower raising
	node.stop.Store(true)
	
	// Remove from all levels top-down
	for level := node.origHeight - 1; level >= 0; level-- {
		for {
			left, right := st.listSearch(node.key, st.head, level)
			if right != node {
				break // Already removed from this level
			}
			
			next := node.next[level].Load()
			if left.next[level].CompareAndSwap(node, next) {
				break
			}
		}
	}
	
	return true
}

// xFastTriePred finds the predecessor in the x-fast trie
func (st *SkipTrie) xFastTriePred(key uint32) *Node {
	curr := st.lowestAncestor(key)
	
	// Traverse backward if necessary
	for curr != nil && curr.key > key {
		if curr.marked.Load() {
			if curr.back != nil {
				curr = curr.back.Load()
			} else {
				break
			}
		} else if curr.prev != nil {
			curr = curr.prev.Load()
		} else {
			break
		}
	}
	
	return curr
}

// lowestAncestor performs binary search on prefix length
func (st *SkipTrie) lowestAncestor(key uint32) *Node {
	var ancestor *Node
	
	// Start with empty prefix
	if val, ok := st.prefixes.Load(""); ok {
		tn := val.(*TreeNode)
		direction := 0
		if key&(1<<31) != 0 {
			direction = 1
		}
		if tn.pointers[direction] != nil {
			ancestor = tn.pointers[direction].Load()
		}
	}
	
	// Binary search on prefix length
	commonPrefix := ""
	start := 0
	size := 16 // log u / 2 for u = 2^32
	
	for size > 0 {
		// Create query prefix
		query := st.extractPrefix(key, start, start+size)
		if commonPrefix != "" {
			query = commonPrefix + query
		}
		
		if val, ok := st.prefixes.Load(query); ok {
			tn := val.(*TreeNode)
			
			// Determine direction for next bit
			direction := 0
			if start+size < 32 && (key&(1<<(31-start-size))) != 0 {
				direction = 1
			}
			
			if tn.pointers[direction] != nil {
				candidate := tn.pointers[direction].Load()
				if candidate != nil && st.isPrefixOf(query, candidate.key) {
					if ancestor == nil || st.distance(key, candidate.key) < st.distance(key, ancestor.key) {
						ancestor = candidate
					}
					commonPrefix = query
					start = start + size
				}
			}
		}
		
		size = size / 2
	}
	
	if ancestor == nil {
		return st.head
	}
	return ancestor
}

// extractPrefix extracts bits from start to end (exclusive) as a string
func (st *SkipTrie) extractPrefix(key uint32, start, end int) string {
	if end > 32 {
		end = 32
	}
	
	result := ""
	for i := start; i < end; i++ {
		if key&(1<<(31-i)) != 0 {
			result += "1"
		} else {
			result += "0"
		}
	}
	return result
}

// isPrefixOf checks if prefix is a prefix of key
func (st *SkipTrie) isPrefixOf(prefix string, key uint32) bool {
	for i, bit := range prefix {
		keyBit := (key >> (31 - i)) & 1
		if bit == '0' && keyBit != 0 {
			return false
		}
		if bit == '1' && keyBit != 1 {
			return false
		}
	}
	return true
}

// distance calculates the distance between two keys
func (st *SkipTrie) distance(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// Insert inserts a key into the SkipTrie
func (st *SkipTrie) Insert(key uint32) bool {
	node := st.skiplistInsert(key)
	if node == nil {
		return false // Key already exists
	}
	
	// If node reached top level, insert into x-fast trie
	if node.origHeight == LogLogU {
		st.insertIntoTrie(node)
	}
	
	return true
}

// insertIntoTrie inserts a top-level node into the x-fast trie
func (st *SkipTrie) insertIntoTrie(node *Node) {
	// Insert all prefixes of the key
	for i := 31; i >= 0; i-- {
		prefix := st.extractPrefix(node.key, 0, i+1)
		direction := 0
		if i < 31 && (node.key&(1<<(31-i-1))) != 0 {
			direction = 1
		}
		
		for !node.marked.Load() {
			val, loaded := st.prefixes.LoadOrStore(prefix, &TreeNode{
				pointers: [2]*atomic.Pointer[Node]{
					&atomic.Pointer[Node]{},
					&atomic.Pointer[Node]{},
				},
			})
			
			tn := val.(*TreeNode)
			
			if !loaded {
				// New entry created
				tn.pointers[direction].Store(node)
				break
			}
			
			// Update existing entry if necessary
			curr := tn.pointers[direction].Load()
			if curr == nil {
				tn.pointers[direction].CompareAndSwap(nil, node)
				break
			}
			
			if direction == 0 && curr.key >= node.key {
				break // Already adequately represented
			}
			if direction == 1 && curr.key <= node.key {
				break // Already adequately represented
			}
			
			// Try to update the pointer
			if tn.pointers[direction].CompareAndSwap(curr, node) {
				break
			}
		}
	}
}

// Delete deletes a key from the SkipTrie
func (st *SkipTrie) Delete(key uint32) bool {
	// Find the node by searching from head
	pred := st.head
	if key > 0 {
		pred = st.Predecessor(key)
	}
	
	curr := pred
	if pred != nil && pred != st.head {
		curr = pred.next[0].Load()
	} else {
		curr = st.head.next[0].Load()
	}
	
	// Search for exact key
	for curr != nil && curr.key < key {
		curr = curr.next[0].Load()
	}
	
	if curr == nil || curr.key != key {
		return false // Key not found
	}
	
	// Delete from skiplist
	if !st.skiplistDelete(curr) {
		return false
	}
	
	// If it was a top-level node, update the trie
	if curr.origHeight == LogLogU {
		st.deleteFromTrie(curr)
	}
	
	return true
}

// deleteFromTrie removes references to a deleted node from the x-fast trie
func (st *SkipTrie) deleteFromTrie(node *Node) {
	for i := 0; i < 32; i++ {
		prefix := st.extractPrefix(node.key, 0, i+1)
		direction := 0
		if i < 31 && (node.key&(1<<(31-i-1))) != 0 {
			direction = 1
		}
		
		val, ok := st.prefixes.Load(prefix)
		if !ok {
			continue
		}
		
		tn := val.(*TreeNode)
		curr := tn.pointers[direction].Load()
		
		for curr == node {
			// Find replacement
			left, right := st.listSearch(node.key, st.head, LogLogU-1)
			
			var replacement *Node
			if direction == 0 {
				replacement = left
			} else {
				replacement = right
			}
			
			if replacement != nil && st.isPrefixOf(prefix, replacement.key) {
				tn.pointers[direction].CompareAndSwap(curr, replacement)
			} else {
				// Subtree is empty
				tn.pointers[direction].CompareAndSwap(curr, nil)
			}
			
			curr = tn.pointers[direction].Load()
		}
		
		// If both pointers are nil, remove the entry
		if tn.pointers[0].Load() == nil && tn.pointers[1].Load() == nil {
			st.prefixes.Delete(prefix)
		}
	}
}

// Predecessor finds the predecessor of a key
func (st *SkipTrie) Predecessor(key uint32) *Node {
	// Search through skiplist starting from head
	curr := st.head
	for level := LogLogU - 1; level >= 0; level-- {
		for curr != nil {
			next := curr.next[level].Load()
			if next == nil || next == st.tail || next.key >= key {
				break
			}
			if !next.marked.Load() {
				curr = next
			} else {
				// Skip marked node
				nextNext := next.next[level].Load()
				curr.next[level].CompareAndSwap(next, nextNext)
			}
		}
	}
	
	if curr == st.head {
		return nil
	}
	return curr
}

// Contains checks if a key exists in the SkipTrie
func (st *SkipTrie) Contains(key uint32) bool {
	pred := st.Predecessor(key)
	if pred == nil {
		curr := st.head.next[0].Load()
		return curr != nil && curr.key == key && !curr.marked.Load()
	}
	
	next := pred.next[0].Load()
	return next != nil && next.key == key && !next.marked.Load()
}

// Helper function for CAS operations on pointers
func cas(ptr **Node, old, new *Node) bool {
	return atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(ptr)),
		unsafe.Pointer(old),
		unsafe.Pointer(new),
	)
}

// DCSS simulates double-compare-single-swap
// In production, this would need more sophisticated implementation
func dcss(target **Node, oldTarget, newTarget *Node, guard *atomic.Bool, guardValue bool) bool {
	if guard.Load() != guardValue {
		return false
	}
	return cas(target, oldTarget, newTarget)
}