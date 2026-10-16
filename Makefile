# 게이트 1(기계 검사). 로컬과 CI 가 같은 명령을 쓴다: make check
.PHONY: check fmt vet build test layers agents tidy cross run

check: fmt tidy vet build test layers agents cross

# go.mod/go.sum 이 정리된 상태인지 (안 쓰는 의존성 흔적이 남으면 실패)
tidy:
	go mod tidy -diff

# .claude/agents 와 .codex/agents 지시문이 같은지
agents:
	bash scripts/check-agents-sync.sh

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
