#покажет детализацию до названий всех тестов
go test -v ./...
#покажет процент покрытия тестами
go test ./... -coverprofile=.coverage.html

#покажет покрытие в файлах
go tool cover -html=coverage.html

#бенчмарки. надо переключаться на директорию и указывать точку, ./... не работает. -benchmem - добавляет данные по оперативке к процессору
cd /home/stanislav/go/metrics/cmd/agent && go test -bench . -benchmem