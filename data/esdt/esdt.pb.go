// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: esdt.proto

package esdt

import (
	bytes "bytes"
	fmt "fmt"
	github_com_ElrondNetwork_elrond_go_data "github.com/ElrondNetwork/elrond-go/data"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_big "math/big"
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

// ESDigitalToken holds the data for a Elrond standard digital token transaction
type ESDigitalToken struct {
	Type          uint32        `protobuf:"varint,1,opt,name=Type,proto3" json:"Type"`
	Value         *math_big.Int `protobuf:"bytes,2,opt,name=Value,proto3,casttypewith=math/big.Int;github.com/ElrondNetwork/elrond-go/data.BigIntCaster" json:"Value"`
	Properties    []byte        `protobuf:"bytes,3,opt,name=Properties,proto3" json:"Properties"`
	TokenMetaData *MetaData     `protobuf:"bytes,4,opt,name=TokenMetaData,proto3" json:"MetaData"`
	Reserved      []byte        `protobuf:"bytes,5,opt,name=Reserved,proto3" json:"Reserved"`
}

func (m *ESDigitalToken) Reset()      { *m = ESDigitalToken{} }
func (*ESDigitalToken) ProtoMessage() {}
func (*ESDigitalToken) Descriptor() ([]byte, []int) {
	return fileDescriptor_e413e402abc6a34c, []int{0}
}
func (m *ESDigitalToken) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ESDigitalToken) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *ESDigitalToken) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ESDigitalToken.Merge(m, src)
}
func (m *ESDigitalToken) XXX_Size() int {
	return m.Size()
}
func (m *ESDigitalToken) XXX_DiscardUnknown() {
	xxx_messageInfo_ESDigitalToken.DiscardUnknown(m)
}

var xxx_messageInfo_ESDigitalToken proto.InternalMessageInfo

func (m *ESDigitalToken) GetType() uint32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *ESDigitalToken) GetValue() *math_big.Int {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *ESDigitalToken) GetProperties() []byte {
	if m != nil {
		return m.Properties
	}
	return nil
}

func (m *ESDigitalToken) GetTokenMetaData() *MetaData {
	if m != nil {
		return m.TokenMetaData
	}
	return nil
}

func (m *ESDigitalToken) GetReserved() []byte {
	if m != nil {
		return m.Reserved
	}
	return nil
}

// ESDTRoles holds the roles for a given token and the given address
type ESDTRoles struct {
	Roles [][]byte `protobuf:"bytes,1,rep,name=Roles,proto3" json:"roles"`
}

func (m *ESDTRoles) Reset()      { *m = ESDTRoles{} }
func (*ESDTRoles) ProtoMessage() {}
func (*ESDTRoles) Descriptor() ([]byte, []int) {
	return fileDescriptor_e413e402abc6a34c, []int{1}
}
func (m *ESDTRoles) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ESDTRoles) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *ESDTRoles) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ESDTRoles.Merge(m, src)
}
func (m *ESDTRoles) XXX_Size() int {
	return m.Size()
}
func (m *ESDTRoles) XXX_DiscardUnknown() {
	xxx_messageInfo_ESDTRoles.DiscardUnknown(m)
}

var xxx_messageInfo_ESDTRoles proto.InternalMessageInfo

func (m *ESDTRoles) GetRoles() [][]byte {
	if m != nil {
		return m.Roles
	}
	return nil
}

// MetaData hold the metadata structure for the ESDT token
type MetaData struct {
	Nonce      uint64   `protobuf:"varint,1,opt,name=Nonce,proto3" json:"Nonce"`
	Name       []byte   `protobuf:"bytes,2,opt,name=Name,proto3" json:"Name"`
	Creator    []byte   `protobuf:"bytes,3,opt,name=Creator,proto3" json:"Creator"`
	Royalties  uint32   `protobuf:"varint,4,opt,name=Royalties,proto3" json:"Royalties"`
	Hash       []byte   `protobuf:"bytes,5,opt,name=Hash,proto3" json:"Hash"`
	URIs       [][]byte `protobuf:"bytes,6,rep,name=URIs,proto3" json:"URIs"`
	Attributes []byte   `protobuf:"bytes,7,opt,name=Attributes,proto3" json:"Attributes"`
}

