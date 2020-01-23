package mock

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/golang/protobuf/proto"
)

var TestingMarshalizers = map[string]marshal.Marshalizer{
	"capnp": &CapnpMarshalizer{},
	"json":  &JsonMarshalizer{},
	"proto": &ProtobufMarshalizer{},
}

//-------- capnp

type CapnpMarshalizer struct{}

func (x *CapnpMarshalizer) Marshal(obj interface{}) ([]byte, error) {
	out := bytes.NewBuffer(nil)

	o := obj.(marshal.CapnpHelper)
	// set the members to capnp struct
	err := o.Save(out)

	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func (x *CapnpMarshalizer) Unmarshal(obj interface{}, buff []byte) error {
	out := bytes.NewBuffer(buff)

	o := obj.(marshal.CapnpHelper)
	// set the members to capnp struct
	err := o.Load(out)

	return err
}

func (x *CapnpMarshalizer) IsInterfaceNil() bool {
	return x == nil
}

//-------- Json

type JsonMarshalizer struct{}

func (j JsonMarshalizer) Marshal(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nil, errors.New("nil object to serialize from")
	}

	return json.Marshal(obj)
}

func (j JsonMarshalizer) Unmarshal(obj interface{}, buff []byte) error {
	if obj == nil {
		return errors.New("nil object to serialize to")
	}
	if buff == nil {
		return errors.New("nil byte buffer to deserialize from")
	}
	if len(buff) == 0 {
		return errors.New("empty byte buffer to deserialize from")
	}

	return json.Unmarshal(buff, obj)
}

func (j *JsonMarshalizer) IsInterfaceNil() bool {
	return j == nil
}

//------- protobuf

type ProtobufMarshalizer struct{}

func (x *ProtobufMarshalizer) Marshal(obj interface{}) ([]byte, error) {
	if msg, ok := obj.(proto.Message); ok {
		enc, err := proto.Marshal(msg)
		if err != nil {
			return nil, err
		}
		return enc, nil
	}
	return nil, errors.New("can not serialize the object")
}

func (x *ProtobufMarshalizer) Unmarshal(obj interface{}, buff []byte) error {
	if msg, ok := obj.(proto.Message); ok {
		return proto.Unmarshal(buff, msg)
	}
	return errors.New("obj does not implement proto.Message")
}

func (x *ProtobufMarshalizer) IsInterfaceNil() bool {
	return x == nil
}

//------- stub

type MarshalizerStub struct {
	MarshalCalled   func(obj interface{}) ([]byte, error)
	UnmarshalCalled func(obj interface{}, buff []byte) error
}

func (ms *MarshalizerStub) Marshal(obj interface{}) ([]byte, error) {
	return ms.MarshalCalled(obj)
}

func (ms *MarshalizerStub) Unmarshal(obj interface{}, buff []byte) error {
	return ms.UnmarshalCalled(obj, buff)
}

func (ms *MarshalizerStub) IsInterfaceNil() bool {
	return ms == nil
}
