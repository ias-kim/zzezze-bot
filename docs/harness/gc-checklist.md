# GC 체크리스트

주기: 월 1회 (직접 실행). 기계가 이미 검사하는 항목(tidy, 계층, 에이전트 동기화, 대사 키)은 `make check`가 맡으므로 여기에는 **수동 점검만** 둔다.

- [ ] `docs/sot.md`에서 "수동" 표시된 행이 원천과 일치하는지
  - 설계 문서 "채널별 기능" 표 ↔ `config/channels.go`
  - AGENTS.md 명령어 절 ↔ `Makefile`
- [ ] README가 값을 복사하지 않고 링크만 걸고 있는지
- [ ] `docs/exec-plans/active/`에 끝난 작업이 남아 있으면 `completed/`로 이동
- [ ] `docs/exec-plans/tech-debt-tracker.md` 최신화
- [ ] AGENTS.md의 "예정" 패키지 중 이미 생긴 것이 있으면 "있는 것"으로 옮기기
- [ ] `docs/solutions/` 교훈 중 이제 코드나 테스트로 강제되는 것은 정리
- [ ] ADR 없이 조용히 뒤집힌 설계 결정이 없는지
- [ ] 채널/기능 이름이 실제 디스코드 채널과 어긋나지 않았는지
