package powercache

import (
	"iter"
)

type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Set(key K, value V)
	Delete(key K)
}

type Doable[K comparable, V any] interface {
	Do(key K, fn func() (V, error)) (V, error)
}

type MultiSetter[K comparable, V any] interface {
	SetFromMap(data map[K]V)
	SetFromIter(data iter.Seq2[K, V])
}
