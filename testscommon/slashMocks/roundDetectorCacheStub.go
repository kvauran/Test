package slashMocks

import (
	"github.com/ElrondNetwork/elrond-go-core/data"
)

// RoundDetectorCacheStub -
type RoundDetectorCacheStub struct {
	AddCalled            func(round uint64, pubKey []byte, headerInfo data.HeaderInfoHandler) error
	GetHeadersCalled     func(round uint64, pubKey []byte) []data.HeaderInfoHandler
	GetPubKeysCalled     func(round uint64) [][]byte
	IsInterfaceNilCalled func() bool
}

// Add -
func (rdc *RoundDetectorCacheStub) Add(round uint64, pubKey []byte, headerInfo data.HeaderInfoHandler) error {
	if rdc.AddCalled != nil {
		return rdc.AddCalled(round, pubKey, headerInfo)
	}
	return nil
}

// GetHeaders -
func (rdc *RoundDetectorCacheStub) GetHeaders(round uint64, pubKey []byte) []data.HeaderInfoHandler {
	if rdc.GetHeadersCalled != nil {
		return rdc.GetHeadersCalled(round, pubKey)
	}
	return nil
}

// GetPubKeys -
func (rdc *RoundDetectorCacheStub) GetPubKeys(round uint64) [][]byte {
	if rdc.GetPubKeysCalled != nil {
		return rdc.GetPubKeysCalled(round)
	}
	return nil
}

// IsInterfaceNil -
func (rdc *RoundDetectorCacheStub) IsInterfaceNil() bool {
	return rdc == nil
}