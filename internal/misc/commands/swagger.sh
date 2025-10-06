#Это вообще надо было сделать при установке go
export PATH=$PATH:$(go env GOPATH)/bin
#Установка сваггера
go install github.com/swaggo/swag/cmd/swag@latest
#Генерирование документации
cd /home/stanislav/go/metrics//internal/server/handlers/ && swag init -g router.go --output ./swagger/