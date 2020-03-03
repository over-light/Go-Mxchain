// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: accountData.proto

package state

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

type AccountData struct {
	Nonce           uint64        `protobuf:"varint,1,opt,name=Nonce,proto3" json:"Nonce,omitempty"`
	Balance         *math_big.Int `protobuf:"bytes,2,opt,name=Balance,proto3,casttypewith=math/big.Int;github.com/ElrondNetwork/elrond-go/data.BigIntCaster" json:"Balance,omitempty"`
	CodeHash        []byte        `protobuf:"bytes,3,opt,name=CodeHash,proto3" json:"CodeHash,omitempty"`
	RootHash        []byte        `protobuf:"bytes,4,opt,name=RootHash,proto3" json:"RootHash,omitempty"`
	Address         []byte        `protobuf:"bytes,5,opt,name=Address,proto3" json:"Address,omitempty"`
	DeveloperReward *math_big.Int `protobuf:"bytes,6,opt,name=DeveloperReward,proto3,casttypewith=math/big.Int;github.com/ElrondNetwork/elrond-go/data.BigIntCaster" json:"DeveloperReward,omitempty"`
	OwnerAddress    []byte        `protobuf:"bytes,7,opt,name=OwnerAddress,proto3" json:"OwnerAddress,omitempty"`
}

func (m *AccountData) Reset()      { *m = AccountData{} }
func (*AccountData) ProtoMessage() {}
func (*AccountData) Descriptor() ([]byte, []int) {
	return fileDescriptor_6c4d48acb3d2c3f3, []int{0}
}
func (m *AccountData) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *AccountData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_AccountData.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *AccountData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AccountData.Merge(m, src)
}
func (m *AccountData) XXX_Size() int {
	return m.Size()
}
func (m *AccountData) XXX_DiscardUnknown() {
	xxx_messageInfo_AccountData.DiscardUnknown(m)
}

var xxx_messageInfo_AccountData proto.InternalMessageInfo

func (m *AccountData) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *AccountData) GetBalance() *math_big.Int {
	if m != nil {
		return m.Balance
	}
	return nil
}

func (m *AccountData) GetCodeHash() []byte {
	if m != nil {
		return m.CodeHash
	}
	return nil
}

func (m *AccountData) GetRootHash() []byte {
	if m != nil {
		return m.RootHash
	}
	return nil
}

func (m *AccountData) GetAddress() []byte {
	if m != nil {
		return m.Address
	}
	return nil
}

func (m *AccountData) GetDeveloperReward() *math_big.Int {
	if m != nil {
		return m.DeveloperReward
	}
	return nil
}

func (m *AccountData) GetOwnerAddress() []byte {
	if m != nil {
		return m.OwnerAddress
	}
	return nil
}

func init() {
	proto.RegisterType((*AccountData)(nil), "proto.AccountData")
}

func init() { proto.RegisterFile("accountData.proto", fileDescriptor_6c4d48acb3d2c3f3) }

