package dblookupext

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go/dblookupext"
)

// HistoryRepositoryStub -
type HistoryRepositoryStub struct {
	RecordBlockCalled                  func(blockHeaderHash []byte, blockHeader data.HeaderHandler, blockBody data.BodyHandler, scrsPool map[string]data.TransactionHandler, receipts map[string]data.TransactionHandler, logs map[string]data.LogHandler) error
	OnNotarizedBlocksCalled            func(shardID uint32, headers []data.HeaderHandler, headersHashes [][]byte)
	GetMiniblockMetadataByTxHashCalled func(hash []byte) (*dblookupext.MiniblockMetadata, error)
	GetEpochByHashCalled               func(hash []byte) (uint32, error)
	GetEventsHashesByTxHashCalled      func(hash []byte, epoch uint32) (*dblookupext.ResultsHashesByTxHash, error)
	IsEnabledCalled                    func() bool
}

// RecordBlock -
func (hp *HistoryRepositoryStub) RecordBlock(
	blockHeaderHash []byte,
	blockHeader data.HeaderHandler,
	blockBody data.BodyHandler,
	scrsPool map[string]data.TransactionHandler,
	receipts map[string]data.TransactionHandler,
	logs map[string]data.LogHandler,
) error {
	if hp.RecordBlockCalled != nil {
		return hp.RecordBlockCalled(blockHeaderHash, blockHeader, blockBody, scrsPool, receipts, logs)
	}
	return nil
}

// OnNotarizedBlocks -
func (hp *HistoryRepositoryStub) OnNotarizedBlocks(shardID uint32, headers []data.HeaderHandler, headersHashes [][]byte) {
	if hp.OnNotarizedBlocksCalled != nil {
		hp.OnNotarizedBlocksCalled(shardID, headers, headersHashes)
	}
}

// GetMiniblockMetadataByTxHash -
func (hp *HistoryRepositoryStub) GetMiniblockMetadataByTxHash(hash []byte) (*dblookupext.MiniblockMetadata, error) {
	if hp.GetMiniblockMetadataByTxHashCalled != nil {
		return hp.GetMiniblockMetadataByTxHashCalled(hash)
	}
	return nil, fmt.Errorf("miniblock metadata not found")
}

// GetEpochByHash -
func (hp *HistoryRepositoryStub) GetEpochByHash(hash []byte) (uint32, error) {
	return hp.GetEpochByHashCalled(hash)
}

// IsEnabled -
func (hp *HistoryRepositoryStub) IsEnabled() bool {
	if hp.IsEnabledCalled != nil {
		return hp.IsEnabledCalled()
	}
	return true
}

// GetResultsHashesByTxHash -
func (hp *HistoryRepositoryStub) GetResultsHashesByTxHash(hash []byte, epoch uint32) (*dblookupext.ResultsHashesByTxHash, error) {
	if hp.GetEventsHashesByTxHashCalled != nil {
		return hp.GetEventsHashesByTxHashCalled(hash, epoch)
	}
	return nil, nil
}

// RevertBlock -
func (hp *HistoryRepositoryStub) RevertBlock(_ data.HeaderHandler, _ data.BodyHandler) error {
	return nil
}

// GetESDTSupply -
func (hp *HistoryRepositoryStub) GetESDTSupply(_ string) (string, error) {
	return "", nil
}

// IsInterfaceNil -
func (hp *HistoryRepositoryStub) IsInterfaceNil() bool {
	return hp == nil
}
