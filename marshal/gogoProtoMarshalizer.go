package marshal

import (
	"github.com/ElrondNetwork/elrond-go/data"
)

// GogoProtoMarshalizer implements marshaling with protobuf
type GogoProtoMarshalizer struct {
}

// Marshal does the actual serialization of an object
// The object to be serialized must implement the gogoProtoObj interface
func (x *GogoProtoMarshalizer) Marshal(obj interface{}) ([]byte, error) {
	if msg, ok := obj.(data.GogoProtoObj); ok {
		return msg.Marshal()
	}

	return nil, ErrMarshallingProto
}

// Unmarshal does the actual deserialization of an object
// The object to be deserialized must implement the gogoProtoObj interface
func (x *GogoProtoMarshalizer) Unmarshal(obj interface{}, buff []byte) error {
	if msg, ok := obj.(data.GogoProtoObj); ok {
		msg.Reset()
		err := msg.Unmarshal(buff)
		if err != nil {
			panic(err)
		}
		return err
	}
	return ErrUnmarshallingProto

}

// IsInterfaceNil returns true if there is no value under the interface
func (x *GogoProtoMarshalizer) IsInterfaceNil() bool {
	return x == nil
}