func (m *MetaData) Reset()      { *m = MetaData{} }
func (*MetaData) ProtoMessage() {}
func (*MetaData) Descriptor() ([]byte, []int) {
	return fileDescriptor_e413e402abc6a34c, []int{2}
}
func (m *MetaData) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MetaData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *MetaData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MetaData.Merge(m, src)
}
func (m *MetaData) XXX_Size() int {
	return m.Size()
}
func (m *MetaData) XXX_DiscardUnknown() {
	xxx_messageInfo_MetaData.DiscardUnknown(m)
}

var xxx_messageInfo_MetaData proto.InternalMessageInfo

func (m *MetaData) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *MetaData) GetName() []byte {
	if m != nil {
		return m.Name
	}
	return nil
}

func (m *MetaData) GetCreator() []byte {
	if m != nil {
		return m.Creator
	}
	return nil
}

func (m *MetaData) GetRoyalties() uint32 {
	if m != nil {
		return m.Royalties
	}
	return 0
}

func (m *MetaData) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

func (m *MetaData) GetURIs() [][]byte {
	if m != nil {
		return m.URIs
	}
	return nil
}

func (m *MetaData) GetAttributes() []byte {
	if m != nil {
		return m.Attributes
	}
	return nil
}

func init() {
	proto.RegisterType((*ESDigitalToken)(nil), "protoBuiltInFunctions.ESDigitalToken")
	proto.RegisterType((*ESDTRoles)(nil), "protoBuiltInFunctions.ESDTRoles")
	proto.RegisterType((*MetaData)(nil), "protoBuiltInFunctions.MetaData")
}

func init() { proto.RegisterFile("esdt.proto", fileDescriptor_e413e402abc6a34c) }

