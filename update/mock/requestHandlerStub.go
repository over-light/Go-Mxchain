package mock

type RequestHandlerStub struct {
	RequestShardHeaderCalled           func(shardId uint32, hash []byte)
	RequestMetaHeaderCalled            func(hash []byte)
	RequestMetaHeaderByNonceCalled     func(nonce uint64)
	RequestShardHeaderByNonceCalled    func(shardId uint32, nonce uint64)
	RequestTransactionHandlerCalled    func(destShardID uint32, txHashes [][]byte)
	RequestScrHandlerCalled            func(destShardID uint32, txHashes [][]byte)
	RequestRewardTxHandlerCalled       func(destShardID uint32, txHashes [][]byte)
	RequestMiniBlockHandlerCalled      func(destShardID uint32, miniblockHash []byte)
	RequestTrieNodesCalled             func(shardId uint32, hash []byte)
	RequestStartOfEpochMetaBlockCalled func(epoch uint32)
}

func (rhs *RequestHandlerStub) SetEpoch(epoch uint32) {
	panic("implement me")
}

func (rhs *RequestHandlerStub) RequestMiniBlocks(destShardID uint32, miniblocksHashes [][]byte) {
	panic("implement me")
}

func (rhs *RequestHandlerStub) RequestTrieNodes(destShardID uint32, hash []byte, topic string) {
	panic("implement me")
}

func (rhs *RequestHandlerStub) RequestShardHeader(shardId uint32, hash []byte) {
	if rhs.RequestShardHeaderCalled == nil {
		return
	}
	rhs.RequestShardHeaderCalled(shardId, hash)
}

func (rhs *RequestHandlerStub) RequestMetaHeader(hash []byte) {
	if rhs.RequestMetaHeaderCalled == nil {
		return
	}
	rhs.RequestMetaHeaderCalled(hash)
}

func (rhs *RequestHandlerStub) RequestMetaHeaderByNonce(nonce uint64) {
	if rhs.RequestMetaHeaderByNonceCalled == nil {
		return
	}
	rhs.RequestMetaHeaderByNonceCalled(nonce)
}

func (rhs *RequestHandlerStub) RequestShardHeaderByNonce(shardId uint32, nonce uint64) {
	if rhs.RequestShardHeaderByNonceCalled == nil {
		return
	}
	rhs.RequestShardHeaderByNonceCalled(shardId, nonce)
}

func (rhs *RequestHandlerStub) RequestTransaction(destShardID uint32, txHashes [][]byte) {
	if rhs.RequestTransactionHandlerCalled == nil {
		return
	}
	rhs.RequestTransactionHandlerCalled(destShardID, txHashes)
}

func (rhs *RequestHandlerStub) RequestUnsignedTransactions(destShardID uint32, txHashes [][]byte) {
	if rhs.RequestScrHandlerCalled == nil {
		return
	}
	rhs.RequestScrHandlerCalled(destShardID, txHashes)
}

func (rhs *RequestHandlerStub) RequestRewardTransactions(destShardID uint32, txHashes [][]byte) {
	if rhs.RequestRewardTxHandlerCalled == nil {
		return
	}
	rhs.RequestRewardTxHandlerCalled(destShardID, txHashes)
}

func (rhs *RequestHandlerStub) RequestMiniBlock(shardId uint32, miniblockHash []byte) {
	if rhs.RequestMiniBlockHandlerCalled == nil {
		return
	}
	rhs.RequestMiniBlockHandlerCalled(shardId, miniblockHash)
}

func (rhs *RequestHandlerStub) RequestStartOfEpochMetaBlock(epoch uint32) {
	if rhs.RequestStartOfEpochMetaBlockCalled != nil {
		rhs.RequestStartOfEpochMetaBlockCalled(epoch)
	}
}

// IsInterfaceNil returns true if there is no value under the interface
func (rhs *RequestHandlerStub) IsInterfaceNil() bool {
	return rhs == nil
}
