#сделать перед использованием curl
curl.snap-acked

#снять 30 сек с оперативной памяти профиля текущей запущенной программы и сохранить
cd /home/stanislav/go/metrics/ && curl http://127.0.0.1:8080/debug/pprof/heap?seconds=30 > ./profiles/base.pprof

#сохраняем метрики из агента
cd /home/stanislav/go/metrics/cmd/agent && go test -bench=. -benchmem -cpuprofile=../../profiles/cpu.pprof -memprofile=../../profiles/base.pprof

#вывести анализ сохраненного профиля
cd /home/stanislav/go/metrics/ && go tool pprof -http=":9090" ./profiles/base.pprof
