#https://protobuf.dev/installation/
sudo apt install -y protobuf-compiler
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

cd /home/stanislav/go/metrics && export PATH=$PATH:$(go env GOPATH)/bin && protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  models/grps/metric.proto
