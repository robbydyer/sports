# twirp-swagger-gen

A Twirp RPC Swagger/OpenAPI 2.0 generator 

# Usage

```
go get github.com/go-bridget/twirp-swagger-gen
go run github.com/go-bridget/twirp-swagger-gen \
	-in example/example.proto \
	-out example/example.swagger.json \
	-host test.example.com
```

# Why?

The project
[elliots/protoc-gen-twirp_swagger](https://github.com/elliots/protoc-gen-twirp_swagger)
is [defunct due to upstream changes to grpc-ecosystem
dependencies](https://github.com/elliots/protoc-gen-twirp_swagger/issues/25).

This project is a rewrite, that relies on both the official OpenAPI
structures, and a generic .proto file parser. The output should be line
compatible - my goal was just to replace the generator with a working one
without still being exposed to breaking changes from the gRPC ecosystem
packages.

The generated output is suitable for Swagger-UI.
