//go:build tools
// +build tools

package tools

import (
	_ "github.com/go-bridget/twirp-swagger-gen/cmd/twirp-swagger-gen"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	_ "github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc"
	_ "github.com/srikrsna/protoc-gen-gotag"
	_ "github.com/thechriswalker/protoc-gen-twirp_js"
	_ "github.com/twitchtv/twirp/protoc-gen-twirp"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
