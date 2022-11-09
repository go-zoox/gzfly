package manager

import "fmt"

type Manager[T any] struct {
	options *Options[T]
	cache   map[string]T
}

type Options[T any] struct {
	Cache map[string]T
	Get   func(id string) (T, error)
}

func New[T any](opts ...*Options[T]) *Manager[T] {
	var options *Options[T]
	cache := make(map[string]T)
	if len(opts) == 1 && opts != nil {
		options = opts[0]

		if options.Cache != nil {
			cache = options.Cache
		}
	}

	return &Manager[T]{
		cache:   cache,
		options: options,
	}
}

func (m *Manager[T]) Get(id string) (T, error) {
	if m.options != nil && m.options.Get != nil {
		return m.options.Get(id)
	}

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
