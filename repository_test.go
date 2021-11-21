package main

import (
	"context"
	"time"
)

func NewMockRepo(fn func(string) (int, error)) Repository {
	return &MockRepository{
		incrFunc: fn,
	}
}

type MockRepository struct {
	incrFunc func(string) (int, error)
}

func (mr MockRepository) Incr(_ context.Context, key string) (int, error) {
	return mr.incrFunc(key)
}

func (mr MockRepository) Get(_ context.Context, _ string) (int, error) {
	panic("not implemented") // TODO: Implement
}

func (mr MockRepository) Set(_ context.Context, _ string, _ interface{}, _ time.Duration) error {
	panic("not implemented") // TODO: Implement
}
