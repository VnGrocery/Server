package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type KV interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Incr(ctx context.Context, key string) (int64, error)
}

type Store struct {
	kv     KV
	prefix string
	ttl    time.Duration
}

func NewStore(kv KV, prefix string, ttl time.Duration) *Store {
	return &Store{
		kv:     kv,
		prefix: prefix,
		ttl:    ttl,
	}
}

func (s *Store) IsEnabled() bool {
	return s != nil && s.kv != nil && s.ttl > 0
}

func (s *Store) Version(ctx context.Context, namespace string) (int64, error) {
	if !s.IsEnabled() {
		return 0, nil
	}
	value, err := s.kv.Get(ctx, s.nsKey(namespace))
	if err != nil {
		if errors.Is(err, ErrCacheMiss) {
			return 0, nil
		}
		return 0, err
	}

	var version int64
	_, scanErr := fmt.Sscanf(value, "%d", &version)
	if scanErr != nil {
		return 0, fmt.Errorf("parse namespace version: %w", scanErr)
	}
	if version < 0 {
		return 0, nil
	}
	return version, nil
}

func (s *Store) BumpVersion(ctx context.Context, namespace string) error {
	if !s.IsEnabled() {
		return nil
	}
	_, err := s.kv.Incr(ctx, s.nsKey(namespace))
	return err
}

func (s *Store) BuildKey(namespace string, version int64, suffix string) string {
	return fmt.Sprintf("%s%s:v%d:%s", s.prefix, namespace, version, suffix)
}

func (s *Store) GetJSON(ctx context.Context, key string, out any) (bool, error) {
	if !s.IsEnabled() {
		return false, nil
	}
	value, err := s.kv.Get(ctx, key)
	if err != nil {
		if errors.Is(err, ErrCacheMiss) {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal([]byte(value), out); err != nil {
		return false, fmt.Errorf("decode cache value: %w", err)
	}
	return true, nil
}

func (s *Store) SetJSON(ctx context.Context, key string, value any) error {
	if !s.IsEnabled() {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode cache value: %w", err)
	}
	return s.kv.Set(ctx, key, string(payload), s.ttl)
}

func (s *Store) nsKey(namespace string) string {
	return fmt.Sprintf("%s%s:version", s.prefix, namespace)
}
