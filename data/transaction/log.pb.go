// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: log.proto

package transaction

import (
	bytes "bytes"
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
	reflect "reflect"
	strings "strings"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// Event holds all the data needed for an event structure
type Event struct {
	Address    []byte   `protobuf:"bytes,1,opt,name=Address,proto3" json:"address"`
	Identifier []byte   `protobuf:"bytes,2,opt,name=Identifier,proto3" json:"identifier"`
	Topics     [][]byte `protobuf:"bytes,3,rep,name=Topics,proto3" json:"topics"`
	Data       []byte   `protobuf:"bytes,4,opt,name=data,proto3" json:"data"`
}

func (m *Event) Reset()      { *m = Event{} }
func (*Event) ProtoMessage() {}
func (*Event) Descriptor() ([]byte, []int) {
	return fileDescriptor_a153da538f858886, []int{0}
}
func (m *Event) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Event) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *Event) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Event.Merge(m, src)
}
func (m *Event) XXX_Size() int {
	return m.Size()
}
func (m *Event) XXX_DiscardUnknown() {
	xxx_messageInfo_Event.DiscardUnknown(m)
}

var xxx_messageInfo_Event proto.InternalMessageInfo

func (m *Event) GetAddress() []byte {
	if m != nil {
		return m.Address
	}
	return nil
}

func (m *Event) GetIdentifier() []byte {
	if m != nil {
		return m.Identifier
	}
	return nil
}

func (m *Event) GetTopics() [][]byte {
	if m != nil {
		return m.Topics
	}
	return nil
}

func (m *Event) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

// Log holds all the data needed for a log structure
type Log struct {
	Address []byte   `protobuf:"bytes,1,opt,name=Address,proto3" json:"address"`
	Events  []*Event `protobuf:"bytes,2,rep,name=Events,proto3" json:"events"`
}

func (m *Log) Reset()      { *m = Log{} }
func (*Log) ProtoMessage() {}
func (*Log) Descriptor() ([]byte, []int) {
	return fileDescriptor_a153da538f858886, []int{1}
}
func (m *Log) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Log) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *Log) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Log.Merge(m, src)
}
func (m *Log) XXX_Size() int {
	return m.Size()
}
func (m *Log) XXX_DiscardUnknown() {
	xxx_messageInfo_Log.DiscardUnknown(m)
}

var xxx_messageInfo_Log proto.InternalMessageInfo

func (m *Log) GetAddress() []byte {
	if m != nil {
		return m.Address
	}
	return nil
}

func (m *Log) GetEvents() []*Event {
	if m != nil {
		return m.Events
	}
	return nil
}

func init() {
	proto.RegisterType((*Event)(nil), "proto.Event")
	proto.RegisterType((*Log)(nil), "proto.Log")
}

func init() { proto.RegisterFile("log.proto", fileDescriptor_a153da538f858886) }

