build:
	go build -o ./ctr ./controller/main.go
	docker build -t envoy:v1 .