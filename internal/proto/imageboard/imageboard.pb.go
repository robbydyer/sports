// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.15.7
// source: imageboard/imageboard.proto

package imageboard

import (
	empty "github.com/golang/protobuf/ptypes/empty"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Status struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Enabled          bool `protobuf:"varint,1,opt,name=enabled,proto3" json:"enabled,omitempty"`
	DiskcacheEnabled bool `protobuf:"varint,2,opt,name=diskcache_enabled,json=diskcacheEnabled,proto3" json:"diskcache_enabled,omitempty"`
	MemcacheEnabled  bool `protobuf:"varint,3,opt,name=memcache_enabled,json=memcacheEnabled,proto3" json:"memcache_enabled,omitempty"`
}

func (x *Status) Reset() {
	*x = Status{}
	if protoimpl.UnsafeEnabled {
		mi := &file_imageboard_imageboard_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Status) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Status) ProtoMessage() {}

func (x *Status) ProtoReflect() protoreflect.Message {
	mi := &file_imageboard_imageboard_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Status.ProtoReflect.Descriptor instead.
func (*Status) Descriptor() ([]byte, []int) {
	return file_imageboard_imageboard_proto_rawDescGZIP(), []int{0}
}

func (x *Status) GetEnabled() bool {
	if x != nil {
		return x.Enabled
	}
	return false
}

func (x *Status) GetDiskcacheEnabled() bool {
	if x != nil {
		return x.DiskcacheEnabled
	}
	return false
}

func (x *Status) GetMemcacheEnabled() bool {
	if x != nil {
		return x.MemcacheEnabled
	}
	return false
}

type SetStatusReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Status *Status `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
}

func (x *SetStatusReq) Reset() {
	*x = SetStatusReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_imageboard_imageboard_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SetStatusReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetStatusReq) ProtoMessage() {}

func (x *SetStatusReq) ProtoReflect() protoreflect.Message {
	mi := &file_imageboard_imageboard_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetStatusReq.ProtoReflect.Descriptor instead.
func (*SetStatusReq) Descriptor() ([]byte, []int) {
	return file_imageboard_imageboard_proto_rawDescGZIP(), []int{1}
}

func (x *SetStatusReq) GetStatus() *Status {
	if x != nil {
		return x.Status
	}
	return nil
}

type StatusResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Status *Status `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
}

func (x *StatusResp) Reset() {
	*x = StatusResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_imageboard_imageboard_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatusResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusResp) ProtoMessage() {}

func (x *StatusResp) ProtoReflect() protoreflect.Message {
	mi := &file_imageboard_imageboard_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusResp.ProtoReflect.Descriptor instead.
func (*StatusResp) Descriptor() ([]byte, []int) {
	return file_imageboard_imageboard_proto_rawDescGZIP(), []int{2}
}

func (x *StatusResp) GetStatus() *Status {
	if x != nil {
		return x.Status
	}
	return nil
}

type JumpReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *JumpReq) Reset() {
	*x = JumpReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_imageboard_imageboard_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JumpReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JumpReq) ProtoMessage() {}

