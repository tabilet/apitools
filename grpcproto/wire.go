package grpcproto

import (
	"fmt"
	"io"
)

func parseBinaryDescriptorSet(data []byte) (*Model, error) {
	reader := wireReader{data: data}
	model := &Model{SourceKind: ArtifactKindDescriptorSet}
	for !reader.done() {
		fieldNumber, wireType, err := reader.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case fieldNumber == 1 && wireType == 2:
			payload, err := reader.bytes()
			if err != nil {
				return nil, err
			}
			file, err := parseFileDescriptorProto(payload)
			if err != nil {
				return nil, err
			}
			if file != nil {
				model.Files = append(model.Files, file)
			}
		default:
			if err := reader.skip(wireType); err != nil {
				return nil, err
			}
		}
	}
	if len(model.Files) == 0 {
		return nil, fmt.Errorf("grpcproto: descriptor set has no file descriptors")
	}
	finalizeModel(model)
	if !modelHasMetadata(model) {
		return nil, fmt.Errorf("grpcproto: descriptor set has no services, messages, or enums")
	}
	return model, nil
}

func parseFileDescriptorProto(data []byte) (*File, error) {
	reader := wireReader{data: data}
	file := &File{}
	for !reader.done() {
		fieldNumber, wireType, err := reader.tag()
		if err != nil {
			return nil, err
		}
		switch fieldNumber {
		case 1:
			file.Name, err = reader.string(wireType)
		case 2:
			file.Package, err = reader.string(wireType)
		case 3:
			value, e := reader.string(wireType)
			err = e
			if value != "" {
				file.Imports = append(file.Imports, value)
			}
		case 4:
			var payload []byte
			payload, err = reader.bytesOfType(wireType)
			if err == nil {
				var message *Message
				message, err = parseDescriptorProto(payload, file.Package, "")
				if message != nil {
					file.Messages = append(file.Messages, message)
				}
			}
		case 5:
			var payload []byte
			payload, err = reader.bytesOfType(wireType)
			if err == nil {
				var enum *Enum
				enum, err = parseEnumDescriptorProto(payload, file.Package, "")
				if enum != nil {
					file.Enums = append(file.Enums, enum)
				}
			}
		case 6:
			var payload []byte
			payload, err = reader.bytesOfType(wireType)
			if err == nil {
				var service *Service
				service, err = parseServiceDescriptorProto(payload, file.Package)
				if service != nil {
					file.Services = append(file.Services, service)
				}
			}
		case 12:
			file.Syntax, err = reader.string(wireType)
		default:
			err = reader.skip(wireType)
		}
		if err != nil {
			return nil, err
		}
	}
	return file, nil
}

func parseServiceDescriptorProto(data []byte, packageName string) (*Service, error) {
	reader := wireReader{data: data}
	service := &Service{Package: packageName}
	for !reader.done() {
		fieldNumber, wireType, err := reader.tag()
		if err != nil {
			return nil, err
		}
		switch fieldNumber {
		case 1:
			service.Name, err = reader.string(wireType)
		case 2:
			var payload []byte
			payload, err = reader.bytesOfType(wireType)
			if err == nil {
				var method *Method
				method, err = parseMethodDescriptorProto(payload, packageName)
				if method != nil {
					service.Methods = append(service.Methods, method)
				}
			}
		default:
			err = reader.skip(wireType)
		}
		if err != nil {
			return nil, err
		}
	}
	if service.Name == "" {
		return nil, nil
	}
	service.FullName = joinName(packageName, service.Name)
	return service, nil
}

func parseMethodDescriptorProto(data []byte, packageName string) (*Method, error) {
	reader := wireReader{data: data}
	method := &Method{Package: packageName}
	for !reader.done() {
		fieldNumber, wireType, err := reader.tag()
		if err != nil {
			return nil, err
		}
		switch fieldNumber {
		case 1:
			method.Name, err = reader.string(wireType)
		case 2:
			method.RequestType, err = reader.string(wireType)
			method.RequestType = normalizeTypeName(method.RequestType)
		case 3:
			method.ResponseType, err = reader.string(wireType)
			method.ResponseType = normalizeTypeName(method.ResponseType)
		case 5:
			method.ClientStreaming, err = reader.bool(wireType)
		case 6:
			method.ServerStreaming, err = reader.bool(wireType)
		default:
			err = reader.skip(wireType)
		}
		if err != nil {
			return nil, err
		}
	}
	if method.Name == "" {
		return nil, nil
	}
	return method, nil
}

func parseDescriptorProto(data []byte, packageName, parent string) (*Message, error) {
	reader := wireReader{data: data}
	message := &Message{}
	for !reader.done() {
		fieldNumber, wireType, err := reader.tag()
		if err != nil {
			return nil, err
		}
		switch fieldNumber {
		case 1:
			message.Name, err = reader.string(wireType)
		case 2:
			var payload []byte
			payload, err = reader.bytesOfType(wireType)
			if err == nil {
				var field *Field
				field, err = parseFieldDescriptorProto(payload)
				if field != nil {
					message.Fields = append(message.Fields, field)
				}
			}
		case 3:
			var payload []byte
			payload, err = reader.bytesOfType(wireType)
			if err == nil {
				var nested *Message
				nested, err = parseDescriptorProto(payload, packageName, joinName(parent, message.Name))
				if nested != nil {
					message.Messages = append(message.Messages, nested)
				}
			}
		case 4:
			var payload []byte
			payload, err = reader.bytesOfType(wireType)
			if err == nil {
				var enum *Enum
				enum, err = parseEnumDescriptorProto(payload, packageName, joinName(parent, message.Name))
				if enum != nil {
					message.Enums = append(message.Enums, enum)
				}
			}
		default:
			err = reader.skip(wireType)
		}
		if err != nil {
			return nil, err
		}
	}
	if message.Name == "" {
		return nil, nil
	}
	message.FullName = joinName(packageName, joinName(parent, message.Name))
	return message, nil
}

