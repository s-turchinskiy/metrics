#Покажет детализацию до названий всех тестов
go test -v ./...
#Покажет процент покрытия тестами, 2 способа
go test ./... -coverprofile=coverage.html
go test ./... -cover

#Покажет покрытие в файлах
go tool cover -html=coverage.html

#Все вместе
go test ./... -coverprofile=coverage.html && go tool cover -html=coverage.html

#Бенчмарки. надо переключаться на директорию и указывать точку, ./... не работает. -benchmem - добавляет данные по оперативке к процессору
cd /home/stanislav/go/metrics/cmd/agent && go test -bench . -benchmem