run:
	go run ./examples/status.go

plot:
	go run ./metrics/plotter/ ${ARGS}

clean:
	kubectl delete deployments -l go.dedis.ch.app=simnet \
	&& kubectl delete service simnet-router

build_monitor:
	docker build -t dedis/simnet-monitor -f monitor/Dockerfile .

build_router:
	docker build -t dedis/simnet-router-init -f router/Init.Dockerfile .
	docker build -t dedis/simnet-router -f router/Dockerfile .
