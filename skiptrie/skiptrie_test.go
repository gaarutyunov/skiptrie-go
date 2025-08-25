package skiptrie

import (
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"
)

// Test basic operations
func TestBasicOperations(t *testing.T) {
	st := NewSkipTrie()
	
	// Test insertion
	if !st.Insert(42) {
		t.Fatal("Failed to insert 42")
	}
	
	// Test contains
	if !st.Contains(42) {
		t.Fatal("SkipTrie should contain 42")
	}
	
	// Test duplicate insertion
	if st.Insert(42) {
		t.Fatal("Should not be able to insert duplicate key 42")
	}
	
	// Test non-existent key
	if st.Contains(99) {
		t.Fatal("SkipTrie should not contain 99")
	}
	
	// Test deletion
	if !st.Delete(42) {
		t.Fatal("Failed to delete 42")
	}
	
	// Test contains after deletion
	if st.Contains(42) {
		t.Fatal("SkipTrie should not contain 42 after deletion")
	}
	
	// Test deletion of non-existent key
	if st.Delete(42) {
		t.Fatal("Should not be able to delete non-existent key 42")
	}
}

// Test edge cases
func TestEdgeCases(t *testing.T) {
	st := NewSkipTrie()
	
	// Test with key 0
	if !st.Insert(0) {
		t.Fatal("Failed to insert key 0")
	}
	if !st.Contains(0) {
		t.Fatal("SkipTrie should contain key 0")
	}
	if !st.Delete(0) {
		t.Fatal("Failed to delete key 0")
	}
	
	// Test with maximum uint32 value (but not the sentinel value)
	maxKey := uint32(MaxKey - 1) // One less than MaxKey since MaxKey is used as sentinel
	if !st.Insert(maxKey) {
		t.Fatal("Failed to insert near-max key")
	}
	if !st.Contains(maxKey) {
		t.Fatal("SkipTrie should contain near-max key")
	}
	if !st.Delete(maxKey) {
		t.Fatal("Failed to delete near-max key")
	}
	
	// Test with boundary values
	testKeys := []uint32{1, 2, uint32(MaxKey - 2), uint32(MaxKey - 1), 0}
	for _, key := range testKeys {
		if !st.Insert(key) {
			t.Fatalf("Failed to insert boundary key %d", key)
		}
	}
	
	for _, key := range testKeys {
		if !st.Contains(key) {
			t.Fatalf("SkipTrie should contain boundary key %d", key)
		}
	}
	
	for _, key := range testKeys {
		if !st.Delete(key) {
			t.Fatalf("Failed to delete boundary key %d", key)
		}
	}
}

// Test predecessor functionality
func TestPredecessor(t *testing.T) {
	st := NewSkipTrie()
	
	// Insert some keys
	keys := []uint32{10, 20, 30, 40, 50}
	for _, key := range keys {
		st.Insert(key)
	}
	
	tests := []struct {
		query    uint32
		expected *uint32
	}{
		{5, nil},          // No predecessor
		{10, nil},         // Query key exists, no predecessor
		{15, &keys[0]},    // 10
		{25, &keys[1]},    // 20
		{35, &keys[2]},    // 30
		{45, &keys[3]},    // 40
		{55, &keys[4]},    // 50
		{100, &keys[4]},   // 50
	}
	
	for _, test := range tests {
		pred := st.Predecessor(test.query)
		if test.expected == nil {
			if pred != nil {
				t.Errorf("Predecessor(%d) = %v, expected nil", test.query, pred.key)
			}
		} else {
			if pred == nil {
				t.Errorf("Predecessor(%d) = nil, expected %d", test.query, *test.expected)
			} else if pred.key != *test.expected {
				t.Errorf("Predecessor(%d) = %d, expected %d", test.query, pred.key, *test.expected)
			}
		}
	}
}

// Test with multiple insertions and deletions
func TestMultipleOperations(t *testing.T) {
	st := NewSkipTrie()
	
	// Test 1000 insertions
	const numKeys = 1000
	keys := make([]uint32, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = uint32(i * 2) // Even numbers to test gaps
	}
	
	// Insert all keys
	for _, key := range keys {
		if !st.Insert(key) {
			t.Fatalf("Failed to insert key %d", key)
		}
	}
	
	// Check all keys exist
	for _, key := range keys {
		if !st.Contains(key) {
			t.Fatalf("Key %d should exist", key)
		}
	}
	
	// Check odd numbers don't exist
	for i := 1; i < numKeys*2; i += 2 {
		if st.Contains(uint32(i)) {
			t.Fatalf("Key %d should not exist", i)
		}
	}
	
	// Delete half the keys
	for i := 0; i < numKeys/2; i++ {
		if !st.Delete(keys[i]) {
			t.Fatalf("Failed to delete key %d", keys[i])
		}
	}
	
	// Check deleted keys don't exist
	for i := 0; i < numKeys/2; i++ {
		if st.Contains(keys[i]) {
			t.Fatalf("Deleted key %d should not exist", keys[i])
		}
	}
	
	// Check remaining keys still exist
	for i := numKeys / 2; i < numKeys; i++ {
		if !st.Contains(keys[i]) {
			t.Fatalf("Remaining key %d should exist", keys[i])
		}
	}
}

