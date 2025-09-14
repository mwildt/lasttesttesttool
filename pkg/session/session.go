package session

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"sync"
)

type (
	Store[T any] struct {
		lock sync.RWMutex
		data map[string]Session[T]
	}

	Session[T any] struct {
		Id    string
		Value T
	}
)

func token() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.
		WithPadding(base64.NoPadding).
		EncodeToString(b), nil
}

func NewSessionStore[T any]() *Store[T] {
	return &Store[T]{
		lock: sync.RWMutex{},
		data: make(map[string]Session[T]),
	}
}

func (store *Store[T]) SessionCount() int {
	store.lock.RLock()
	defer store.lock.RUnlock()
	return len(store.data)
}

func (store *Store[T]) Create(value T) (Session[T], error) {
	sessionId, err := token()
	if err != nil {
		return Session[T]{}, err
	}
	log.Printf("create new session with id %s\n", sessionId)
	session := Session[T]{Id: sessionId, Value: value}
	store.lock.Lock()
	defer store.lock.Unlock()
	store.data[sessionId] = session
	return session, nil
}

func (store *Store[T]) Delete(key string) {
	log.Printf("delete session (id: %s)\n", key)
	store.lock.Lock()
	defer store.lock.Unlock()
	delete(store.data, key)
}

func (store *Store[T]) Reset() {
	log.Println("Store Reset")
	store.lock.Lock()
	defer store.lock.Unlock()
	store.data = make(map[string]Session[T])
}

func (store *Store[T]) Get(key string) (Session[T], bool) {
	store.lock.RLock()
	defer store.lock.RUnlock()
	session, ok := store.data[key]
	return session, ok
}

func (store *Store[T]) GetValue(key string) (T, bool) {
	session, ok := store.Get(key)
	return session.Value, ok
}
