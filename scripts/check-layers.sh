#!/usr/bin/env bash
# 계층(의존 방향) 규칙 검사. make check 에서 실행된다.
#
# 규칙 (docs/adr/0003-pure-core-shared-session.md):
#   1. discordgo 는 루트(main) 와 features/ 만 import 한다. (_test.go 포함)
#   2. 루트와 features/ 가 아닌 모든 패키지는 features/ 를 import 하지 않는다.
#   3. store/ 를 import 할 수 있는 것은 루트와 features/ 뿐이다.
# 허용 목록이 아니라 "예외만 적는" 방식이라, 새 패키지가 생겨도 자동으로 검사된다.
set -euo pipefail

mod="$(go list -m)"
discord="github.com/bwmarrin/discordgo"
fail=0

while read -r pkg imports; do
	rel="${pkg#"$mod"}"
	rel="${rel#/}"
	top="${rel%%/*}"

	case "$top" in
	"" | features) continue ;; # 연결 계층: 무엇이든 import 가능
	esac

	for imp in $imports; do
		case "$imp" in
		"$discord" | "$discord"/*)
			echo "✘ $pkg 가 discordgo 를 import 함 (main, features/ 만 허용)"
			fail=1
			;;
		"$mod/features" | "$mod/features"/*)
			echo "✘ $pkg 가 $imp 를 import 함 (features/ 는 연결 계층이라 아래에서 참조 금지)"
			fail=1
			;;
		"$mod/store" | "$mod/store"/*)
			if [[ "$top" != store ]]; then
				echo "✘ $pkg 가 $imp 를 import 함 (store/ 는 main, features/ 만 사용)"
				fail=1
			fi
			;;
		esac
	done
	# 테스트 import(TestImports, XTestImports)까지 한 줄에 합쳐서 검사한다.
done < <(go list -f '{{.ImportPath}} {{join .Imports " "}} {{join .TestImports " "}} {{join .XTestImports " "}}' ./...)

if [[ $fail -ne 0 ]]; then
	echo "계층 규칙 위반. AGENTS.md '계층/의존 방향 규칙' 참고."
	exit 1
fi
echo "✔ 계층 규칙 통과"
