#Почему-то слетает иногда
export PATH=$PATH:$(go env GOPATH)/bin
#Установка сваггера
go install github.com/swaggo/swag/cmd/swag@latest
#Генерирование документации
#--parseDependency надо ставить если модели находятся в других пакетах. например models.Metric
cd /home/stanislav/go/metrics/internal/server/handlers/ && swag init -g router.go --output ./swagger/ --parseDependency