package track

import (
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/process"
)

type shardBlockTrack struct {
	*baseBlockTrack
}

// NewShardBlockTrack creates an object for tracking the received shard blocks
func NewShardBlockTrack(arguments ArgShardTracker) (*shardBlockTrack, error) {
	err := checkTrackerNilParameters(arguments.ArgBaseTracker)
	if err != nil {
		return nil, err
	}

	bbt, err := createBaseBlockTrack(arguments.ArgBaseTracker)
	if err != nil {
		return nil, err
	}

	err = bbt.initNotarizedHeaders(arguments.StartHeaders)
	if err != nil {
		return nil, err
	}

	sbt := shardBlockTrack{
		baseBlockTrack: bbt,
	}

	argBlockProcessor := ArgBlockProcessor{
		HeaderValidator:               arguments.HeaderValidator,
		RequestHandler:                arguments.RequestHandler,
		ShardCoordinator:              arguments.ShardCoordinator,
		BlockTracker:                  &sbt,
		CrossNotarizer:                bbt.crossNotarizer,
		CrossNotarizedHeadersNotifier: bbt.crossNotarizedHeadersNotifier,
		SelfNotarizedHeadersNotifier:  bbt.selfNotarizedHeadersNotifier,
		Rounder:                       arguments.Rounder,
	}

	blockProcessorObject, err := NewBlockProcessor(argBlockProcessor)
	if err != nil {
		return nil, err
	}

	sbt.blockProcessor = blockProcessorObject
	sbt.headers = make(map[uint32]map[uint64][]*HeaderInfo)
	sbt.headersPool.RegisterHandler(sbt.receivedHeader)

	return &sbt, nil
}

// GetSelfHeaders gets a slice of self headers from a given metablock
func (sbt *shardBlockTrack) GetSelfHeaders(headerHandler data.HeaderHandler) []*HeaderInfo {
	selfHeadersInfo := make([]*HeaderInfo, 0)

	metaBlock, ok := headerHandler.(*block.MetaBlock)
	if !ok {
		log.Debug("GetSelfHeaders", "error", process.ErrWrongTypeAssertion)
		return selfHeadersInfo
	}

	for _, shardInfo := range metaBlock.ShardInfo {
		if shardInfo.ShardID != sbt.shardCoordinator.SelfId() {
			continue
		}

		header, err := process.GetShardHeader(shardInfo.HeaderHash, sbt.headersPool, sbt.marshalizer, sbt.store)
		if err != nil {
			log.Trace("GetSelfHeaders.GetShardHeader", "error", err.Error())
			continue
		}

		selfHeadersInfo = append(selfHeadersInfo, &HeaderInfo{Hash: shardInfo.HeaderHash, Header: header})
	}

	return selfHeadersInfo
}

// CleanupInvalidCrossHeaders cleans headers added to the block tracker that have become invalid after processing
func (sbt *shardBlockTrack) CleanupInvalidCrossHeaders(_ uint32, _ uint64) {
	// no rule for shard
}

// ComputeLongestSelfChain computes the longest chain from self shard
func (sbt *shardBlockTrack) ComputeLongestSelfChain() (data.HeaderHandler, []byte, []data.HeaderHandler, [][]byte) {
	lastSelfNotarizedHeader, lastSelfNotarizedHeaderHash, err := sbt.selfNotarizer.GetLastNotarizedHeader(core.MetachainShardId)
	if err != nil {
		log.Warn("ComputeLongestSelfChain.GetLastNotarizedHeader", "error", err.Error())
		return nil, nil, nil, nil
	}

	headers, hashes := sbt.ComputeLongestChain(sbt.shardCoordinator.SelfId(), lastSelfNotarizedHeader)
	return lastSelfNotarizedHeader, lastSelfNotarizedHeaderHash, headers, hashes
}

// ComputeNumPendingMiniBlocks computes the number of pending miniblocks from a given slice of metablocks
func (sbt *shardBlockTrack) ComputeNumPendingMiniBlocks(headers []data.HeaderHandler) {
	lenHeaders := len(headers)
	if lenHeaders == 0 {
		return
	}

	metaBlock, ok := headers[lenHeaders-1].(*block.MetaBlock)
	if !ok {
		log.Debug("ComputeNumPendingMiniBlocks", "error", process.ErrWrongTypeAssertion)
		return
	}

	for _, shardInfo := range metaBlock.ShardInfo {
		sbt.blockBalancer.SetNumPendingMiniBlocks(shardInfo.ShardID, shardInfo.NumPendingMiniBlocks)
		sbt.blockBalancer.SetLastShardProcessedMetaNonce(shardInfo.ShardID, shardInfo.LastIncludedMetaNonce)
	}

	for shardID := uint32(0); shardID < sbt.shardCoordinator.NumberOfShards(); shardID++ {
		log.Debug("pending miniblocks",
			"shard", shardID,
			"num", sbt.blockBalancer.GetNumPendingMiniBlocks(shardID))
	}

	log.Debug("pending miniblocks",
		"shard", core.AllShardId,
		"num", sbt.blockBalancer.GetNumPendingMiniBlocks(core.AllShardId))
}