var fileDescriptor_6c4d48acb3d2c3f3 = []byte{
	// 330 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x91, 0xbf, 0x6e, 0xea, 0x30,
	0x18, 0xc5, 0x63, 0x2e, 0x81, 0x2b, 0x17, 0xa9, 0xaa, 0xd5, 0x21, 0x62, 0xf8, 0x8a, 0x98, 0x58,
	0x20, 0x43, 0x47, 0x26, 0x02, 0x48, 0x65, 0xa1, 0x52, 0xc6, 0x2e, 0x95, 0x93, 0xb8, 0x01, 0x15,
	0xfc, 0x21, 0xc7, 0x94, 0xb5, 0x8f, 0xd0, 0xc7, 0xa8, 0xfa, 0x24, 0x1d, 0x19, 0xd9, 0x5a, 0xcc,
	0x52, 0xa9, 0x0b, 0x8f, 0x50, 0xc5, 0x28, 0xfd, 0x37, 0x77, 0xb2, 0x7f, 0xe7, 0xc8, 0xe7, 0x1c,
	0xc9, 0xf4, 0x84, 0xc7, 0x31, 0x2e, 0xa5, 0x1e, 0x70, 0xcd, 0x3b, 0x0b, 0x85, 0x1a, 0x99, 0x6b,
	0x8f, 0x7a, 0x3b, 0x9d, 0xea, 0xc9, 0x32, 0xea, 0xc4, 0x38, 0xf7, 0x53, 0x4c, 0xd1, 0xb7, 0x72,
	0xb4, 0xbc, 0xb1, 0x64, 0xc1, 0xde, 0x0e, 0xaf, 0x9a, 0xef, 0x25, 0x7a, 0xd4, 0xfb, 0xca, 0x62,
	0xa7, 0xd4, 0x1d, 0xa3, 0x8c, 0x85, 0x47, 0x1a, 0xa4, 0x55, 0x0e, 0x0f, 0xc0, 0xae, 0x69, 0x35,
	0xe0, 0x33, 0x9e, 0xeb, 0xa5, 0x06, 0x69, 0xd5, 0x82, 0xe1, 0xd3, 0xcb, 0x59, 0x6f, 0xce, 0xf5,
	0xc4, 0x8f, 0xa6, 0x69, 0x67, 0x24, 0x75, 0xf7, 0x5b, 0xed, 0x70, 0xa6, 0x50, 0x26, 0x63, 0xa1,
	0x57, 0xa8, 0x6e, 0x7d, 0x61, 0xa9, 0x9d, 0xa2, 0x9f, 0xe4, 0x63, 0x83, 0x69, 0x3a, 0x92, 0xba,
	0xcf, 0x33, 0x2d, 0x54, 0x58, 0xa4, 0xb2, 0x3a, 0xfd, 0xdf, 0xc7, 0x44, 0x5c, 0xf0, 0x6c, 0xe2,
	0xfd, 0xcb, 0x1b, 0xc2, 0x4f, 0xce, 0xbd, 0x10, 0x51, 0x5b, 0xaf, 0x7c, 0xf0, 0x0a, 0x66, 0x1e,
	0xad, 0xf6, 0x92, 0x44, 0x89, 0x2c, 0xf3, 0x5c, 0x6b, 0x15, 0xc8, 0x90, 0x1e, 0x0f, 0xc4, 0x9d,
	0x98, 0xe1, 0x42, 0xa8, 0x50, 0xac, 0xb8, 0x4a, 0xbc, 0xca, 0x5f, 0x4e, 0xff, 0x9d, 0xce, 0x9a,
	0xb4, 0x76, 0xb9, 0x92, 0x42, 0x15, 0x7b, 0xaa, 0x76, 0xcf, 0x0f, 0x2d, 0xe8, 0xae, 0xb7, 0xe0,
	0x6c, 0xb6, 0xe0, 0xec, 0xb7, 0x40, 0xee, 0x0d, 0x90, 0x47, 0x03, 0xe4, 0xd9, 0x00, 0x59, 0x1b,
	0x20, 0xaf, 0x06, 0xc8, 0x9b, 0x01, 0x67, 0x6f, 0x80, 0x3c, 0xec, 0xc0, 0x59, 0xef, 0xc0, 0xd9,
	0xec, 0xc0, 0xb9, 0x72, 0x33, 0xcd, 0xb5, 0x88, 0x2a, 0xf6, 0xc7, 0xce, 0x3f, 0x02, 0x00, 0x00,
	0xff, 0xff, 0x41, 0x57, 0x1f, 0x50, 0xfc, 0x01, 0x00, 0x00,
}

