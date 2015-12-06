// Code generated by protoc-gen-gogo.
// source: internal/internal.proto
// DO NOT EDIT!

/*
Package internal is a generated protocol buffer package.

It is generated from these files:
	internal/internal.proto

It has these top-level messages:
	TargetSnapshot
	FileSnapshot
*/
package internal

import proto "github.com/gogo/protobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type TargetSnapshot struct {
	Name             *string         `protobuf:"bytes,1,req" json:"Name,omitempty"`
	Hash             *string         `protobuf:"bytes,2,req" json:"Hash,omitempty"`
	Inputs           []*FileSnapshot `protobuf:"bytes,3,rep" json:"Inputs,omitempty"`
	XXX_unrecognized []byte          `json:"-"`
}

func (m *TargetSnapshot) Reset()         { *m = TargetSnapshot{} }
func (m *TargetSnapshot) String() string { return proto.CompactTextString(m) }
func (*TargetSnapshot) ProtoMessage()    {}

func (m *TargetSnapshot) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *TargetSnapshot) GetHash() string {
	if m != nil && m.Hash != nil {
		return *m.Hash
	}
	return ""
}

func (m *TargetSnapshot) GetInputs() []*FileSnapshot {
	if m != nil {
		return m.Inputs
	}
	return nil
}

type FileSnapshot struct {
	Name             *string `protobuf:"bytes,1,req" json:"Name,omitempty"`
	Hash             *string `protobuf:"bytes,2,req" json:"Hash,omitempty"`
	Content          *string `protobuf:"bytes,3,req" json:"Content,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *FileSnapshot) Reset()         { *m = FileSnapshot{} }
func (m *FileSnapshot) String() string { return proto.CompactTextString(m) }
func (*FileSnapshot) ProtoMessage()    {}

func (m *FileSnapshot) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *FileSnapshot) GetHash() string {
	if m != nil && m.Hash != nil {
		return *m.Hash
	}
	return ""
}

func (m *FileSnapshot) GetContent() string {
	if m != nil && m.Content != nil {
		return *m.Content
	}
	return ""
}

func init() {
}
