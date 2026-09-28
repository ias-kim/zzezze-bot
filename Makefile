# 게이트 1(기계 검사). 로컬과 CI 가 같은 명령을 쓴다: make check
.PHONY: check fmt vet build test layers cross run

check: fmt vet build test layers cross

fmt:
	@out="$$(gofmt -l .)"; \
	if [ -n "$$out" ]; then echo "gofmt 로 정리 안 된 파일:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

build:
	go build ./...

# -race: session 패키지의 타이머/클릭 경합을 잡는다.
test:
	go test -race ./...

layers:
	bash scripts/check-layers.sh

# EC2(t4g, arm64) 배포 대상 크로스 빌드가 깨지지 않았는지 확인.
cross:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /dev/null .

run:
	go run .