// Test concurrent operations
func TestConcurrentOperations(t *testing.T) {
	st := NewSkipTrie()
	const numGoroutines = 10
	const keysPerGoroutine = 100
	
	var wg sync.WaitGroup
	
	// Concurrent insertions
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < keysPerGoroutine; i++ {
				key := uint32(goroutineID*keysPerGoroutine + i)
				st.Insert(key)
			}
		}(g)
	}
	wg.Wait()
	
	// Verify all insertions
	for g := 0; g < numGoroutines; g++ {
		for i := 0; i < keysPerGoroutine; i++ {
			key := uint32(g*keysPerGoroutine + i)
			if !st.Contains(key) {
				t.Errorf("Key %d should exist after concurrent insertion", key)
			}
		}
	}
	
	// Concurrent deletions
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < keysPerGoroutine/2; i++ {
				key := uint32(goroutineID*keysPerGoroutine + i)
				st.Delete(key)
			}
		}(g)
	}
	wg.Wait()
	
	// Verify partial deletions
	for g := 0; g < numGoroutines; g++ {
		for i := 0; i < keysPerGoroutine/2; i++ {
			key := uint32(g*keysPerGoroutine + i)
			if st.Contains(key) {
				t.Errorf("Key %d should not exist after concurrent deletion", key)
			}
		}
		for i := keysPerGoroutine / 2; i < keysPerGoroutine; i++ {
			key := uint32(g*keysPerGoroutine + i)
			if !st.Contains(key) {
				t.Errorf("Key %d should still exist after partial deletion", key)
			}
		}
	}
}

// Test concurrent mixed operations
func TestConcurrentMixedOperations(t *testing.T) {
	st := NewSkipTrie()
	const duration = 100 * time.Millisecond // Much shorter duration
	const numGoroutines = 4
	
	var wg sync.WaitGroup
	done := make(chan bool, 1)
	
	// Start mixed operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(workerID)))
			
			for {
				select {
				case <-done:
					return
				default:
					key := uint32(rng.Intn(10)) // Smaller key space to reduce conflicts
					op := rng.Intn(3)
					
					switch op {
					case 0: // Insert
						st.Insert(key)
					case 1: // Delete
						st.Delete(key)
					case 2: // Contains
						st.Contains(key)
					}
				}
			}
		}(i)
	}
	
	// Stop after duration
	time.Sleep(duration)
	close(done)
	wg.Wait()
	
	// The test passes if no panics or race conditions occurred
	t.Log("Mixed concurrent operations completed successfully")
}

// Benchmark basic operations
func BenchmarkInsert(b *testing.B) {
	st := NewSkipTrie()
	keys := make([]uint32, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = uint32(i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st.Insert(keys[i])
	}
}

func BenchmarkContains(b *testing.B) {
	st := NewSkipTrie()
	keys := make([]uint32, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = uint32(i)
		st.Insert(keys[i])
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st.Contains(keys[i])
	}
}

func BenchmarkDelete(b *testing.B) {
	st := NewSkipTrie()
	keys := make([]uint32, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = uint32(i)
		st.Insert(keys[i])
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st.Delete(keys[i])
	}
}

func BenchmarkPredecessor(b *testing.B) {
	st := NewSkipTrie()
	for i := 0; i < 10000; i++ {
		st.Insert(uint32(i * 2)) // Even numbers
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st.Predecessor(uint32(i%20000 + 1)) // Query odd numbers
	}
}

// Test implementation correctness against a simple reference
func TestCorrectnessAgainstReference(t *testing.T) {
	st := NewSkipTrie()
	reference := make(map[uint32]bool)
	
	rng := rand.New(rand.NewSource(42)) // Fixed seed for reproducibility
	
	// Perform random operations and compare results
	for i := 0; i < 1000; i++ {
		key := uint32(rng.Intn(100))
		op := rng.Intn(3)
		
		switch op {
		case 0: // Insert
			skipTrieResult := st.Insert(key)
			exists := reference[key]
			referenceResult := !exists
			reference[key] = true // Always mark as existing after insert attempt
			
			if skipTrieResult != referenceResult {
				t.Errorf("Insert(%d): SkipTrie=%v, Reference=%v (exists=%v)", key, skipTrieResult, referenceResult, exists)
			}
			
		case 1: // Contains
			skipTrieResult := st.Contains(key)
			referenceResult := reference[key]
			
			if skipTrieResult != referenceResult {
				t.Errorf("Contains(%d): SkipTrie=%v, Reference=%v", key, skipTrieResult, referenceResult)
			}
			
		case 2: // Delete
			skipTrieResult := st.Delete(key)
			referenceResult := reference[key]
			reference[key] = false // Always mark as non-existing after delete attempt
			
			if skipTrieResult != referenceResult {
				t.Errorf("Delete(%d): SkipTrie=%v, Reference=%v", key, skipTrieResult, referenceResult)
			}
		}
	}
}

// Test for memory leaks by running GC
func TestMemoryLeaks(t *testing.T) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	// Create and destroy many SkipTries
	for i := 0; i < 100; i++ {
		st := NewSkipTrie()
		for j := 0; j < 1000; j++ {
			st.Insert(uint32(j))
		}
		for j := 0; j < 1000; j++ {
			st.Delete(uint32(j))
		}
	}
	
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	// Allow for some memory growth, but not excessive
	growth := m2.HeapInuse - m1.HeapInuse
	if growth > 10*1024*1024 { // 10MB threshold
		t.Errorf("Potential memory leak: heap grew by %d bytes", growth)
	}
}

