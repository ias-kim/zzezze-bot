# Handoff

> 하루 작업이 끝날 때마다 이 파일을 **덮어써서** 갱신한다. 지난 기록은 `git log -p docs/handoff.md`로 본다.
> 새 세션은 이 파일 → `docs/exec-plans/active/` 순서로 읽고 시작한다.

- 마지막 갱신: 2026-10-16 (작업일 2026-10-15)
- 브랜치: `claude/stoic-brown-623ddx` (`feat/phase1-chicken`은 PR #1로 main에 병합됨)
- 진행 중 계획: [phase1-meetup.md](exec-plans/active/phase1-meetup.md)
- 2026-10-17 방향 전환: 1차 중심 기능을 치킨 판에서 **약속 카드**로 바꿨다([ADR 0006](adr/0006-meetup-first-rollout.md)). 아래 내용 중 치킨 판 관련 항목은 이 결정으로 대체된다.

## 오늘 한 것

- reviewer 서브에이전트의 문서·코드 불일치 감사(23건)를 반영했다.
  - README를 링크 중심으로 다시 썼다. 값을 복사하지 않아서 다시 어긋나지 않는다.
  - `make check`에 `tidy`(go.sum 정리 여부)와 `agents`(.claude/.codex 지시문 일치)를 추가했다.
  - 계층 검사를 강화했다. 테스트 파일 import와 새 패키지도 자동으로 검사한다. 일부러 위반을 넣어 실패하는지 확인했다.
  - voice 대사 키 목록과 대사 표가 정확히 같은 집합인지 테스트한다.
  - `main.go`가 `config.Lookup()`을 쓰도록 바꿨다.
  - profile.md 안의 모순을 해소하고 ADR 0005(store/ 리뷰 필수)를 기록했다.
  - 재검증(reviewer, sonnet): 22건은 바로 해결로 확인됐다. 남은 1건(설계 문서의 `Roller` 표기)과 새로 생긴 경고 2건(AGENTS.md 의존 방향 문장, 검사 항목 목록 사본)도 고쳤다.
  - 서브에이전트 실측: 감사 73k 토큰(opus), 재검증 44k 토큰(sonnet)
- 비용을 추정했고 배포 서버를 t4g.nano(월 약 $8)로 정했다.
- 서브에이전트 모델을 지정했다(explorer=haiku, reviewer=sonnet).

## 다음 할 것

1. **1차 3단계**: `session/registry.go`를 TDD로 만든다(`Touch`, `Finish` 1회 보장, `CloseAll`, 유저 역인덱스, `go test -race`).
2. 4단계: `features/autoreply.go`

## 막힌 것 / 사용자에게 필요한 것

- **2번방 문장 수집 과제**: 약속 얘기가 나온 실제 문장을 원문 그대로 더 모아야 한다(특히 `~맥` 문장). 초안은 `meetup/testdata/{hit,miss}.txt`에 있고, hit이 3줄뿐이다. 5단계를 시작하기 전에 필요하다.
- **`voice/lines.go` 대사 다듬기**: 지금은 초안이다. 방 말투에 맞게 사용자가 고쳐야 한다.
- 작업 폴더의 `.obsidian/`, `StudyVault/`는 커밋하지 않은 상태다(사용자 개인 노트로 보임). 레포에 올릴지, `.gitignore`에 넣을지 사용자가 정해야 한다.

## 새로 내린 결정

- t4g.nano로 배포한다.
- Codex 무료 플랜은 게이트에 쓰지 않는다. 검증은 Claude 서브에이전트가 맡는다.
- 하루 마무리 때 이 파일을 갱신하고 push한다.
- `store/` PR은 reviewer 필수 (ADR 0005)
