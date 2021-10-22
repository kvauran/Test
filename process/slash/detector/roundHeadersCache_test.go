package detector

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/slash"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/stretchr/testify/require"
)

func TestRoundDataCache_Add_OneRound_FourHeaders(t *testing.T) {
	t.Parallel()

	dataCache := NewRoundHeadersCache(1)

	err := dataCache.Add(1, &slash.HeaderInfo{Header: &testscommon.HeaderHandlerStub{TimestampField: 1}, Hash: []byte("hash1")})
	require.Nil(t, err)

	err = dataCache.Add(1, &slash.HeaderInfo{Header: &testscommon.HeaderHandlerStub{TimestampField: 2}, Hash: []byte("hash1")})
	require.Equal(t, process.ErrHeadersNotDifferentHashes, err)

	err = dataCache.Add(1, &slash.HeaderInfo{Header: &testscommon.HeaderHandlerStub{TimestampField: 3}, Hash: []byte("hash2")})
	require.Nil(t, err)

	err = dataCache.Add(1, &slash.HeaderInfo{Header: &testscommon.HeaderHandlerStub{TimestampField: 4}, Hash: []byte("hash3")})
	require.Nil(t, err)

	require.Len(t, dataCache.cache, 1)
	require.Len(t, dataCache.cache[1], 3)

	require.Equal(t, []byte("hash1"), dataCache.cache[1][0].Hash)
	require.Equal(t, uint64(1), dataCache.cache[1][0].Header.GetTimeStamp())

	require.Equal(t, []byte("hash2"), dataCache.cache[1][1].Hash)
	require.Equal(t, uint64(3), dataCache.cache[1][1].Header.GetTimeStamp())

	require.Equal(t, []byte("hash3"), dataCache.cache[1][2].Hash)
	require.Equal(t, uint64(4), dataCache.cache[1][2].Header.GetTimeStamp())

}

func TestRoundDataCache_Add_CacheSizeTwo_FourEntriesInCache_ExpectOldestRoundInCacheRemoved(t *testing.T) {
	t.Parallel()

	dataCache := NewRoundHeadersCache(2)

	err := dataCache.Add(1, &slash.HeaderInfo{Header: &testscommon.HeaderHandlerStub{TimestampField: 1}, Hash: []byte("hash1")})
	require.Nil(t, err)

	err = dataCache.Add(2, &slash.HeaderInfo{Header: &testscommon.HeaderHandlerStub{TimestampField: 2}, Hash: []byte("hash2")})
	require.Nil(t, err)

	require.Len(t, dataCache.cache, 2)
	require.Len(t, dataCache.cache[1], 1)
	require.Len(t, dataCache.cache[2], 1)

	require.Equal(t, []byte("hash1"), dataCache.cache[1][0].Hash)
	require.Equal(t, uint64(1), dataCache.cache[1][0].Header.GetTimeStamp())

	require.Equal(t, []byte("hash2"), dataCache.cache[2][0].Hash)
	require.Equal(t, uint64(2), dataCache.cache[2][0].Header.GetTimeStamp())

	err = dataCache.Add(0, &slash.HeaderInfo{Header: &testscommon.HeaderHandlerStub{}, Hash: []byte("hash0")})
	require.Equal(t, process.ErrHeaderRoundNotRelevant, err)

	require.Len(t, dataCache.cache, 2)
	require.Len(t, dataCache.cache[1], 1)
	require.Len(t, dataCache.cache[2], 1)

	require.Equal(t, []byte("hash1"), dataCache.cache[1][0].Hash)
	require.Equal(t, uint64(1), dataCache.cache[1][0].Header.GetTimeStamp())

	require.Equal(t, []byte("hash2"), dataCache.cache[2][0].Hash)
	require.Equal(t, uint64(2), dataCache.cache[2][0].Header.GetTimeStamp())

	err = dataCache.Add(3, &slash.HeaderInfo{Header: &testscommon.HeaderHandlerStub{TimestampField: 3}, Hash: []byte("hash3")})
	require.Nil(t, err)

	require.Len(t, dataCache.cache, 2)
	require.Len(t, dataCache.cache[2], 1)
	require.Len(t, dataCache.cache[3], 1)

	require.Equal(t, []byte("hash2"), dataCache.cache[2][0].Hash)
	require.Equal(t, uint64(2), dataCache.cache[2][0].Header.GetTimeStamp())

	require.Equal(t, []byte("hash3"), dataCache.cache[3][0].Hash)
	require.Equal(t, uint64(3), dataCache.cache[3][0].Header.GetTimeStamp())

	err = dataCache.Add(4, &slash.HeaderInfo{Header: &testscommon.HeaderHandlerStub{TimestampField: 4}, Hash: []byte("hash4")})
	require.Nil(t, err)

	require.Len(t, dataCache.cache, 2)
	require.Len(t, dataCache.cache[3], 1)
	require.Len(t, dataCache.cache[4], 1)

	require.Equal(t, []byte("hash3"), dataCache.cache[3][0].Hash)
	require.Equal(t, uint64(3), dataCache.cache[3][0].Header.GetTimeStamp())

	require.Equal(t, []byte("hash4"), dataCache.cache[4][0].Hash)
	require.Equal(t, uint64(4), dataCache.cache[4][0].Header.GetTimeStamp())
}

func TestRoundDataCache_Contains(t *testing.T) {
	t.Parallel()

	dataCache := NewRoundHeadersCache(2)

	err := dataCache.Add(1, &slash.HeaderInfo{Header: &testscommon.HeaderHandlerStub{TimestampField: 1}, Hash: []byte("hash1")})
	require.Nil(t, err)

	err = dataCache.Add(1, &slash.HeaderInfo{Header: &testscommon.HeaderHandlerStub{TimestampField: 2}, Hash: []byte("hash2")})
	require.Nil(t, err)

	err = dataCache.Add(2, &slash.HeaderInfo{Header: &testscommon.HeaderHandlerStub{TimestampField: 3}, Hash: []byte("hash3")})
	require.Nil(t, err)

	require.True(t, dataCache.contains(1, []byte("hash1")))
	require.True(t, dataCache.contains(1, []byte("hash2")))
	require.True(t, dataCache.contains(2, []byte("hash3")))

	require.False(t, dataCache.contains(1, []byte("hash3")))
	require.False(t, dataCache.contains(3, []byte("hash1")))
}

func TestRoundValidatorsDataCache_IsInterfaceNil(t *testing.T) {
	cache := NewRoundValidatorHeaderCache(1)
	require.False(t, cache.IsInterfaceNil())
	cache = nil
	require.True(t, cache.IsInterfaceNil())
}