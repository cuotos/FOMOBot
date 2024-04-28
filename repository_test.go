package main

import (
	"context"
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

func (mr MockRepository) Healthy(_ context.Context) error {
	return nil
}
