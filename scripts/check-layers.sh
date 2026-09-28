#!/usr/bin/env bash
# 계층(의존 방향) 규칙 검사. make check 에서 실행된다.
#
# 규칙 (docs/adr/0003-pure-core-shared-session.md):
#   1. discordgo 는 루트(main) 와 features/ 만 import 한다.
#   2. 순수 패키지(config, interactions, voice, session, yacht)는
#      features/ 와 store/ 를 import 하지 않는다.
set -euo pipefail

mod="$(go list -m)"
discord="github.com/bwmarrin/discordgo"
fail=0

while read -r pkg imports; do
	rel="${pkg#"$mod"}"
	rel="${rel#/}"
	top="${rel%%/*}"

	# 규칙 1: discordgo 허용 패키지
	case "$top" in
	"" | features) ;;
	*)
		if [[ " $imports " == *" $discord "* ]]; then
			echo "✘ $pkg 가 discordgo 를 import 함 (main, features/ 만 허용)"
			fail=1
		fi
		;;
	esac

	# 규칙 2: 순수 패키지는 features/, store/ 를 모른다
	case "$top" in
	config | interactions | voice | session | yacht)
		for imp in $imports; do
			case "$imp" in
			"$mod/features"* | "$mod/store"*)
				echo "✘ $pkg 가 $imp 를 import 함 (순수 패키지는 features/, store/ 금지)"
				fail=1
				;;
			esac
		done
		;;
	esac
done < <(go list -f '{{.ImportPath}} {{join .Imports " "}}' ./...)

if [[ $fail -ne 0 ]]; then
	echo "계층 규칙 위반. docs/design/six-member.md '구조와 의존 방향' 참고."
	exit 1
fi
echo "✔ 계층 규칙 통과"
