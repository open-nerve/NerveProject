#!/usr/bin/env bash
# 运行持续集成第 SHARD 路（共 SHARDS 路，从 0 起）的 Go 测试，由 make test-shard 调用；本地用 make test，不分路。
# bootstrap 包串行运行数百个组合测试，独自逼近 go test 的整包期限（M6/P2，负责人 2026-11-08 批准分路）：
# 它的测试按名字排序后轮流分配，第 k 个（从 0 起）归第 k % SHARDS 路；server 的其他包和 server/tools 模块只在第 0 路运行。
set -euo pipefail

usage="usage: bash tools/go-test-shard.sh <shard> <shards> <timeout>"
[ $# -eq 3 ] || { echo "$usage" >&2; exit 2; }
shard=$1 shards=$2 timeout=$3
[[ $shard =~ ^[0-9]+$ && $shards =~ ^[0-9]+$ ]] && ((shard < shards)) || { echo "$usage" >&2; exit 2; }

cd "$(dirname "$0")/../server"
pkg=./internal/bootstrap
# go test -list 先列名字，最后一行是包的结果；测试名是标识符，不含正则的元字符
list() { go test -list "$1" "$pkg" | grep -E '^[A-Za-z_][A-Za-z0-9_]*$'; }
all=$(list .)
mine=$(LC_ALL=C sort <<<"$all" | awk -v n="$shards" -v k="$shard" '(NR - 1) % n == k')
[ -n "$mine" ] || { echo "第 $shard 路没有分到 bootstrap 包的测试" >&2; exit 1; }
run="^($(paste -sd '|' - <<<"$mine"))\$"
# 模式选中的必须恰好是分到的这些测试，免得一个也没跑却通过
[ "$(list "$run" | LC_ALL=C sort)" = "$mine" ] || { echo "第 $shard 路的 -run 模式没有恰好选中分到的测试" >&2; exit 1; }
echo "第 $shard 路（共 $shards 路）：bootstrap 包 $(wc -l <<<"$all" | tr -d ' ') 个测试中的 $(wc -l <<<"$mine" | tr -d ' ') 个"

status=0
if ((shard == 0)); then
	go test -count=1 -timeout="$timeout" $(go list ./... | grep -vx "$(go list -m)/internal/bootstrap") || status=1
	go -C tools test -count=1 -timeout="$timeout" ./... || status=1
fi
go test -count=1 -timeout="$timeout" -run "$run" "$pkg" || status=1
exit "$status"