func parseFieldDescriptorProto(data []byte) (*Field, error) {
	reader := wireReader{data: data}
	field := &Field{}
	for !reader.done() {
		fieldNumber, wireType, err := reader.tag()
		if err != nil {
			return nil, err
		}
		switch fieldNumber {
		case 1:
			field.Name, err = reader.string(wireType)
		case 3:
			field.Number, err = reader.int(wireType)
		case 4:
			var label int
			label, err = reader.int(wireType)
			field.Label = protoLabelNumberName(label)
			field.Repeated = field.Label == "repeated"
			field.Required = field.Label == "required"
		case 5:
			var typ int
			typ, err = reader.int(wireType)
			field.Type = protoTypeNumberName(typ)
		case 6:
			field.TypeName, err = reader.string(wireType)
			field.TypeName = normalizeTypeName(field.TypeName)
		case 10:
			field.JSONName, err = reader.string(wireType)
		default:
			err = reader.skip(wireType)
		}
		if err != nil {
			return nil, err
		}
	}
	if field.Name == "" {
		return nil, nil
	}
	return field, nil
}

func parseEnumDescriptorProto(data []byte, packageName, parent string) (*Enum, error) {
	reader := wireReader{data: data}
	enum := &Enum{}
	for !reader.done() {
		fieldNumber, wireType, err := reader.tag()
		if err != nil {
			return nil, err
		}
		switch fieldNumber {
		case 1:
			enum.Name, err = reader.string(wireType)
		case 2:
			var payload []byte
			payload, err = reader.bytesOfType(wireType)
			if err == nil {
				var value *EnumValue
				value, err = parseEnumValueDescriptorProto(payload)
				if value != nil {
					enum.Values = append(enum.Values, value)
				}
			}
		default:
			err = reader.skip(wireType)
		}
		if err != nil {
			return nil, err
		}
	}
	if enum.Name == "" {
		return nil, nil
	}
	enum.FullName = joinName(packageName, joinName(parent, enum.Name))
	return enum, nil
}

func parseEnumValueDescriptorProto(data []byte) (*EnumValue, error) {
	reader := wireReader{data: data}
	value := &EnumValue{}
	for !reader.done() {
		fieldNumber, wireType, err := reader.tag()
		if err != nil {
			return nil, err
		}
		switch fieldNumber {
		case 1:
			value.Name, err = reader.string(wireType)
		case 2:
			value.Number, err = reader.int(wireType)
		default:
			err = reader.skip(wireType)
		}
		if err != nil {
			return nil, err
		}
	}
	if value.Name == "" {
		return nil, nil
	}
	return value, nil
}

type wireReader struct {
	data []byte
	pos  int
}

func (r *wireReader) done() bool {
	return r.pos >= len(r.data)
}

func (r *wireReader) tag() (int, int, error) {
	value, err := r.varint()
	if err != nil {
		return 0, 0, err
	}
	return int(value >> 3), int(value & 0x7), nil
}

func (r *wireReader) string(wireType int) (string, error) {
	payload, err := r.bytesOfType(wireType)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func (r *wireReader) bytes() ([]byte, error) {
	size, err := r.varint()
	if err != nil {
		return nil, err
	}
	if size > uint64(len(r.data)-r.pos) {
		return nil, io.ErrUnexpectedEOF
	}
	out := r.data[r.pos : r.pos+int(size)]
	r.pos += int(size)
	return out, nil
}

func (r *wireReader) bytesOfType(wireType int) ([]byte, error) {
	if wireType != 2 {
		return nil, fmt.Errorf("grpcproto: expected length-delimited field, got wire type %d", wireType)
	}
	return r.bytes()
}

func (r *wireReader) bool(wireType int) (bool, error) {
	value, err := r.int(wireType)
	return value != 0, err
}

func (r *wireReader) int(wireType int) (int, error) {
	if wireType != 0 {
		return 0, fmt.Errorf("grpcproto: expected varint field, got wire type %d", wireType)
	}
	value, err := r.varint()
	return int(value), err
}

func (r *wireReader) varint() (uint64, error) {
	var value uint64
	for shift := 0; shift < 64; shift += 7 {
		if r.pos >= len(r.data) {
			return 0, io.ErrUnexpectedEOF
		}
		b := r.data[r.pos]
		r.pos++
		value |= uint64(b&0x7f) << shift
		if b < 0x80 {
			return value, nil
		}
	}
	return 0, fmt.Errorf("grpcproto: varint overflow")
}

func (r *wireReader) skip(wireType int) error {
	switch wireType {
	case 0:
		_, err := r.varint()
		return err
	case 1:
		return r.advance(8)
	case 2:
		_, err := r.bytes()
		return err
	case 5:
		return r.advance(4)
	default:
		return fmt.Errorf("grpcproto: unsupported wire type %d", wireType)
	}
}

func (r *wireReader) advance(n int) error {
	if n > len(r.data)-r.pos {
		return io.ErrUnexpectedEOF
	}
	r.pos += n
	return nil
}
