// Package grpcproto parses metadata-only gRPC/protobuf source artifacts.
//
// It accepts local .proto files, JSON descriptor artifacts, and binary
// google.protobuf.FileDescriptorSet bytes. The package never opens gRPC
// channels, uses server reflection, executes RPCs, negotiates TLS, or resolves
// credentials.
package grpcproto
