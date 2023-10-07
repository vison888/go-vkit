package codec

import (
	"encoding/json"
	"errors"

	"google.golang.org/grpc/encoding"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	ErrInvalidMessage = errors.New("invalid message")
)

type JsonCodec struct{}
type ProtoCodec struct{}
type WrapCodec struct{ encoding.Codec }

var jsonpbMarshaler = &protojson.MarshalOptions{
	//UseEnumNumbers: true,
	UseProtoNames:   true,
	EmitUnpopulated: true,
}

var jsonpbUnmarshaler = &protojson.UnmarshalOptions{
	DiscardUnknown: true,
}

var (
	DefaultGRPCCodecs = map[string]encoding.Codec{
		"application/json":                  JsonCodec{},
		"application/x-www-form-urlencoded": JsonCodec{},
		"multipart/form-data":               JsonCodec{},
		"application/proto":                 ProtoCodec{},
		"application/protobuf":              ProtoCodec{},
		"application/octet-stream":          ProtoCodec{},
		"application/grpc":                  ProtoCodec{},
		"application/grpc+proto":            ProtoCodec{},
	}
)

func (ProtoCodec) Marshal(v any) ([]byte, error) {
	m, ok := v.(proto.Message)
	if !ok {
		return nil, ErrInvalidMessage
	}
	return proto.Marshal(m)
}

func (ProtoCodec) Unmarshal(data []byte, v any) error {
	m, ok := v.(proto.Message)
	if !ok {
		return ErrInvalidMessage
	}

	return proto.Unmarshal(data, m)
}

func (ProtoCodec) Name() string {
	return "proto"
}

func (JsonCodec) Marshal(v any) ([]byte, error) {
	if pb, ok := v.(proto.Message); ok {
		s, err := jsonpbMarshaler.Marshal(pb)
		return s, err
	}

	if raw, ok := v.([]byte); ok {
		return raw, nil
	}

	return json.Marshal(v)
}

func (JsonCodec) Unmarshal(data []byte, v any) error {
	if len(data) == 0 {
		return nil
	}
	if pb, ok := v.(proto.Message); ok {
		return jsonpbUnmarshaler.Unmarshal(data, pb)
	}
	return json.Unmarshal(data, v)
}

func (JsonCodec) Name() string {
	return "json"
}
