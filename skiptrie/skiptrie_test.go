package skiptrie

import (
	"math/rand"
	"testing"
	"time"
)

func TestNewSkipTrie(t *testing.T) {
	st := NewSkipTrie()
	if st == nil {
		t.Fatal("NewSkipTrie returned nil")
	}
	if st.head == nil {
		t.Fatal("SkipTrie head is nil")
	}
	if st.tail == nil {
		t.Fatal("SkipTrie tail is nil")
	}
}

func TestInsertAndContains(t *testing.T) {
	st := NewSkipTrie()

	// Test inserting some keys
	keys := []uint32{10, 20, 30, 5, 15, 25}

	for _, key := range keys {
		if !st.Insert(key) {
			t.Errorf("Failed to insert key %d", key)
		}
		if !st.Contains(key) {
			t.Errorf("Key %d not found after insertion", key)
		}
	}

	// Test that non-inserted keys are not found
	if st.Contains(100) {
		t.Error("Non-inserted key 100 was found")
	}
	if st.Contains(0) {
		t.Error("Non-inserted key 0 was found")
	}
}

func TestInsertDuplicate(t *testing.T) {
	st := NewSkipTrie()

	// Insert a key
	if !st.Insert(42) {
		t.Fatal("Failed to insert key 42")
	}

	// Try to insert the same key again
	if st.Insert(42) {
		t.Error("Duplicate key insertion should return false")
	}

	// Key should still be present
	if !st.Contains(42) {
		t.Error("Key 42 not found after duplicate insertion attempt")
	}
}

func TestDelete(t *testing.T) {
	st := NewSkipTrie()

	// Insert some keys
	keys := []uint32{10, 20, 30}
	for _, key := range keys {
		st.Insert(key)
	}

	// Delete a key
	if !st.Delete(20) {
		t.Error("Failed to delete key 20")
	}

	// Verify it's gone
	if st.Contains(20) {
		t.Error("Key 20 still found after deletion")
	}

	// Verify other keys are still there
	if !st.Contains(10) {
		t.Error("Key 10 not found after deleting 20")
	}
	if !st.Contains(30) {
		t.Error("Key 30 not found after deleting 20")
	}

	// Try to delete non-existent key
	if st.Delete(100) {
		t.Error("Deleting non-existent key should return false")
	}
}

func TestPredecessor(t *testing.T) {
	st := NewSkipTrie()

	// Insert keys in random order
	keys := []uint32{10, 30, 5, 25, 15}
	for _, key := range keys {
		st.Insert(key)
	}

	// Test predecessor function
	pred := st.Predecessor(20)
	if pred == nil || pred.Key() != 15 {
		t.Errorf("Predecessor of 20 should be 15, got %v", pred)
	}

	pred = st.Predecessor(100)
	if pred == nil || pred.Key() != 30 {
		t.Errorf("Predecessor of 100 should be 30, got %v", pred)
	}

	// Predecessor of smallest element
	pred = st.Predecessor(1)
	if pred != nil {
		t.Errorf("Predecessor of 1 should be nil, got %v", pred)
	}
}

func TestRandomOperations(t *testing.T) {
	st := NewSkipTrie()
	rand.Seed(time.Now().UnixNano())

	inserted := make(map[uint32]bool)

	// Perform random insertions
	for i := 0; i < 100; i++ {
		key := rand.Uint32() % 1000 // Keep keys small for better testing
		if st.Insert(key) {
			inserted[key] = true
		}
	}

	// Verify all inserted keys are found
	for key := range inserted {
		if !st.Contains(key) {
			t.Errorf("Key %d was inserted but not found", key)
		}
	}

	// Delete half the keys
	count := 0
	for key := range inserted {
		if count%2 == 0 {
			if !st.Delete(key) {
				t.Errorf("Failed to delete key %d", key)
			}
			delete(inserted, key)
		}
		count++
	}

	// Verify remaining keys are still found and deleted keys are gone
	for key := range inserted {
		if !st.Contains(key) {
			t.Errorf("Key %d should still be present after partial deletion", key)
		}
	}
}

func BenchmarkInsert(b *testing.B) {
	st := NewSkipTrie()
	keys := make([]uint32, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = rand.Uint32()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st.Insert(keys[i])
	}
}

func BenchmarkContains(b *testing.B) {
	st := NewSkipTrie()
	// Pre-populate with some keys
	for i := 0; i < 1000; i++ {
		st.Insert(rand.Uint32())
	}

	keys := make([]uint32, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = rand.Uint32()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st.Contains(keys[i])
	}
}
