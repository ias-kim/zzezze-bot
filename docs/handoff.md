# Handoff

> 하루 작업이 끝날 때마다 이 파일을 **덮어써서** 갱신한다. 지난 기록은 `git log -p docs/handoff.md`로 본다.
> 새 세션은 이 파일 → `docs/exec-plans/active/` 순서로 읽고 시작한다.

- 마지막 갱신: 2026-10-15
- 브랜치: `feat/phase1-chicken` (PR #1, main 미병합)
- 진행 중 계획: [phase1-chicken.md](exec-plans/active/phase1-chicken.md)

## 오늘 한 것

- reviewer 서브에이전트의 문서·코드 불일치 감사(23건)를 반영했다.
  - README를 링크 중심으로 다시 썼다. 값을 복사하지 않아서 다시 어긋나지 않는다.
  - `make check`에 `tidy`(go.sum 정리 여부)와 `agents`(.claude/.codex 지시문 일치)를 추가했다.
  - 계층 검사를 강화했다. 테스트 파일 import와 새 패키지도 자동으로 검사한다. 일부러 위반을 넣어 실패하는지 확인했다.
  - voice 대사 키 목록과 대사 표가 정확히 같은 집합인지 테스트한다.
  - `main.go`가 `config.Lookup()`을 쓰도록 바꿨다.
  - profile.md 안의 모순을 해소하고 ADR 0005(store/ 리뷰 필수)를 기록했다.
- 비용을 추정했고 배포 서버를 t4g.nano(월 약 $8)로 정했다.
- 서브에이전트 모델을 지정했다(explorer=haiku, reviewer=sonnet).

## 다음 할 것

1. **1차 3단계**: `session/registry.go`를 TDD로 만든다(`Touch`, `Finish` 1회 보장, `CloseAll`, 유저 역인덱스, `go test -race`).
2. 4단계: `features/autoreply.go`

## 막힌 것 / 사용자에게 필요한 것

- **2번방 문장 수집 과제**: 치킨이나 시간 얘기가 나온 실제 문장을 원문 그대로 모아야 한다. 5단계(nudge)를 시작하기 전에 필요하다.
- **`voice/lines.go` 대사 다듬기**: 지금은 초안이다. 방 말투에 맞게 사용자가 고쳐야 한다.
- PR #1 병합 여부는 사용자가 결정한다.

## 새로 내린 결정

- t4g.nano로 배포한다.
- Codex 무료 플랜은 게이트에 쓰지 않는다. 검증은 Claude 서브에이전트가 맡는다.
- 하루 마무리 때 이 파일을 갱신하고 push한다.
- `store/` PR은 reviewer 필수 (ADR 0005)
