package detector

import (
	"bytes"
	"math"
	"sync"

	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/slash"
)

type roundHeadersCache struct {
	cache       map[uint64]slash.HeaderInfoList
	cacheMutex  sync.RWMutex
	oldestRound uint64
	cacheSize   uint64
}

// NewRoundHeadersCache creates an instance of roundHeadersCache, which is a header-hash-based cache
func NewRoundHeadersCache(maxRounds uint64) *roundHeadersCache {
	return &roundHeadersCache{
		cache:       make(map[uint64]slash.HeaderInfoList),
		cacheMutex:  sync.RWMutex{},
		oldestRound: math.MaxUint64,
		cacheSize:   maxRounds,
	}
}

// Add adds a header-hash in cache, in a given round.
// It has an eviction mechanism which always removes the oldest round entry when cache is full
func (rhc *roundHeadersCache) Add(round uint64, header *slash.HeaderInfo) error {
	rhc.cacheMutex.Lock()
	defer rhc.cacheMutex.Unlock()

	if rhc.contains(round, header.Hash) {
		return process.ErrHeadersNotDifferentHashes
	}

	if rhc.isCacheFull(round) {
		if round < rhc.oldestRound {
			return process.ErrHeaderRoundNotRelevant
		}
		delete(rhc.cache, rhc.oldestRound)
		rhc.updateOldestRound()
	}

	if round < rhc.oldestRound {
		rhc.oldestRound = round
	}

	rhc.cache[round] = append(rhc.cache[round], header)
	return nil
}

func (rhc *roundHeadersCache) contains(round uint64, hash []byte) bool {
	hashHeaderList, exist := rhc.cache[round]
	if !exist {
		return false
	}

	for _, currData := range hashHeaderList {
		if bytes.Equal(currData.Hash, hash) {
			return true
		}
	}

	return false
}

func (rhc *roundHeadersCache) isCacheFull(currRound uint64) bool {
	_, currRoundInCache := rhc.cache[currRound]
	return len(rhc.cache) >= int(rhc.cacheSize) && !currRoundInCache
}

func (rhc *roundHeadersCache) updateOldestRound() {
	min := uint64(math.MaxUint64)

	for round := range rhc.cache {
		if round < min {
			min = round
		}
	}

	rhc.oldestRound = min
}

// IsInterfaceNil checks if the underlying pointer is nil
func (rhc *roundHeadersCache) IsInterfaceNil() bool {
	return rhc == nil
}