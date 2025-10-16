#Покажет детализацию до названий всех тестов
go test -v ./...
#Покажет процент покрытия тестами, 2 способа
go test ./... -coverprofile=coverage.html
go test ./... -cover
go test -v -coverpkg=./... -coverprofile=coverage.html ./... #покажет еще и общее покрытие тестами

#Покажет покрытие в файлах
go tool cover -html=coverage.html

#Все вместе
go test ./... -v -coverpkg=./... -coverprofile=coverage.html && go tool cover -html=coverage.html

#Покажет общее покрытие тестами, вывод в консоль, не в html
go test -v -coverpkg=./... -coverprofile=coverage.html ./... && go tool cover -func coverage.html
#Исключая сгенерированные модули, их надо исключать
go test -v -coverpkg=./... -coverprofile=coverage.html ./...
/internal/server/models/models_easyjson /internal/server/repository/mock/mock_store #вручную удаляем
go tool cover -func coverage.html

#Бенчмарки. надо переключаться на директорию и указывать точку, ./... не работает. -benchmem - добавляет данные по оперативке к процессору
cd /home/stanislav/go/metrics/cmd/agent && go test -bench . -benchmem