package trie

import (
	"github.com/ElrondNetwork/elrond-go/common"
)

// DataTrieTrackerStub -
type DataTrieTrackerStub struct {
	ClearDataCachesCalled func()
	DirtyDataCalled       func() map[string][]byte
	RetrieveValueCalled   func(key []byte) ([]byte, error)
	SaveKeyValueCalled    func(key []byte, value []byte) error
	SetDataTrieCalled     func(tr common.Trie)
	DataTrieCalled        func() common.Trie
}

// ClearDataCaches -
func (dtts *DataTrieTrackerStub) ClearDataCaches() {
	dtts.ClearDataCachesCalled()
}

// DirtyData -
func (dtts *DataTrieTrackerStub) DirtyData() map[string][]byte {
	return dtts.DirtyDataCalled()
}

// RetrieveValue -
func (dtts *DataTrieTrackerStub) RetrieveValue(key []byte) ([]byte, error) {
	return dtts.RetrieveValueCalled(key)
}

// SaveKeyValue -
func (dtts *DataTrieTrackerStub) SaveKeyValue(key []byte, value []byte) error {
	return dtts.SaveKeyValueCalled(key, value)
}

// SetDataTrie -
func (dtts *DataTrieTrackerStub) SetDataTrie(tr common.Trie) {
	dtts.SetDataTrieCalled(tr)
}

// DataTrie -
func (dtts *DataTrieTrackerStub) DataTrie() common.Trie {
	return dtts.DataTrieCalled()
}

// IsInterfaceNil returns true if there is no value under the interface
func (dtts *DataTrieTrackerStub) IsInterfaceNil() bool {
	return dtts == nil
}
