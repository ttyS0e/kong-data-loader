build:
	go build

build-image:
	docker build -t ttys0e/kong-data-loader:latest .
