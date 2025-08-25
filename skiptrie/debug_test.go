package skiptrie

import (
	"testing"
	"time"
)

// Test for potential infinite loops in listSearch
func TestListSearchInfiniteLoop(t *testing.T) {
	st := NewSkipTrie()
	
	// Add some keys
	keys := []uint32{10, 20, 30}
	for _, key := range keys {
		st.Insert(key)
	}
	
	// Test listSearch with timeout
	done := make(chan bool, 1)
	
	go func() {
		// This should complete quickly
		left, right := st.listSearch(25, st.head, 0)
		if left.key != 20 || (right != nil && right.key != 30) {
			t.Errorf("listSearch(25) returned incorrect results: left=%v, right=%v", 
				left.key, right)
		}
		done <- true
	}()
	
	select {
	case <-done:
		// Test passed
	case <-time.After(5 * time.Second):
		t.Fatal("listSearch appears to be in infinite loop")
	}
}

// Test for ABA problem and concurrent modification
func TestConcurrentModificationABA(t *testing.T) {
	st := NewSkipTrie()
	
	// Insert initial key
	st.Insert(50)
	
	done := make(chan bool, 2)
	
	// Goroutine 1: constantly insert/delete
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			st.Insert(uint32(i))
			st.Delete(uint32(i))
		}
	}()
	
	// Goroutine 2: constantly search
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			st.Contains(25)
		}
	}()
	
	// Wait for both to complete with timeout
	select {
	case <-done:
		<-done // Wait for second goroutine
		t.Log("Concurrent modification test passed")
	case <-time.After(10 * time.Second):
		t.Fatal("Concurrent modification test timed out - possible infinite loop")
	}
}