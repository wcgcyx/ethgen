package idgen

import "sync/atomic"

type IdGenerator struct {
	id int64
}

func NewIdGenerator() IdGenerator {
	return IdGenerator{id: 0}
}

func (gen *IdGenerator) Next() int64 {
	return atomic.AddInt64(&gen.id, 1)
}
