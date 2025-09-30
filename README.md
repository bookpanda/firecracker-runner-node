# firecracker-runner-node
# Generate Protobuf
```bash
protoc \
  --go_out=. \
  --go-grpc_out=. \
  ./proto/**/*.proto

```