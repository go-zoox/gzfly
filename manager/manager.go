package manager

import "fmt"

type Manager[T any] struct {
	cache map[string]T
}

func New[T any]() *Manager[T] {
	return &Manager[T]{
		cache: make(map[string]T),
	}
}

func (m *Manager[T]) Get(id string) (T, error) {
	if instance, ok := m.cache[id]; ok {
		return instance, nil
	}

	var t T
	return t, fmt.Errorf("id %s not found", id)
}

func (m *Manager[T]) Set(id string, instance T) error {
	m.cache[id] = instance
	return nil
}

func (m *Manager[T]) GetOrCreate(id string, creator func() T) (T, error) {
	if instance, err := m.Get(id); err == nil {
		return instance, nil
	}

	m.cache[id] = creator()
	return m.cache[id], nil
}