var fileDescriptor_e413e402abc6a34c = []byte{
	// 508 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x53, 0x41, 0x8b, 0xd3, 0x40,
	0x14, 0xce, 0x74, 0xdb, 0xdd, 0x76, 0xb6, 0xdd, 0x43, 0x40, 0x08, 0x22, 0x93, 0x52, 0x10, 0x0a,
	0xba, 0x29, 0xe8, 0x51, 0x10, 0x36, 0xdb, 0x8a, 0x3d, 0x58, 0x64, 0x5a, 0x3d, 0x78, 0x9b, 0x34,
	0x63, 0x1a, 0x36, 0xcd, 0x94, 0xc9, 0x44, 0xd9, 0x9b, 0x57, 0x6f, 0xfe, 0x0c, 0xf1, 0x6f, 0x78,
	0xf1, 0xd8, 0x63, 0x4f, 0xd1, 0xa6, 0x17, 0xc9, 0x69, 0x7f, 0x82, 0xcc, 0xcb, 0x66, 0x5b, 0x61,
	0x2f, 0x2f, 0xdf, 0xf7, 0xcd, 0x63, 0xde, 0x9b, 0xef, 0x23, 0x18, 0xf3, 0xc4, 0x57, 0xce, 0x4a,
	0x0a, 0x25, 0xcc, 0x07, 0xf0, 0x71, 0xd3, 0x30, 0x52, 0xe3, 0xf8, 0x55, 0x1a, 0xcf, 0x55, 0x28,
	0xe2, 0xe4, 0xe1, 0x79, 0x10, 0xaa, 0x45, 0xea, 0x39, 0x73, 0xb1, 0x1c, 0x04, 0x22, 0x10, 0x03,
	0x68, 0xf3, 0xd2, 0x8f, 0xc0, 0x80, 0x00, 0x2a, 0x6f, 0xe9, 0xfd, 0xac, 0xe1, 0xb3, 0xd1, 0x74,
	0x18, 0x06, 0xa1, 0x62, 0xd1, 0x4c, 0x5c, 0xf1, 0xd8, 0x7c, 0x84, 0xeb, 0xb3, 0xeb, 0x15, 0xb7,
	0x50, 0x17, 0xf5, 0x3b, 0x6e, 0xb3, 0xc8, 0x6c, 0xe0, 0x14, 0xaa, 0xe9, 0xe3, 0xc6, 0x7b, 0x16,
	0xa5, 0xdc, 0xaa, 0x75, 0x51, 0xbf, 0xed, 0x4e, 0x8a, 0xcc, 0x2e, 0x85, 0x1f, 0xbf, 0xed, 0x8b,
	0x25, 0x53, 0x8b, 0x81, 0x17, 0x06, 0xce, 0x38, 0x56, 0x2f, 0x0e, 0x16, 0x19, 0x45, 0x52, 0xc4,
	0xfe, 0x84, 0xab, 0xcf, 0x42, 0x5e, 0x0d, 0x38, 0xb0, 0xf3, 0x40, 0x0c, 0x7c, 0xa6, 0x98, 0xe3,
	0x86, 0xc1, 0x38, 0x56, 0x97, 0x2c, 0x51, 0x5c, 0xd2, 0xf2, 0x2e, 0xd3, 0xc1, 0xf8, 0xad, 0x14,
	0x2b, 0x2e, 0x55, 0xc8, 0x13, 0xeb, 0x08, 0x46, 0x9d, 0x15, 0x99, 0x7d, 0xa0, 0xd2, 0x03, 0x6c,
	0x4e, 0x71, 0x07, 0x96, 0x7f, 0xc3, 0x15, 0x1b, 0x32, 0xc5, 0xac, 0x7a, 0x17, 0xf5, 0x4f, 0x9f,
	0xd9, 0xce, 0xbd, 0x26, 0x39, 0x55, 0x9b, 0xdb, 0x2e, 0x32, 0xbb, 0x59, 0x31, 0xfa, 0xff, 0x1d,
	0x66, 0x1f, 0x37, 0x29, 0x4f, 0xb8, 0xfc, 0xc4, 0x7d, 0xab, 0x01, 0x2b, 0x40, 0x7b, 0xa5, 0xd1,
	0x3b, 0xd4, 0x7b, 0x8a, 0x5b, 0xa3, 0xe9, 0x70, 0x46, 0x45, 0xc4, 0x13, 0xd3, 0xc6, 0x0d, 0x00,
	0x16, 0xea, 0x1e, 0xf5, 0xdb, 0x6e, 0x4b, 0x3b, 0x24, 0xb5, 0x40, 0x4b, 0xbd, 0xf7, 0xb5, 0x86,
	0xef, 0x66, 0xea, 0xee, 0x89, 0x88, 0xe7, 0xa5, 0xdd, 0xf5, 0xb2, 0x1b, 0x04, 0x5a, 0x7e, 0x74,
	0x1c, 0x13, 0xb6, 0xac, 0xfc, 0x86, 0x38, 0x34, 0xa7, 0x50, 0xcd, 0xc7, 0xf8, 0xe4, 0x52, 0x72,
	0xa6, 0x84, 0xbc, 0x75, 0xe9, 0xb4, 0xc8, 0xec, 0x4a, 0xa2, 0x15, 0x30, 0x9f, 0xe0, 0x16, 0x15,
	0xd7, 0x2c, 0x02, 0x3b, 0xeb, 0x10, 0x6c, 0xa7, 0xc8, 0xec, 0xbd, 0x48, 0xf7, 0x50, 0x4f, 0x7c,
	0xcd, 0x92, 0xc5, 0xed, 0x9b, 0x61, 0xa2, 0xe6, 0x14, 0xaa, 0x3e, 0x7d, 0x47, 0xc7, 0x89, 0x75,
	0x0c, 0xaf, 0x83, 0x53, 0xcd, 0x29, 0x54, 0x1d, 0xdc, 0x85, 0x52, 0x32, 0xf4, 0x52, 0xc5, 0x13,
	0xeb, 0x64, 0x1f, 0xdc, 0x5e, 0xa5, 0x07, 0xd8, 0x7d, 0xb9, 0xde, 0x12, 0x63, 0xb3, 0x25, 0xc6,
	0xcd, 0x96, 0xa0, 0x2f, 0x39, 0x41, 0xdf, 0x73, 0x82, 0x7e, 0xe5, 0x04, 0xad, 0x73, 0x82, 0x36,
	0x39, 0x41, 0x7f, 0x72, 0x82, 0xfe, 0xe6, 0xc4, 0xb8, 0xc9, 0x09, 0xfa, 0xb6, 0x23, 0xc6, 0x7a,
	0x47, 0x8c, 0xcd, 0x8e, 0x18, 0x1f, 0xea, 0xfa, 0x5f, 0xf0, 0x8e, 0x21, 0xe0, 0xe7, 0xff, 0x02,
	0x00, 0x00, 0xff, 0xff, 0x45, 0x91, 0x1a, 0x6d, 0x1a, 0x03, 0x00, 0x00,
}

