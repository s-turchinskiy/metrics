#Установка, эта программа будет лежать в /home/stanislav/go/bin
go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
#скрытые переменные, полезно
go vet -vettool=../bin/shadow ./...

#staticcheck
go install honnef.co/go/tools/cmd/staticcheck@latest
export PATH=$PATH:$(go env GOPATH)/bin && staticcheck ./...
https://staticcheck.dev/docs/checks/#S1030 или staticcheck -explain SA5000

#staticlint
cd /home/stanislav/go/metrics && go build ./cmd/staticlint
./staticlint ./...