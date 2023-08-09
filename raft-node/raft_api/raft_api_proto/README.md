# GRPC & Protobuf

Run the following in this directory to auto-generate grpc code in GO.

```bash
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    raft_api.proto
```
