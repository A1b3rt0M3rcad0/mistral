package persistence

import "errors"

type Version uint64

type Record[T any] struct {
	Value   T       `json:"value"`
	Version Version `json:"version"`
}

var (
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("record already exists")
	ErrConflict      = errors.New("optimistic concurrency conflict")
)
