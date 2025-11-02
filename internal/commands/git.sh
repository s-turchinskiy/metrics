#Если вы хотите удалить файл из удаленного репозитория, но оставить его на вашем компьютере (например, вы ошибочно запушили файл с логами)
git rm --cached ./cmd/server/certificate/cert.pem && git commit -m "remove cert data" && git push origin main
