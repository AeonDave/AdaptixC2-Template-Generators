package main

import "sync"

type Map struct {
	mu sync.RWMutex
	m  map[string]interface{}
}

func NewMap() Map {
	return Map{
		m: make(map[string]interface{}),
	}
}

func (m *Map) Contains(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.m[key]
	return ok
}

func (m *Map) Put(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[key] = value
}

func (m *Map) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.m[key]
	return v, ok
}

func (m *Map) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, key)
}

func (m *Map) GetDelete(key string) (interface{}, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.m[key]
	if ok {
		delete(m.m, key)
	}
	return v, ok
}

func (m *Map) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.m)
}

func (m *Map) ForEach(fn func(key string, value interface{}) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k, v := range m.m {
		if !fn(k, v) {
			break
		}
	}
}

func (m *Map) DirectLock() {
	m.mu.Lock()
}

func (m *Map) DirectUnlock() {
	m.mu.Unlock()
}

func (m *Map) DirectMap() map[string]interface{} {
	return m.m
}

func (m *Map) CutMap() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.m
	m.m = make(map[string]interface{})
	return old
}
