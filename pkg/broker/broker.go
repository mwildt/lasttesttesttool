package broker

import (
	"crypto/rand"
	"sync"
)

type (
	RegistrationToken [32]byte
	Broker[T any]     struct {
		mu      sync.Mutex
		clients map[RegistrationToken]chan T
	}
)

func newRegistrationToken() (RegistrationToken, error) {
	var b RegistrationToken
	_, err := rand.Read(b[:])
	if err != nil {
		return b, err
	}
	return b, nil
}

func NewBroker[T comparable]() *Broker[T] {
	return &Broker[T]{
		clients: make(map[RegistrationToken]chan T),
	}
}

func (b *Broker[T]) Register() (RegistrationToken, chan T, error) {
	key, err := newRegistrationToken()
	if err != nil {
		return RegistrationToken{}, nil, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan T, 1000)
	b.clients[key] = ch
	return key, ch, nil
}

func (b *Broker[T]) Cancel(key RegistrationToken) {
	ch, exists := b.clients[key]
	if exists {
		b.mu.Lock()
		defer b.mu.Unlock()
		close(ch)
		delete(b.clients, key)
	}
}

func (b *Broker[T]) Broadcast(msg T) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, client := range b.clients {
		client <- msg
	}
}
