package mock

import (
	"sync"
	"time"

	nodeFactory "github.com/ElrondNetwork/elrond-go/cmd/node/factory"
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/endProcess"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/factory"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/ntp"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
)

// CoreComponentsMock -
type CoreComponentsMock struct {
	IntMarsh                    marshal.Marshalizer
	TxMarsh                     marshal.Marshalizer
	VmMarsh                     marshal.Marshalizer
	Hash                        hashing.Hasher
	TxSignHasherField           hashing.Hasher
	UInt64ByteSliceConv         typeConverters.Uint64ByteSliceConverter
	AddrPubKeyConv              core.PubkeyConverter
	ValPubKeyConv               core.PubkeyConverter
	StatusHdlUtils              nodeFactory.StatusHandlersUtils
	AppStatusHdl                core.AppStatusHandler
	mutStatus                   sync.RWMutex
	PathHdl                     storage.PathManagerHandler
	WatchdogTimer               core.WatchdogTimer
	AlarmSch                    core.TimersScheduler
	NtpSyncTimer                ntp.SyncTimer
	GenesisBlockTime            time.Time
	ChainIdCalled               func() string
	MinTransactionVersionCalled func() uint32
	mutIntMarshalizer           sync.RWMutex
	RoundHandlerField           consensus.RoundHandler
	EconomicsHandler            process.EconomicsHandler
	RatingsConfig               process.RatingsInfoHandler
	RatingHandler               sharding.PeerAccountListAndRatingHandler
	NodesConfig                 sharding.GenesisNodesSetupHandler
	Shuffler                    sharding.NodesShuffler
	EpochChangeNotifier         process.EpochNotifier
	EpochNotifierWithConfirm    factory.EpochStartNotifierWithConfirm
	TxVersionCheckHandler       process.TxVersionCheckerHandler
	ChanStopProcess             chan endProcess.ArgEndProcess
	StartTime                   time.Time
}

// InternalMarshalizer -
func (ccm *CoreComponentsMock) InternalMarshalizer() marshal.Marshalizer {
	ccm.mutIntMarshalizer.RLock()
	defer ccm.mutIntMarshalizer.RUnlock()

	return ccm.IntMarsh
}

// SetInternalMarshalizer -
func (ccm *CoreComponentsMock) SetInternalMarshalizer(m marshal.Marshalizer) error {
	ccm.mutIntMarshalizer.Lock()
	ccm.IntMarsh = m
	ccm.mutIntMarshalizer.Unlock()

	return nil
}

// TxMarshalizer -
func (ccm *CoreComponentsMock) TxMarshalizer() marshal.Marshalizer {
	return ccm.TxMarsh
}

// VmMarshalizer -
func (ccm *CoreComponentsMock) VmMarshalizer() marshal.Marshalizer {
	return ccm.VmMarsh
}

// Hasher -
func (ccm *CoreComponentsMock) Hasher() hashing.Hasher {
	return ccm.Hash
}

// TxSignHasher -
func (ccm *CoreComponentsMock) TxSignHasher() hashing.Hasher {
	return ccm.TxSignHasherField
}

// Uint64ByteSliceConverter -
func (ccm *CoreComponentsMock) Uint64ByteSliceConverter() typeConverters.Uint64ByteSliceConverter {
	return ccm.UInt64ByteSliceConv
}

// AddressPubKeyConverter -
func (ccm *CoreComponentsMock) AddressPubKeyConverter() core.PubkeyConverter {
	return ccm.AddrPubKeyConv
}

// ValidatorPubKeyConverter -
func (ccm *CoreComponentsMock) ValidatorPubKeyConverter() core.PubkeyConverter {
	return ccm.ValPubKeyConv
}

// StatusHandlerUtils -
func (ccm *CoreComponentsMock) StatusHandlerUtils() nodeFactory.StatusHandlersUtils {
	ccm.mutStatus.RLock()
	defer ccm.mutStatus.RUnlock()

	return ccm.StatusHdlUtils
}

// StatusHandler -
func (ccm *CoreComponentsMock) StatusHandler() core.AppStatusHandler {
	ccm.mutStatus.RLock()
	defer ccm.mutStatus.RUnlock()

	return ccm.AppStatusHdl
}

// PathHandler -
func (ccm *CoreComponentsMock) PathHandler() storage.PathManagerHandler {
	return ccm.PathHdl
}

// Watchdog -
func (ccm *CoreComponentsMock) Watchdog() core.WatchdogTimer {
	return ccm.WatchdogTimer
}

// AlarmScheduler -
func (ccm *CoreComponentsMock) AlarmScheduler() core.TimersScheduler {
	return ccm.AlarmSch
}

// SyncTimer -
func (ccm *CoreComponentsMock) SyncTimer() ntp.SyncTimer {
	return ccm.NtpSyncTimer
}

// GenesisTime -
func (ccm *CoreComponentsMock) GenesisTime() time.Time {
	return ccm.GenesisBlockTime
}

// ChainID -
func (ccm *CoreComponentsMock) ChainID() string {
	if ccm.ChainIdCalled != nil {
		return ccm.ChainIdCalled()
	}
	return "undefined"
}

// MinTransactionVersion -
func (ccm *CoreComponentsMock) MinTransactionVersion() uint32 {
	if ccm.MinTransactionVersionCalled != nil {
		return ccm.MinTransactionVersionCalled()
	}
	return 1
}

// TxVersionChecker -
func (ccm *CoreComponentsMock) TxVersionChecker() process.TxVersionCheckerHandler {
	return ccm.TxVersionCheckHandler
}

// RoundHandler -
func (ccm *CoreComponentsMock) RoundHandler() consensus.RoundHandler {
	return ccm.RoundHandlerField
}

// EconomicsData -
func (ccm *CoreComponentsMock) EconomicsData() process.EconomicsHandler {
	return ccm.EconomicsHandler
}

// RatingsData -
func (ccm *CoreComponentsMock) RatingsData() process.RatingsInfoHandler {
	return ccm.RatingsConfig
}

// Rater -
func (ccm *CoreComponentsMock) Rater() sharding.PeerAccountListAndRatingHandler {
	return ccm.RatingHandler
}

// GenesisNodesSetup -
func (ccm *CoreComponentsMock) GenesisNodesSetup() sharding.GenesisNodesSetupHandler {
	return ccm.NodesConfig
}

// NodesShuffler -
func (ccm *CoreComponentsMock) NodesShuffler() sharding.NodesShuffler {
	return ccm.Shuffler
}

// EpochNotifier -
func (ccm *CoreComponentsMock) EpochNotifier() process.EpochNotifier {
	return ccm.EpochChangeNotifier
}

// EpochStartNotifierWithConfirm -
func (ccm *CoreComponentsMock) EpochStartNotifierWithConfirm() factory.EpochStartNotifierWithConfirm {
	return ccm.EpochNotifierWithConfirm
}

// ChanStopNodeProcess -
func (ccm *CoreComponentsMock) ChanStopNodeProcess() chan endProcess.ArgEndProcess {
	return ccm.ChanStopProcess
}

// IsInterfaceNil -
func (ccm *CoreComponentsMock) IsInterfaceNil() bool {
	return ccm == nil
}