var fileDescriptor_a153da538f858886 = []byte{
	// 303 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x8f, 0xbf, 0x4e, 0xf3, 0x30,
	0x14, 0xc5, 0xed, 0xa6, 0x4d, 0xbf, 0xcf, 0xad, 0x18, 0x32, 0x45, 0x08, 0xdd, 0x54, 0x91, 0x90,
	0xb2, 0x90, 0x22, 0x78, 0x02, 0x22, 0x3a, 0x20, 0x31, 0x45, 0x4c, 0x0c, 0x48, 0xce, 0x9f, 0x06,
	0x4b, 0x10, 0x57, 0x89, 0xcb, 0xcc, 0x23, 0xf0, 0x08, 0x8c, 0x3c, 0x0a, 0x63, 0xc6, 0x4c, 0x11,
	0x71, 0x16, 0x94, 0xa9, 0x8f, 0x80, 0x74, 0x53, 0x10, 0x23, 0x93, 0x7d, 0x7f, 0xe7, 0xd8, 0xe7,
	0x1e, 0xf6, 0xff, 0x41, 0x66, 0xfe, 0xa6, 0x90, 0x4a, 0x5a, 0x13, 0x3c, 0x0e, 0x4f, 0x32, 0xa1,
	0xee, 0xb7, 0x91, 0x1f, 0xcb, 0xc7, 0x65, 0x26, 0x33, 0xb9, 0x44, 0x1c, 0x6d, 0xd7, 0x38, 0xe1,
	0x80, 0xb7, 0xe1, 0x95, 0xfb, 0x4a, 0xd9, 0x64, 0xf5, 0x94, 0xe6, 0xca, 0x3a, 0x66, 0xd3, 0x8b,
	0x24, 0x29, 0xd2, 0xb2, 0xb4, 0xe9, 0x82, 0x7a, 0xf3, 0x60, 0xd6, 0x37, 0xce, 0x94, 0x0f, 0x28,
	0xfc, 0xd6, 0x2c, 0x9f, 0xb1, 0xab, 0x24, 0xcd, 0x95, 0x58, 0x8b, 0xb4, 0xb0, 0x47, 0xe8, 0x3c,
	0xe8, 0x1b, 0x87, 0x89, 0x1f, 0x1a, 0xfe, 0x72, 0x58, 0x2e, 0x33, 0x6f, 0xe4, 0x46, 0xc4, 0xa5,
	0x6d, 0x2c, 0x0c, 0x6f, 0x1e, 0xb0, 0xbe, 0x71, 0x4c, 0x85, 0x24, 0xdc, 0x2b, 0xd6, 0x11, 0x1b,
	0x5f, 0x72, 0xc5, 0xed, 0x31, 0xfe, 0xf6, 0xaf, 0x6f, 0x9c, 0x71, 0xc2, 0x15, 0x0f, 0x91, 0xba,
	0x77, 0xcc, 0xb8, 0x96, 0xd9, 0x5f, 0xf7, 0x3b, 0x65, 0x26, 0xf6, 0x29, 0xed, 0xd1, 0xc2, 0xf0,
	0x66, 0x67, 0xf3, 0xa1, 0xa8, 0x8f, 0x70, 0x48, 0x4f, 0x51, 0x0f, 0xf7, 0xbe, 0x60, 0x55, 0xb5,
	0x40, 0xea, 0x16, 0xc8, 0xae, 0x05, 0xfa, 0xac, 0x81, 0xbe, 0x69, 0xa0, 0xef, 0x1a, 0x68, 0xa5,
	0x81, 0xd6, 0x1a, 0xe8, 0x87, 0x06, 0xfa, 0xa9, 0x81, 0xec, 0x34, 0xd0, 0x97, 0x0e, 0x48, 0xd5,
	0x01, 0xa9, 0x3b, 0x20, 0xb7, 0x33, 0x55, 0xf0, 0xbc, 0xe4, 0xb1, 0x12, 0x32, 0x8f, 0x4c, 0xcc,
	0x39, 0xff, 0x0a, 0x00, 0x00, 0xff, 0xff, 0x4b, 0x55, 0xcf, 0x0b, 0x93, 0x01, 0x00, 0x00,
}

