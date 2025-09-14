package store

import (
	"github.com/mwildt/load-monitor/pkg/broker"
	"strings"
	"sync"
	"time"
)

// der store hält die Daten -> er wird von den Listeners aktualisiert

type (
	Event[T any] struct {
		Key   string
		Value T
	}
	Store struct {
		values        map[string]any
		defaultValues map[string]any
		broker        *broker.Broker[Event[any]]
		mu            sync.RWMutex
	}
	Predicate func(string) bool
)

func cloneMap(original map[string]any) map[string]any {
	clone := make(map[string]any)
	for k, v := range original {
		clone[k] = v
	}
	return clone
}

func All() Predicate {
	return func(key string) bool {
		return true
	}
}

func Select(prefix string) Predicate {
	return func(key string) bool {
		return strings.HasPrefix(key, prefix)
	}
}

func NewStore(defaultValues map[string]any) *Store {
	return &Store{
		defaultValues: defaultValues,
		values:        cloneMap(defaultValues),
		broker:        broker.NewBroker[Event[any]](),
		mu:            sync.RWMutex{},
	}
}

func (s *Store) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	s.broadcast(Event[any]{key, value})
}

func (s *Store) Reduce(key string, reducer func(any) any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value := reducer(s.values[key])
	s.values[key] = value
	s.broadcast(Event[any]{key, value})
}

func (s *Store) Get(key string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.values[key]
}

func (s *Store) Cancel(reg broker.RegistrationToken) {
	s.broker.Cancel(reg)
}

func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = cloneMap(s.defaultValues)
	s.broadcastAll()
}

func (s *Store) broadcast(event Event[any]) {
	s.broker.Broadcast(event)
}

func (s *Store) RegisterThrottled(predicate Predicate, delay time.Duration) (broker.RegistrationToken, chan []Event[any], error) {
	registration, in, err := s.broker.Register()
	if err != nil {
		return registration, nil, err
	}
	out := make(chan []Event[any])
	go func() {
		var (
			timeBarrier = time.Now()
			timeout     <-chan time.Time
			cache       = make(map[string]Event[any])
		)
		for {
			select {
			case event, ok := <-in:
				{
					if !ok {
						return
					}
					if predicate(event.Key) {
						now := time.Now()
						if now.After(timeBarrier) {
							out <- []Event[any]{event}
							timeBarrier = now.Add(delay)
						} else {
							cache[event.Key] = event
							if timeout == nil {
								after := timeBarrier.Sub(now)
								timeout = time.After(after)
							}
						}
					}
				}
			case <-timeout:
				{
					if cache != nil {
						events := make([]Event[any], 0)
						for _, event := range cache {
							events = append(events, event)
						}
						out <- events
						cache = make(map[string]Event[any])
					}
					timeBarrier = time.Now().Add(delay)
					timeout = nil
				}
			}
		}
		close(out)
	}()
	return registration, out, nil
}

func (s *Store) Register(predicate Predicate) (broker.RegistrationToken, chan Event[any], error) {
	registration, in, err := s.broker.Register()
	if err != nil {
		return registration, nil, err
	}
	out := make(chan Event[any])
	go func() {
		for event := range in {
			if predicate(event.Key) {
				out <- event
			}
		}
		close(out)
	}()
	return registration, out, nil
}

func (s *Store) Entries() map[string]any {
	return s.values
}

func (s *Store) broadcastAll() {
	for key, value := range s.values {
		s.broadcast(Event[any]{key, value})
	}
}
