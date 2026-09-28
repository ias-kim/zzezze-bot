# GC 체크리스트

주기: 월 1회 (직접 실행. 정기 스케줄 자동 실행은 아직 걸지 않음)

- [ ] 죽은 코드 / 미사용 의존성 (`go vet ./...`, `go mod tidy` 후 diff 확인)
- [ ] `docs/sot.md`와 어긋난 문서(README 등)가 없는지
- [ ] `docs/exec-plans/active/`에 이미 끝난 작업이 남아 있으면 `completed/`로 이동
- [ ] `docs/exec-plans/tech-debt-tracker.md` 최신화
- [ ] 채널/기능 이름(`config/channels.go`)과 실제 디스코드 채널 이름이 드리프트하지 않았는지
- [ ] ADR 없이 조용히 뒤집힌 설계 결정이 없는지