func (this *AccountData) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*AccountData)
	if !ok {
		that2, ok := that.(AccountData)
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
	{
		__caster := &github_com_ElrondNetwork_elrond_go_data.BigIntCaster{}
		if !__caster.Equal(this.Balance, that1.Balance) {
			return false
		}
	}
	if !bytes.Equal(this.CodeHash, that1.CodeHash) {
		return false
	}
	if !bytes.Equal(this.RootHash, that1.RootHash) {
		return false
	}
	if !bytes.Equal(this.Address, that1.Address) {
		return false
	}
	{
		__caster := &github_com_ElrondNetwork_elrond_go_data.BigIntCaster{}
		if !__caster.Equal(this.DeveloperReward, that1.DeveloperReward) {
			return false
		}
	}
	if !bytes.Equal(this.OwnerAddress, that1.OwnerAddress) {
		return false
	}
	return true
}
func (this *AccountData) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 11)
	s = append(s, "&state.AccountData{")
	s = append(s, "Nonce: "+fmt.Sprintf("%#v", this.Nonce)+",\n")
	s = append(s, "Balance: "+fmt.Sprintf("%#v", this.Balance)+",\n")
	s = append(s, "CodeHash: "+fmt.Sprintf("%#v", this.CodeHash)+",\n")
	s = append(s, "RootHash: "+fmt.Sprintf("%#v", this.RootHash)+",\n")
	s = append(s, "Address: "+fmt.Sprintf("%#v", this.Address)+",\n")
	s = append(s, "DeveloperReward: "+fmt.Sprintf("%#v", this.DeveloperReward)+",\n")
	s = append(s, "OwnerAddress: "+fmt.Sprintf("%#v", this.OwnerAddress)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringAccountData(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *AccountData) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *AccountData) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *AccountData) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.OwnerAddress) > 0 {
		i -= len(m.OwnerAddress)
		copy(dAtA[i:], m.OwnerAddress)
		i = encodeVarintAccountData(dAtA, i, uint64(len(m.OwnerAddress)))
		i--
		dAtA[i] = 0x3a
	}
	{
		__caster := &github_com_ElrondNetwork_elrond_go_data.BigIntCaster{}
		size := __caster.Size(m.DeveloperReward)
		i -= size
		if _, err := __caster.MarshalTo(m.DeveloperReward, dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintAccountData(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x32
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintAccountData(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0x2a
	}
	if len(m.RootHash) > 0 {
		i -= len(m.RootHash)
		copy(dAtA[i:], m.RootHash)
		i = encodeVarintAccountData(dAtA, i, uint64(len(m.RootHash)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.CodeHash) > 0 {
		i -= len(m.CodeHash)
		copy(dAtA[i:], m.CodeHash)
		i = encodeVarintAccountData(dAtA, i, uint64(len(m.CodeHash)))
		i--
		dAtA[i] = 0x1a
	}
	{
		__caster := &github_com_ElrondNetwork_elrond_go_data.BigIntCaster{}
		size := __caster.Size(m.Balance)
		i -= size
		if _, err := __caster.MarshalTo(m.Balance, dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintAccountData(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if m.Nonce != 0 {
		i = encodeVarintAccountData(dAtA, i, uint64(m.Nonce))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintAccountData(dAtA []byte, offset int, v uint64) int {
	offset -= sovAccountData(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *AccountData) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Nonce != 0 {
		n += 1 + sovAccountData(uint64(m.Nonce))
	}
	{
		__caster := &github_com_ElrondNetwork_elrond_go_data.BigIntCaster{}
		l = __caster.Size(m.Balance)
		n += 1 + l + sovAccountData(uint64(l))
	}
	l = len(m.CodeHash)
	if l > 0 {
		n += 1 + l + sovAccountData(uint64(l))
	}
	l = len(m.RootHash)
	if l > 0 {
		n += 1 + l + sovAccountData(uint64(l))
	}
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovAccountData(uint64(l))
	}
	{
		__caster := &github_com_ElrondNetwork_elrond_go_data.BigIntCaster{}
		l = __caster.Size(m.DeveloperReward)
		n += 1 + l + sovAccountData(uint64(l))
	}
	l = len(m.OwnerAddress)
	if l > 0 {
		n += 1 + l + sovAccountData(uint64(l))
	}
	return n
}

func sovAccountData(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozAccountData(x uint64) (n int) {
	return sovAccountData(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *AccountData) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&AccountData{`,
		`Nonce:` + fmt.Sprintf("%v", this.Nonce) + `,`,
		`Balance:` + fmt.Sprintf("%v", this.Balance) + `,`,
		`CodeHash:` + fmt.Sprintf("%v", this.CodeHash) + `,`,
		`RootHash:` + fmt.Sprintf("%v", this.RootHash) + `,`,
		`Address:` + fmt.Sprintf("%v", this.Address) + `,`,
		`DeveloperReward:` + fmt.Sprintf("%v", this.DeveloperReward) + `,`,
		`OwnerAddress:` + fmt.Sprintf("%v", this.OwnerAddress) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringAccountData(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *AccountData) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAccountData
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
			return fmt.Errorf("proto: AccountData: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: AccountData: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nonce", wireType)
			}
			m.Nonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAccountData
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
				return fmt.Errorf("proto: wrong wireType = %d for field Balance", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAccountData
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
				return ErrInvalidLengthAccountData
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthAccountData
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			{
				__caster := &github_com_ElrondNetwork_elrond_go_data.BigIntCaster{}
				if tmp, err := __caster.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
					return err
				} else {
					m.Balance = tmp
				}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CodeHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAccountData
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
				return ErrInvalidLengthAccountData
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthAccountData
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.CodeHash = append(m.CodeHash[:0], dAtA[iNdEx:postIndex]...)
			if m.CodeHash == nil {
				m.CodeHash = []byte{}
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RootHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAccountData
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
				return ErrInvalidLengthAccountData
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthAccountData
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RootHash = append(m.RootHash[:0], dAtA[iNdEx:postIndex]...)
			if m.RootHash == nil {
				m.RootHash = []byte{}
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAccountData
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
				return ErrInvalidLengthAccountData
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthAccountData
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = append(m.Address[:0], dAtA[iNdEx:postIndex]...)
			if m.Address == nil {
				m.Address = []byte{}
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DeveloperReward", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAccountData
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
				return ErrInvalidLengthAccountData
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthAccountData
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			{
				__caster := &github_com_ElrondNetwork_elrond_go_data.BigIntCaster{}
				if tmp, err := __caster.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
					return err
				} else {
					m.DeveloperReward = tmp
				}
			}
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OwnerAddress", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAccountData
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
				return ErrInvalidLengthAccountData
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthAccountData
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.OwnerAddress = append(m.OwnerAddress[:0], dAtA[iNdEx:postIndex]...)
			if m.OwnerAddress == nil {
				m.OwnerAddress = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAccountData(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthAccountData
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthAccountData
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
func skipAccountData(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowAccountData
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
					return 0, ErrIntOverflowAccountData
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
					return 0, ErrIntOverflowAccountData
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
				return 0, ErrInvalidLengthAccountData
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupAccountData
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthAccountData
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthAccountData        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowAccountData          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupAccountData = fmt.Errorf("proto: unexpected end of group")
)