func (x *JumpReq) ProtoReflect() protoreflect.Message {
	mi := &file_imageboard_imageboard_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JumpReq.ProtoReflect.Descriptor instead.
func (*JumpReq) Descriptor() ([]byte, []int) {
	return file_imageboard_imageboard_proto_rawDescGZIP(), []int{3}
}

func (x *JumpReq) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

var File_imageboard_imageboard_proto protoreflect.FileDescriptor

var file_imageboard_imageboard_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x2f, 0x69, 0x6d, 0x61,
	0x67, 0x65, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0d, 0x69,
	0x6d, 0x61, 0x67, 0x65, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x2e, 0x76, 0x31, 0x1a, 0x1b, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d,
	0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x7a, 0x0a, 0x06, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x12, 0x2b, 0x0a,
	0x11, 0x64, 0x69, 0x73, 0x6b, 0x63, 0x61, 0x63, 0x68, 0x65, 0x5f, 0x65, 0x6e, 0x61, 0x62, 0x6c,
	0x65, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x10, 0x64, 0x69, 0x73, 0x6b, 0x63, 0x61,
	0x63, 0x68, 0x65, 0x45, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x12, 0x29, 0x0a, 0x10, 0x6d, 0x65,
	0x6d, 0x63, 0x61, 0x63, 0x68, 0x65, 0x5f, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x0f, 0x6d, 0x65, 0x6d, 0x63, 0x61, 0x63, 0x68, 0x65, 0x45, 0x6e,
	0x61, 0x62, 0x6c, 0x65, 0x64, 0x22, 0x3d, 0x0a, 0x0c, 0x53, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x52, 0x65, 0x71, 0x12, 0x2d, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x62, 0x6f, 0x61,
	0x72, 0x64, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x22, 0x3b, 0x0a, 0x0a, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65,
	0x73, 0x70, 0x12, 0x2d, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x15, 0x2e, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x2e,
	0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x22, 0x1d, 0x0a, 0x07, 0x4a, 0x75, 0x6d, 0x70, 0x52, 0x65, 0x71, 0x12, 0x12, 0x0a, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x32, 0xc6, 0x01, 0x0a, 0x0a, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x42, 0x6f, 0x61, 0x72, 0x64, 0x12,
	0x40, 0x0a, 0x09, 0x53, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x1b, 0x2e, 0x69,
	0x6d, 0x61, 0x67, 0x65, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x74,
	0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x71, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74,
	0x79, 0x12, 0x3e, 0x0a, 0x09, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x16,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x19, 0x2e, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x62, 0x6f,
	0x61, 0x72, 0x64, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x12, 0x36, 0x0a, 0x04, 0x4a, 0x75, 0x6d, 0x70, 0x12, 0x16, 0x2e, 0x69, 0x6d, 0x61, 0x67,
	0x65, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x2e, 0x76, 0x31, 0x2e, 0x4a, 0x75, 0x6d, 0x70, 0x52, 0x65,
	0x71, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x42, 0x37, 0x5a, 0x35, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x6f, 0x62, 0x62, 0x79, 0x64, 0x79, 0x65,
	0x72, 0x2f, 0x73, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61,
	0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x62, 0x6f, 0x61,
	0x72, 0x64, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_imageboard_imageboard_proto_rawDescOnce sync.Once
	file_imageboard_imageboard_proto_rawDescData = file_imageboard_imageboard_proto_rawDesc
)

func file_imageboard_imageboard_proto_rawDescGZIP() []byte {
	file_imageboard_imageboard_proto_rawDescOnce.Do(func() {
		file_imageboard_imageboard_proto_rawDescData = protoimpl.X.CompressGZIP(file_imageboard_imageboard_proto_rawDescData)
	})
	return file_imageboard_imageboard_proto_rawDescData
}

var file_imageboard_imageboard_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_imageboard_imageboard_proto_goTypes = []interface{}{
	(*Status)(nil),       // 0: imageboard.v1.Status
	(*SetStatusReq)(nil), // 1: imageboard.v1.SetStatusReq
	(*StatusResp)(nil),   // 2: imageboard.v1.StatusResp
	(*JumpReq)(nil),      // 3: imageboard.v1.JumpReq
	(*empty.Empty)(nil),  // 4: google.protobuf.Empty
}
var file_imageboard_imageboard_proto_depIdxs = []int32{
	0, // 0: imageboard.v1.SetStatusReq.status:type_name -> imageboard.v1.Status
	0, // 1: imageboard.v1.StatusResp.status:type_name -> imageboard.v1.Status
	1, // 2: imageboard.v1.ImageBoard.SetStatus:input_type -> imageboard.v1.SetStatusReq
	4, // 3: imageboard.v1.ImageBoard.GetStatus:input_type -> google.protobuf.Empty
	3, // 4: imageboard.v1.ImageBoard.Jump:input_type -> imageboard.v1.JumpReq
	4, // 5: imageboard.v1.ImageBoard.SetStatus:output_type -> google.protobuf.Empty
	2, // 6: imageboard.v1.ImageBoard.GetStatus:output_type -> imageboard.v1.StatusResp
	4, // 7: imageboard.v1.ImageBoard.Jump:output_type -> google.protobuf.Empty
	5, // [5:8] is the sub-list for method output_type
	2, // [2:5] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_imageboard_imageboard_proto_init() }
func file_imageboard_imageboard_proto_init() {
	if File_imageboard_imageboard_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_imageboard_imageboard_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Status); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_imageboard_imageboard_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SetStatusReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_imageboard_imageboard_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StatusResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_imageboard_imageboard_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*JumpReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_imageboard_imageboard_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_imageboard_imageboard_proto_goTypes,
		DependencyIndexes: file_imageboard_imageboard_proto_depIdxs,
		MessageInfos:      file_imageboard_imageboard_proto_msgTypes,
	}.Build()
	File_imageboard_imageboard_proto = out.File
	file_imageboard_imageboard_proto_rawDesc = nil
	file_imageboard_imageboard_proto_goTypes = nil
	file_imageboard_imageboard_proto_depIdxs = nil
}
