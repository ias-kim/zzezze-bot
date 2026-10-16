# 하네스 프로파일

이 프로젝트에 하네스를 어떻게 적용하는지 정하는 설정값이다. 값이 바뀌면 이 파일을 먼저 고치고, 결정이 바뀌면 ADR을 남긴다.

## 프로젝트 조건

| 항목 | 값 |
|---|---|
| 프로젝트 성격 | 개인 학습/사이드 프로젝트. 사용자는 친구 5명 (BM 증바람 서버) |
| 스택 | Go, discordgo, godotenv. 버전은 `go.mod`가 원천 |
| DB | 1차: 없음. 2차: SQLite(`modernc.org/sqlite`) 가상 포인트 원장 |
| API 형태 | 없음 (Discord 게이트웨이 이벤트 기반) |
| 코딩 에이전트 | Claude Code (Pro 플랜) + Codex (무료 플랜) |
| CI | GitHub Actions `make check`. 공개 레포라 무료 |
| 배포 | EC2 t4g.nano (서울, arm64), systemd |
| 위험 영역 | `store/` (2차, 잃으면 신뢰가 깨지는 데이터), `.env`의 `DISCORD_TOKEN` 유출 |
| 교차 모델 검증 | 선택 사항. Codex 무료 플랜은 품질이 낮아 게이트로 쓰지 않는다 |
| 목표 레벨 | Lv.2 + `store/`에만 게이트 2 |

## 이 조건이라서 이렇게 한다

| 조건 | 선택 | 하지 않는 것 |
|---|---|---|
| 혼자 개발, 사용자 5명 | 문서는 지도(`AGENTS.md`), 설계, 진행 중 계획, handoff 위주 | 팀용 절차 (GC 정기 자동 실행, 승인 워크플로) |
| 배우면서 만듦 | 규칙을 스크립트로 강제 (`make check`) | 문서에만 있는 규칙 |
| Codex가 약함 | 검증은 Claude `reviewer` 서브에이전트가 맡음 | Codex 결과에 기대는 게이트 |
| Pro 플랜 사용량 한도 | 서브에이전트는 검증에만 쓰고 가벼운 모델 지정 (explorer=haiku, reviewer=sonnet). 단계마다 세션을 새로 시작하고 `docs/handoff.md`로 이어감 | 구현을 여러 에이전트로 쪼개기 |
| 공개 레포 | `.gitignore`로 비밀값 제외, 커밋 전 토큰 검사 | 비밀값 커밋 |
| 2차부터 포인트 원장 | `store/` 변경 PR만 `reviewer` 필수 ([ADR 0005](../adr/0005-store-review-gate.md)) | 모든 PR에 무거운 리뷰 |

## 게이트

| 게이트 | 내용 | 적용 |
|---|---|---|
| 1 기계 | `make check` (로컬 + CI) | 모든 PR |
| 2 의미 | `reviewer` 서브에이전트가 SoT, ADR, exec-plan 완료 기준과 대조 | `store/` 변경 PR은 필수. 그 외는 선택 |
| 3 교차 모델 | 다른 벤더 모델 리뷰 | 사용하지 않음 |

## 비용 (월 추정)

| 항목 | 비용 |
|---|---|
| Claude Code | Pro 정액. 추가 사용 결제는 꺼 둠. 관리 대상은 사용량 한도 (서브에이전트 1회 약 7만 토큰 실측) |
| Codex | 0 (무료 플랜) |
| GitHub Actions | 0 (공개 레포) |
| EC2 t4g.nano + 공인 IPv4 + EBS 8GB | 약 $8 (서울. 결제 전에 AWS 가격 계산기로 확인) |
| (2차) S3 백업 | 몇 센트 |

## 다음에 올릴 레벨

결정론적 산출물이 생길 때(예: 채널 설정을 YAML 원천에서 코드로 생성) `make generate && git diff --exit-code`를 게이트 1에 추가한다. 함께 기여하는 사람이 생기면 게이트 2를 모든 PR로 넓힌다.
