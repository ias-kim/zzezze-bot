# ADR 0001: 하네스 엔지니어링 도입 (Lv.2)

## 상태

승인됨 (2026-09-28)

## 맥락

개인 사이드 프로젝트지만 Claude Code와 Codex를 함께 쓰기 시작하면서, 에이전트가 매번 같은 질문(명령어가 뭔지, 어디에 뭘 적는지)을 하지 않도록 최소한의 구조가 필요했다. `AGENTS.md`가 비어 있었고 CI도 없었다.

## 결정

- `AGENTS.md`를 공통 지도로 두고, `CLAUDE.md`는 그쪽을 따르라는 안내 + Claude Code 전용 메모만 담는다.
- 사실은 `docs/sot.md` 한 곳에 표로만 관리한다 (DB/API SoT는 이 프로젝트에 해당 없음).
- 위험 영역이 없어 게이트는 Lv.1(기계 게이트: build/vet/test/gofmt, GitHub Actions)만 도입한다. 의미 게이트(reviewer 서브에이전트)는 만들어두되 필수로 강제하지 않는다.
- 결정론적 산출물(생성기 → 산출물 diff 체크)은 현재 생성되는 파생 파일이 없어 도입하지 않는다.

## 후속 변경

- 2026-09-28: 게이트 1은 `make check`(Makefile)로 묶였다. 지금 무엇을 검사하는지의 원천은 Makefile이다.
- 2026-10-15: `store/`에 한해 의미 게이트를 필수로 바꿨다 → [ADR 0005](0005-store-review-gate.md).

## 결과

- 신규 기여(미래의 나 포함)가 `AGENTS.md` 하나만 읽으면 시작할 수 있다.
- PR마다 `go build/vet/test/gofmt`가 CI에서 자동으로 돈다.
- 프로젝트가 커지면(여러 서버 확장, 영속 저장소 도입 등) `docs/harness/profile.md`를 갱신하고 레벨을 올린다.