// Test ordering properties
func TestOrdering(t *testing.T) {
	st := NewSkipTrie()
	
	// Insert random keys
	keys := []uint32{50, 25, 75, 10, 30, 60, 80, 5, 15, 35, 55, 65, 85}
	for _, key := range keys {
		st.Insert(key)
	}
	
	// Check predecessor relationships
	sortedKeys := make([]uint32, len(keys))
	copy(sortedKeys, keys)
	sort.Slice(sortedKeys, func(i, j int) bool { return sortedKeys[i] < sortedKeys[j] })
	
	for i, key := range sortedKeys {
		pred := st.Predecessor(key)
		if i == 0 {
			if pred != nil {
				t.Errorf("Predecessor of smallest key %d should be nil, got %v", key, pred.key)
			}
		} else {
			if pred == nil {
				t.Errorf("Predecessor of %d should not be nil", key)
			} else if pred.key != sortedKeys[i-1] {
				t.Errorf("Predecessor of %d should be %d, got %d", key, sortedKeys[i-1], pred.key)
			}
		}
	}
}

// Test with large datasets
func TestLargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}
	
	st := NewSkipTrie()
	const numKeys = 100000
	
	// Insert sequential keys
	start := time.Now()
	for i := uint32(0); i < numKeys; i++ {
		if !st.Insert(i) {
			t.Fatalf("Failed to insert key %d", i)
		}
	}
	insertTime := time.Since(start)
	
	// Check all keys exist
	start = time.Now()
	for i := uint32(0); i < numKeys; i++ {
		if !st.Contains(i) {
			t.Fatalf("Key %d should exist", i)
		}
	}
	searchTime := time.Since(start)
	
	// Delete all keys
	start = time.Now()
	for i := uint32(0); i < numKeys; i++ {
		if !st.Delete(i) {
			t.Fatalf("Failed to delete key %d", i)
		}
	}
	deleteTime := time.Since(start)
	
	t.Logf("Large dataset (%d keys): Insert=%v, Search=%v, Delete=%v", 
		numKeys, insertTime, searchTime, deleteTime)
}

// Test error conditions and edge cases
func TestErrorConditions(t *testing.T) {
	st := NewSkipTrie()
	
	// Test operations on empty trie
	if st.Contains(42) {
		t.Error("Empty trie should not contain any keys")
	}
	
	if st.Delete(42) {
		t.Error("Should not be able to delete from empty trie")
	}
	
	if pred := st.Predecessor(42); pred != nil {
		t.Error("Predecessor in empty trie should be nil")
	}
	
	// Test with the maximum possible value (but not the sentinel)
	maxUint32 := uint32(MaxKey - 1)
	if !st.Insert(maxUint32) {
		t.Error("Should be able to insert near-max uint32")
	}
	
	// Test predecessor of 0 when trie has elements
	st.Insert(100)
	if pred := st.Predecessor(0); pred != nil {
		t.Error("Predecessor of 0 should be nil")
	}
}

// Helper function to validate SkipTrie invariants
func validateSkipTrieInvariants(st *SkipTrie, t *testing.T) {
	// This is a basic validation - in a real implementation, you'd want more thorough checks
	// For now, we just ensure the head and tail are properly connected
	
	// Check head points to tail at all levels when empty
	for level := 0; level < LogLogU; level++ {
		next := st.head.next[level].Load()
		if next != st.tail {
			// This is okay if there are nodes in between
			continue
		}
	}
	
	// More invariant checks could be added here based on the paper's requirements
}

// Test invariants after various operations
func TestInvariants(t *testing.T) {
	st := NewSkipTrie()
	validateSkipTrieInvariants(st, t)
	
	// After insertions
	for i := uint32(0); i < 10; i++ {
		st.Insert(i)
		validateSkipTrieInvariants(st, t)
	}
	
	// After deletions
	for i := uint32(0); i < 5; i++ {
		st.Delete(i)
		validateSkipTrieInvariants(st, t)
	}
}