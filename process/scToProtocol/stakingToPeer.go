package scToProtocol

import (
	"bytes"
	"math/big"
	"sort"
	"sync"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/batch"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/node/external"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/vm/factory"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

// ArgStakingToPeer is struct that contain all components that are needed to create a new stakingToPeer object
type ArgStakingToPeer struct {
	AdrConv          state.AddressConverter
	Hasher           hashing.Hasher
	ProtoMarshalizer marshal.Marshalizer
	VmMarshalizer    marshal.Marshalizer
	PeerState        state.AccountsAdapter
	BaseState        state.AccountsAdapter

	ArgParser process.ArgumentsParser
	CurrTxs   dataRetriever.TransactionCacher
	ScQuery   external.SCQueryService
}

// stakingToPeer defines the component which will translate changes from staking SC state
// to validator statistics trie
type stakingToPeer struct {
	adrConv          state.AddressConverter
	hasher           hashing.Hasher
	protoMarshalizer marshal.Marshalizer
	vmMarshalizer    marshal.Marshalizer
	peerState        state.AccountsAdapter
	baseState        state.AccountsAdapter

	argParser process.ArgumentsParser
	currTxs   dataRetriever.TransactionCacher
	scQuery   external.SCQueryService

	mutPeerChanges sync.Mutex
	peerChanges    map[string]block.PeerData
}

// NewStakingToPeer creates the component which moves from staking sc state to peer state
func NewStakingToPeer(args ArgStakingToPeer) (*stakingToPeer, error) {
	err := checkIfNil(args)
	if err != nil {
		return nil, err
	}

	st := &stakingToPeer{
		adrConv:          args.AdrConv,
		hasher:           args.Hasher,
		protoMarshalizer: args.ProtoMarshalizer,
		vmMarshalizer:    args.VmMarshalizer,
		peerState:        args.PeerState,
		baseState:        args.BaseState,
		argParser:        args.ArgParser,
		currTxs:          args.CurrTxs,
		scQuery:          args.ScQuery,
		mutPeerChanges:   sync.Mutex{},
		peerChanges:      make(map[string]block.PeerData),
	}

	return st, nil
}

func checkIfNil(args ArgStakingToPeer) error {
	if args.AdrConv == nil || args.AdrConv.IsInterfaceNil() {
		return process.ErrNilAddressConverter
	}
	if args.Hasher == nil || args.Hasher.IsInterfaceNil() {
		return process.ErrNilHasher
	}
	if args.ProtoMarshalizer == nil || args.ProtoMarshalizer.IsInterfaceNil() {
		return process.ErrNilMarshalizer
	}
	if args.VmMarshalizer == nil || args.VmMarshalizer.IsInterfaceNil() {
		return process.ErrNilMarshalizer
	}
	if args.PeerState == nil || args.PeerState.IsInterfaceNil() {
		return process.ErrNilPeerAccountsAdapter
	}
	if args.BaseState == nil || args.BaseState.IsInterfaceNil() {
		return process.ErrNilAccountsAdapter
	}
	if args.ArgParser == nil || args.ArgParser.IsInterfaceNil() {
		return process.ErrNilArgumentParser
	}
	if args.CurrTxs == nil || args.CurrTxs.IsInterfaceNil() {
		return process.ErrNilTxForCurrentBlockHandler
	}
	if args.ScQuery == nil || args.ScQuery.IsInterfaceNil() {
		return process.ErrNilSCDataGetter
	}

	return nil
}

func (stp *stakingToPeer) getPeerAccount(key []byte) (*state.PeerAccount, error) {
	adrSrc, err := stp.adrConv.CreateAddressFromPublicKeyBytes(key)
	if err != nil {
		return nil, err
	}

	account, err := stp.peerState.GetAccountWithJournal(adrSrc)
	if err != nil {
		return nil, err
	}

	peerAcc, ok := account.(*state.PeerAccount)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	return peerAcc, nil
}

// UpdateProtocol applies changes from staking smart contract to peer state and creates the actual peer changes
func (stp *stakingToPeer) UpdateProtocol(body *block.Body, nonce uint64) error {
	stp.mutPeerChanges.Lock()
	stp.peerChanges = make(map[string]block.PeerData)
	stp.mutPeerChanges.Unlock()

	affectedStates, err := stp.getAllModifiedStates(body)
	if err != nil {
		return err
	}

	for _, key := range affectedStates {
		blsPubKey := []byte(key)
		var peerAcc *state.PeerAccount
		peerAcc, err = stp.getPeerAccount(blsPubKey)
		if err != nil {
			return err
		}

		query := process.SCQuery{
			ScAddress: factory.StakingSCAddress,
			FuncName:  "get",
			Arguments: [][]byte{blsPubKey},
		}
		var vmOutput *vmcommon.VMOutput
		vmOutput, err = stp.scQuery.ExecuteQuery(&query)
		if err != nil {
			return err
		}

		var data []byte
		if len(vmOutput.ReturnData) > 0 {
			data = vmOutput.ReturnData[0]
		}
		// no data under key -> peer can be deleted from trie
		if len(data) == 0 {
			err = stp.peerUnregistered(peerAcc, nonce)
			if err != nil {
				return err
			}

			var adrSrc state.AddressContainer
			adrSrc, err = stp.adrConv.CreateAddressFromPublicKeyBytes(blsPubKey)
			if err != nil {
				return err
			}

			err = stp.peerState.RemoveAccount(adrSrc)
			if err != nil {
				return err
			}

			continue
		}

		var stakingData systemSmartContracts.StakingData
		err = stp.vmMarshalizer.Unmarshal(&stakingData, data)
		if err != nil {
			return err
		}

		err = stp.createPeerChangeData(stakingData, peerAcc, nonce, blsPubKey)
		if err != nil {
			return err
		}

		err = stp.updatePeerState(stakingData, peerAcc, blsPubKey)
		if err != nil {
			return err
		}
	}

	return nil
}

func (stp *stakingToPeer) peerUnregistered(account *state.PeerAccount, nonce uint64) error {
	stp.mutPeerChanges.Lock()
	defer stp.mutPeerChanges.Unlock()

	actualPeerChange := block.PeerData{
		Address:     account.RewardAddress,
		PublicKey:   account.BLSPublicKey,
		Action:      block.PeerDeregistration,
		TimeStamp:   nonce,
		ValueChange: account.Stake,
	}

	peerHash, err := core.CalculateHash(stp.protoMarshalizer, stp.hasher, &actualPeerChange)
	if err != nil {
		return err
	}

	stp.peerChanges[string(peerHash)] = actualPeerChange
	return nil
}

func (stp *stakingToPeer) updatePeerState(
	stakingData systemSmartContracts.StakingData,
	account *state.PeerAccount,
	blsPubKey []byte,
) error {
	if !bytes.Equal(stakingData.Address, account.RewardAddress) {
		err := account.SetRewardAddressWithJournal(stakingData.Address)
		if err != nil {
			return err
		}
	}

	if !bytes.Equal(blsPubKey, account.BLSPublicKey) {
		err := account.SetBLSPublicKeyWithJournal(blsPubKey)
		if err != nil {
			return err
		}
	}

	if stakingData.StakeValue.Cmp(account.Stake) != 0 {
		err := account.SetStakeWithJournal(stakingData.StakeValue)
		if err != nil {
			return err
		}
	}

	if stakingData.StartNonce != account.Nonce {
		err := account.SetNonceWithJournal(stakingData.StartNonce)
		if err != nil {
			return err
		}

		err = account.SetNodeInWaitingListWithJournal(true)
		if err != nil {
			return err
		}
	}

	if stakingData.UnStakedNonce != account.UnStakedNonce {
		err := account.SetUnStakedNonceWithJournal(stakingData.UnStakedNonce)
		if err != nil {
			return err
		}
	}

	return nil
}

func (stp *stakingToPeer) createPeerChangeData(
	stakingData systemSmartContracts.StakingData,
	account *state.PeerAccount,
	nonce uint64,
	blsKey []byte,
) error {
	stp.mutPeerChanges.Lock()
	defer stp.mutPeerChanges.Unlock()

	actualPeerChange := block.PeerData{
		Address:     account.RewardAddress,
		PublicKey:   account.BLSPublicKey,
		Action:      0,
		TimeStamp:   nonce,
		ValueChange: big.NewInt(0),
	}

	if len(account.RewardAddress) == 0 {
		actualPeerChange.Action = block.PeerRegistration
		actualPeerChange.TimeStamp = stakingData.StartNonce
		actualPeerChange.ValueChange.Set(stakingData.StakeValue)
		actualPeerChange.Address = stakingData.Address
		actualPeerChange.PublicKey = blsKey

		peerHash, err := core.CalculateHash(stp.protoMarshalizer, stp.hasher, &actualPeerChange)
		if err != nil {
			return err
		}

		stp.peerChanges[string(peerHash)] = actualPeerChange

		return nil
	}

	if account.Stake.Cmp(stakingData.StakeValue) != 0 {
		actualPeerChange.ValueChange.Sub(account.Stake, stakingData.StakeValue)
		if account.Stake.Cmp(stakingData.StakeValue) < 0 {
			actualPeerChange.Action = block.PeerSlashed
		} else {
			actualPeerChange.Action = block.PeerReStake
		}
	}

	if stakingData.StartNonce == nonce {
		actualPeerChange.Action = block.PeerRegistration
	}

	if stakingData.UnStakedNonce == nonce {
		actualPeerChange.Action = block.PeerUnstaking
	}

	peerHash, err := core.CalculateHash(stp.protoMarshalizer, stp.hasher, &actualPeerChange)
	if err != nil {
		return err
	}

	stp.peerChanges[string(peerHash)] = actualPeerChange

	return nil
}

func (stp *stakingToPeer) getAllModifiedStates(body *block.Body) ([]string, error) {
	affectedStates := make([]string, 0)

	for _, miniBlock := range body.MiniBlocks {
		if miniBlock.Type != block.SmartContractResultBlock {
			continue
		}
		if miniBlock.SenderShardID != core.MetachainShardId {
			continue
		}

		for _, txHash := range miniBlock.TxHashes {
			tx, err := stp.currTxs.GetTx(txHash)
			if err != nil {
				continue
			}

			if !bytes.Equal(tx.GetRcvAddr(), factory.StakingSCAddress) {
				continue
			}

			scr, ok := tx.(*smartContractResult.SmartContractResult)
			if !ok {
				return nil, process.ErrWrongTypeAssertion
			}

			storageUpdates, err := stp.argParser.GetStorageUpdates(string(scr.Data))
			if err != nil {
				return nil, err
			}

			for _, storageUpdate := range storageUpdates {
				affectedStates = append(affectedStates, string(storageUpdate.Offset))
			}
		}
	}

	return affectedStates, nil
}

// PeerChanges returns peer changes created in current round
func (stp *stakingToPeer) PeerChanges() []block.PeerData {
	stp.mutPeerChanges.Lock()
	peersData := make([]block.PeerData, 0, len(stp.peerChanges))
	for _, peerData := range stp.peerChanges {
		peersData = append(peersData, peerData)
	}
	stp.mutPeerChanges.Unlock()

	sort.Slice(peersData, func(i, j int) bool {
		return string(peersData[i].Address) < string(peersData[j].Address)
	})

	return peersData
}

func (stp *stakingToPeer) batchPeerData(pc []block.PeerData) (*batch.Batch, error) {
	mrsPc := make([][]byte, len(pc))
	for i := range pc {
		var err error
		mrsPc[i], err = stp.protoMarshalizer.Marshal(&pc[i])
		if err != nil {
			return nil, err
		}
	}
	return &batch.Batch{Data: mrsPc}, nil
}

// VerifyPeerChanges verifies if peer changes from header is the same as the one created while processing
func (stp *stakingToPeer) VerifyPeerChanges(peerChanges []block.PeerData) error {
	createdPeersData := stp.PeerChanges()
	bcpd, err := stp.batchPeerData(createdPeersData)
	if err != nil {
		return err
	}

	createdHash, err := core.CalculateHash(stp.protoMarshalizer, stp.hasher, bcpd)
	if err != nil {
		return err
	}

	bpd, err := stp.batchPeerData(peerChanges)
	if err != nil {
		return err
	}

	receivedHash, err := core.CalculateHash(stp.protoMarshalizer, stp.hasher, bpd)
	if err != nil {
		return err
	}

	if !bytes.Equal(createdHash, receivedHash) {
		return process.ErrPeerChangesHashDoesNotMatch
	}

	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (stp *stakingToPeer) IsInterfaceNil() bool {
	if stp == nil {
		return true
	}
	return false
}
