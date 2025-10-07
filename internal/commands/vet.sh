#Установка, эта программа будет лежать в /home/stanislav/go/bin
go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
#скрытые переменные, полезно
go vet -vettool=../bin/shadow ./...