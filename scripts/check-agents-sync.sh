#!/usr/bin/env bash
# Claude Code(.claude/agents/*.md)와 Codex(.codex/agents/*.toml) 서브에이전트
# 지시문 본문이 같은지 검사한다. 두 에이전트가 서로 다른 규칙을 읽지 않게 하기 위함.
# 비교 대상: md 는 frontmatter(---) 다음 본문, toml 은 developer_instructions 의 """ 사이.
# 빈 줄과 앞뒤 공백은 무시한다.
set -euo pipefail

norm() { sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' | grep -v '^$' || true; }
fail=0

for md in .claude/agents/*.md; do
	name="$(basename "$md" .md)"
	toml=".codex/agents/$name.toml"
	if [[ ! -f "$toml" ]]; then
		echo "✘ $md 에 대응하는 $toml 이 없음"
		fail=1
		continue
	fi
	claude_body="$(awk 'c>=2{print} /^---$/{c++}' "$md" | norm)"
	codex_body="$(awk '/developer_instructions = """/{f=1; sub(/.*"""/,""); if($0!="")print; next} f&&/"""/{sub(/""".*/,""); if($0!="")print; f=0} f{print}' "$toml" | norm)"
	if [[ "$claude_body" != "$codex_body" ]]; then
		echo "✘ $name: .claude 와 .codex 지시문이 다름"
		diff <(echo "$claude_body") <(echo "$codex_body") || true
		fail=1
	fi
done

for toml in .codex/agents/*.toml; do
	name="$(basename "$toml" .toml)"
	[[ -f ".claude/agents/$name.md" ]] || { echo "✘ $toml 에 대응하는 .claude/agents/$name.md 가 없음"; fail=1; }
done

[[ $fail -eq 0 ]] && echo "✔ 에이전트 지시문 동기화 통과" || exit 1
