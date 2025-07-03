package queue

import (
	"testing"
)

func TestNewQueue(t *testing.T) {
	q := NewQueue()
	if q == nil {
		t.Fatal("NewQueue() returned nil")
	}
	
	if q.current != -1 {
		t.Errorf("Expected current to be -1, got %d", q.current)
	}
	
	if len(q.tracks) != 0 {
		t.Errorf("Expected tracks length to be 0, got %d", len(q.tracks))
	}
}

func TestQueueAdd(t *testing.T) {
	q := NewQueue()
	
	track := Track{
		Title:    "Test Song",
		Artist:   "Test Artist",
		URL:      "test-url",
		Platform: "yt",
		Duration: 180,
		Genre:    "rock",
	}
	
	q.Add(track)
	
	if len(q.tracks) != 1 {
		t.Errorf("Expected tracks length to be 1, got %d", len(q.tracks))
	}
	
	if q.current != 0 {
		t.Errorf("Expected current to be 0, got %d", q.current)
	}
	
	current, err := q.Current()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if current.Title != track.Title {
		t.Errorf("Expected title %s, got %s", track.Title, current.Title)
	}
}

func TestQueueEmpty(t *testing.T) {
	q := NewQueue()
	
	_, err := q.Current()
	if err != ErrQueueEmpty {
		t.Errorf("Expected ErrQueueEmpty, got %v", err)
	}
}

func TestQueueClear(t *testing.T) {
	q := NewQueue()
	
	track := Track{
		Title:    "Test Song",
		Artist:   "Test Artist",
		URL:      "test-url",
		Platform: "yt",
		Duration: 180,
		Genre:    "rock",
	}
	
	q.Add(track)
	q.Clear()
	
	if len(q.tracks) != 0 {
		t.Errorf("Expected tracks length to be 0 after clear, got %d", len(q.tracks))
	}
	
	if q.current != -1 {
		t.Errorf("Expected current to be -1 after clear, got %d", q.current)
	}
}