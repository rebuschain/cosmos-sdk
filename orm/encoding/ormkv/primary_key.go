package ormkv

import (
	"bytes"
	"io"

	"github.com/cosmos/cosmos-sdk/orm/types/ormerrors"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type PrimaryKeyCodec struct {
	*KeyCodec
	Type protoreflect.MessageType
}

func (p PrimaryKeyCodec) DecodeIndexKey(k, _ []byte) (indexFields, primaryKey []protoreflect.Value, err error) {
	indexFields, err = p.Decode(bytes.NewReader(k))

	// got prefix key
	if err == io.EOF {
		return indexFields, nil, nil
	} else if err != nil {
		return nil, nil, err
	}

	if len(indexFields) == len(p.FieldCodecs) {
		// for primary keys the index fields are the primary key
		// but only if we don't have a prefix key
		primaryKey = indexFields
	}
	return indexFields, primaryKey, nil

}

var _ IndexCodec = PrimaryKeyCodec{}

func (p PrimaryKeyCodec) DecodeKV(k, v []byte) (Entry, error) {
	values, err := p.Decode(bytes.NewReader(k))
	if err != nil {
		return nil, err
	}

	msg := p.Type.New().Interface()
	err = proto.Unmarshal(v, msg)
	if err != nil {
		return nil, err
	}

	return PrimaryKeyEntry{
		Key:   values,
		Value: msg,
	}, nil
}

func (p PrimaryKeyCodec) EncodeKV(entry Entry) (k, v []byte, err error) {
	pkEntry, ok := entry.(PrimaryKeyEntry)
	if !ok {
		return nil, nil, ormerrors.BadDecodeEntry
	}

	if pkEntry.Value.ProtoReflect().Descriptor().FullName() != p.Type.Descriptor().FullName() {
		return nil, nil, ormerrors.BadDecodeEntry
	}

	bz, err := p.KeyCodec.Encode(pkEntry.Key)
	if err != nil {
		return nil, nil, err
	}

	v, err = proto.MarshalOptions{Deterministic: true}.Marshal(pkEntry.Value)
	if err != nil {
		return nil, nil, err
	}

	return bz, v, nil
}

func (p *PrimaryKeyCodec) ClearValues(message protoreflect.Message) {
	for _, f := range p.FieldDescriptors {
		message.Clear(f)
	}
}

func (p *PrimaryKeyCodec) Unmarshal(key []protoreflect.Value, value []byte, message proto.Message) error {
	err := proto.Unmarshal(value, message)
	if err != nil {
		return err
	}

	// rehydrate primary key
	p.SetValues(message.ProtoReflect(), key)
	return nil
}