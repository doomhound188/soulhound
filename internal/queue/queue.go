package queue

import (
	"errors"
	"sync"
)

type Track struct {
	Title    string
	Artist   string
	URL      string
	Platform string
	Duration int
	Genre    string
}

type Queue struct {
	tracks  []Track
	mu      sync.Mutex
	current int
}

var (
	ErrQueueEmpty   = errors.New("queue is empty")
	ErrInvalidIndex = errors.New("invalid track index")
)

func NewQueue() *Queue {
	return &Queue{
		tracks:  make([]Track, 0),
		current: -1,
	}
}

func (q *Queue) Add(track Track) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.tracks = append(q.tracks, track)
	if q.current == -1 {
		q.current = 0
	}
}

func (q *Queue) Remove(index int) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if index < 0 || index >= len(q.tracks) {
		return ErrInvalidIndex
	}

	q.tracks = append(q.tracks[:index], q.tracks[index+1:]...)
	if len(q.tracks) == 0 {
		q.current = -1
	} else if index <= q.current {
		q.current--
	}
	return nil
}

func (q *Queue) Current() (*Track, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.current < 0 || q.current >= len(q.tracks) {
		return nil, ErrQueueEmpty
	}
	return &q.tracks[q.current], nil
}

func (q *Queue) Next() (*Track, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tracks) == 0 {
		return nil, ErrQueueEmpty
	}

	q.current = (q.current + 1) % len(q.tracks)
	return &q.tracks[q.current], nil
}

func (q *Queue) List() []Track {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]Track{}, q.tracks...)
}

func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.tracks = q.tracks[:0]
	q.current = -1
}
