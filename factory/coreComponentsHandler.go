package factory

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/storage"
)

var _ ComponentHandler = (*managedCoreComponents)(nil)
var _ CoreComponentsHolder = (*managedCoreComponents)(nil)
var _ CoreComponentsHandler = (*managedCoreComponents)(nil)

// CoreComponentsHandlerArgs holds the arguments required to create a core components handler
type CoreComponentsHandlerArgs CoreComponentsFactoryArgs

// managedCoreComponents is an implementation of core components handler that can create, close and access the core components
type managedCoreComponents struct {
	coreComponentsFactory *coreComponentsFactory
	*coreComponents
	mutCoreComponents sync.RWMutex
}

// NewManagedCoreComponents creates a new core components handler implementation
func NewManagedCoreComponents(args CoreComponentsHandlerArgs) (*managedCoreComponents, error) {
	coreComponentsFactory := NewCoreComponentsFactory(CoreComponentsFactoryArgs(args))
	mcc := &managedCoreComponents{
		coreComponents:        nil,
		coreComponentsFactory: coreComponentsFactory,
	}
	return mcc, nil
}

// Create creates the core components
func (mcc *managedCoreComponents) Create() error {
	cc, err := mcc.coreComponentsFactory.Create()
	if err != nil {
		return err
	}

	mcc.mutCoreComponents.Lock()
	mcc.coreComponents = cc
	mcc.mutCoreComponents.Unlock()

	return nil
}

// Close closes the managed core components
func (mcc *managedCoreComponents) Close() error {
	mcc.mutCoreComponents.Lock()
	mcc.coreComponents.statusHandler.Close()
	mcc.coreComponents = nil
	mcc.mutCoreComponents.Unlock()

	return nil
}

// InternalMarshalizer returns the core components internal marshalizer
func (mcc *managedCoreComponents) InternalMarshalizer() marshal.Marshalizer {
	mcc.mutCoreComponents.RLock()
	defer mcc.mutCoreComponents.RUnlock()

	if mcc.coreComponents == nil {
		return nil
	}

	return mcc.coreComponents.internalMarshalizer
}

// SetInternalMarshalizer sets the internal marshalizer to the one given as parameter
func (mcc *managedCoreComponents) SetInternalMarshalizer(m marshal.Marshalizer) error {
	mcc.mutCoreComponents.Lock()
	defer mcc.mutCoreComponents.Unlock()

	if mcc.coreComponents == nil {
		return ErrNilCoreComponents
	}

	mcc.coreComponents.internalMarshalizer = m

	return nil
}

// TxMarshalizer returns the core components tx marshalizer
func (mcc *managedCoreComponents) TxMarshalizer() marshal.Marshalizer {
	mcc.mutCoreComponents.RLock()
	defer mcc.mutCoreComponents.RUnlock()

	if mcc.coreComponents == nil {
		return nil
	}

	return mcc.coreComponents.txSignMarshalizer
}

// VmMarshalizer returns the core components vm marshalizer
func (mcc *managedCoreComponents) VmMarshalizer() marshal.Marshalizer {
	mcc.mutCoreComponents.RLock()
	defer mcc.mutCoreComponents.RUnlock()

	if mcc.coreComponents == nil {
		return nil
	}

	return mcc.coreComponents.vmMarshalizer
}

// Hasher returns the core components Hasher
func (mcc *managedCoreComponents) Hasher() hashing.Hasher {
	mcc.mutCoreComponents.RLock()
	defer mcc.mutCoreComponents.RUnlock()

	if mcc.coreComponents == nil {
		return nil
	}

	return mcc.coreComponents.hasher
}

// Uint64ByteSliceConverter returns the core component converter between a byte slice and uint64
func (mcc *managedCoreComponents) Uint64ByteSliceConverter() typeConverters.Uint64ByteSliceConverter {
	mcc.mutCoreComponents.RLock()
	defer mcc.mutCoreComponents.RUnlock()

	if mcc.coreComponents == nil {
		return nil
	}

	return mcc.coreComponents.uint64ByteSliceConverter
}

// AddressPubKeyConverter returns the address to public key converter
func (mcc *managedCoreComponents) AddressPubKeyConverter() state.PubkeyConverter {
	mcc.mutCoreComponents.RLock()
	defer mcc.mutCoreComponents.RUnlock()

	if mcc.coreComponents == nil {
		return nil
	}

	return mcc.coreComponents.addressPubKeyConverter
}

// ValidatorPubKeyConverter returns the validator public key converter
func (mcc *managedCoreComponents) ValidatorPubKeyConverter() state.PubkeyConverter {
	mcc.mutCoreComponents.RLock()
	defer mcc.mutCoreComponents.RUnlock()

	if mcc.coreComponents == nil {
		return nil
	}

	return mcc.coreComponents.validatorPubKeyConverter
}

// StatusHandler returns the core components status handler
func (mcc *managedCoreComponents) StatusHandler() core.AppStatusHandler {
	mcc.mutCoreComponents.RLock()
	defer mcc.mutCoreComponents.RUnlock()

	if mcc.coreComponents == nil {
		return nil
	}

	return mcc.coreComponents.statusHandler
}

// SetStatusHandler allows the change of the status handler
func (mcc *managedCoreComponents) SetStatusHandler(statusHandler core.AppStatusHandler) error {
	if check.IfNil(statusHandler) {
		return ErrNilStatusHandler
	}

	mcc.mutCoreComponents.Lock()
	defer mcc.mutCoreComponents.Unlock()

	if mcc.coreComponents == nil {
		return ErrNilCoreComponents
	}

	mcc.coreComponents.statusHandler = statusHandler

	return nil
}

// PathHandler returns the core components path handler
func (mcc *managedCoreComponents) PathHandler() storage.PathManagerHandler {
	mcc.mutCoreComponents.RLock()
	defer mcc.mutCoreComponents.RUnlock()

	if mcc.coreComponents == nil {
		return nil
	}

	return mcc.coreComponents.pathHandler
}

// ChainID returns the core components chainID
func (mcc *managedCoreComponents) ChainID() string {
	mcc.mutCoreComponents.RLock()
	defer mcc.mutCoreComponents.RUnlock()

	if mcc.coreComponents == nil {
		return ""
	}

	return mcc.coreComponents.chainID
}

// IsInterfaceNil returns true if there is no value under the interface
func (mcc *managedCoreComponents) IsInterfaceNil() bool {
	return mcc == nil
}
