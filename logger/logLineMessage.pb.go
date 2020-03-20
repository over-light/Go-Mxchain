// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: logLineMessage.proto

package logger

import (
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

type LogLineMessage struct {
	Message    string   `protobuf:"bytes,1,opt,name=Message,proto3" json:"Message,omitempty"`
	LogLevel   int32    `protobuf:"varint,2,opt,name=LogLevel,proto3" json:"LogLevel,omitempty"`
	Args       []string `protobuf:"bytes,3,rep,name=Args,proto3" json:"Args,omitempty"`
	Timestamp  int64    `protobuf:"varint,4,opt,name=Timestamp,proto3" json:"Timestamp,omitempty"`
	LoggerName string   `protobuf:"bytes,5,opt,name=LoggerName,proto3" json:"LoggerName,omitempty"`
}

func (m *LogLineMessage) Reset()      { *m = LogLineMessage{} }
func (*LogLineMessage) ProtoMessage() {}
func (*LogLineMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_dc96a1223a5fcf02, []int{0}
}
func (m *LogLineMessage) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *LogLineMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *LogLineMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LogLineMessage.Merge(m, src)
}
func (m *LogLineMessage) XXX_Size() int {
	return m.Size()
}
func (m *LogLineMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_LogLineMessage.DiscardUnknown(m)
}

var xxx_messageInfo_LogLineMessage proto.InternalMessageInfo

func (m *LogLineMessage) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *LogLineMessage) GetLogLevel() int32 {
	if m != nil {
		return m.LogLevel
	}
	return 0
}

func (m *LogLineMessage) GetArgs() []string {
	if m != nil {
		return m.Args
	}
	return nil
}

func (m *LogLineMessage) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *LogLineMessage) GetLoggerName() string {
	if m != nil {
		return m.LoggerName
	}
	return ""
}

func init() {
	proto.RegisterType((*LogLineMessage)(nil), "proto.LogLineMessage")
}

func init() { proto.RegisterFile("logLineMessage.proto", fileDescriptor_dc96a1223a5fcf02) }

var fileDescriptor_dc96a1223a5fcf02 = []byte{
	// 250 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x54, 0x8f, 0x3d, 0x4e, 0xc4, 0x30,
	0x10, 0x85, 0x3d, 0x64, 0xb3, 0x10, 0x17, 0x14, 0x16, 0x85, 0xb5, 0x42, 0xa3, 0x88, 0x2a, 0x0d,
	0xbb, 0x05, 0x17, 0x00, 0xea, 0x40, 0x11, 0x51, 0xd1, 0x25, 0xc8, 0x98, 0x48, 0x09, 0x5e, 0xc5,
	0x59, 0x6a, 0x8e, 0x40, 0xc9, 0x11, 0x38, 0x0a, 0x65, 0xca, 0x94, 0x64, 0xd2, 0x50, 0xe6, 0x08,
	0x88, 0xe1, 0x77, 0x2b, 0xbf, 0xef, 0x93, 0xe7, 0x8d, 0x46, 0x1e, 0x54, 0xce, 0xa6, 0xe5, 0xbd,
	0xb9, 0x30, 0xde, 0xe7, 0xd6, 0x2c, 0xd7, 0x8d, 0x6b, 0x9d, 0x0a, 0xf9, 0x59, 0x1c, 0xdb, 0xb2,
	0xbd, 0xdb, 0x14, 0xcb, 0x1b, 0x57, 0xaf, 0xac, 0xb3, 0x6e, 0xc5, 0xba, 0xd8, 0xdc, 0x32, 0x31,
	0x70, 0xfa, 0x9a, 0x3a, 0x7a, 0x06, 0xb9, 0x9f, 0x6e, 0xd5, 0x29, 0x2d, 0x77, 0xbf, 0xa3, 0x86,
	0x18, 0x92, 0x28, 0xfb, 0x41, 0xb5, 0x90, 0x7b, 0x9f, 0x7f, 0xcd, 0x83, 0xa9, 0xf4, 0x4e, 0x0c,
	0x49, 0x98, 0xfd, 0xb2, 0x52, 0x72, 0x76, 0xd6, 0x58, 0xaf, 0x83, 0x38, 0x48, 0xa2, 0x8c, 0xb3,
	0x3a, 0x94, 0xd1, 0x55, 0x59, 0x1b, 0xdf, 0xe6, 0xf5, 0x5a, 0xcf, 0x62, 0x48, 0x82, 0xec, 0x4f,
	0x28, 0x94, 0x32, 0x75, 0xd6, 0x9a, 0xe6, 0x32, 0xaf, 0x8d, 0x0e, 0x79, 0xd5, 0x3f, 0x73, 0x7e,
	0xda, 0x0d, 0x28, 0xfa, 0x01, 0xc5, 0x34, 0x20, 0x3c, 0x12, 0xc2, 0x0b, 0x21, 0xbc, 0x12, 0x42,
	0x47, 0x08, 0x3d, 0x21, 0xbc, 0x11, 0xc2, 0x3b, 0xa1, 0x98, 0x08, 0xe1, 0x69, 0x44, 0xd1, 0x8d,
	0x28, 0xfa, 0x11, 0xc5, 0xf5, 0xbc, 0xe2, 0x96, 0x62, 0xce, 0x37, 0x9e, 0x7c, 0x04, 0x00, 0x00,
	0xff, 0xff, 0x3e, 0xa4, 0x6d, 0x1f, 0x31, 0x01, 0x00, 0x00,
}

