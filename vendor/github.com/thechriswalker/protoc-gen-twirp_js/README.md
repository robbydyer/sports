# `protoc-gen-twirp_js`

Creates javascript bindings compatible with both the browser and node.js as
common-js modules. This means that you should run your `protoc` with options
for the `--js-out` as `--js-out=import_style=commonjs,binary:<path>`.

The resulting javascript files `<service>_pb.js` and `<service>_pb_twirp.js` will
be compatible with all commonjs aware module systems, for example, nodejs, browserify,
webpack, rollup, etc...

## Caveats

1. I am not completely happy with the JSON support, I do not believe that the serialisation
   the [Google PB JS](https://github.com/google/protobuf/tree/master/js) output provides 
   matches the protocol buffers [JSON serialisation definition](https://developers.google.com/protocol-buffers/docs/proto3#json). 
   Nor does the other prominent alternative [dcodeIO/protobuf.js](https://github.com/dcodeIO/protobuf.js),
   which leaves implementing it myself, and then probably the entire generator, or not bothering. Given there
   is little incentive to actually use JSON on the wire, I lean towards removing/ignoring the JSON interop - this library deals
   with plain objects either way, only the wire format changes.
2. Twirp is adding Streaming support. I don't really want to implement that. The simplicity 
   of the RPC (in contrast to grpc for example) was a major selling point to me.