func (this *ESDigitalToken) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*ESDigitalToken)
	if !ok {
		that2, ok := that.(ESDigitalToken)
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
	if this.Type != that1.Type {
		return false
	}
	{
		__caster := &github_com_ElrondNetwork_elrond_go_data.BigIntCaster{}
		if !__caster.Equal(this.Value, that1.Value) {
			return false
		}
	}
	if !bytes.Equal(this.Properties, that1.Properties) {
		return false
	}
	if !this.TokenMetaData.Equal(that1.TokenMetaData) {
		return false
	}
	if !bytes.Equal(this.Reserved, that1.Reserved) {
		return false
	}
	return true
}
func (this *ESDTRoles) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*ESDTRoles)
	if !ok {
		that2, ok := that.(ESDTRoles)
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
	if len(this.Roles) != len(that1.Roles) {
		return false
	}
	for i := range this.Roles {
		if !bytes.Equal(this.Roles[i], that1.Roles[i]) {
			return false
		}
	}
	return true
}
func (this *MetaData) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*MetaData)
	if !ok {
		that2, ok := that.(MetaData)
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
	if this.Nonce != that1.Nonce {
		return false
	}
	if !bytes.Equal(this.Name, that1.Name) {
		return false
	}
	if !bytes.Equal(this.Creator, that1.Creator) {
		return false
	}
	if this.Royalties != that1.Royalties {
		return false
	}
	if !bytes.Equal(this.Hash, that1.Hash) {
		return false
	}
	if len(this.URIs) != len(that1.URIs) {
		return false
	}
	for i := range this.URIs {
		if !bytes.Equal(this.URIs[i], that1.URIs[i]) {
			return false
		}
	}
	if !bytes.Equal(this.Attributes, that1.Attributes) {
		return false
	}
	return true
}
func (this *ESDigitalToken) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 9)
	s = append(s, "&esdt.ESDigitalToken{")
	s = append(s, "Type: "+fmt.Sprintf("%#v", this.Type)+",\n")
	s = append(s, "Value: "+fmt.Sprintf("%#v", this.Value)+",\n")
	s = append(s, "Properties: "+fmt.Sprintf("%#v", this.Properties)+",\n")
	if this.TokenMetaData != nil {
		s = append(s, "TokenMetaData: "+fmt.Sprintf("%#v", this.TokenMetaData)+",\n")
	}
	s = append(s, "Reserved: "+fmt.Sprintf("%#v", this.Reserved)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *ESDTRoles) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 5)
	s = append(s, "&esdt.ESDTRoles{")
	s = append(s, "Roles: "+fmt.Sprintf("%#v", this.Roles)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *MetaData) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 11)
	s = append(s, "&esdt.MetaData{")
	s = append(s, "Nonce: "+fmt.Sprintf("%#v", this.Nonce)+",\n")
	s = append(s, "Name: "+fmt.Sprintf("%#v", this.Name)+",\n")
	s = append(s, "Creator: "+fmt.Sprintf("%#v", this.Creator)+",\n")
	s = append(s, "Royalties: "+fmt.Sprintf("%#v", this.Royalties)+",\n")
	s = append(s, "Hash: "+fmt.Sprintf("%#v", this.Hash)+",\n")
	s = append(s, "URIs: "+fmt.Sprintf("%#v", this.URIs)+",\n")
	s = append(s, "Attributes: "+fmt.Sprintf("%#v", this.Attributes)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringEsdt(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *ESDigitalToken) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ESDigitalToken) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ESDigitalToken) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Reserved) > 0 {
		i -= len(m.Reserved)
		copy(dAtA[i:], m.Reserved)
		i = encodeVarintEsdt(dAtA, i, uint64(len(m.Reserved)))
		i--
		dAtA[i] = 0x2a
	}
	if m.TokenMetaData != nil {
		{
			size, err := m.TokenMetaData.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintEsdt(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x22
	}
	if len(m.Properties) > 0 {
		i -= len(m.Properties)
		copy(dAtA[i:], m.Properties)
		i = encodeVarintEsdt(dAtA, i, uint64(len(m.Properties)))
		i--
		dAtA[i] = 0x1a
	}
	{
		__caster := &github_com_ElrondNetwork_elrond_go_data.BigIntCaster{}
		size := __caster.Size(m.Value)
		i -= size
		if _, err := __caster.MarshalTo(m.Value, dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintEsdt(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if m.Type != 0 {
		i = encodeVarintEsdt(dAtA, i, uint64(m.Type))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *ESDTRoles) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ESDTRoles) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ESDTRoles) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Roles) > 0 {
		for iNdEx := len(m.Roles) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Roles[iNdEx])
			copy(dAtA[i:], m.Roles[iNdEx])
			i = encodeVarintEsdt(dAtA, i, uint64(len(m.Roles[iNdEx])))
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func (m *MetaData) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MetaData) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MetaData) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Attributes) > 0 {
		i -= len(m.Attributes)
		copy(dAtA[i:], m.Attributes)
		i = encodeVarintEsdt(dAtA, i, uint64(len(m.Attributes)))
		i--
		dAtA[i] = 0x3a
	}
	if len(m.URIs) > 0 {
		for iNdEx := len(m.URIs) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.URIs[iNdEx])
			copy(dAtA[i:], m.URIs[iNdEx])
			i = encodeVarintEsdt(dAtA, i, uint64(len(m.URIs[iNdEx])))
			i--
			dAtA[i] = 0x32
		}
	}
	if len(m.Hash) > 0 {
		i -= len(m.Hash)
		copy(dAtA[i:], m.Hash)
		i = encodeVarintEsdt(dAtA, i, uint64(len(m.Hash)))
		i--
		dAtA[i] = 0x2a
	}
	if m.Royalties != 0 {
		i = encodeVarintEsdt(dAtA, i, uint64(m.Royalties))
		i--
		dAtA[i] = 0x20
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintEsdt(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = encodeVarintEsdt(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0x12
	}
	if m.Nonce != 0 {
		i = encodeVarintEsdt(dAtA, i, uint64(m.Nonce))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintEsdt(dAtA []byte, offset int, v uint64) int {
	offset -= sovEsdt(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *ESDigitalToken) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Type != 0 {
		n += 1 + sovEsdt(uint64(m.Type))
	}
	{
		__caster := &github_com_ElrondNetwork_elrond_go_data.BigIntCaster{}
		l = __caster.Size(m.Value)
		n += 1 + l + sovEsdt(uint64(l))
	}
	l = len(m.Properties)
	if l > 0 {
		n += 1 + l + sovEsdt(uint64(l))
	}
	if m.TokenMetaData != nil {
		l = m.TokenMetaData.Size()
		n += 1 + l + sovEsdt(uint64(l))
	}
	l = len(m.Reserved)
	if l > 0 {
		n += 1 + l + sovEsdt(uint64(l))
	}
	return n
}

func (m *ESDTRoles) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Roles) > 0 {
		for _, b := range m.Roles {
			l = len(b)
			n += 1 + l + sovEsdt(uint64(l))
		}
	}
	return n
}

func (m *MetaData) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Nonce != 0 {
		n += 1 + sovEsdt(uint64(m.Nonce))
	}
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovEsdt(uint64(l))
	}
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovEsdt(uint64(l))
	}
	if m.Royalties != 0 {
		n += 1 + sovEsdt(uint64(m.Royalties))
	}
	l = len(m.Hash)
	if l > 0 {
		n += 1 + l + sovEsdt(uint64(l))
	}
	if len(m.URIs) > 0 {
		for _, b := range m.URIs {
			l = len(b)
			n += 1 + l + sovEsdt(uint64(l))
		}
	}
	l = len(m.Attributes)
	if l > 0 {
		n += 1 + l + sovEsdt(uint64(l))
	}
	return n
}

