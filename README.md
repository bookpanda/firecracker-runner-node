# firecracker-runner-node
```bash
go run cmd/main.go -port=50051

```
# Generate Protobuf
```bash
protoc \
  --go_out=. \
  --go-grpc_out=. \
  ./proto/**/*.proto

```