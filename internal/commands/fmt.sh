#Переформатирует все файлы
gofmt -w .
#Только покажет что можно переформатировать
gofmt -d .

#Установка goimports
sudo apt install golang-golang-x-tools
#Все то же самое, что gofmt, но еще сортирует импорты
goimports -local "github.com/s-turchinskiy/metrics" -w .
