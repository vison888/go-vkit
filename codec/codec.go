package codec

import (
	b "bytes"
	"encoding/json"
	"errors"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/encoding"
)

var (
	ErrInvalidMessage = errors.New("invalid message")
)

type JsonCodec struct{}
type ProtoCodec struct{}
type WrapCodec struct{ encoding.Codec }

var jsonpbMarshaler = &jsonpb.Marshaler{
	EnumsAsInts:  false,
	EmitDefaults: true,
	OrigName:     true,
}

var jsonpbUnmarshaler = &jsonpb.Unmarshaler{
	AllowUnknownFields: true,
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

func (ProtoCodec) Marshal(v interface{}) ([]byte, error) {
	m, ok := v.(proto.Message)
	if !ok {
		return nil, ErrInvalidMessage
	}
	return proto.Marshal(m)
}

func (ProtoCodec) Unmarshal(data []byte, v interface{}) error {
	m, ok := v.(proto.Message)
	if !ok {
		return ErrInvalidMessage
	}

	return proto.Unmarshal(data, m)
}

func (ProtoCodec) Name() string {
	return "proto"
}

func (JsonCodec) Marshal(v interface{}) ([]byte, error) {
	if pb, ok := v.(proto.Message); ok {
		s, err := jsonpbMarshaler.MarshalToString(pb)
		return []byte(s), err
	}

	if raw, ok := v.([]byte); ok {
		return raw, nil
	}

	return json.Marshal(v)
}

func (JsonCodec) Unmarshal(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	if pb, ok := v.(proto.Message); ok {
		return jsonpbUnmarshaler.Unmarshal(b.NewReader(data), pb)
	}
	return json.Unmarshal(data, v)
}

func (JsonCodec) Name() string {
	return "json"
}