func (this *LogLineMessage) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*LogLineMessage)
	if !ok {
		that2, ok := that.(LogLineMessage)
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
	if this.Message != that1.Message {
		return false
	}
	if this.LogLevel != that1.LogLevel {
		return false
	}
	if len(this.Args) != len(that1.Args) {
		return false
	}
	for i := range this.Args {
		if this.Args[i] != that1.Args[i] {
			return false
		}
	}
	if this.Timestamp != that1.Timestamp {
		return false
	}
	if this.LoggerName != that1.LoggerName {
		return false
	}
	return true
}
func (this *LogLineMessage) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 9)
	s = append(s, "&logger.LogLineMessage{")
	s = append(s, "Message: "+fmt.Sprintf("%#v", this.Message)+",\n")
	s = append(s, "LogLevel: "+fmt.Sprintf("%#v", this.LogLevel)+",\n")
	s = append(s, "Args: "+fmt.Sprintf("%#v", this.Args)+",\n")
	s = append(s, "Timestamp: "+fmt.Sprintf("%#v", this.Timestamp)+",\n")
	s = append(s, "LoggerName: "+fmt.Sprintf("%#v", this.LoggerName)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringLogLineMessage(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *LogLineMessage) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *LogLineMessage) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *LogLineMessage) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.LoggerName) > 0 {
		i -= len(m.LoggerName)
		copy(dAtA[i:], m.LoggerName)
		i = encodeVarintLogLineMessage(dAtA, i, uint64(len(m.LoggerName)))
		i--
		dAtA[i] = 0x2a
	}
	if m.Timestamp != 0 {
		i = encodeVarintLogLineMessage(dAtA, i, uint64(m.Timestamp))
		i--
		dAtA[i] = 0x20
	}
	if len(m.Args) > 0 {
		for iNdEx := len(m.Args) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Args[iNdEx])
			copy(dAtA[i:], m.Args[iNdEx])
			i = encodeVarintLogLineMessage(dAtA, i, uint64(len(m.Args[iNdEx])))
			i--
			dAtA[i] = 0x1a
		}
	}
	if m.LogLevel != 0 {
		i = encodeVarintLogLineMessage(dAtA, i, uint64(m.LogLevel))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Message) > 0 {
		i -= len(m.Message)
		copy(dAtA[i:], m.Message)
		i = encodeVarintLogLineMessage(dAtA, i, uint64(len(m.Message)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintLogLineMessage(dAtA []byte, offset int, v uint64) int {
	offset -= sovLogLineMessage(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *LogLineMessage) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Message)
	if l > 0 {
		n += 1 + l + sovLogLineMessage(uint64(l))
	}
	if m.LogLevel != 0 {
		n += 1 + sovLogLineMessage(uint64(m.LogLevel))
	}
	if len(m.Args) > 0 {
		for _, s := range m.Args {
			l = len(s)
			n += 1 + l + sovLogLineMessage(uint64(l))
		}
	}
	if m.Timestamp != 0 {
		n += 1 + sovLogLineMessage(uint64(m.Timestamp))
	}
	l = len(m.LoggerName)
	if l > 0 {
		n += 1 + l + sovLogLineMessage(uint64(l))
	}
	return n
}

func sovLogLineMessage(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozLogLineMessage(x uint64) (n int) {
	return sovLogLineMessage(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *LogLineMessage) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&LogLineMessage{`,
		`Message:` + fmt.Sprintf("%v", this.Message) + `,`,
		`LogLevel:` + fmt.Sprintf("%v", this.LogLevel) + `,`,
		`Args:` + fmt.Sprintf("%v", this.Args) + `,`,
		`Timestamp:` + fmt.Sprintf("%v", this.Timestamp) + `,`,
		`LoggerName:` + fmt.Sprintf("%v", this.LoggerName) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringLogLineMessage(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *LogLineMessage) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowLogLineMessage
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
			return fmt.Errorf("proto: LogLineMessage: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: LogLineMessage: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Message", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLogLineMessage
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthLogLineMessage
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthLogLineMessage
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Message = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LogLevel", wireType)
			}
			m.LogLevel = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLogLineMessage
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LogLevel |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Args", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLogLineMessage
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthLogLineMessage
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthLogLineMessage
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Args = append(m.Args, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Timestamp", wireType)
			}
			m.Timestamp = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLogLineMessage
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Timestamp |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LoggerName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowLogLineMessage
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthLogLineMessage
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthLogLineMessage
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.LoggerName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipLogLineMessage(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthLogLineMessage
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthLogLineMessage
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
func skipLogLineMessage(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowLogLineMessage
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
					return 0, ErrIntOverflowLogLineMessage
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
					return 0, ErrIntOverflowLogLineMessage
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
				return 0, ErrInvalidLengthLogLineMessage
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupLogLineMessage
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthLogLineMessage
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthLogLineMessage        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowLogLineMessage          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupLogLineMessage = fmt.Errorf("proto: unexpected end of group")
)