func sovEsdt(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozEsdt(x uint64) (n int) {
	return sovEsdt(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *ESDigitalToken) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&ESDigitalToken{`,
		`Type:` + fmt.Sprintf("%v", this.Type) + `,`,
		`Value:` + fmt.Sprintf("%v", this.Value) + `,`,
		`Properties:` + fmt.Sprintf("%v", this.Properties) + `,`,
		`TokenMetaData:` + strings.Replace(this.TokenMetaData.String(), "MetaData", "MetaData", 1) + `,`,
		`Reserved:` + fmt.Sprintf("%v", this.Reserved) + `,`,
		`}`,
	}, "")
	return s
}
func (this *ESDTRoles) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&ESDTRoles{`,
		`Roles:` + fmt.Sprintf("%v", this.Roles) + `,`,
		`}`,
	}, "")
	return s
}
func (this *MetaData) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&MetaData{`,
		`Nonce:` + fmt.Sprintf("%v", this.Nonce) + `,`,
		`Name:` + fmt.Sprintf("%v", this.Name) + `,`,
		`Creator:` + fmt.Sprintf("%v", this.Creator) + `,`,
		`Royalties:` + fmt.Sprintf("%v", this.Royalties) + `,`,
		`Hash:` + fmt.Sprintf("%v", this.Hash) + `,`,
		`URIs:` + fmt.Sprintf("%v", this.URIs) + `,`,
		`Attributes:` + fmt.Sprintf("%v", this.Attributes) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringEsdt(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *ESDigitalToken) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEsdt
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
			return fmt.Errorf("proto: ESDigitalToken: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ESDigitalToken: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Type", wireType)
			}
			m.Type = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEsdt
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Type |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEsdt
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
				return ErrInvalidLengthEsdt
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthEsdt
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			{
				__caster := &github_com_ElrondNetwork_elrond_go_data.BigIntCaster{}
				if tmp, err := __caster.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
					return err
				} else {
					m.Value = tmp
				}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Properties", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEsdt
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
				return ErrInvalidLengthEsdt
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthEsdt
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Properties = append(m.Properties[:0], dAtA[iNdEx:postIndex]...)
			if m.Properties == nil {
				m.Properties = []byte{}
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TokenMetaData", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEsdt
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
				return ErrInvalidLengthEsdt
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEsdt
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.TokenMetaData == nil {
				m.TokenMetaData = &MetaData{}
			}
			if err := m.TokenMetaData.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Reserved", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEsdt
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
				return ErrInvalidLengthEsdt
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthEsdt
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Reserved = append(m.Reserved[:0], dAtA[iNdEx:postIndex]...)
			if m.Reserved == nil {
				m.Reserved = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEsdt(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthEsdt
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthEsdt
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
func (m *ESDTRoles) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEsdt
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
			return fmt.Errorf("proto: ESDTRoles: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ESDTRoles: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Roles", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEsdt
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
				return ErrInvalidLengthEsdt
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthEsdt
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Roles = append(m.Roles, make([]byte, postIndex-iNdEx))
			copy(m.Roles[len(m.Roles)-1], dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEsdt(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthEsdt
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthEsdt
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
func (m *MetaData) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEsdt
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
			return fmt.Errorf("proto: MetaData: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MetaData: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nonce", wireType)
			}
			m.Nonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEsdt
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Nonce |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEsdt
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
				return ErrInvalidLengthEsdt
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthEsdt
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = append(m.Name[:0], dAtA[iNdEx:postIndex]...)
			if m.Name == nil {
				m.Name = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEsdt
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
				return ErrInvalidLengthEsdt
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthEsdt
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = append(m.Creator[:0], dAtA[iNdEx:postIndex]...)
			if m.Creator == nil {
				m.Creator = []byte{}
			}
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Royalties", wireType)
			}
			m.Royalties = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEsdt
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Royalties |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Hash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEsdt
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
				return ErrInvalidLengthEsdt
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthEsdt
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Hash = append(m.Hash[:0], dAtA[iNdEx:postIndex]...)
			if m.Hash == nil {
				m.Hash = []byte{}
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field URIs", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEsdt
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
				return ErrInvalidLengthEsdt
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthEsdt
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.URIs = append(m.URIs, make([]byte, postIndex-iNdEx))
			copy(m.URIs[len(m.URIs)-1], dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Attributes", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEsdt
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
				return ErrInvalidLengthEsdt
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthEsdt
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Attributes = append(m.Attributes[:0], dAtA[iNdEx:postIndex]...)
			if m.Attributes == nil {
				m.Attributes = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEsdt(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthEsdt
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthEsdt
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
func skipEsdt(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowEsdt
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
					return 0, ErrIntOverflowEsdt
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
					return 0, ErrIntOverflowEsdt
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
				return 0, ErrInvalidLengthEsdt
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupEsdt
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthEsdt
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthEsdt        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowEsdt          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupEsdt = fmt.Errorf("proto: unexpected end of group")
)
