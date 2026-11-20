#!/usr/bin/env bash
# 运行持续集成第 SHARD 路（共 SHARDS 路，从 0 起）的 Go 测试，由 make test-shard 调用；本地用 make test，不分路。
# bootstrap 包串行运行数百个组合测试，独自逼近 go test 的整包期限（M6/P2，负责人 2026-11-08 批准分路）。
# 它的每个顶层测试按名字的哈希固定在一路（负责人 2026-11-20 批准）：桶号是名字的 POSIX cksum（CRC32）对 100 取余，
# 只取决于名字本身，加减测试不让别的测试换路。第 0 路分到桶 [0, p0)，其余的桶平分给第 1..SHARDS-1 路；只有一路时全归第 0 路。
# server 的其他包和 server/tools 模块只在第 0 路运行，所以第 0 路少分 bootstrap 的测试。
# 某个测试在哪个桶：printf %s TestName | cksum，第一个数对 100 取余。
set -euo pipefail

usage="usage: bash tools/go-test-shard.sh <shard> <shards> <timeout>"
[ $# -eq 3 ] || { echo "$usage" >&2; exit 2; }
shard=$1 shards=$2 timeout=$3
[[ $shard =~ ^[0-9]+$ && $shards =~ ^[0-9]+$ ]] && ((shard < shards)) || { echo "$usage" >&2; exit 2; }

# 第 0 路的桶数 p0：第 0 路另跑其他包、lint 等步骤（持续集成上约 300 秒）。按持续集成上逐个测试的时间（并行子测试的时间
# 不算进父测试：TestPermissionMatrix 在桶 46、约 85 秒，报告里只有 2 秒）折算，47 让两路离各自期限（作业 25 分钟、
# go test 20 分钟）最近的那一路余量最大（M7 分路段的阶段记录）
p0=47
if ((shard == 0)); then
	lo=0 hi=$((shards == 1 ? 100 : p0))
else
	lo=$((p0 + (shard - 1) * (100 - p0) / (shards - 1))) hi=$((p0 + shard * (100 - p0) / (shards - 1)))
fi

cd "$(dirname "$0")/../server"
pkg=./internal/bootstrap
# go test -list 先列名字，最后一行是包的结果（含空白）；测试名是 Go 标识符（可含非 ASCII 字母），不含空白和正则的元字符
list() { go test -list "$1" "$pkg" | grep -E '^[^[:space:]]+$'; }
all=$(list .)
mine=$(while IFS= read -r name; do
	read -r crc _ < <(printf %s "$name" | cksum)
	[[ $crc =~ ^[0-9]+$ ]] || { echo "cksum 没有给出 $name 的校验值" >&2; exit 1; }
	if ((crc % 100 >= lo && crc % 100 < hi)); then echo "$name"; fi
done <<<"$all" | LC_ALL=C sort)
[ -n "$mine" ] || { echo "第 $shard 路没有分到 bootstrap 包的测试" >&2; exit 1; }
run="^($(paste -sd '|' - <<<"$mine"))\$"
# 模式选中的必须恰好是分到的这些测试，免得一个也没跑却通过
[ "$(list "$run" | LC_ALL=C sort)" = "$mine" ] || { echo "第 $shard 路的 -run 模式没有恰好选中分到的测试" >&2; exit 1; }
# 名单的 cksum 用来对照本地与持续集成日志里的分法是否相同
echo "第 $shard 路（共 $shards 路，桶 ${lo}–$((hi - 1))）：bootstrap 包 $(wc -l <<<"$all" | tr -d ' ') 个测试中的 $(wc -l <<<"$mine" | tr -d ' ') 个，名单 cksum $(cksum <<<"$mine")"

status=0
if ((shard == 0)); then
	go test -count=1 -timeout="$timeout" $(go list ./... | grep -vx "$(go list -m)/internal/bootstrap") || status=1
	go -C tools test -count=1 -timeout="$timeout" ./... || status=1
fi
go test -count=1 -timeout="$timeout" -run "$run" "$pkg" || status=1
exit "$status"
