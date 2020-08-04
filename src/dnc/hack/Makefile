.PHONY: build-dockerfile
build-dockerfile:
	docker build -t $(IMAGE) -f hack/Dockerfile.dev .
	docker push $(IMAGE)

.PHONY: pull-cni
pull-cni: 
	mkdir -p ./bin
	wget -q $(shell curl -sL https://api.github.com/repos/Azure/azure-container-networking/releases/latest | jq -r '.assets[].browser_download_url' | grep cni-linux)
	tar -zxvf *cni-linux*.tgz -C ./bin/
	rm *cni-linux*.tgz

.PHONY: start-dlv
start-dlv:
	echo "Cleaning up previous proc"
	if pgrep -f -x "./bin/dnc" > /dev/null; then kill $(pgrep -f -x "./bin/dnc"); fi
	dlv --listen=0.0.0.0:4000 --headless --api-version=2  exec ./bin/dnc
	