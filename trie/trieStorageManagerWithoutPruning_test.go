package trie

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/stretchr/testify/assert"
)

func TestNewTrieStorageManagerWithoutPruningWithNilDb(t *testing.T) {
	t.Parallel()

	ts, err := NewTrieStorageManagerWithoutPruning(nil)
	assert.Nil(t, ts)
	assert.Equal(t, ErrNilDatabase, err)
}

func TestNewTrieStorageManagerWithoutPruning(t *testing.T) {
	t.Parallel()

	ts, err := NewTrieStorageManagerWithoutPruning(testscommon.NewMemDbMock())
	assert.Nil(t, err)
	assert.NotNil(t, ts)
}

func TestTrieStorageManagerWithoutPruning_TakeSnapshotShouldWork(t *testing.T) {
	t.Parallel()

	ts, _ := NewTrieStorageManagerWithoutPruning(testscommon.NewMemDbMock())
	ts.TakeSnapshot([]byte{}, true, nil)

	chLeaves := make(chan core.KeyValueHolder)
	ts.TakeSnapshot([]byte("rootHash"), true, chLeaves)

	select {
	case <-chLeaves:
	default:
		assert.Fail(t, "unclosed channel")
	}
}

func TestTrieStorageManagerWithoutPruning_SetCheckpointShouldWork(t *testing.T) {
	t.Parallel()

	ts, _ := NewTrieStorageManagerWithoutPruning(testscommon.NewMemDbMock())
	ts.SetCheckpoint([]byte{}, nil)

	chLeaves := make(chan core.KeyValueHolder)
	ts.SetCheckpoint([]byte("rootHash"), chLeaves)

	select {
	case <-chLeaves:
	default:
		assert.Fail(t, "unclosed channel")
	}
}

func TestTrieStorageManagerWithoutPruning_IsPruningEnabled(t *testing.T) {
	t.Parallel()

	ts, _ := NewTrieStorageManagerWithoutPruning(testscommon.NewMemDbMock())
	assert.False(t, ts.IsPruningEnabled())
}

func TestTrieStorageManagerWithoutPruning_Close(t *testing.T) {
	t.Parallel()

	closeCalled := false
	ts, _ := NewTrieStorageManagerWithoutPruning(&testscommon.StorerStub{
		CloseCalled: func() error {
			closeCalled = true
			return nil
		},
	})
	err := ts.Close()
	assert.Nil(t, err)
	assert.True(t, closeCalled)
}

func TestTrieStorageManagerWithoutPruning_AddDirtyCheckpointHashes(t *testing.T) {
	t.Parallel()

	ts, _ := NewTrieStorageManagerWithoutPruning(testscommon.NewMemDbMock())
	assert.False(t, ts.AddDirtyCheckpointHashes([]byte("key"), make(common.ModifiedHashes)))
}

func TestTrieStorageManagerWithoutPruning_Remove(t *testing.T) {
	t.Parallel()

	ts, _ := NewTrieStorageManagerWithoutPruning(testscommon.NewMemDbMock())
	assert.Nil(t, ts.Remove([]byte("key")))
}
