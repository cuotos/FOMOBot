package database

import "context"

// TODO we are not actually using this interface anywhere!
type Database interface {
	Incr(context.Context, string) (int, error)
	Healthy(context.Context) error
}
