#Удалить файл из удаленного репозитория, но оставить его на компьютере (например ошибочно запушили файл с логами)
git rm --cached ./cmd/server/certificate/cert.pem && git commit -m "remove cert data" && git push