func (this *Event) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Event)
	if !ok {
		that2, ok := that.(Event)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if !bytes.Equal(this.Address, that1.Address) {
		return false
	}
	if !bytes.Equal(this.Identifier, that1.Identifier) {
		return false
	}
	if len(this.Topics) != len(that1.Topics) {
		return false
	}
	for i := range this.Topics {
		if !bytes.Equal(this.Topics[i], that1.Topics[i]) {
			return false
		}
	}
	if !bytes.Equal(this.Data, that1.Data) {
		return false
	}
	return true
}
func (this *Log) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Log)
	if !ok {
		that2, ok := that.(Log)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if !bytes.Equal(this.Address, that1.Address) {
		return false
	}
	if len(this.Events) != len(that1.Events) {
		return false
	}
	for i := range this.Events {
		if !this.Events[i].Equal(that1.Events[i]) {
			return false
		}
	}
	return true
}
func (this *Event) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 8)
	s = append(s, "&transaction.Event{")
	s = append(s, "Address: "+fmt.Sprintf("%#v", this.Address)+",\n")
	s = append(s, "Identifier: "+fmt.Sprintf("%#v", this.Identifier)+",\n")
	s = append(s, "Topics: "+fmt.Sprintf("%#v", this.Topics)+",\n")
	s = append(s, "data: "+fmt.Sprintf("%#v", this.Data)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *Log) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 6)
	s = append(s, "&transaction.Log{")
	s = append(s, "Address: "+fmt.Sprintf("%#v", this.Address)+",\n")
	if this.Events != nil {
		s = append(s, "Events: "+fmt.Sprintf("%#v", this.Events)+",\n")
	}
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringLog(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *Event) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Event) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Event) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Data) > 0 {
		i -= len(m.Data)
		copy(dAtA[i:], m.Data)
		i = encodeVarintLog(dAtA, i, uint64(len(m.Data)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.Topics) > 0 {
		for iNdEx := len(m.Topics) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Topics[iNdEx])
			copy(dAtA[i:], m.Topics[iNdEx])
			i = encodeVarintLog(dAtA, i, uint64(len(m.Topics[iNdEx])))
			i--
			dAtA[i] = 0x1a
		}
	}
	if len(m.Identifier) > 0 {
		i -= len(m.Identifier)
		copy(dAtA[i:], m.Identifier)
		i = encodeVarintLog(dAtA, i, uint64(len(m.Identifier)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintLog(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *Log) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Log) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Log) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Events) > 0 {
		for iNdEx := len(m.Events) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Events[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintLog(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintLog(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintLog(dAtA []byte, offset int, v uint64) int {
	offset -= sovLog(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Event) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovLog(uint64(l))
	}
	l = len(m.Identifier)
	if l > 0 {
		n += 1 + l + sovLog(uint64(l))
	}
	if len(m.Topics) > 0 {
		for _, b := range m.Topics {
			l = len(b)
			n += 1 + l + sovLog(uint64(l))
		}
	}
	l = len(m.Data)
	if l > 0 {
		n += 1 + l + sovLog(uint64(l))
	}
	return n
}

func (m *Log) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovLog(uint64(l))
	}
	if len(m.Events) > 0 {
		for _, e := range m.Events {
			l = e.Size()
			n += 1 + l + sovLog(uint64(l))
		}
	}
	return n
}

func sovLog(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozLog(x uint64) (n int) {
	return sovLog(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *Event) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&Event{`,
		`Address:` + fmt.Sprintf("%v", this.Address) + `,`,
		`Identifier:` + fmt.Sprintf("%v", this.Identifier) + `,`,
		`Topics:` + fmt.Sprintf("%v", this.Topics) + `,`,
		`data:` + fmt.Sprintf("%v", this.Data) + `,`,
		`}`,
	}, "")
	return s
}
func (this *Log) String() string {
	if this == nil {
		return "nil"
	}
	repeatedStringForEvents := "[]*Event{"
	for _, f := range this.Events {
		repeatedStringForEvents += strings.Replace(f.String(), "Event", "Event", 1) + ","
	}
	repeatedStringForEvents += "}"
	s := strings.Join([]string{`&Log{`,
		`Address:` + fmt.Sprintf("%v", this.Address) + `,`,
		`Events:` + repeatedStringForEvents + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringLog(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *Event) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowLog
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Event: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Event: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthLog
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = append(m.Address[:0], dAtA[iNdEx:postIndex]...)
			if m.Address == nil {
				m.Address = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Identifier", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthLog
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Identifier = append(m.Identifier[:0], dAtA[iNdEx:postIndex]...)
			if m.Identifier == nil {
				m.Identifier = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Topics", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthLog
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Topics = append(m.Topics, make([]byte, postIndex-iNdEx))
			copy(m.Topics[len(m.Topics)-1], dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field data", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthLog
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Data = append(m.Data[:0], dAtA[iNdEx:postIndex]...)
			if m.Data == nil {
				m.Data = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipLog(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthLog
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthLog
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Log) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowLog
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Log: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Log: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthLog
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = append(m.Address[:0], dAtA[iNdEx:postIndex]...)
			if m.Address == nil {
				m.Address = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Events", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLog
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthLog
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthLog
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Events = append(m.Events, &Event{})
			if err := m.Events[len(m.Events)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipLog(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthLog
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthLog
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipLog(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowLog
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowLog
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowLog
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthLog
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupLog
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthLog
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthLog        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowLog          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupLog = fmt.Errorf("proto: unexpected end of group")
)
