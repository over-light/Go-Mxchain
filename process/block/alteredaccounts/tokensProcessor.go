package alteredaccounts

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

const (
	idxTokenIDInTopics         = 0
	idxTokenNonceInTopics      = 1
	idxReceiverAddressInTopics = 3
)

type tokensProcessor struct {
	shardCoordinator            sharding.Coordinator
	fungibleTokensIdentifiers   map[string]struct{}
	nonFungibleTokensIdentifier map[string]struct{}
}

func newTokensProcessor(shardCoordinator sharding.Coordinator) *tokensProcessor {
	return &tokensProcessor{
		fungibleTokensIdentifiers: map[string]struct{}{
			core.BuiltInFunctionESDTTransfer:         {},
			core.BuiltInFunctionESDTBurn:             {},
			core.BuiltInFunctionESDTLocalMint:        {},
			core.BuiltInFunctionESDTLocalBurn:        {},
			core.BuiltInFunctionESDTWipe:             {},
			core.BuiltInFunctionMultiESDTNFTTransfer: {},
		},
		nonFungibleTokensIdentifier: map[string]struct{}{
			core.BuiltInFunctionESDTNFTTransfer:      {},
			core.BuiltInFunctionESDTNFTBurn:          {},
			core.BuiltInFunctionESDTNFTAddQuantity:   {},
			core.BuiltInFunctionESDTNFTCreate:        {},
			core.BuiltInFunctionMultiESDTNFTTransfer: {},
		},
		shardCoordinator: shardCoordinator,
	}
}

func (tp *tokensProcessor) extractESDTAccounts(
	txPool *indexer.Pool,
	markedAlteredAccounts map[string]*markedAlteredAccount,
) error {
	var err error
	for _, txLog := range txPool.Logs {
		for _, event := range txLog.LogHandler.GetLogEvents() {
			err = tp.processEvent(event, markedAlteredAccounts)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (tp *tokensProcessor) processEvent(
	event data.EventHandler,
	markedAlteredAccounts map[string]*markedAlteredAccount,
) error {
	_, isEsdtOperation := tp.fungibleTokensIdentifiers[string(event.GetIdentifier())]
	if isEsdtOperation {
		err := tp.extractEsdtData(event, zeroBigInt, markedAlteredAccounts)
		if err != nil {
			log.Debug("cannot extract esdt data", "error", err)
			return err
		}

		return nil
	}

	_, isNftOperation := tp.nonFungibleTokensIdentifier[string(event.GetIdentifier())]
	if isNftOperation {
		topics := event.GetTopics()
		if len(topics) == 0 {
			return nil
		}

		nonce := topics[idxTokenNonceInTopics]
		nonceBigInt := big.NewInt(0).SetBytes(nonce)
		err := tp.extractEsdtData(event, nonceBigInt, markedAlteredAccounts)
		if err != nil {
			log.Debug("cannot extract nft data", "error", err)
			return nil
		}

		return nil
	}

	return nil
}

func (tp *tokensProcessor) extractEsdtData(
	event data.EventHandler,
	nonce *big.Int,
	markedAlteredAccounts map[string]*markedAlteredAccount,
) error {
	address := event.GetAddress()
	topics := event.GetTopics()
	if len(topics) == 0 {
		return nil
	}

	// in case of esdt, nft or multi esdt transfers, the 3rd index of the topics contains the destination address
	tokenID := topics[idxTokenIDInTopics]
	err := tp.processEsdtDataForAddress(address, nonce, string(tokenID), markedAlteredAccounts)
	if err != nil {
		return err
	}

	if len(topics) > idxReceiverAddressInTopics {
		destinationAddress := topics[idxReceiverAddressInTopics]
		err = tp.processEsdtDataForAddress(destinationAddress, nonce, string(tokenID), markedAlteredAccounts)
		if err != nil {
			return err
		}
	}

	return nil
}

func (tp *tokensProcessor) processEsdtDataForAddress(
	address []byte,
	nonce *big.Int,
	tokenID string,
	markedAlteredAccounts map[string]*markedAlteredAccount,
) error {
	if !tp.isSameShard(address) {
		return nil
	}

	addressStr := string(address)
	_, exists := markedAlteredAccounts[addressStr]
	if !exists {
		markedAlteredAccounts[addressStr] = &markedAlteredAccount{}
	}

	markedAccount := markedAlteredAccounts[addressStr]
	if markedAccount.tokens == nil {
		markedAccount.tokens = make(map[string]*markedAlteredAccountToken)
	}

	tokenKey := tokenID + string(nonce.Bytes())
	_, alreadyExists := markedAccount.tokens[tokenKey]
	if alreadyExists {
		return nil
	}

	markedAccount.tokens[tokenKey] = &markedAlteredAccountToken{
		identifier: tokenID,
		nonce:      nonce.Uint64(),
	}

	return nil
}

func (tp *tokensProcessor) isSameShard(address []byte) bool {
	return tp.shardCoordinator.SelfId() == tp.shardCoordinator.ComputeId(address)
}
