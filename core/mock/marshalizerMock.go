package mock

import (
	"encoding/json"
	"errors"
)

var ErrMockMarshalizer = errors.New("MarshalizerMock generic error")
var ErrNilObjectToMarshal = errors.New("nil object to serialize from")

// MarshalizerMock that will be used for testing
type MarshalizerMock struct {
	Fail bool
}

// Marshal converts the input object in a slice of bytes
func (mm *MarshalizerMock) Marshal(obj interface{}) ([]byte, error) {
	if mm.Fail {
		return nil, ErrMockMarshalizer
	}

	if obj == nil {
		return nil, ErrNilObjectToMarshal
	}

	return json.Marshal(obj)
}

// Unmarshal applies the serialized values over an instantiated object
func (mm *MarshalizerMock) Unmarshal(obj interface{}, buff []byte) error {
	if mm.Fail {
		return ErrMockMarshalizer
	}

	if obj == nil {
		return errors.New("nil object to serilize to")
	}

	if buff == nil {
		return errors.New("nil byte buffer to deserialize from")
	}

	if len(buff) == 0 {
		return errors.New("empty byte buffer to deserialize from")
	}

	return json.Unmarshal(buff, obj)
}
