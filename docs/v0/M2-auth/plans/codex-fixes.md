# M2/Codex 修复 codex-fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 按控制者的分诊修好 Codex 对全部 M2 的对抗性评审的每一条（spec 2.1），每一处都在根因上修、有能失败的回归：请求体只有一种读法（Critical 1），令牌管理器在每个 `await` 之后重读记录（Important 1），服务角色的权限只有一个出处并由测试守住（Important 2），会话的主题只在一处设置（Minor 1），A8–A10 共用业务断言（Minor 2），文档漂移（Minor 3），一个 store 的修改一个接一个发出（报告第 6 节），完整的恢复说明（第 7 节），`AllowAll` 拒绝重复的检查。M2 在这一阶段合并时再次完成。

**Architecture:** Task 0 随 spec 和本计划提交（M2 设计第 15 节的一行）。之后 11 个 Task，前 10 个各一个提交：`bodyshape` 的扫描器和字段码（T1）→ 整程序测试和 Codex 的请求（T2）→ 令牌管理器（T3）→ 服务角色的权限（T4）→ `AllowAll`（T5）→ 共用断言（T6）→ 修改的队列（T7）→ 会话的主题（T8）→ M2 设计（T9）→ 上级文档、README 的恢复、交接、收尾的数字（T10）→ 全部门禁（T11，不提交）。每一处改动都写成下面的块，由 `$M2TMP/codexfix/planapply.mjs` 精确地应用；生成的代码不写成块，由 `make gen` 生成（T1）。状态翻转在 codex-fixes review 的提交里，由控制者做，替换的全文在最后一节。

**Tech Stack:** Go 1.27.1、golangci-lint 2.13.2、River 0.47.0、PostgreSQL 18.6（testcontainers）、Node 24、pnpm 11.10.0、turbo 2.10.11、vitest、Playwright 1.63.0；不加依赖。

**Spec:** `docs/v0/M2-auth/specs/codex-fixes.md`（上级：M2 设计 3.10、3.11、6.1、7.1、7.5、7.7、8.5、8.7、11.2、第 15 节、17.3；总体设计 3.5、4.3、7.7；输入：`docs/v0/M2-auth/reviews/M2-codex-adversarial-review.md`）

**原型：** `$M2TMP/codexfix/proto`，本分支在 `37b7e9c` 的克隆。下面每个块都由原型的文件生成（`$M2TMP/codexfix/mkblocks.py`），块里的代码就是原型上跑过的代码；每个 Task 的检查、先红后绿（spec 附录 A）、变异（附录 B）和全部门禁（附录 C）都在原型上做过。本计划的块依次应用到 `37b7e9c` 的另一份干净副本上，得到的树与原型逐字节相同（文末"块的核对"）。

---

## Global Constraints

### 命令与环境

- **所有命令在 worktree 的根目录执行**，不要 `cd`，也不要把 `cd` 和任何命令写在一起。在包里执行用 `pnpm -C <目录>`，Go 用 `go -C server`。开始之前执行一次 `pnpm install --frozen-lockfile`；不做全局安装，不运行 `corepack enable`。
- **`$M2TMP`** 是
  `/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-m2`。
  本阶段的一次性脚本都在 `$M2TMP/codexfix/`，不进仓库；不要用裸的 `/tmp`。
- **一个 Bash 调用只跑一个 git 命令**：不用 `;`、`&&`、管道串联 git，不用 `git -C`、`stash`、`clean`、`reset --hard`；不碰 `plane/`、`refer/`。
- **容器**：`make test`、`make e2e` 和 `runtime_role_test.go` 用自己的 testcontainers；建角色（`CREATE ROLE`）只在它们里面发生。开发库 `nerve-dev-db-1` 不停、不重建，不执行 `make dev-db-down`、`make dev-db-reset`；不碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`）。2026-09-26、27 建的 14 个 `Created` 状态的旧 testcontainers 不是本阶段建的，不动它们。跑完之后等一分钟，`docker ps -a --filter label=org.testcontainers=true --format '{{.Names}} {{.Status}} {{.CreatedAt}}'` 中没有这次运行建的容器。
- **文件内容**：块由 `planapply.mjs` 写入；手工补写时用编辑工具，不用带反引号的 heredoc。写完 `grep -rn -E '&lt;|&gt;|&amp;'` 本 Task 改过的文件，应当没有输出（Edit、Write 工具曾把 `<` 写成 `&lt;`）。
- **不留后台进程**：`make e2e`、Playwright 自己停掉它起的 nerve 和容器。

### 块的写法和应用

- `` ````file <路径> ``：新文件的全文（块的每一行，外加结尾的换行）。
- `` ````whole <路径> ``：已有文件改写之后的全文（同样的写法）。只用于改写得多、逐处替换反而难读的三个文件：两个 store 的测试和 `store-wrapper.tsx`。
- `` ````old <路径> `` 紧跟 `` ````new <路径> ``：把恰好出现一次的旧文本换成新文本。
- 每个 Task 的块用一条命令应用：
  `node $M2TMP/codexfix/planapply.mjs docs/v0/M2-auth/plans/codex-fixes.md apply . <Task 编号>`
  它先核对本 Task 的每一块（旧文本在当前文件里恰好一处，新文件还不存在，改写的文件存在），全部成立才写，否则什么都不写并报出是哪一块。
- 全部块的核对：`node $M2TMP/codexfix/planapply.mjs docs/v0/M2-auth/plans/codex-fixes.md check $M2TMP/codexfix/base`（`base/` 是 `git archive 37b7e9c` 解开的树）。输出见文末"块的核对"。

### 本阶段的规则

1. **分诊有约束力**（spec 2.1）。架构层面的事（跨模块的规则、超出"重复键和非 UTF-8 是 400"的前后端契约、v0 或 M0 的架构决定、负责人的裁定、把 M3 的工作拉进 M2）停下来报告，不自己决定。
2. **根因上修，不打补丁**；变成死代码的删掉（两个 store 的 `updatesSent`、`updateWritten`；切换账户弹窗的 `setTheme`；`startSession` 写的主题）。
3. **测试咬得住**：顺序和竞争的测试用门、假时钟、可控的锁强制顺序，从不 `sleep`；每个等待都有期限；store 的依赖图在模块顶层导入，不在测试的超时之内冷导入。每个新测试都由变异证明（spec 附录 B），本计划的块就是原型上被变异验证过的代码。
4. **不手改生成的代码**：`api/common.yaml` 改了之后 `make gen`。
5. **执行方式沿用 P5**：实现者不指定模型；T1–T8 按常规评审，T9、T10 由评审者对照 spec 核对文字。

### 每个 Task 的固定节奏

1. 应用本 Task 的块（上面的命令），读 `git diff`，确认每一处都对应本 Task 的一条说明。
2. `grep -rn -E '&lt;|&gt;|&amp;'` 本 Task 改过的文件 → 没有输出。
3. 本 Task 写明的检查，预期是原型上的结果。
4. 提交：一个 Task 一个提交，提交信息用英文，最后一行是
   `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`。提交前 `git status --short` 只有本 Task 的文件（T1 另有三个生成的文件）。

### 风险点（每个 Task 报告必答）

1. **只改本 Task 的文件**：`git diff --stat HEAD~1` 与本 Task 在文件表里的行一致。
2. **测试咬得住**：本 Task 的新测试在 spec 附录 B 里有对应的变异；评审可以在原型上用 `python3 $M2TMP/codexfix/mutants/run.py <变异>` 重跑。
3. **没有冷导入、没有 `sleep`**：新的 vitest 测试在顶层导入，e2e 的等待都有 `timeout`。

### 一次性脚本

都在 `$M2TMP/codexfix/`。

| 脚本 | 用途 |
|---|---|
| `planapply.mjs <plan> check <基树>`、`… apply <根> <Task>`、`… list` | 核对全部块；应用一个 Task 的块；列出块 |
| `mkblocks.py` | 由 `base/` 和原型生成本计划的块（`planparts/T<n>.md`），并核对依次应用得到原型 |
| `swap.sh <标签> '<命令>' -- <文件>=<替换>…` | 在原型里临时换掉几个文件跑一条命令再换回（先红、变异），输出在 `logs/<标签>.log` |
| `mutants/make.py`、`mutants/run.py <变异>…` | 写出变异、跑变异（spec 附录 B） |
| `gates.sh <根> <标签> [门禁…]` | 七个门禁，摘要在 `logs/<标签>.summary`；原型没有提交，`gen-check` 换成 `make gen` 之后逐字节比较（spec D8） |
| `treediff.mjs <树 a> <树 b>` | 两棵树逐字节比较，不含 `node_modules`、`.git` 和构建产物 |
| `probes/a9-codex-experiment.spec.ts` | Codex 的页面实验原样（spec 附录 A.6） |

### 文件结构

| 文件 | 改动 | Task |
|---|---|---|
| `docs/v0/M2-auth/M2-design.md` | 第 15 节"Codex 修复"一行（随 spec 提交） | 0 |
| `server/internal/platform/httpserver/bodyshape/ambiguity.go` | 新文件：一种读法的扫描器 | 1 |
| `server/internal/platform/httpserver/bodyshape/ambiguity_test.go` | 新文件：24 个子测试和 `duplicate` 的消息 | 1 |
| `server/internal/platform/httpserver/bodyshape/bodyshape.go` | `Check` 先扫描；`codeDuplicate` 和它的消息；包和函数的注释 | 1 |
| `server/internal/shared/error.go` | `FieldDuplicate`，进 `FieldCodes()` | 1 |
| `api/common.yaml` | `FieldError.code` 的枚举加 `duplicate` | 1 |
| `api/dist/openapi.yaml`、`server/internal/platform/httpserver/apigen/components.gen.go`、`web/packages/api-client/src/schema.gen.ts` | `make gen` 生成 | 1 |
| `web/packages/i18n/src/locales/{en,zh-CN}/auth.json`、`web/apps/web/helpers/authentication.helper.ts` | `duplicate` 的文案和字段码表 | 1 |
| `server/internal/platform/httpserver/apitest/operations.go`、`operations_test.go` | 整程序用例第 7–9 项（两处重复、非 UTF-8） | 2 |
| `server/internal/bootstrap/contract_test.go` | 第四个整程序测试的注释 | 2 |
| `e2e/stories/identity/a10-onboarding-profile.spec.ts` | 新故事：Codex 的两个请求 | 2 |
| `web/apps/web/core/lib/auth/token-manager.ts`、`token-manager.session-change.test.ts` | 退出和第一次续期在 `await` 之后重读；两个测试 | 3 |
| `deploy/runtime-grants.sql` | 新文件：服务角色的权限 | 4 |
| `server/internal/bootstrap/runtime_role_test.go` | 新文件：两个测试 | 4 |
| `README.md` | "部署"的"迁移"一条；River 索引的删除 | 4 |
| `server/internal/platform/ratelimit/ratelimit.go`、`ratelimit_test.go` | 重复的检查 panic；一个测试 | 5 |
| `e2e/fixtures/assert/identity.ts` | `expectAccountChanged`、`expectPreferences`、`expectProfileStepTaken`、`OnboardingSteps` | 6 |
| `e2e/stories/identity/a8-update-me.spec.ts` | 调用 `expectAccountChanged` | 6 |
| `e2e/stories/identity/a9-preferences.spec.ts` | 调用 `expectPreferences`，删 `profileOf` | 6 |
| `e2e/stories/identity/a10-onboarding-profile.spec.ts` | 调用 `expectProfileStepTaken` | 6 |
| `web/apps/web/core/lib/one-at-a-time.ts` | 新文件：队列 | 7 |
| `web/apps/web/core/store/user/profile.store.ts`、`index.ts` | 修改经队列发出；删编号 | 7 |
| `web/apps/web/core/store/user/profile.store.test.ts`、`index.test.ts` | 改写：6 个测试 | 7 |
| `e2e/stories/identity/a9-preferences.spec.ts` | 新故事：Codex 的页面实验 | 7 |
| `web/apps/web/core/lib/wrappers/store-wrapper.tsx` | 主题按会话设置 | 8 |
| `web/apps/web/core/lib/store-context.tsx`、`store-context.test.ts` | 删 `startSession` 写的主题 | 8 |
| `web/apps/web/core/components/onboarding/switch-account-modal.tsx` | 删 `setTheme` | 8 |
| `e2e/stories/identity/a5-refresh-reuse.spec.ts`、`a6-sign-out.spec.ts`、`a12-deactivate.spec.ts` | 回到默认主题的断言 | 8 |
| `docs/v0/M2-auth/M2-design.md` | 3.10、3.11、6.1、7.1、7.5、7.7、8.5、8.7、11.2、17.3 | 9 |
| `docs/v0/v0-design.md` | 3.5、4.3、7.7 | 10 |
| `docs/v0/plane-diff.md` | 第三节"请求体" | 10 |
| `README.md` | "令牌泄露后的恢复" | 10 |
| `docs/v0/M4-issue-core/handoffs/M2-closeout.md`、`docs/v0/M8-open-release/handoffs/M2-closeout.md` | 第 6 节、第 3 节各一条和关闭条件 | 10 |
| `docs/v0/M2-auth/specs/closeout.md`、`docs/v0/M2-auth/reviews/closeout-review.md` | 13 个操作、约 48 行 | 10 |
| — | 全部门禁 | 11 |

codex-fixes review 的提交（控制者）另改 `docs/v0/M2-auth/M2-design.md` 的第 14、15 节，见最后一节。总体设计 9.4 不变（仍是"已完成"）。

---

## Tasks

### Task 0: M2 设计第 15 节的"Codex 修复"一行（已随 spec 和本计划提交）

不由实现者做：架构师提交 spec 和本计划时，同一个提交加上这一行（进行中，带两个链接）。写在这里，是为了让块的核对从 `37b7e9c` 出发、按顺序走到 review 提交的替换。

````old docs/v0/M2-auth/M2-design.md
| 收尾 | closeout | 已完成 | [spec](specs/closeout.md) | [plan](plans/closeout.md) | [review](reviews/closeout-review.md) |
````

````new docs/v0/M2-auth/M2-design.md
| 收尾 | closeout | 已完成 | [spec](specs/closeout.md) | [plan](plans/closeout.md) | [review](reviews/closeout-review.md) |
| Codex 修复 | codex-fixes | 进行中 | [spec](specs/codex-fixes.md) | [plan](plans/codex-fixes.md) | — |
````

### Task 1: 请求体只有一种读法：`bodyshape` 的扫描器和字段码 `duplicate`

spec 2.2。`ambiguity.go` 在 `Check` 按表检查之前扫描 `json.Valid` 已确认的请求体：对象里按解码后的名字计数，第二次出现时报 `duplicate`（每个名字一次）；字符串用 `utf8.Valid` 和代理项配对判断，不合法报 `invalid_format`。有这两种问题的请求体只得到它们（spec D3）。新码 `duplicate` 进入 3.11 列出和核对字段码的每一处：契约的枚举、`shared.FieldCodes()`、前端的字段码表、中英文案。

- [ ] **Step 1：应用块**（`… apply . 1`）：

````file server/internal/platform/httpserver/bodyshape/ambiguity.go
package bodyshape

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"
)

// A document that json.Valid accepts can still be read two ways (M2 design
// 3.11). Check reads its objects as maps, where the last of two members with
// one name wins, while the generated handler decodes them into Go structs,
// which merge the two: a member Check never saw would reach the domain. And a
// string that is not valid Unicode decodes to U+FFFD, a text the client never
// sent. So Check first requires what I-JSON (RFC 7493) requires of both: each
// member name at most once in its object, compared as decoded, so that "a"
// and its \u escape are one name; each string, name or value, valid Unicode:
// UTF-8 bytes, and a \u escape of a surrogate only as one half of a pair.
// Then every decoder reads the one document the client wrote.

// ambiguities returns what lets value, one valid JSON value, be read two
// ways: a member name repeated in its object (duplicate, once per name) and a
// string that is not valid Unicode (invalid_format), each at its JSON path.
func ambiguities(value []byte) []FieldError {
	s := scanner{data: value}
	s.value("")
	return s.errs
}

// scanner reads a valid JSON value byte by byte; it does not check the
// syntax again.
type scanner struct {
	data []byte
	pos  int
	errs []FieldError
}

func (s *scanner) value(path string) {
	s.space()
	switch s.data[s.pos] {
	case '{':
		s.object(path)
	case '[':
		s.array(path)
	case '"':
		if _, ok := s.str(); !ok {
			s.errs = append(s.errs, FieldError{path, codeInvalidFormat})
		}
	default: // a number, true, false or null
		for s.pos < len(s.data) && !isEnd(s.data[s.pos]) {
			s.pos++
		}
	}
}

func (s *scanner) object(path string) {
	seen := map[string]int{}
	s.pos++ // {
	if s.space(); s.data[s.pos] == '}' {
		s.pos++
		return
	}
	for {
		s.space()
		name, ok := s.str()
		at := join(path, name)
		seen[name]++
		switch {
		case !ok:
			s.errs = append(s.errs, FieldError{at, codeInvalidFormat})
		case seen[name] == 2:
			s.errs = append(s.errs, FieldError{at, codeDuplicate})
		}
		s.space()
		s.pos++ // :
		s.value(at)
		s.space()
		s.pos++ // , or }
		if s.data[s.pos-1] == '}' {
			return
		}
	}
}

func (s *scanner) array(path string) {
	s.pos++ // [
	if s.space(); s.data[s.pos] == ']' {
		s.pos++
		return
	}
	for i := 0; ; i++ {
		s.value(path + "[" + strconv.Itoa(i) + "]")
		s.space()
		s.pos++ // , or ]
		if s.data[s.pos-1] == ']' {
			return
		}
	}
}

// str reads the string at s.pos: decoded, as encoding/json decodes it, and
// whether it is valid Unicode.
func (s *scanner) str() (string, bool) {
	start := s.pos
	for s.pos++; s.data[s.pos] != '"'; s.pos++ {
		if s.data[s.pos] == '\\' {
			s.pos++
		}
	}
	s.pos++
	raw := s.data[start:s.pos]
	var decoded string
	_ = json.Unmarshal(raw, &decoded) // the whole document is valid JSON
	return decoded, utf8.Valid(raw) && pairedSurrogates(raw)
}

// pairedSurrogates reports whether every \u escape of a surrogate in raw, a
// valid JSON string with its quotes, is one half of a pair: a high one right
// before a low one. A lone half decodes to U+FFFD.
func pairedSurrogates(raw []byte) bool {
	high := false // the escape just read is a high surrogate
	for i := 0; i < len(raw); {
		unit := -1
		switch {
		case raw[i] != '\\':
			i++
		case raw[i+1] != 'u':
			i += 2
		default:
			n, _ := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
			unit = int(n)
			i += 6
		}
		if low := unit >= 0xDC00 && unit <= 0xDFFF; low != high {
			return false // a high half without its low one, or a low half without its high one
		}
		high = unit >= 0xD800 && unit <= 0xDBFF
	}
	return !high
}

func (s *scanner) space() {
	for s.pos < len(s.data) && isSpace(s.data[s.pos]) {
		s.pos++
	}
}

func isSpace(c byte) bool { return strings.IndexByte(jsonSpace, c) >= 0 }

// isEnd reports whether c ends a number or a literal.
func isEnd(c byte) bool { return c == ',' || c == ']' || c == '}' || isSpace(c) }
````

````file server/internal/platform/httpserver/bodyshape/ambiguity_test.go
package bodyshape

import (
	"errors"
	"slices"
	"testing"
	"unicode/utf8"
)

// A body that can be read two ways is refused before its structure is
// checked: a member name twice in one object, at any depth, compared as
// decoded; a string, name or value, that is not valid Unicode (M2 design
// 3.11). The JSON escapes are written \\u in these Go strings, so the body
// holds the escape itself.
func TestCheckReadsABodyOneWayOnly(t *testing.T) {
	const valid = `"name":"a","nested":{"a":"y"}`
	replacement := string(utf8.RuneError)
	tests := []struct {
		name string
		body string
		want []FieldError
	}{
		{"a top-level name twice", `{` + valid + `,"name":"b"}`, []FieldError{{"name", "duplicate"}}},
		// Read as a map, the second nested wins and the first one's x goes unseen; decoded into a struct,
		// the two merge and x is set. Either way the body is refused, whatever the objects hold.
		{"an object twice, the first with what the contract refuses", `{"name":"a","nested":{"a":"y","x":1},"nested":{"a":"y"}}`,
			[]FieldError{{"nested", "duplicate"}}},
		{"a nested name twice", `{"name":"a","nested":{"a":"y","a":"z"}}`, []FieldError{{"nested.a", "duplicate"}}},
		{"a name three times, said once", `{` + valid + `,"name":"b","name":"c"}`, []FieldError{{"name", "duplicate"}}},
		{"an escaped name is the name it decodes to", "{" + valid + ",\"n\\u0061me\":\"b\"}", []FieldError{{"name", "duplicate"}}},
		{"two escapes of one name", "{\"\\u006eame\":\"a\",\"n\\u0061me\":\"b\",\"nested\":{\"a\":\"y\"}}",
			[]FieldError{{"name", "duplicate"}}},
		{"a name twice in an item of an array", `{` + valid + `,"tags":[{"name":"t"},{"name":"t","name":"u"}]}`,
			[]FieldError{{"tags[1].name", "duplicate"}}},
		{"a key twice in an open map", `{` + valid + `,"labels":{"k":"v","k":"w"}}`, []FieldError{{"labels.k", "duplicate"}}},
		{"a name twice where the structure check never looks", `{` + valid + `,"extra":{"q":1,"q":2}}`,
			[]FieldError{{"extra.q", "duplicate"}}},
		{"names that differ in case are two names: the undeclared one is refused", `{` + valid + `,"Name":"b"}`,
			[]FieldError{{"Name", "not_allowed"}}},
		{"the same name in two objects is no duplicate", `{"name":"a","nested":{"a":"y"},"maybe":{"a":"z"}}`, nil},
		{"bytes that are not UTF-8 in a value", "{\"name\":\"A\xffB\",\"nested\":{\"a\":\"y\"}}",
			[]FieldError{{"name", "invalid_format"}}},
		{"bytes that are not UTF-8 in a nested value", "{\"name\":\"a\",\"nested\":{\"a\":\"\xc3\"}}",
			[]FieldError{{"nested.a", "invalid_format"}}},
		{"bytes that are not UTF-8 in an item", "{" + valid + ",\"tags\":[{\"name\":\"\xed\xa0\x80\"}]}",
			[]FieldError{{"tags[0].name", "invalid_format"}}},
		{"bytes that are not UTF-8 in a name", "{" + valid + ",\"n\xffx\":1}", []FieldError{{"n" + replacement + "x", "invalid_format"}}},
		{"a lone high surrogate", "{\"name\":\"A\\ud800B\",\"nested\":{\"a\":\"y\"}}", []FieldError{{"name", "invalid_format"}}},
		{"a high surrogate at the end", "{\"name\":\"A\\udbff\",\"nested\":{\"a\":\"y\"}}", []FieldError{{"name", "invalid_format"}}},
		{"a lone low surrogate", "{\"name\":\"\\udc00\",\"nested\":{\"a\":\"y\"}}", []FieldError{{"name", "invalid_format"}}},
		{"two high surrogates", "{\"name\":\"\\ud83d\\ud83d\\ude00\",\"nested\":{\"a\":\"y\"}}", []FieldError{{"name", "invalid_format"}}},
		{"a surrogate pair", "{\"name\":\"\\ud83d\\ude00\",\"nested\":{\"a\":\"y\"}}", nil},
		{"an escaped backslash before u", "{\"name\":\"\\\\ud800\",\"nested\":{\"a\":\"y\"}}", nil},
		{"U+FFFD itself", "{\"name\":\"A" + replacement + "B\\ufffd\",\"nested\":{\"a\":\"y\"}}", nil},
		// The structure of a body with two readings would be checked on one of them: only its ambiguities come back.
		{"every ambiguity, and nothing of the structure", "{\"nested\":{\"a\":\"\xff\"},\"nested\":{},\"zzz\":1}",
			[]FieldError{{"nested", "duplicate"}, {"nested.a", "invalid_format"}}},
		{"a string body that is not UTF-8", "\"\xff\"", []FieldError{{"", "invalid_format"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := things().Check(pattern, []byte(tt.body))

			var shape *Error
			if err != nil && !errors.As(err, &shape) {
				t.Fatalf("Check(%q) = %v, want nil or a *Error", tt.body, err)
			}
			var got []FieldError
			if shape != nil {
				got = shape.Fields
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Check(%q) = %v, want %v", tt.body, got, tt.want)
			}
		})
	}
}

func TestADuplicateSaysWhy(t *testing.T) {
	if got := (FieldError{"name", "duplicate"}).Error(); got != "appears more than once in its object" {
		t.Errorf("Error() = %q", got)
	}
}
````

````old server/internal/platform/httpserver/bodyshape/bodyshape.go
// 3.11): JSON types, undeclared properties, null where the contract does not
// allow it, missing required properties, and the string formats that the
// generated code decodes into Go types. Every problem is collected in one
// pass. Values (lengths, enums, ranges, e-mail syntax) are the domain's.
````

````new server/internal/platform/httpserver/bodyshape/bodyshape.go
// 3.11): first that the body can be read one way only (no member name twice
// in an object, no string that is not valid Unicode), then JSON types,
// undeclared properties, null where the contract does not allow it, missing
// required properties, and the string formats that the generated code
// decodes into Go types. Every problem of a kind is collected in one pass.
// Values (lengths, enums, ranges, e-mail syntax) are the domain's.
````

````old server/internal/platform/httpserver/bodyshape/bodyshape.go
	codeNotAllowed    = "not_allowed"
````

````new server/internal/platform/httpserver/bodyshape/bodyshape.go
	codeNotAllowed    = "not_allowed"
	codeDuplicate     = "duplicate"
````

````old server/internal/platform/httpserver/bodyshape/bodyshape.go
		return "is not a property of this request"
````

````new server/internal/platform/httpserver/bodyshape/bodyshape.go
		return "is not a property of this request"
	case codeDuplicate:
		return "appears more than once in its object"
````

````old server/internal/platform/httpserver/bodyshape/bodyshape.go
// generated decoder. A body that is not one valid JSON document is ErrNotJSON;
// one that breaks the structure is an *Error with every problem, sorted by
// path.
````

````new server/internal/platform/httpserver/bodyshape/bodyshape.go
// generated decoder. A body that is not one valid JSON document is ErrNotJSON.
// One that can be read two ways (ambiguity.go) is an *Error with each
// ambiguity: its structure is checked once it has one reading. One that
// breaks the structure is an *Error with every problem. The problems are
// sorted by path.
````

````old server/internal/platform/httpserver/bodyshape/bodyshape.go
	var errs []FieldError
	t.walk(root, value, "", &errs)
````

````new server/internal/platform/httpserver/bodyshape/bodyshape.go
	errs := ambiguities(value)
	if len(errs) == 0 {
		t.walk(root, value, "", &errs)
	}
````

````old server/internal/shared/error.go
	FieldNotAllowed     = "not_allowed"
````

````new server/internal/shared/error.go
	FieldNotAllowed     = "not_allowed"
	FieldDuplicate      = "duplicate"
````

````old server/internal/shared/error.go
		FieldNotAllowed, FieldWeakPassword, FieldCommonPassword, FieldMustBeFuture, FieldContainsURL,
````

````new server/internal/shared/error.go
		FieldNotAllowed, FieldDuplicate, FieldWeakPassword, FieldCommonPassword, FieldMustBeFuture, FieldContainsURL,
````

````old api/common.yaml
            - not_allowed
````

````new api/common.yaml
            - not_allowed
            - duplicate
````

````old web/packages/i18n/src/locales/en/auth.json
        "not_allowed": "Not allowed",
````

````new web/packages/i18n/src/locales/en/auth.json
        "not_allowed": "Not allowed",
        "duplicate": "Given more than once",
````

````old web/packages/i18n/src/locales/zh-CN/auth.json
        "not_allowed": "不允许",
````

````new web/packages/i18n/src/locales/zh-CN/auth.json
        "not_allowed": "不允许",
        "duplicate": "重复出现",
````

````old web/apps/web/helpers/authentication.helper.ts
  not_allowed: "auth.errors.field.not_allowed",
````

````new web/apps/web/helpers/authentication.helper.ts
  not_allowed: "auth.errors.field.not_allowed",
  duplicate: "auth.errors.field.duplicate",
````

- [ ] **Step 2：生成**：`make gen`。`git status --short` 多出三个生成的文件，都只是枚举多了 `duplicate`：`api/dist/openapi.yaml`、`server/internal/platform/httpserver/apigen/components.gen.go`、`web/packages/api-client/src/schema.gen.ts`。

- [ ] **Step 3：检查**：
  - `go -C server test -count=1 ./internal/platform/httpserver/bodyshape/ -run 'TestCheckReadsABodyOneWayOnly|TestADuplicateSaysWhy' -v` → 24 个子测试和 `TestADuplicateSaysWhy` 都是 `PASS`；
  - `go -C server test -count=1 ./internal/platform/httpserver/... ./internal/shared/` → 全部 `ok`；`go -C server test -count=1 ./internal/bootstrap/ -run TestFieldCodesAreTheContractsEnum` → `ok`（契约的枚举和 `shared.FieldCodes()` 相等）；
  - `pnpm -C web/apps/web exec vitest run helpers/authentication.helper.test.ts` → 全部通过（字段码表覆盖枚举、文案键存在）；
  - `make lint-go` → 两段 `0 issues.`。

- [ ] **Step 4：提交**（`fix(M2/codex-fixes): bodyshape reads a body one way only; a duplicate name or invalid Unicode is 400`）

### Task 2: 每个带请求体的操作都拒绝重复和非 UTF-8；A10 发 Codex 的请求

spec 2.2 的回归。`apitest.BodyCases` 为每个带请求体的操作加三种用例：第一个字段出现两次、嵌套对象的第一个字段在其中出现两次（有嵌套对象时）、第一个字符串字段的值是字节 `0xff`。它们用 `raw` 把 `json.Marshal` 写不出的文字放进合法的请求体。第四个整程序测试据此发给真实组合的程序。A10 的新故事用 Playwright 的 `request` 原样发 Codex 的两个请求体（生成的客户端的对象装不下重复的键）。

- [ ] **Step 1：应用块**（`… apply . 2`）：

````old server/internal/platform/httpserver/apitest/operations.go
import (
````

````new server/internal/platform/httpserver/apitest/operations.go
import (
	"bytes"
````

````old server/internal/platform/httpserver/apitest/operations.go
const unknownField = "nerve_undeclared"

````

````new server/internal/platform/httpserver/apitest/operations.go
const unknownField = "nerve_undeclared"

// rawMark stands for a value that json.Marshal cannot write, until BodyCases
// puts the JSON text in its place.
const rawMark = "nerve_raw_value"

````

````old server/internal/platform/httpserver/apitest/operations.go
//  7. every kind of 1–4 and 6 that the schema has, each on a property of its
````

````new server/internal/platform/httpserver/apitest/operations.go
//  7. the first property twice: duplicate (a body read two ways);
//  8. the first property of the nested object twice, in it;
//  9. bytes that are not UTF-8 in the first string property: invalid_format;
//  10. every kind of 1–4 and 6 that the schema has, each on a property of its
````

````old server/internal/platform/httpserver/apitest/operations.go
//     has only the first kind.
````

````new server/internal/platform/httpserver/apitest/operations.go
//     has only the first kind. 7–9 are not in it: a body read two ways gets
//     only those answers, before its structure is checked.
````

````old server/internal/platform/httpserver/apitest/operations.go
	wrong := func(name string) string { return "not-a-" + s.Properties[name].Value.Format }
````

````new server/internal/platform/httpserver/apitest/operations.go
	wrong := func(name string) string { return "not-a-" + s.Properties[name].Value.Format }
	// raw is the valid body with name's value written as text, which json.Marshal would not write: a
	// second member of the same name, bytes that are not UTF-8.
	raw := func(name, text string) []byte {
		body := maps.Clone(valid)
		body[name] = rawMark
		out, _ := json.Marshal(body)
		return bytes.Replace(out, []byte(`"`+rawMark+`"`), []byte(text), 1)
	}
	// twice is the text of name's value v, then of a second member name: v.
	twice := func(name string, v any) string {
		value, _ := json.Marshal(v)
		return string(value) + `,"` + name + `":` + string(value)
	}
	// valueOf is name's value in values, else a valid one for its schema.
	valueOf := func(schema *openapi3.Schema, name string, values map[string]any) any {
		if v, ok := values[name]; ok {
			return v
		}
		return validValue(schema.Properties[name].Value)
	}
````

````old server/internal/platform/httpserver/apitest/operations.go
			Body: with(func(b map[string]any) { b[name] = wrong(name) }), Fields: []FieldProblem{{name, "invalid_format"}}})
	}
````

````new server/internal/platform/httpserver/apitest/operations.go
			Body: with(func(b map[string]any) { b[name] = wrong(name) }), Fields: []FieldProblem{{name, "invalid_format"}}})
	}
	if len(names) > 0 {
		name := names[0]
		cases = append(cases, BodyCase{Name: name + " twice", Body: raw(name, twice(name, valueOf(s, name, valid))),
			Fields: []FieldProblem{{name, "duplicate"}}})
	}
	if nested != "" {
		n := s.Properties[nested].Value
		if inner := slices.Sorted(maps.Keys(n.Properties)); len(inner) > 0 {
			text := `{"` + inner[0] + `":` + twice(inner[0], valueOf(n, inner[0], nil)) + `}`
			cases = append(cases, BodyCase{Name: nested + "." + inner[0] + " twice", Body: raw(nested, text),
				Fields: []FieldProblem{{nested + "." + inner[0], "duplicate"}}})
		}
	}
	if i := slices.IndexFunc(names, func(name string) bool { return s.Properties[name].Value.Type.Includes("string") }); i >= 0 {
		cases = append(cases, BodyCase{Name: "not UTF-8 in " + names[i], Body: raw(names[i], "\"\xff\""),
			Fields: []FieldProblem{{names[i], "invalid_format"}}})
	}
````

````old server/internal/platform/httpserver/apitest/operations.go
	// Case 7: each kind takes the first property no earlier kind took.
````

````new server/internal/platform/httpserver/apitest/operations.go
	// Case 10: each kind takes the first property no earlier kind took.
````

````old server/internal/platform/httpserver/apitest/operations_test.go
		`wrong uuid in owner_id {"name":"x","owner_id":"not-a-uuid"}`,
````

````new server/internal/platform/httpserver/apitest/operations_test.go
		`wrong uuid in owner_id {"name":"x","owner_id":"not-a-uuid"}`,
		`count twice {"count":1,"count":1,` + valid + `}`,
		`settings.notify twice {` + valid + `,"settings":{"notify":false,"notify":false}}`,
		"not UTF-8 in kind {\"kind\":\"\xff\"," + valid + `}`,
````

````old server/internal/bootstrap/contract_test.go
// schema allows it. Operations that need a token get a valid one, so the
// body check, not the authentication, answers.
````

````new server/internal/bootstrap/contract_test.go
// schema allows it; a body that can be read two ways (a property twice, at
// the top and in the nested object; bytes that are not UTF-8) gets 400 too,
// before its structure is checked. Operations that need a token get a valid
// one, so the body check, not the authentication, answers.
````

````old e2e/stories/identity/a10-onboarding-profile.spec.ts
  expect(await onboardingStepsOf(db, email)).toEqual(merged);
});

````

````new e2e/stories/identity/a10-onboarding-profile.spec.ts
  expect(await onboardingStepsOf(db, email)).toEqual(merged);
});

test("A10 (API): onboarding_step twice is the platform's 400, whatever the other one holds, and nothing changes", async ({
  api,
  db,
  request,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const stepsBefore = await onboardingStepsOf(db, email);
  // The body as written, byte for byte: an object of the typed client cannot hold a key twice.
  const send = async (body: string) => {
    const response = await request.patch("/api/v0/me/profile", {
      data: body,
      headers: { ...bearer(pat.token), "Content-Type": "application/json" },
    });
    const problem = (await response.json()) as { code: string; errors?: { field: string; code: string }[] };
    return {
      status: response.status(),
      code: problem.code,
      errors: problem.errors?.map(({ field, code }) => ({ field, code })),
    };
  };

  // WORKSPACE_INVITE is not a step: the platform's 400.
  expect(await send('{"onboarding_step":{"WORKSPACE_INVITE":true}}')).toEqual({
    status: 400,
    code: "bad_request",
    errors: [{ field: "onboarding_step.WORKSPACE_INVITE", code: "not_allowed" }],
  });
  // The same, and onboarding_step again, empty (M2 Codex review, Critical 1): a decoder that merges the two would
  // take WORKSPACE_INVITE for workspace_invite. The body can be read two ways, which is a 400 before any reading.
  expect(await send('{"onboarding_step":{"WORKSPACE_INVITE":true},"onboarding_step":{}}')).toEqual({
    status: 400,
    code: "bad_request",
    errors: [{ field: "onboarding_step", code: "duplicate" }],
  });
  expect(await onboardingStepsOf(db, email)).toEqual(stepsBefore);
});

````

- [ ] **Step 2：检查**：
  - `go -C server test -count=1 ./internal/platform/httpserver/apitest/ -run TestBodyCases` → `ok`；
  - `go -C server test -count=1 ./internal/bootstrap/ -run TestBodiesThatBreakTheStructureAnswer400 -v` → 新的 17 个子测试（8 个操作的"`<字段>` twice"和"not UTF-8 in `<字段>`"，`onboarding_step.profile_complete twice`）和原有的都 `PASS`；
  - `make build`，然后 `pnpm -C e2e exec playwright test a10-onboarding` → `5 passed`。

- [ ] **Step 3：提交**（`test(M2/codex-fixes): every body operation refuses a duplicate and invalid UTF-8; A10 sends Codex's request`）

### Task 3: 令牌管理器在每个 `await` 之后重读记录

spec 2.3。`signOut` 在 `logout` 回来之后经新的 `#end(loginId)` 重读记录：仍是自己的会话才删除，否则跟随；`endSession` 用同一个 `#end`。`#firstRefresh` 在续期失败之后、设"会话暂不可用"之前重读：已是另一个会话就跟随，不设状态、不排重试。两个测试用 `gate` 扣住请求、写过期的租约、用包着 `locks.run` 的锁推迟交回，强制出 Codex 的顺序。

- [ ] **Step 1：应用块**（`… apply . 3`）：

````old web/apps/web/core/lib/auth/token-manager.ts
// every decision reads the record there now, never a copy: a record of another session is followed.
````

````new web/apps/web/core/lib/auth/token-manager.ts
// every decision reads the record there now, never a copy: a record of another session is followed. After
// each await (a request, the lock), the record is read again before anything is written or decided: without
// navigator.locks, the lease is not atomic, and another tab may sign in meanwhile (M2 design 7.1).
````

````old web/apps/web/core/lib/auth/token-manager.ts
   * replaced this session's refresh token, so the browser has nothing of it left to log out with.
````

````new web/apps/web/core/lib/auth/token-manager.ts
   * replaced this session's refresh token, so the browser has nothing of it left to log out with. The same
   * holds when another tab signs in while the logout is out (the lease is not atomic): the new record stays.
````

````old web/apps/web/core/lib/auth/token-manager.ts
      await this.#call("/api/v0/auth/logout", record.refresh_token);
      this.deps.storage.removeItem(AUTH_KEY);
      this.#signedOut();
````

````new web/apps/web/core/lib/auth/token-manager.ts
      await this.#call("/api/v0/auth/logout", record.refresh_token);
      this.#end(loginId);
````

````old web/apps/web/core/lib/auth/token-manager.ts
    return this.deps.lock.run(async () => {
      const record = this.#read();
      if (!isRecordOf(record, loginId)) {
        this.#switchTo(record);
        return false;
      }
      this.deps.storage.removeItem(AUTH_KEY);
      this.#signedOut();
      return true;
    });
````

````new web/apps/web/core/lib/auth/token-manager.ts
    return this.deps.lock.run(async () => this.#end(loginId));
````

````old web/apps/web/core/lib/auth/token-manager.ts
  /** The refresh that decides a starting or unavailable session; a passing failure makes it unavailable. */
  async #firstRefresh(): Promise<void> {
````

````new web/apps/web/core/lib/auth/token-manager.ts
  /**
   * The refresh that decides a starting or unavailable session; a passing failure makes it unavailable, while
   * the record is still that session's. navigator.locks hands the failure back in a later task, after another
   * tab's change of the record may have come in: the tab then follows the record.
   */
  async #firstRefresh(): Promise<void> {
    const loginId = this.#state.loginId;
````

````old web/apps/web/core/lib/auth/token-manager.ts
      this.#set({ status: "unavailable", loginId: this.#state.loginId, retryAt: error.retryAt });
````

````new web/apps/web/core/lib/auth/token-manager.ts
      const record = this.#read();
      if (!isRecordOf(record, loginId)) {
        this.#switchTo(record);
        return;
      }
      this.#set({ status: "unavailable", loginId, retryAt: error.retryAt });
````

````old web/apps/web/core/lib/auth/token-manager.ts
  }

  /** Follows another tab's change of the record, and stops the request that was meant for the old session. */
````

````new web/apps/web/core/lib/auth/token-manager.ts
  }

  /**
   * Ends the session loginId, under the lock: removes the record and signs the tab out when the record is still
   * that session's, else follows the record. Returns whether it ended that session.
   */
  #end(loginId: string | undefined): boolean {
    const record = this.#read();
    if (!isRecordOf(record, loginId)) {
      this.#switchTo(record);
      return false;
    }
    this.deps.storage.removeItem(AUTH_KEY);
    this.#signedOut();
    return true;
  }

  /** Follows another tab's change of the record, and stops the request that was meant for the old session. */
````

````old web/apps/web/core/lib/auth/token-manager.session-change.test.ts
import { RecordingLock, SharedStorage } from "./fake-browser";
import { FakeNerve, json, problem } from "./fake-nerve";
````

````new web/apps/web/core/lib/auth/token-manager.session-change.test.ts
import { RecordingLock, SharedStorage, gate } from "./fake-browser";
import { FakeNerve, json, noContent, problem } from "./fake-nerve";
````

````old web/apps/web/core/lib/auth/token-manager.session-change.test.ts
import { leaseLock, webLock } from "./refresh-lock";
````

````new web/apps/web/core/lib/auth/token-manager.session-change.test.ts
import { LEASE_KEY, leaseLock, webLock } from "./refresh-lock";
````

````old web/apps/web/core/lib/auth/token-manager.session-change.test.ts
// does not rest on the fake's.
````

````new web/apps/web/core/lib/auth/token-manager.session-change.test.ts
// does not rest on the fake's. The same holds when the change comes while the operation awaits, after it
// read the record: it reads the record again before it writes or decides anything (the last describe).
````

````old web/apps/web/core/lib/auth/token-manager.session-change.test.ts
  });
});

````

````new web/apps/web/core/lib/auth/token-manager.session-change.test.ts
  });
});

describe("after an await", () => {
  it("keeps the sign-in another tab made while a sign-out's logout was out and the lease had run out", async () => {
    const { storage, nerve, tab, tabA, stored } = browser("the lease");
    const a = await tabA();
    const out = track(a.tm.signOut());
    await until(() => nerve.to(LOGOUT).length === 1, "A's logout");
    expect(nerve.to(LOGOUT)[0]?.body).toEqual({ refresh_token: "rt-1" });

    // A is frozen past its lease while the logout is out: it hears nothing, and to the other tabs its lease has
    // run out. Fake time moves every tab at once, so the test runs A's lease out by hand; A's own 8 s timeout,
    // which the freeze holds back as well, does not come into it. Tab B then signs in as Y, lease and all.
    storage.hold();
    storage.write(LEASE_KEY, JSON.stringify({ owner: "A", expires: Date.now() - 1 }));
    const b = tab("B");
    const signedIn = track(b.tm.signIn({ ...nerve.tokens(), refresh_token: "rt-y" }));
    await until(() => signedIn.settled, "B's sign-in");
    expect(stored()).toEqual({ refresh_token: "rt-y", login_id: Y });

    // A thaws as nerve answers its logout, before it hears of B's sign-in: Y's record stays, and A follows it.
    nerve.to(LOGOUT)[0]?.answer(noContent());
    await until(() => out.settled, "A's sign-out");
    storage.deliver();

    expect(out.error).toBeUndefined();
    expect(stored()).toEqual({ refresh_token: "rt-y", login_id: Y });
    expect([a.tm.state, b.tm.state]).toEqual([
      { status: "signed-in", loginId: Y },
      { status: "signed-in", loginId: Y },
    ]);
    expect(a.changes).toEqual([`signed-in ${Y}`]);
    expect(nerve.to(LOGOUT)).toHaveLength(1);
  });

  it("follows another tab's sign-in that came in before navigator.locks handed a failed first refresh back", async () => {
    const storage = new SharedStorage();
    storage.data.set(AUTH_KEY, JSON.stringify({ refresh_token: "rt-0", login_id: X }));
    const nerve = new FakeNerve();
    const locks = new RecordingLock();
    const handBack = gate();
    const view = storage.tab("A");
    const a = new TokenManager({
      storage: view,
      // navigator.locks settles request() in a task of its own once the task is done, so another tab's storage
      // event may come in first: this lock holds the settling back until the test lets it go.
      lock: { run: (task) => locks.run(task).finally(() => handBack.promise) },
      client: nerve.client(),
      now: () => Date.now(),
      randomHex: (bytes) => "a".repeat(bytes * 2),
    });
    view.onStorage((key) => {
      if (key === AUTH_KEY) a.handleStorageChange();
    });
    const changes: string[] = [];
    a.subscribe(() => changes.push(`${a.state.status} ${a.state.loginId ?? "-"}`));

    // A's first refresh, as X, fails for a passing reason; another tab signs in as Y before the lock hands the
    // failure back, and A follows Y.
    const started = track(a.start());
    await until(() => nerve.calls.length === 1, "A's first refresh");
    nerve.calls[0]?.answer(problem(503, "server_busy"));
    await until(() => !locks.held, "the refresh's end under the lock");
    storage.write(AUTH_KEY, JSON.stringify({ refresh_token: "rt-y", login_id: Y }));
    await until(() => a.state.loginId === Y, "A following Y");
    handBack.open();
    await until(() => started.settled, "A's start");

    // X's failure leaves Y's session as it is: not unavailable, and no retry of it is due.
    expect(a.state).toEqual({ status: "signed-in", loginId: Y });
    expect(changes).toEqual([`starting ${X}`, `signed-in ${Y}`]);
    await vi.advanceTimersByTimeAsync(60_000);
    expect(nerve.calls).toHaveLength(1);
  });
});

````

- [ ] **Step 2：检查**：`pnpm -C web/apps/web exec vitest run core/lib/auth/` → 全部通过，其中 `token-manager.session-change.test.ts` 为 16 个（新的 2 个在 `describe("after an await")` 下）。

- [ ] **Step 3：提交**（`fix(M2/codex-fixes): the token manager reads the record again after each await`）

### Task 4: 服务角色的权限：`deploy/runtime-grants.sql` 和守住它的测试

spec 2.4。文件逐个列出表和序列，授权给组角色 `nerve_runtime`。`runtime_role_test.go` 在自己的 testcontainers 库里建所有者和服务两个角色：所有者迁移并执行文件，服务角色运行整个 nerve；另一个测试核对库里每张表、每个序列的实际权限等于文件的规则（多给、少给都失败）。README 的"迁移"一条改为指向文件；River 索引的删除一句补上分离角色的情况。

- [ ] **Step 1：应用块**（`… apply . 4`）：

````file deploy/runtime-grants.sql
-- What the role nerve serves with needs when another role, the owner of the tables, runs the migrations
-- (README, "部署"). It grants to the group role nerve_runtime; the login role of nerve serve is a member of it.
-- Run it as the owner of the tables after every `nerve migrate up`: it names each table and sequence, and a new
-- migration may add one. Running it again changes nothing.
--
-- server/internal/bootstrap/runtime_role_test.go runs nerve on a role with exactly these grants (ready, River's
-- jobs, River's reindex), and fails when a table or sequence of the schema has no grant here.

GRANT USAGE ON SCHEMA public TO nerve_runtime;

-- The modules' tables.
GRANT SELECT, INSERT, UPDATE, DELETE ON users, profiles, auth_sessions, api_tokens TO nerve_runtime;

-- River's tables and the sequences of their ids. River rebuilds the indexes of river_job every day with
-- REINDEX INDEX CONCURRENTLY, which takes MAINTAIN on the table (PostgreSQL 17 and later).
GRANT SELECT, INSERT, UPDATE, DELETE ON river_job, river_leader, river_queue, river_notification TO nerve_runtime;
GRANT USAGE ON SEQUENCE river_job_id_seq, river_notification_id_seq TO nerve_runtime;
GRANT MAINTAIN ON river_job TO nerve_runtime;

-- goose's record of the migrations: /readyz reads it to tell whether every migration is applied.
GRANT SELECT ON goose_db_version TO nerve_runtime;
````

````file server/internal/bootstrap/runtime_role_test.go
package bootstrap

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The role nerve serves with when another role owns the tables and runs the
// migrations (README "部署"): a login role in the group role nerve_runtime,
// which deploy/runtime-grants.sql grants to. These tests run nerve on such a
// role, and hold the file to every table and sequence of the schema.

// splitRoles is a database whose tables belong to an owner role that ran the
// migrations and then deploy/runtime-grants.sql, and a login role in
// nerve_runtime to serve with.
type splitRoles struct {
	owner      *pgxpool.Pool // the owner of the tables
	server     *pgxpool.Pool // the login role nerve serves with
	serverName string
	serverURL  string
}

func newSplitRoles(t *testing.T) splitRoles {
	t.Helper()
	ctx := context.Background()
	adminURL := pgtest.NewEmptyDatabase(t)
	admin := openPool(t, adminURL)
	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatal(err)
	}
	db := strings.TrimPrefix(u.Path, "/")
	// The roles belong to the cluster, which the tests of this binary share: named after the database, which
	// is new; nerve_runtime once.
	ownerName, serverName := db+"_owner", db+"_server"
	for _, sql := range []string{
		"CREATE ROLE " + pgx.Identifier{ownerName}.Sanitize() + " LOGIN PASSWORD 'owner'",
		"ALTER DATABASE " + pgx.Identifier{db}.Sanitize() + " OWNER TO " + pgx.Identifier{ownerName}.Sanitize(),
		"DO $$ BEGIN CREATE ROLE nerve_runtime NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$",
		"CREATE ROLE " + pgx.Identifier{serverName}.Sanitize() + " LOGIN PASSWORD 'server' IN ROLE nerve_runtime",
	} {
		if _, err := admin.Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	as := func(name, password string) string {
		v := *u
		v.User = url.UserPassword(name, password)
		return v.String()
	}
	owner := openPool(t, as(ownerName, "owner"))
	m, err := postgres.NewMigrator(owner, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })
	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("migrate as the owner: %v", err)
	}
	grants, err := os.ReadFile(grantsFile())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, string(grants)); err != nil {
		t.Fatalf("run deploy/runtime-grants.sql as the owner: %v", err)
	}
	serverURL := as(serverName, "server")
	return splitRoles{owner: owner, server: openPool(t, serverURL), serverName: serverName, serverURL: serverURL}
}

// grantsFile is deploy/runtime-grants.sql, found from this file, three
// directories below the repository root (tests run without -trimpath).
func grantsFile() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "deploy", "runtime-grants.sql")
}

// nerve serves on the role: ready, River's jobs run (the session cleanup
// completes as the app starts), and River's daily reindex goes through, as
// River runs it, index by index.
func TestTheRuntimeRoleServesWithTheGrantsFile(t *testing.T) {
	roles := newSplitRoles(t)
	ctx := context.Background()
	user, expired, live := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for _, insert := range []struct {
		sql  string
		args []any
	}{
		{"INSERT INTO users (id, email, password, display_name) VALUES ($1, 'runtime@example.com', 'x', 'runtime')", []any{user}},
		{`INSERT INTO auth_sessions (id, user_id, token_hash, expires_at) VALUES
			($1, $3, sha256('a'), now() - interval '1 minute'), ($2, $3, sha256('b'), now() + interval '1 hour')`, []any{expired, live, user}},
	} {
		if _, err := roles.owner.Exec(ctx, insert.sql, insert.args...); err != nil {
			t.Fatal(err)
		}
	}
	var logs lockedBuffer
	base := startAppLogging(t, testConfig(t, roles.serverURL, false), migrations.FS(), slog.New(slog.NewTextHandler(&logs, nil)))

	if status, p := getReadyz(t, base); status != 200 {
		t.Errorf("/readyz = %d %+v, want 200", status, p)
	}
	for deadline := time.Now().Add(15 * time.Second); cleanupRuns(t, roles.owner) == 0; time.Sleep(50 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("the session cleanup did not run as %s; logs:\n%s", roles.serverName, logs.String())
		}
	}
	if sessionExists(t, roles.owner, expired) || !sessionExists(t, roles.owner, live) {
		t.Errorf("after the cleanup: expired session left %v, live one left %v; want only the live one",
			sessionExists(t, roles.owner, expired), sessionExists(t, roles.owner, live))
	}
	if all := logs.String(); !strings.Contains(all, `msg="jobs started"`) || strings.Contains(all, "permission denied") {
		t.Errorf("logs want jobs started and no permission denied:\n%s", all)
	}
	for _, index := range river.ReindexerIndexNamesDefault() {
		if _, err := roles.server.Exec(ctx, "REINDEX INDEX CONCURRENTLY "+pgx.Identifier{index}.Sanitize()); err != nil {
			t.Errorf("REINDEX INDEX CONCURRENTLY %s as %s: %v", index, roles.serverName, err)
		}
	}
}

// Every table and sequence the migrations leave has its grant in the file,
// and no more: a migration that adds one fails here until the file grants
// it. The runtime role reads and writes every table but goose's record,
// which it only reads; it uses the sequences of the tables it writes; it
// may reindex River's jobs.
func TestTheGrantsFileCoversEveryTableAndSequence(t *testing.T) {
	roles := newSplitRoles(t)
	rows, err := roles.owner.Query(context.Background(), `
		SELECT c.relname, c.relkind = 'S',
		       CASE WHEN c.relkind = 'S' THEN ARRAY[has_sequence_privilege($1, c.oid, 'USAGE'), false, false, false, false, false, false, false]
		            ELSE ARRAY[has_table_privilege($1, c.oid, 'SELECT'), has_table_privilege($1, c.oid, 'INSERT'),
		                       has_table_privilege($1, c.oid, 'UPDATE'), has_table_privilege($1, c.oid, 'DELETE'),
		                       has_table_privilege($1, c.oid, 'TRUNCATE'), has_table_privilege($1, c.oid, 'REFERENCES'),
		                       has_table_privilege($1, c.oid, 'TRIGGER'), has_table_privilege($1, c.oid, 'MAINTAIN')] END
		  FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		 WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p', 'S')
		 ORDER BY c.relname`, roles.serverName)
	if err != nil {
		t.Fatal(err)
	}
	tablePrivileges := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER", "MAINTAIN"}
	dml := []string{"SELECT", "INSERT", "UPDATE", "DELETE"}
	seen := 0
	for rows.Next() {
		var name string
		var sequence bool
		var has []bool
		if err := rows.Scan(&name, &sequence, &has); err != nil {
			t.Fatal(err)
		}
		seen++
		var got, want []string
		for i, ok := range has {
			if !ok {
				continue
			}
			if sequence {
				got = append(got, "USAGE")
			} else {
				got = append(got, tablePrivileges[i])
			}
		}
		switch {
		case name == "goose_db_version_id_seq":
		case sequence:
			want = []string{"USAGE"}
		case name == "goose_db_version":
			want = []string{"SELECT"}
		case name == "river_job":
			want = append(slices.Clone(dml), "MAINTAIN")
		default:
			want = dml
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: %s has %v, want %v: deploy/runtime-grants.sql grants each table and sequence", name, roles.serverName, got, want)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if seen == 0 {
		t.Fatal("no table or sequence in the schema public")
	}
}
````

````old README.md
- **迁移**：prod 默认不在启动时迁移（`database.auto_migrate: false`），先执行 `nerve migrate up`，再 `nerve serve`。用单独的数据库角色执行迁移时，运行服务的角色除了读写业务表，还要能读 `goose_db_version`（`GRANT SELECT ON goose_db_version TO <服务的角色>`）：`/readyz` 靠它判断迁移是否已完成。
````

````new README.md
- **迁移**：prod 默认不在启动时迁移（`database.auto_migrate: false`），先执行 `nerve migrate up`，再 `nerve serve`。
  - **迁移和服务分用两个数据库角色时**（表的所有者执行迁移，服务用另一个角色），服务的角色需要的权限全部写在 [`deploy/runtime-grants.sql`](deploy/runtime-grants.sql)：业务表的读写，River 的表和序列，`river_job` 的 `MAINTAIN`（River 每天 00:00 UTC 用 `REINDEX INDEX CONCURRENTLY` 重建它的索引，需要 PostgreSQL 17 起），`goose_db_version` 的读（`/readyz` 靠它判断迁移是否已完成）。文件授权给组角色 `nerve_runtime`：先建一次组角色，服务登录用的角色加入它，例如 `CREATE ROLE nerve_runtime NOLOGIN;`、`CREATE ROLE nerve_app LOGIN PASSWORD '…' IN ROLE nerve_runtime;`。
  - **每次 `nerve migrate up` 之后**，以表的所有者执行一次这个文件，例如 `psql -v ON_ERROR_STOP=1 -f deploy/runtime-grants.sql`：新的迁移可能加了表，文件逐个列出表和序列；重复执行没有影响。
  - 少了权限时 `/readyz` 仍是 200，但 River 的任务（会话清理）和索引重建因 `permission denied`（42501）失败，只记在日志里。`server/internal/bootstrap/runtime_role_test.go` 用恰好这些权限的角色运行 nerve（就绪、会话清理、索引重建），库里有表或序列的权限与文件不符时失败。
````

````old README.md
- **停机时 River 的日志**：nerve 启动后的最初几秒内停机时，River 可能记几条 ERROR：它的维护服务还在错开启动时是 `maintenance.PeriodicJobEnqueuer: Error starting transaction`（`context canceled`）；nerve 还没记 `jobs started` 时，还可能有 `notifier.Notifier: Error running listener (will attempt reconnect after backoff)` 等几条 `conn closed`。这是 River 停止时自己记的，不是故障：退出码为 0，没有任务停在 `running`，没有连接泄漏；这次没投递的会话清理，下次启动时投递。运行中的 nerve 停机，只有恰好落在清理任务投递的那几毫秒（每 `auth.session_cleanup_interval` 一次）才会出现第一条。River 每天 00:00 UTC 用 `REINDEX INDEX CONCURRENTLY` 重建 `river_job` 的索引；停机打断重建时，River 先删掉没建完的 `*_ccnew` 索引再停下，这次删除最多等 15 秒。有访问过 `river_job` 的长事务挡住删除、超过 `jobs.shutdown_timeout` 加 1 秒时，nerve 记 ERROR（`jobs still running …`）并以退出码 1 退出；挡满 15 秒时索引留下，River 此后跳过这个索引，每次重建都记 WARN `maintenance.Reindexer: Found reindex artifact … skipping reindex`：用 `DROP INDEX CONCURRENTLY` 删掉这条 WARN 的 `artifact_names` 中列出的索引。
````

````new README.md
- **停机时 River 的日志**：nerve 启动后的最初几秒内停机时，River 可能记几条 ERROR：它的维护服务还在错开启动时是 `maintenance.PeriodicJobEnqueuer: Error starting transaction`（`context canceled`）；nerve 还没记 `jobs started` 时，还可能有 `notifier.Notifier: Error running listener (will attempt reconnect after backoff)` 等几条 `conn closed`。这是 River 停止时自己记的，不是故障：退出码为 0，没有任务停在 `running`，没有连接泄漏；这次没投递的会话清理，下次启动时投递。运行中的 nerve 停机，只有恰好落在清理任务投递的那几毫秒（每 `auth.session_cleanup_interval` 一次）才会出现第一条。River 每天 00:00 UTC 用 `REINDEX INDEX CONCURRENTLY` 重建 `river_job` 的索引；停机打断重建时，River 先删掉没建完的 `*_ccnew` 索引再停下，这次删除最多等 15 秒。有访问过 `river_job` 的长事务挡住删除、超过 `jobs.shutdown_timeout` 加 1 秒时，nerve 记 ERROR（`jobs still running …`）并以退出码 1 退出；挡满 15 秒时索引留下，River 此后跳过这个索引，每次重建都记 WARN `maintenance.Reindexer: Found reindex artifact … skipping reindex`：用 `DROP INDEX CONCURRENTLY` 删掉这条 WARN 的 `artifact_names` 中列出的索引。迁移和服务分用两个角色时，索引属于表的所有者，服务的角色删不掉它（River 停机时的那次删除也会因此失败，索引留下）：由表的所有者执行这条删除。
````

- [ ] **Step 2：检查**：`go -C server test -count=1 ./internal/bootstrap/ -run 'TestTheRuntimeRoleServesWithTheGrantsFile|TestTheGrantsFileCoversEveryTableAndSequence' -v` → 两个都 `PASS`（原型上约 4.5 秒）。

- [ ] **Step 3：提交**（`feat(M2/codex-fixes): the runtime role's grants in deploy/runtime-grants.sql, proven by serving on them`）

### Task 5: `AllowAll` 拒绝同一个桶和键出现两次

spec 2.10。与"别的限流器的桶"同样 panic。

- [ ] **Step 1：应用块**（`… apply . 5`）：

````old server/internal/platform/ratelimit/ratelimit.go
import (
````

````new server/internal/platform/ratelimit/ratelimit.go
import (
	"slices"
````

````old server/internal/platform/ratelimit/ratelimit.go
// denied is nil when every bucket gave a unit. Every bucket must belong to l.
````

````new server/internal/platform/ratelimit/ratelimit.go
// denied is nil when every bucket gave a unit. Every bucket must belong to l,
// and each check must be another bucket or key: one check twice would see
// the unit that is left twice and take two, below zero.
````

````old server/internal/platform/ratelimit/ratelimit.go
			panic("ratelimit: AllowAll with a bucket of another limiter")
````

````new server/internal/platform/ratelimit/ratelimit.go
			panic("ratelimit: AllowAll with a bucket of another limiter")
		}
		if slices.Contains(checks[:i], c) {
			panic("ratelimit: AllowAll with the same bucket and key twice")
````

````old server/internal/platform/ratelimit/ratelimit_test.go
}

// Reserve takes a unit at once; its refund gives it back once, and never
````

````new server/internal/platform/ratelimit/ratelimit_test.go
}

// The same bucket and key twice would take two units where one is left: a
// programming error, like a bucket of another limiter.
func TestAllowAllRejectsTheSameCheckTwice(t *testing.T) {
	l := New(newClock().now)
	b := l.Bucket("login_ip", Rate{PerMinute: 1, Burst: 1})
	defer func() {
		if recover() == nil {
			t.Error("AllowAll() with the same bucket and key twice did not panic")
		}
	}()
	l.AllowAll(Check{Bucket: b, Key: "k"}, Check{Bucket: b, Key: "k"})
}

// Reserve takes a unit at once; its refund gives it back once, and never
````

- [ ] **Step 2：检查**：`go -C server test -count=1 ./internal/platform/ratelimit/ ./internal/modules/identity/...` → 全部 `ok`（登录的两个桶不同，不受影响）。

- [ ] **Step 3：提交**（`fix(M2/codex-fixes): AllowAll refuses the same bucket and key twice`）

### Task 6: A8、A9、A10 的页面版本和接口版本调用同一组业务断言

spec 2.6。三个带参数的断言；A9 文件内的 `profileOf` 删除；A10 的接口版本先建好"另一步已完成"的起点、记下 `stepsBefore`，再改名字和步骤，与页面版本走同一个断言。

- [ ] **Step 1：应用块**（`… apply . 6`）：

````old e2e/fixtures/assert/identity.ts
}

/** The onboarding steps of the account of email, a lowercased address, as its profile holds them. */
export async function onboardingStepsOf(db: Database, email: string): Promise<unknown> {
````

````new e2e/fixtures/assert/identity.ts
}

/** The names and the time zone of an account, which A8 changes. */
type AccountChange = Partial<Pick<AccountRow, "first_name" | "last_name" | "display_name" | "user_timezone">>;

/**
 * A8: the account of email, a lowercased address, is the account `before` with change, and nothing else changed but
 * its updated_at, which moved on. Returns it now.
 */
export async function expectAccountChanged(
  db: Database,
  email: string,
  before: AccountRow,
  change: AccountChange
): Promise<AccountRow> {
  const after = await accountOf(db, email);
  expect(after).toEqual({ ...before, ...change, updated_at: after.updated_at });
  expect(after.updated_at.getTime()).toBeGreaterThan(before.updated_at.getTime());
  return after;
}

/** The preferences of a profile, which A9 changes. */
export interface Preferences {
  theme: string;
  language: string;
  start_of_the_week: number;
}

/** A9: the profile of the account of email, a lowercased address, holds preferences. */
export async function expectPreferences(db: Database, email: string, preferences: Preferences): Promise<void> {
  const { id } = await accountOf(db, email);
  expect(
    await db.query("SELECT theme, language, start_of_the_week FROM profiles WHERE user_id = $1", [id]),
    `the preferences of ${email}`
  ).toEqual([preferences]);
}

/** A profile's onboarding steps: done or not. */
export interface OnboardingSteps {
  profile_complete: boolean;
  workspace_create: boolean;
  workspace_invite: boolean;
  workspace_join: boolean;
}

/** The onboarding steps of the account of email, a lowercased address, as its profile holds them. */
export async function onboardingStepsOf(db: Database, email: string): Promise<OnboardingSteps> {
````

````old e2e/fixtures/assert/identity.ts
  const rows = await db.query<{ onboarding_step: unknown }>("SELECT onboarding_step FROM profiles WHERE user_id = $1", [
    id,
  ]);
````

````new e2e/fixtures/assert/identity.ts
  const rows = await db.query<{ onboarding_step: OnboardingSteps }>(
    "SELECT onboarding_step FROM profiles WHERE user_id = $1",
    [id]
  );
````

````old e2e/fixtures/assert/identity.ts
  return rows[0]?.onboarding_step;
````

````new e2e/fixtures/assert/identity.ts
  return rows[0]?.onboarding_step as OnboardingSteps;
}

/**
 * A10: the account of email, a lowercased address, took the profile step: its first name is firstName, and its
 * onboarding steps are stepsBefore with the profile's done; the others keep their values (M2 design 3.14).
 */
export async function expectProfileStepTaken(
  db: Database,
  email: string,
  firstName: string,
  stepsBefore: OnboardingSteps
): Promise<void> {
  expect((await accountOf(db, email)).first_name).toBe(firstName);
  expect(await onboardingStepsOf(db, email)).toEqual({ ...stepsBefore, profile_complete: true });
````

````old e2e/stories/identity/a8-update-me.spec.ts
import { accountOf } from "../../fixtures/assert/identity";
````

````new e2e/stories/identity/a8-update-me.spec.ts
import { accountOf, expectAccountChanged } from "../../fixtures/assert/identity";
````

````old e2e/stories/identity/a8-update-me.spec.ts
  const change = { first_name: "Ada", last_name: "Lovelace", display_name: "ada", user_timezone: "Asia/Shanghai" };
  const after = await accountOf(db, email);
  expect(after).toMatchObject(change);
  expect(after.updated_at.getTime()).toBeGreaterThan(before.updated_at.getTime());
````

````new e2e/stories/identity/a8-update-me.spec.ts
  await expectAccountChanged(db, email, before, {
    first_name: "Ada",
    last_name: "Lovelace",
    display_name: "ada",
    user_timezone: "Asia/Shanghai",
  });
````

````old e2e/stories/identity/a8-update-me.spec.ts
  const watch = await watchPage(page);
  await page.goto("/settings/profile/general");
````

````new e2e/stories/identity/a8-update-me.spec.ts
  const watch = await watchPage(page);
  await page.goto("/settings/profile/general");
  const before = await accountOf(db, email);
````

````old e2e/stories/identity/a8-update-me.spec.ts
  const saved = await accountOf(db, email);
  expect(saved).toMatchObject({ first_name: "Ada", last_name: "", display_name: "Ada Lovelace" });
````

````new e2e/stories/identity/a8-update-me.spec.ts
  const saved = await expectAccountChanged(db, email, before, {
    first_name: "Ada",
    last_name: "",
    display_name: "Ada Lovelace",
  });
````

````old e2e/stories/identity/a8-update-me.spec.ts
  expect(data).toMatchObject(change);
  const after = await accountOf(db, email);
  expect(after).toMatchObject(change);
  expect(after.updated_at.getTime()).toBeGreaterThan(before.updated_at.getTime());
````

````new e2e/stories/identity/a8-update-me.spec.ts
  expect(data).toMatchObject(change);
  const after = await expectAccountChanged(db, email, before, change);
````

````old e2e/stories/identity/a9-preferences.spec.ts
import { accountOf } from "../../fixtures/assert/identity";
````

````new e2e/stories/identity/a9-preferences.spec.ts
import { expectPreferences } from "../../fixtures/assert/identity";
````

````old e2e/stories/identity/a9-preferences.spec.ts
// its button in both languages (7.7, 9.6).

const profileOf = async (db: Parameters<typeof accountOf>[0], email: string) =>
  db.query("SELECT theme, language, start_of_the_week FROM profiles WHERE user_id = $1", [
    (await accountOf(db, email)).id,
  ]);
````

````new e2e/stories/identity/a9-preferences.spec.ts
// its button in both languages (7.7, 9.6).
````

````old e2e/stories/identity/a9-preferences.spec.ts
  const change = { theme: "dark", language: "zh-CN", start_of_the_week: 1 };
  expect(await profileOf(db, email)).toEqual([change]);
````

````new e2e/stories/identity/a9-preferences.spec.ts
  await expectPreferences(db, email, { theme: "dark", language: "zh-CN", start_of_the_week: 1 });
````

````old e2e/stories/identity/a9-preferences.spec.ts
  expect(await profileOf(db, email)).toEqual([{ theme: "system", language: "en", start_of_the_week: 0 }]);
````

````new e2e/stories/identity/a9-preferences.spec.ts
  await expectPreferences(db, email, { theme: "system", language: "en", start_of_the_week: 0 });
````

````old e2e/stories/identity/a9-preferences.spec.ts
  expect(await profileOf(db, x)).toEqual([{ theme: "dark", language: "en", start_of_the_week: 0 }]);
  expect(await profileOf(db, y)).toEqual([{ theme: "light-contrast", language: "en", start_of_the_week: 0 }]);
````

````new e2e/stories/identity/a9-preferences.spec.ts
  await expectPreferences(db, x, { theme: "dark", language: "en", start_of_the_week: 0 });
  await expectPreferences(db, y, { theme: "light-contrast", language: "en", start_of_the_week: 0 });
````

````old e2e/stories/identity/a9-preferences.spec.ts
  expect(data).toMatchObject(change);
  expect(await profileOf(db, email)).toEqual([change]);
````

````new e2e/stories/identity/a9-preferences.spec.ts
  expect(data).toMatchObject(change);
  await expectPreferences(db, email, change);
````

````old e2e/stories/identity/a10-onboarding-profile.spec.ts
import { accountOf, onboardingStepsOf } from "../../fixtures/assert/identity";
````

````new e2e/stories/identity/a10-onboarding-profile.spec.ts
import { accountOf, expectProfileStepTaken, onboardingStepsOf } from "../../fixtures/assert/identity";
````

````old e2e/stories/identity/a10-onboarding-profile.spec.ts
async function takeProfileStep(page: Page, watch: PageWatch, db: Database, email: string): Promise<void> {
  await expect(page.getByText("Create your profile.")).toBeVisible();
````

````new e2e/stories/identity/a10-onboarding-profile.spec.ts
async function takeProfileStep(page: Page, watch: PageWatch, db: Database, email: string): Promise<void> {
  await expect(page.getByText("Create your profile.")).toBeVisible();
  const stepsBefore = await onboardingStepsOf(db, email);
````

````old e2e/stories/identity/a10-onboarding-profile.spec.ts
  expect(await onboardingStepsOf(db, email)).toEqual({
    profile_complete: true,
    workspace_create: false,
    workspace_invite: false,
    workspace_join: false,
  });
  expect((await accountOf(db, email)).first_name).toBe("Ada");
````

````new e2e/stories/identity/a10-onboarding-profile.spec.ts
  await expectProfileStepTaken(db, email, "Ada", stepsBefore);
````

````old e2e/stories/identity/a10-onboarding-profile.spec.ts

  const named = await api.PATCH("/api/v0/me", { body: { first_name: "Ada" }, headers: bearer(pat.token) });
  expect(named.response.status).toBe(200);
  expect((await accountOf(db, email)).first_name).toBe("Ada");

````

````new e2e/stories/identity/a10-onboarding-profile.spec.ts

````

````old e2e/stories/identity/a10-onboarding-profile.spec.ts
  expect(joined.response.status).toBe(200);
````

````new e2e/stories/identity/a10-onboarding-profile.spec.ts
  expect(joined.response.status).toBe(200);
  const stepsBefore = await onboardingStepsOf(db, email);
````

````old e2e/stories/identity/a10-onboarding-profile.spec.ts
  // One key: it is merged in, the other three keep their values (M2 design 3.14).
````

````new e2e/stories/identity/a10-onboarding-profile.spec.ts
  // The name, then the step: one key, merged in, the other three keep their values.
  const named = await api.PATCH("/api/v0/me", { body: { first_name: "Ada" }, headers: bearer(pat.token) });
  expect(named.response.status).toBe(200);
````

````old e2e/stories/identity/a10-onboarding-profile.spec.ts
  const merged = { profile_complete: true, workspace_create: false, workspace_invite: false, workspace_join: true };
  expect(await onboardingStepsOf(db, email)).toEqual(merged);
````

````new e2e/stories/identity/a10-onboarding-profile.spec.ts
  await expectProfileStepTaken(db, email, "Ada", stepsBefore);
  const merged = await onboardingStepsOf(db, email);
````

- [ ] **Step 2：检查**：`pnpm -C e2e run check:types`、`pnpm -C e2e run check:lint`、`pnpm -C e2e run check:format` 通过；`make build`，然后 `pnpm -C e2e exec playwright test a8-update a9-pref a10-onboarding` → `12 passed`。

- [ ] **Step 3：提交**（`test(M2/codex-fixes): A8, A9 and A10 call shared business assertions`）

### Task 7: 一个 store 的修改一个接一个发出

spec 2.8。`oneAtATime()` 返回一个队列：每个任务在前一个结束（成功或失败）之后才开始。`ProfileStore.updateUserProfile` 和 `UserStore.updateCurrentUser` 经它发出，编号和"丢弃较旧的应答"删除。两个 store 的测试改写（`whole` 块）。A9 的新故事是 Codex 的页面实验：第一个修改在路由上扣住，再选一次，一个已应答的探测请求证明第二个修改没有发出（spec D7），放行之后页面、接口、数据库和刷新都是最后的选择。

- [ ] **Step 1：应用块**（`… apply . 7`）：

````file web/apps/web/core/lib/one-at-a-time.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

/**
 * A queue that runs the tasks given to it one at a time, in the order given: each starts once the one before it
 * has settled, whether it succeeded or failed. A store sends its changes of one resource through one, so nerve
 * applies them in the order they were made, and the last change made is what nerve holds (v0 design 7.7).
 */
export function oneAtATime(): <T>(task: () => Promise<T>) => Promise<T> {
  let last: Promise<unknown> = Promise.resolve();
  return <T>(task: () => Promise<T>): Promise<T> => {
    const run = last.then(task);
    last = run.catch(() => undefined);
    return run;
  };
}
````

````old web/apps/web/core/store/user/profile.store.ts
import type { ApiClient, Profile, ProfileUpdate, Theme } from "@nerve/api-client";
````

````new web/apps/web/core/store/user/profile.store.ts
import type { ApiClient, Profile, ProfileUpdate, Theme } from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
````

````old web/apps/web/core/store/user/profile.store.ts
  /** The number of the last update sent, and of the one whose answer the profile holds (updateUserProfile). */
  private updatesSent = 0;
  private updateWritten = 0;
````

````new web/apps/web/core/store/user/profile.store.ts
  /** The profile's changes, sent one at a time (updateUserProfile). */
  private readonly changes = oneAtATime();
````

````old web/apps/web/core/store/user/profile.store.ts
   * The profile becomes nerve's answer, never the change asked for: a refused change leaves it as it was. The
   * updates are numbered as they are sent, and an answer older than the one the profile holds is dropped: this
   * assumes nerve applies the updates in the order they were sent, as the PAT store assumes of its requests, so
   * the older answer is an older profile. Only answers count: when the newer update fails, the older one's
   * answer still writes.
````

````new web/apps/web/core/store/user/profile.store.ts
   * The profile becomes nerve's answer, never the change asked for: a refused change leaves it as it was. A change
   * is sent once the one before it is answered, or has failed: nerve applies them in the order they were made, and
   * the answer to the last one is the profile nerve holds.
````

````old web/apps/web/core/store/user/profile.store.ts
  updateUserProfile = async (data: ProfileUpdate): Promise<Profile> => {
    const update = ++this.updatesSent;
    const profile = await this.userService.updateCurrentUserProfile(data);
    if (update > this.updateWritten) {
      this.updateWritten = update;
````

````new web/apps/web/core/store/user/profile.store.ts
  updateUserProfile = (data: ProfileUpdate): Promise<Profile> =>
    this.changes(async () => {
      const profile = await this.userService.updateCurrentUserProfile(data);
````

````old web/apps/web/core/store/user/profile.store.ts
    }
    return profile;
  };
````

````new web/apps/web/core/store/user/profile.store.ts
      return profile;
    });
````

````whole web/apps/web/core/store/user/profile.store.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Profile } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import type { RootStore } from "@/store/root.store";
import { ProfileStore } from "@/store/user/profile.store";

// The profile's changes against a fake nerve that answers each when the test says: a change goes out once the one
// before it is answered or has failed, so nerve applies them in the order they were made and the last answer is
// what nerve holds. The page's language and theme follow the profile (StoreWrapper).

const PROFILE = "/api/v0/me/profile";

/** nerve's answer to a change of the language: the profile with it. */
const profileIn = (language: "en" | "zh-CN") => ({ language, theme: "system" }) as Profile;

/** A store, and two changes made one after the other, to Chinese and then to English: only the first is out. */
async function twoChanges() {
  const nerve = new FakeNerve();
  const store = new ProfileStore({} as RootStore, nerve.client());
  const first = track(store.updateUserProfile({ language: "zh-CN" }));
  const second = track(store.updateUserProfile({ language: "en" }));
  await until(() => nerve.calls.length === 1, "the first change");
  await vi.advanceTimersByTimeAsync(1_000);
  expect(nerve.calls.map((call) => [call.method, call.path, call.body])).toEqual([
    ["PATCH", PROFILE, { language: "zh-CN" }],
  ]);
  return { nerve, store, first, second };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ProfileStore.updateUserProfile", () => {
  it("sends a change once nerve has answered the one before it", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.answer(json(200, profileIn("zh-CN")));
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.value).toEqual(profileIn("zh-CN"));
    expect(store.data).toEqual(profileIn("zh-CN"));
    expect(nerve.calls[1]?.body).toEqual({ language: "en" });
    nerve.calls[1]?.answer(json(200, profileIn("en")));
    await until(() => second.settled, "the second answer");

    expect(second.value).toEqual(profileIn("en"));
    expect(store.data).toEqual(profileIn("en"));
  });

  it("sends the next change when nerve refuses the one before it, which leaves the profile as it was", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.answer(problem(500, "internal_error"));
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(store.data).toBeUndefined();
    nerve.calls[1]?.answer(json(200, profileIn("en")));
    await until(() => second.settled, "the second answer");

    expect(store.data).toEqual(profileIn("en"));
  });

  it("sends the next change when the one before it gets no answer", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.fail();
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.error).toBeInstanceOf(TypeError);
    nerve.calls[1]?.answer(json(200, profileIn("en")));
    await until(() => second.settled, "the second answer");

    expect(store.data).toEqual(profileIn("en"));
  });
});
````

````old web/apps/web/core/store/user/index.ts
import { SessionChangedError } from "@/lib/auth/token-manager";
````

````new web/apps/web/core/store/user/index.ts
import { SessionChangedError } from "@/lib/auth/token-manager";
import { oneAtATime } from "@/lib/one-at-a-time";
````

````old web/apps/web/core/store/user/index.ts
  /** The number of the last update sent, and of the one whose answer data holds (updateCurrentUser). */
  private updatesSent = 0;
  private updateWritten = 0;
````

````new web/apps/web/core/store/user/index.ts
  /** The account's changes, sent one at a time (updateCurrentUser). */
  private readonly changes = oneAtATime();
````

````old web/apps/web/core/store/user/index.ts
   * @description updates the account's names or time zone; fails, writing nothing, when nerve refuses. The
   * updates are numbered as they are sent, and an answer older than the one data holds is dropped: this assumes
   * nerve applies the updates in the order they were sent, as the PAT store assumes of its requests, so the older
   * answer is an older account. Only answers count: when the newer update fails, the older one's answer still
   * writes.
````

````new web/apps/web/core/store/user/index.ts
   * @description updates the account's names or time zone; fails, writing nothing, when nerve refuses. An update
   * is sent once the one before it is answered, or has failed: nerve applies them in the order they were made, and
   * the answer to the last one is the account nerve holds.
````

````old web/apps/web/core/store/user/index.ts
  updateCurrentUser = async (data: UserUpdate): Promise<User> => {
    const update = ++this.updatesSent;
    const user = await this.userService.updateCurrentUser(data);
    if (update > this.updateWritten) {
      this.updateWritten = update;
````

````new web/apps/web/core/store/user/index.ts
  updateCurrentUser = (data: UserUpdate): Promise<User> =>
    this.changes(async () => {
      const user = await this.userService.updateCurrentUser(data);
````

````old web/apps/web/core/store/user/index.ts
    }
    return user;
  };
````

````new web/apps/web/core/store/user/index.ts
      return user;
    });
````

````whole web/apps/web/core/store/user/index.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { User } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import type { RootStore } from "@/store/root.store";
import { UserStore } from "@/store/user";

// The account's changes (names, time zone) against a fake nerve that answers each when the test says: a change goes
// out once the one before it is answered or has failed, so nerve applies them in the order they were made and the
// last answer is what nerve holds. The store gets its session's client from RootStore; the token manager it
// imports is not used here.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

const ME = "/api/v0/me";

/** nerve's answer to a change of the time zone: the account in it. */
const accountIn = (user_timezone: string) => ({ first_name: "Ada", user_timezone }) as User;

/** A store, and two changes made one after the other, to Shanghai and then to Berlin: only the first is out. */
async function twoChanges() {
  const nerve = new FakeNerve();
  const store = new UserStore({} as RootStore, nerve.client());
  const first = track(store.updateCurrentUser({ user_timezone: "Asia/Shanghai" }));
  const second = track(store.updateCurrentUser({ user_timezone: "Europe/Berlin" }));
  await until(() => nerve.calls.length === 1, "the first change");
  await vi.advanceTimersByTimeAsync(1_000);
  expect(nerve.calls.map((call) => [call.method, call.path, call.body])).toEqual([
    ["PATCH", ME, { user_timezone: "Asia/Shanghai" }],
  ]);
  return { nerve, store, first, second };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("UserStore.updateCurrentUser", () => {
  it("sends a change once nerve has answered the one before it", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.answer(json(200, accountIn("Asia/Shanghai")));
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.value).toEqual(accountIn("Asia/Shanghai"));
    expect(store.data).toEqual(accountIn("Asia/Shanghai"));
    expect(nerve.calls[1]?.body).toEqual({ user_timezone: "Europe/Berlin" });
    nerve.calls[1]?.answer(json(200, accountIn("Europe/Berlin")));
    await until(() => second.settled, "the second answer");

    expect(second.value).toEqual(accountIn("Europe/Berlin"));
    expect(store.data).toEqual(accountIn("Europe/Berlin"));
  });

  it("sends the next change when nerve refuses the one before it, which leaves the account as it was", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.answer(problem(500, "internal_error"));
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(store.data).toBeUndefined();
    nerve.calls[1]?.answer(json(200, accountIn("Europe/Berlin")));
    await until(() => second.settled, "the second answer");

    expect(store.data).toEqual(accountIn("Europe/Berlin"));
  });

  it("sends the next change when the one before it gets no answer", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.fail();
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.error).toBeInstanceOf(TypeError);
    nerve.calls[1]?.answer(json(200, accountIn("Europe/Berlin")));
    await until(() => second.settled, "the second answer");

    expect(store.data).toEqual(accountIn("Europe/Berlin"));
  });
});
````

````old e2e/stories/identity/a9-preferences.spec.ts
import { expectPreferences } from "../../fixtures/assert/identity";
````

````new e2e/stories/identity/a9-preferences.spec.ts
import type { Request } from "@playwright/test";

import { expectPreferences } from "../../fixtures/assert/identity";
````

````old e2e/stories/identity/a9-preferences.spec.ts
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
  });
````

````new e2e/stories/identity/a9-preferences.spec.ts
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
  });
});

/** Whether request is a change of the profile. */
const isProfileChange = (request: Request) =>
  request.method() === "PATCH" && new URL(request.url()).pathname === "/api/v0/me/profile";

test("A9 (page): two changes of the language in a row reach nerve one after the other; the page, nerve and a reload agree on the last", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const tokens = await registerOnboarded(api, email);
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto("/settings/profile/preferences");
  const language = page.getByRole("button", { name: "English" });
  const html = page.locator("html");
  await expect(language).toBeVisible();
  // The languages of the changes the page sends, in order, and in the order they go on to nerve: the first waits on
  // its way until the test lets it go (M2 Codex review, 6).
  const sent: string[] = [];
  const passed: string[] = [];
  let letFirstGo!: () => void;
  const firstMayGo = new Promise<void>((resolve) => {
    letFirstGo = resolve;
  });
  await page.route("**/api/v0/me/profile", async (route) => {
    if (!isProfileChange(route.request())) {
      await route.fallback();
      return;
    }
    const { language: lng } = route.request().postDataJSON() as { language: string };
    sent.push(lng);
    if (sent.length === 1) await firstMayGo;
    passed.push(lng);
    await route.fallback();
  });

  // 简体中文: the change leaves the page and waits. Until nerve answers it, the page is in English: English again.
  await language.click();
  const first = page.waitForRequest(isProfileChange, { timeout: 10_000 });
  await page.getByRole("option", { name: "简体中文" }).click();
  await first;
  await language.click();
  await page.getByRole("option", { name: "English" }).click();
  // A request the page makes after the second choice, answered: a change sent at that choice would have reached the
  // route before it. The page holds the second change until nerve has answered the first.
  expect(await page.evaluate(async () => (await fetch("/api/v0/instance")).status)).toBe(200);
  expect(sent).toEqual(["zh-CN"]);

  // The first goes on to nerve; once it is answered, the second goes.
  const second = page.waitForResponse(
    (response) =>
      isProfileChange(response.request()) &&
      (response.request().postDataJSON() as { language: string }).language === "en",
    { timeout: 10_000 }
  );
  letFirstGo();
  expect((await second).status()).toBe(200);
  expect(passed).toEqual(["zh-CN", "en"]);

  // The page, nerve and a reload agree on the last choice: English.
  await expect(page.getByText("Language & Time")).toBeVisible();
  await expect(html).toHaveAttribute("lang", "en");
  await expectPreferences(db, email, { theme: "system", language: "en", start_of_the_week: 0 });
  const read = await api.GET("/api/v0/me/profile", { headers: bearer(tokens.access_token) });
  expect(read.data?.language).toBe("en");
  await page.reload();
  await expect(page.getByText("Language & Time")).toBeVisible();
  await expect(language).toBeVisible();
  await expect(html).toHaveAttribute("lang", "en");

  expect(watch.apiRequests.filter((request) => request.startsWith("PATCH "))).toEqual(
    Array(2).fill("PATCH /api/v0/me/profile")
  );
  expect(watch.apiFailures).toEqual([]);
  expect(watch.oldApiRequests).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  // Two loads: the first and the reload.
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING] });
````

- [ ] **Step 2：检查**：
  - `pnpm -C web/apps/web exec vitest run core/store/user/` → 全部通过，其中两个改写的文件共 6 个；
  - `make knip` → 没有问题；
  - `make build`，然后 `pnpm -C e2e exec playwright test a9-pref` → `5 passed`。

- [ ] **Step 3：提交**（`fix(M2/codex-fixes): a store sends its changes one at a time`）

### Task 8: 页面的主题只在 `StoreWrapper` 按会话设置

spec 2.5。没有会话时设"跟随系统"，每个会话第一次取到资料时用资料的主题；`startSession` 写的主题和切换账户弹窗的 `setTheme` 删除（`store-context.test.ts` 随之不再记主题）。A5、A6、A12 的页面版本先让资料的主题是深色，断言页面先是 `dark`，会话结束（续期被拒、退出、停用）之后是 `light`。

- [ ] **Step 1：应用块**（`… apply . 8`）：

````whole web/apps/web/core/lib/wrappers/store-wrapper.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useEffect, useRef } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { useTheme } from "next-themes";
// nerve imports
import { setLanguage } from "@nerve/i18n";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useRouterParams } from "@/hooks/store/use-router-params";
import { useUser, useUserProfile } from "@/hooks/store/user";
import type { IUserStore } from "@/store/user";
// lib
import { useSession } from "@/lib/auth/use-session";

type TStoreWrapper = {
  children: ReactNode;
};

function StoreWrapper(props: TStoreWrapper) {
  const { children } = props;
  // theme
  const { setTheme } = useTheme();
  // router
  const params = useParams();
  // store hooks
  const { setQuery } = useRouterParams();
  const { sidebarCollapsed, toggleSidebar } = useAppTheme();
  const user = useUser();
  const { data: userProfile } = useUserProfile();
  const { status } = useSession();
  // The session whose profile's theme the page has taken: once for each session (a new one has a new user store).
  const themedBy = useRef<IUserStore | undefined>(undefined);

  /**
   * Sidebar collapsed fetching from local storage
   */
  useEffect(() => {
    const localValue = localStorage && localStorage.getItem("app_sidebar_collapsed");
    const localBoolValue = localValue === "true";
    if (localValue && sidebarCollapsed === undefined) toggleSidebar(localBoolValue);
  }, [sidebarCollapsed, setTheme, toggleSidebar]);

  /**
   * The page's theme follows the tab's session (v0 design 7.7), here and nowhere else: without a session (signed out
   * in this tab or another, the account deactivated, a refresh refused), the default; with one, its profile's, once,
   * when the profile first arrives. Later changes are applied by the component that makes them, once nerve holds
   * them (theme-switcher.tsx). Until a new session's profile arrives, the page keeps the theme it shows.
   */
  useEffect(() => {
    if (status === "signed-out") {
      themedBy.current = undefined;
      setTheme("system");
      return;
    }
    if (!userProfile?.theme || themedBy.current === user) return;
    themedBy.current = user;
    setTheme(userProfile.theme);
  }, [status, user, userProfile?.theme, setTheme]);

  /**
   * The page's language is the profile's of the tab's session now, as nerve answered it: the stores never set it.
   * A refused change leaves the profile, and so the page, as it was; a retired session's store, answered late,
   * changes only its own profile, which nothing here reads. Signed out, or before the profile loads, the page keeps
   * the language it has (a new session starts with the default: store-context.tsx).
   */
  useEffect(() => {
    if (userProfile?.language) void setLanguage(userProfile.language);
  }, [userProfile?.language]);

  useEffect(() => {
    if (!params) return;
    setQuery(params);
  }, [params, setQuery]);

  return <>{children}</>;
}

export default observer(StoreWrapper);
````

````old web/apps/web/core/lib/store-context.tsx
 * old session's: nothing they still do reaches the new one. The theme and the language were the account's; the
 * new session shows the defaults until its profile sets them.
````

````new web/apps/web/core/lib/store-context.tsx
 * old session's: nothing they still do reaches the new one. The language was the account's: the new session shows
 * the default until its profile sets it. The theme follows the session in StoreWrapper.
````

````old web/apps/web/core/lib/store-context.tsx
  loginId = next;
  localStorage.setItem("theme", "system");
````

````new web/apps/web/core/lib/store-context.tsx
  loginId = next;
````

````old web/apps/web/core/lib/store-context.test.ts
/** The page's theme and language, each time something set them. */
const page = vi.hoisted(() => ({ themes: [] as string[], languages: [] as string[] }));
vi.stubGlobal("localStorage", {
  getItem: () => null,
  setItem: (key: string, value: string) => {
    if (key === "theme") page.themes.push(value);
  },
  removeItem: () => {},
});
````

````new web/apps/web/core/lib/store-context.test.ts
/** The page's language, each time something set it. The theme follows the session in StoreWrapper. */
const page = vi.hoisted(() => ({ languages: [] as string[] }));
````

````old web/apps/web/core/lib/store-context.test.ts
  vi.resetModules();
  page.themes = [];
````

````new web/apps/web/core/lib/store-context.test.ts
  vi.resetModules();
````

````old web/apps/web/core/lib/store-context.test.ts
    // Each new session starts with the system's theme and the default language, until its profile sets them.
    expect(page.themes).toEqual(["system", "system", "system", "system"]);
````

````new web/apps/web/core/lib/store-context.test.ts
    // Each new session starts with the default language, until its profile sets it.
````

````old web/apps/web/core/components/onboarding/switch-account-modal.tsx

import { useTheme } from "next-themes";
````

````new web/apps/web/core/components/onboarding/switch-account-modal.tsx

````

````old web/apps/web/core/components/onboarding/switch-account-modal.tsx

  const { setTheme } = useTheme();

````

````new web/apps/web/core/components/onboarding/switch-account-modal.tsx

````

````old web/apps/web/core/components/onboarding/switch-account-modal.tsx
      await signOut();
      setTheme("system");
````

````new web/apps/web/core/components/onboarding/switch-account-modal.tsx
      await signOut();
````

````old e2e/stories/identity/a5-refresh-reuse.spec.ts
import { emailFor, recordOf, refresh, register } from "../../fixtures/auth";
````

````new e2e/stories/identity/a5-refresh-reuse.spec.ts
import { bearer, emailFor, recordOf, refresh, register } from "../../fixtures/auth";
````

````old e2e/stories/identity/a5-refresh-reuse.spec.ts
  const page = await signedInPage(await register(api, emailFor(testInfo)));
````

````new e2e/stories/identity/a5-refresh-reuse.spec.ts
  const tokens = await register(api, emailFor(testInfo));
  // A theme the page shows once the profile arrives, and no longer without a session.
  const themed = await api.PATCH("/api/v0/me/profile", {
    body: { theme: "dark" },
    headers: bearer(tokens.access_token),
  });
  expect(themed.response.status).toBe(200);
  const page = await signedInPage(tokens);
````

````old e2e/stories/identity/a5-refresh-reuse.spec.ts
  await expect(page.getByText("Create your profile.")).toBeVisible();
````

````new e2e/stories/identity/a5-refresh-reuse.spec.ts
  await expect(page.getByText("Create your profile.")).toBeVisible();
  const html = page.locator("html");
  await expect(html).toHaveAttribute("data-theme", "dark");
````

````old e2e/stories/identity/a5-refresh-reuse.spec.ts
  // the page goes to the sign-in page, which comes back here.
````

````new e2e/stories/identity/a5-refresh-reuse.spec.ts
  // the page goes to the sign-in page, which comes back here, in the default theme.
````

````old e2e/stories/identity/a5-refresh-reuse.spec.ts
  await expect(page.getByRole("button", { name: "Go to workspace" })).toBeVisible();
````

````new e2e/stories/identity/a5-refresh-reuse.spec.ts
  await expect(page.getByRole("button", { name: "Go to workspace" })).toBeVisible();
  await expect(html).toHaveAttribute("data-theme", "light");
````

````old e2e/stories/identity/a6-sign-out.spec.ts
  const tabA = await signedInPage(await register(api, email));
````

````new e2e/stories/identity/a6-sign-out.spec.ts
  const tokens = await register(api, email);
  // A theme the tabs show once the profile arrives, and no longer without a session.
  const themed = await api.PATCH("/api/v0/me/profile", {
    body: { theme: "dark" },
    headers: bearer(tokens.access_token),
  });
  expect(themed.response.status).toBe(200);
  const tabA = await signedInPage(tokens);
````

````old e2e/stories/identity/a6-sign-out.spec.ts
  await openOnboarding(tabB, email);
````

````new e2e/stories/identity/a6-sign-out.spec.ts
  await openOnboarding(tabB, email);
  await expect(tabA.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(tabB.locator("html")).toHaveAttribute("data-theme", "dark");
````

````old e2e/stories/identity/a6-sign-out.spec.ts
  // Both tabs are on the sign-in page, which comes back to where they were; the record is gone.
````

````new e2e/stories/identity/a6-sign-out.spec.ts
  // Both tabs are on the sign-in page, which comes back to where they were, in the default theme; the record is gone.
````

````old e2e/stories/identity/a6-sign-out.spec.ts
      await expect(tab.getByRole("button", { name: "Go to workspace" })).toBeVisible();
````

````new e2e/stories/identity/a6-sign-out.spec.ts
      await expect(tab.getByRole("button", { name: "Go to workspace" })).toBeVisible();
      await expect(tab.locator("html")).toHaveAttribute("data-theme", "light");
````

````old e2e/stories/identity/a12-deactivate.spec.ts
  const pat = await createPAT(api, tokens.access_token);
````

````new e2e/stories/identity/a12-deactivate.spec.ts
  const pat = await createPAT(api, tokens.access_token);
  // A theme the page shows once the profile arrives, and no longer without a session.
  const themed = await api.PATCH("/api/v0/me/profile", {
    body: { theme: "dark" },
    headers: bearer(tokens.access_token),
  });
  expect(themed.response.status).toBe(200);
````

````old e2e/stories/identity/a12-deactivate.spec.ts
  const watch = await watchPage(page);
  await page.goto("/settings/profile/general");
  const before = await accountOf(db, email);
````

````new e2e/stories/identity/a12-deactivate.spec.ts
  const watch = await watchPage(page);
  await page.goto("/settings/profile/general");
  const html = page.locator("html");
  await expect(html).toHaveAttribute("data-theme", "dark");
  const before = await accountOf(db, email);
````

````old e2e/stories/identity/a12-deactivate.spec.ts
  // The page is back at sign-in, which comes back to the general page, and the browser keeps no session.
````

````new e2e/stories/identity/a12-deactivate.spec.ts
  // The page is back at sign-in, which comes back to the general page, in the default theme; the browser keeps no
  // session.
````

````old e2e/stories/identity/a12-deactivate.spec.ts
  await expect(page.getByText("Your account is deactivated.")).toBeVisible();
````

````new e2e/stories/identity/a12-deactivate.spec.ts
  await expect(page.getByText("Your account is deactivated.")).toBeVisible();
  await expect(html).toHaveAttribute("data-theme", "light");
````

- [ ] **Step 2：检查**：
  - `pnpm -C web/apps/web exec vitest run core/lib/store-context.test.ts` → 6 个通过；
  - `make lint-web` → `keywords: 60 rules, 3 exceptions, no hits.`，`Tasks: 54 successful, 54 total`；
  - `make build`，然后 `pnpm -C e2e exec playwright test a5-refresh a6-sign-out a12-deact` → `9 passed`。

- [ ] **Step 3：提交**（`fix(M2/codex-fixes): the page's theme follows the session in StoreWrapper only`）

### Task 9: M2 设计记下这些修复和 Codex 的评审（17.3）

spec 2.12。3.10（`AllowAll`）、3.11（一种读法、不变式、`Content-Type`、整程序测试第 8、9 项、字段码）、6.1（分开的角色和权限文件）、7.1（`await` 之后重读、退出）、7.5（两个 store 的一格）、7.7（修改的顺序、会话的主题）、8.5（恢复步骤）、8.7（两行）、11.2（只用标准库）、新的 17.3。

- [ ] **Step 1：应用块**（`… apply . 9`）：

````old docs/v0/M2-auth/M2-design.md
  - `platform/ratelimit` 提供一次检查多个键的 `AllowAll(checks…)`：在同一把锁下先看每个桶是否都有余额，都有才一起扣；有一个没有，就都不扣，返回最长的等待时间。另有 `Reserve(check)`：扣一个单位，同时返回一个只能用一次的退回函数（3.6 的失败闸门用它）。spike：`login_ip_email` 拒绝 10 次之后，`login_ip` 的余额不变；失败闸门在 50 个并发请求下只放过额度内的 3 个。
````

````new docs/v0/M2-auth/M2-design.md
  - `platform/ratelimit` 提供一次检查多个键的 `AllowAll(checks…)`：在同一把锁下先看每个桶是否都有余额，都有才一起扣；有一个没有，就都不扣，返回最长的等待时间。同一个桶和键在一次调用里出现两次是编程错误，`AllowAll` panic：它会看两次同一份余额、扣两个单位，余额可能变成负数（codex-fixes）。另有 `Reserve(check)`：扣一个单位，同时返回一个只能用一次的退回函数（3.6 的失败闸门用它）。spike：`login_ip_email` 拒绝 10 次之后，`login_ip` 的余额不变；失败闸门在 50 个并发请求下只放过额度内的 3 个。
````

````old docs/v0/M2-auth/M2-design.md
      - spike（仓库锁定的 Go 1.27.1）：16 种写法上检查器与解码器的结论全部相同。`date-time`：带 `Z` 和带时区偏移的 RFC 3339 接受；空格分隔、只有日期、数字、布尔、任意字符串拒绝；`Z` 写成 JSON 转义时同样接受。`uuid`：标准写法、大写、无连字符、带花括号、带 `urn:uuid:` 前缀、含 JSON 转义的都接受，非法字符串和数字拒绝。标准库的解析比 RFC 9562 的标准写法宽松，边界与解码器一致，领域拿到的是同一个值。
    - 以上一律 400 `bad_request`，`errors[{field, code}]`，一次收集全部问题，`field` 是 JSON 路径（`onboarding_step.profile_completed`、`tags[1].name`）。
````

````new docs/v0/M2-auth/M2-design.md
      - spike（仓库锁定的 Go 1.27.1）：16 种写法上检查器与解码器的结论全部相同。`date-time`：带 `Z` 和带时区偏移的 RFC 3339 接受；空格分隔、只有日期、数字、布尔、任意字符串拒绝；`Z` 写成 JSON 转义时同样接受。`uuid`：标准写法、大写、无连字符、带花括号、带 `urn:uuid:` 前缀、含 JSON 转义的都接受，非法字符串和数字拒绝。标准库的解析比 RFC 9562 的标准写法宽松，边界与解码器一致，领域拿到的是同一个值。
    - **一份请求体只有一种读法**（Codex 对 M2 的评审 Critical 1，codex-fixes）：同一个对象里同名的成员出现两次，拒绝，字段码 `duplicate`，`field` 是这个成员的路径，一个名字只报一次；名字按解码之后比较，`"a"` 和它的 `\u` 转义是同一个名字。字符串（成员名或值）不是合法的 Unicode，拒绝，字段码 `invalid_format`：不是 UTF-8 的字节，或者不成对的代理项转义。这是 I-JSON（RFC 7493）的两条要求。这两项先于其余各项检查：有这两种问题的请求体只得到它们，结构在请求体只有一种读法之后才检查。
      - **为什么**：结构检查把对象读成 map，同名的成员以后一个为准；生成的 handler 用 `encoding/json` 把同一个请求体解码进结构体，同名的对象合并，字段名先精确匹配、再不区分大小写地匹配。Codex 的复现：`PATCH /me/profile` 的 `{"onboarding_step":{"WORKSPACE_INVITE":true}}` 得到 400 `not_allowed`，后面再加一个 `"onboarding_step":{}` 就得到 200，`workspace_invite` 写成了 `true`。不是 UTF-8 的字节和落单的代理项被解码器换成 U+FFFD，领域拿到的不是客户端写的文字。
      - **不变式：结构检查接受的请求体，解码器读不出结构检查没看过的东西。**①每个对象里的名字各不相同（按解码之后比较），map 和结构体读到的是同一组成员，没有可合并的；②每个对象 schema 都是 `additionalProperties: false`（`apitest` 的 `closedObject` 规则），接受的名字恰好是声明的名字，解码器按它们精确匹配，不区分大小写的退路用不上：只差大小写的名字是未声明的字段，400 `not_allowed`；以后的开放 map 由生成的代码按键原样读取；③转义由两边同一个 `encoding/json` 解码，结构检查比较、检查的正是解码之后的名字；④字符串都是合法的 Unicode，解码出的文字就是客户端写的；⑤请求体是一个 JSON 值，前后只有 JSON 空白（`json.Valid`，原有的一条）。
      - 做法：`bodyshape/ambiguity.go` 在按表检查之前把请求体扫描一遍，只用标准库。第四个整程序测试的第 8、9 项让每个带请求体的操作都拒绝重复的成员和不是 UTF-8 的字节。
    - 以上一律 400 `bad_request`，`errors[{field, code}]`，一次收集全部问题，`field` 是 JSON 路径（`onboarding_step.profile_completed`、`tags[1].name`）。
    - **`Content-Type` 照旧宽松**：`text/plain` 的请求体同样检查、同样解码。Nerve 没有 Cookie 认证，非公开操作都要 `Authorization` 头，跨站的"简单请求"带不上它，所以没有 CSRF 的风险（codex-fixes 核对后保留）。
````

````old docs/v0/M2-auth/M2-design.md
    7. 同一个请求里同时有未知字段、缺少的必填字段和写错的格式：一次返回全部问题。
````

````new docs/v0/M2-auth/M2-design.md
    7. 同一个请求里同时有未知字段、缺少的必填字段和写错的格式：一次返回全部问题。
    8. 第一个字段出现两次；嵌套对象的第一个字段在嵌套对象里出现两次（有嵌套对象时）：400 `duplicate`（codex-fixes）；
    9. 第一个字符串字段的值是不合 UTF-8 的字节：400 `invalid_format`（codex-fixes）。M2 的 8 个带请求体的操作都有第 8、9 项。它们不并进第 7 项：能有两种读法的请求体只得到这两种问题。
````

````old docs/v0/M2-auth/M2-design.md
  - 字段码是一个封闭的小集合：`required`、`invalid_format`、`too_short`、`too_long`、`out_of_range`、`not_allowed`、`weak_password`、`common_password`、`must_be_future`、`contains_url`。
````

````new docs/v0/M2-auth/M2-design.md
  - 字段码是一个封闭的小集合：`required`、`invalid_format`、`too_short`、`too_long`、`out_of_range`、`not_allowed`、`duplicate`（codex-fixes）、`weak_password`、`common_password`、`must_be_future`、`contains_url`。
````

````old docs/v0/M2-auth/M2-design.md
- **只读的迁移角色**：生产环境用单独的数据库角色执行迁移时，运行服务的角色需要 `goose_db_version` 的 SELECT 权限（M0-P2 交接 7），写进 README 的部署说明。
````

````new docs/v0/M2-auth/M2-design.md
- **分开的迁移角色和服务角色**：生产环境用表的所有者执行迁移、服务用另一个角色时，服务的角色需要的权限全部写在 `deploy/runtime-grants.sql`，授权给组角色 `nerve_runtime`，每次迁移之后由所有者执行：业务表的读写；River 的表和序列；`river_job` 的 `MAINTAIN`（River 每天的 `REINDEX INDEX CONCURRENTLY`，PostgreSQL 17 起）；`goose_db_version` 的 SELECT（`/readyz`，M0-P2 交接 7）。README 的部署说明指向它。少了其中一项时 `/readyz` 仍是 200，River 的任务却因 42501 失败，只记在日志里（Codex 对 M2 的评审 Important 2，codex-fixes）。
  - 文件逐个列出表和序列，不用 `ON ALL TABLES IN SCHEMA` 加 `ALTER DEFAULT PRIVILEGES`：默认权限只对执行它的那个角色以后建的对象生效，换了执行迁移的角色就悄悄失效，要到部署之后才发现；它还会把 `goose_db_version` 的写权限、`TRUNCATE`、`REFERENCES`、`TRIGGER` 一并给出。
  - 列表不会漏：`server/internal/bootstrap/runtime_role_test.go` 在库里任何一张表或一个序列的权限与文件的规则不符时失败（新的迁移加了表而文件没有跟上，就是这样），并用恰好这些权限运行 nerve：就绪、River 的清理任务完成、River 的每个索引都能 `REINDEX INDEX CONCURRENTLY`。
  - 服务的角色不是表的所有者，删不掉索引：River 的重建被停机打断、留下 `*_ccnew` 索引时，由所有者删除（README"部署"）。
````

````old docs/v0/M2-auth/M2-design.md
  - **每次写 `nerve.auth` 都在同一把锁（或租约）下**（控制者复核 R4）：登录、注册写入新记录，续期换令牌，退出删除记录，都先拿续期用的那把锁。否则登录写入新记录的同时，另一个标签页的续期可能把旧会话的令牌写回去，盖掉新记录。
````

````new docs/v0/M2-auth/M2-design.md
  - **每次写 `nerve.auth` 都在同一把锁（或租约）下**（控制者复核 R4）：登录、注册写入新记录，续期换令牌，退出删除记录，都先拿续期用的那把锁。否则登录写入新记录的同时，另一个标签页的续期可能把旧会话的令牌写回去，盖掉新记录。
  - **每个 `await` 之后先重读记录**（Codex 对 M2 的评审 Important 1，codex-fixes）：一个请求或等锁的 `await` 回来之后，写或删 `nerve.auth`、改本标签页的状态之前，都再读一次记录；它已不是这次操作所属的会话（`login_id` 变了，或者记录没了），就不写、不删，跟随记录（见"其他标签页"）。租约不是原子的：持有者的请求超过租期时，另一个标签页可以拿到租约登录。逐条核对过：续期的写回（"续期"）；退出的删除（"退出"）；启动时第一次续期失败之后进入"会话暂不可用"之前（`navigator.locks` 把失败交回来是在之后的任务里，另一个标签页的登录可能先到）；登录、注册和结束会话在锁内没有 `await`，读和写之间不会插进别的标签页。
````

````old docs/v0/M2-auth/M2-design.md
  - 尽力而为：失败也清本地。然后结束会话，回到登录页。
````

````new docs/v0/M2-auth/M2-design.md
  - 尽力而为：失败也清本地。`logout` 回来之后在锁内重读记录：仍是这次退出的会话，才删除记录、结束会话、回到登录页；已是另一个会话（租约在请求期间过期，另一个标签页登录了），就不删，跟随它。
````

````old docs/v0/M2-auth/M2-design.md
| `store/user/index.ts`（`UserStore`） | `IUser`；登录时先取账户，再并行取资料、设置、工作区三样 | `data: User`；没有 `nerve.auth` 时不取数（7.1）；`fetchCurrentUser` 只取 `/me` 和 `/me/profile`；`signIn`、`signUp`、`signOut`、`deactivate` 通过令牌管理器；删除死成员（`reset`、`isAuthenticated`、`error` 等） |
| `store/user/profile.store.ts` | `TUserProfile`；`updateUserProfile` 吞掉错误（`:138-148`），调用方的错误提示从不出现 | `data: Profile`；失败时抛出。`finishUserOnboarding` 合并为一次 `PATCH /me/profile`（部分的 `onboarding_step` + `is_onboarded` + `last_workspace_id`）；`updateTourCompleted`、`updateUserTheme` 同样用这一个接口 |
````

````new docs/v0/M2-auth/M2-design.md
| `store/user/index.ts`（`UserStore`） | `IUser`；登录时先取账户，再并行取资料、设置、工作区三样 | `data: User`；没有 `nerve.auth` 时不取数（7.1）；`fetchCurrentUser` 只取 `/me` 和 `/me/profile`；`signIn`、`signUp`、`signOut`、`deactivate` 通过令牌管理器；删除死成员（`reset`、`isAuthenticated`、`error` 等）；`updateCurrentUser` 一个接一个发出（7.7，codex-fixes） |
| `store/user/profile.store.ts` | `TUserProfile`；`updateUserProfile` 吞掉错误（`:138-148`），调用方的错误提示从不出现 | `data: Profile`；失败时抛出。`finishUserOnboarding` 合并为一次 `PATCH /me/profile`（部分的 `onboarding_step` + `is_onboarded` + `last_workspace_id`）；`updateTourCompleted`、`updateUserTheme` 同样用这一个接口；修改一个接一个发出（7.7，codex-fixes） |
````

````old docs/v0/M2-auth/M2-design.md
  - 列表、撤销。列表显示创建时间和最后使用时间，便于认出不认识的令牌（8.5）。
````

````new docs/v0/M2-auth/M2-design.md
  - 列表、撤销。列表显示创建时间和最后使用时间，便于认出不认识的令牌（8.5）。
- **修改的顺序**（Codex 对 M2 的评审第 6 节，codex-fixes）：`ProfileStore` 和 `UserStore` 的修改经 `core/lib/one-at-a-time.ts` 的队列发出，前一个修改有了应答或失败之后，下一个才发出。于是 nerve 按做出的顺序应用它们，最后一个应答就是 nerve 保存的值；失败也放行队列。P5 原来的做法（按发出的顺序给修改编号，丢弃较旧的应答）假定 nerve 按发出的顺序处理，Codex 的页面实验推翻了它：先发出的修改后到，页面显示 English，nerve 保存的是简体中文。它随之删除。
  - preferences、general、新手引导、主题切换和命令面板的主题命令都经这两个 store 修改资料和账户。
  - 只在一个标签页之内成立；多个标签页或多个客户端之间以后写入的为准（v0 不做乐观锁，总体设计 3.6）。
  - PAT 的创建和撤销不依赖 nerve 的处理顺序：撤销要用创建返回的 id，两者本来有先后；列表"取最新"靠提交先于确认，不靠请求的顺序。
- **会话的主题**（Codex 对 M2 的评审 Minor 1，codex-fixes）：没有会话时（本标签页或其他标签页退出、账户停用、续期被拒）页面用默认的"跟随系统"；有会话时，这个会话第一次取到资料时用资料的主题。两者都只在 `StoreWrapper` 设置（总体设计 7.7）；`startSession` 不再写 localStorage 的主题，切换账户弹窗的退出不再自己设主题。
````

````old docs/v0/M2-auth/M2-design.md
- **怀疑泄露时的恢复步骤**（README 的安全说明，8.7）：
  1. 在个人设置的 api-tokens 页或安全页查看 PAT 列表，按创建时间和最后使用时间认出不认识的令牌，逐个撤销；
  2. 或者请服务器管理员执行 `nerve users reset-password`：撤销全部会话和全部 PAT（3.5）。
````

````new docs/v0/M2-auth/M2-design.md
- **怀疑泄露时的恢复步骤**（README 的安全说明，8.7；顺序按 Codex 对 M2 的评审第 7 节收紧，codex-fixes）：
  1. 先在安全页修改密码：账户的其他会话全部撤销（3.5），被盗的刷新令牌和由它换来的访问令牌在下一个请求就失效，对方不能再经会话创建 PAT；
  2. 再在 api-tokens 页或安全页按创建时间和最后使用时间认出不认识的 PAT，全部撤销，然后重新核对列表：PAT 能创建 PAT，撤销期间可能又出现新的，列表里不再有不认识的令牌才算完成；
  3. 只撤销 PAT、不先改密码是不够的：对方仍有会话，随时能再建一个 PAT。对方不断建新令牌、自助撤销跟不上，或者用户已无法登录时，请服务器管理员执行 `nerve users reset-password`：一个事务里撤销全部会话和全部 PAT（3.5）。
````

````old docs/v0/M2-auth/M2-design.md
| 迁移角色的权限：用单独的角色执行迁移时，运行服务的角色要能读 `goose_db_version`（6.1）。第一批迁移在 P1，所以放在 P1 | P1 |
````

````new docs/v0/M2-auth/M2-design.md
| 迁移角色的权限：用单独的角色执行迁移时，运行服务的角色要能读 `goose_db_version`（6.1）。第一批迁移在 P1，所以放在 P1；codex-fixes 改为指向列出全部权限的 `deploy/runtime-grants.sql` | P1、codex-fixes |
````

````old docs/v0/M2-auth/M2-design.md
| 刷新令牌泄露时可能派生 PAT，以及恢复步骤（8.5）：恢复要用 `reset-password` | P3b |
````

````new docs/v0/M2-auth/M2-design.md
| 刷新令牌泄露时可能派生 PAT，以及恢复步骤（8.5）：自助先改密码、再撤销 PAT 并重新核对，管理员用 `reset-password` | P3b、codex-fixes |
````

````old docs/v0/M2-auth/M2-design.md
  1. **请求体的结构校验**（3.11）：新增一个平台子包 `httpserver/bodyshape` 和一个构建时的生成器 `tools/bodyshapegen`。生成器在工具模块，用 oapi-codegen 自己的加载器和 `type-mapping` 读契约，不链接进 nerve；运行时只用标准库和 oapi-codegen 的运行时类型。
````

````new docs/v0/M2-auth/M2-design.md
  1. **请求体的结构校验**（3.11）：新增一个平台子包 `httpserver/bodyshape` 和一个构建时的生成器 `tools/bodyshapegen`。生成器在工具模块，用 oapi-codegen 自己的加载器和 `type-mapping` 读契约，不链接进 nerve；运行时只用标准库（核验 F1 之后，格式检查把原始值 `json.Unmarshal` 进映射的类型，不再依赖 oapi-codegen 的运行时类型，3.11）。
````

````old docs/v0/M2-auth/M2-design.md
- 标准库的 `uuid.UUID` 除了标准写法，还接受无连字符、带花括号和带 `urn:uuid:` 前缀的写法；边界的检查与解码器一致，照样接受（3.11）。

````

````new docs/v0/M2-auth/M2-design.md
- 标准库的 `uuid.UUID` 除了标准写法，还接受无连字符、带花括号和带 `urn:uuid:` 前缀的写法；边界的检查与解码器一致，照样接受（3.11）。

### 17.3 实现之后的 Codex 对抗性评审（codex-fixes 落实）
M2 合并之后（`9ef4a4a`），Codex 对全部 M2 做了对抗性评审（[报告](reviews/M2-codex-adversarial-review.md)）：Critical 1、Important 2、Minor 3，另有对已知事项（报告第 6 节）和负责人裁定（第 7 节）的意见。负责人的指示：仔细处理报告，干净地完成 M2，不开始 M3。控制者逐条裁定，codex-fixes 落实（[spec](specs/codex-fixes.md)、[plan](plans/codex-fixes.md)）；每一条的先红后绿、变异表和门禁在 spec 的附录。

| 编号 | 问题 | 裁定 | 落点 |
|---|---|---|---|
| Critical 1 | 重复的 JSON 键绕过请求体的结构检查：结构检查按 map 读（后一个为准），生成的 handler 按结构体解码（合并，字段名不区分大小写） | 在平台的边界修：同一个对象里重复的成员名（按解码之后比较）和不是合法 Unicode 的字符串，400 `bad_request`；新的字段码 `duplicate`；证明结构检查接受的请求体，解码器读不出别的 | 3.11 的"一份请求体只有一种读法"和不变式、字段码、第四个整程序测试的第 8、9 项；`bodyshape/ambiguity.go`；A10 的接口版本发 Codex 的两个请求 |
| Important 1 | 旧标签页的退出在租约过期之后回来，删掉另一个标签页的新登录 | 修这一类：锁内每个 `await` 之后先重读记录 | 7.1 的"每个 `await` 之后先重读记录"和"退出"；核对出同类的第二处，启动时第一次续期失败之后的"会话暂不可用"；总体设计 4.3 |
| Important 2 | 迁移和服务分用两个角色时 README 只写了 `goose_db_version`：River 因 42501 失败，`/readyz` 仍是 200；River 的索引重建要 `MAINTAIN` | 权限写进一个文件，逐个列出，由测试守住完整 | 6.1；8.7；`deploy/runtime-grants.sql`；`server/internal/bootstrap/runtime_role_test.go`；README"部署"；M4、M8 的交接 |
| Minor 1 | 自助停用之后当前标签页保留旧账户的主题，直到刷新 | 主题只在 `StoreWrapper` 按会话设置，删除退出按钮上的补丁 | 7.7 的"会话的主题"；总体设计 7.7；A5、A6、A12 的页面版本 |
| Minor 2 | A8–A10 的页面版本和接口版本共用查询，没有共用业务断言 | 抽出三个带参数的小断言，不建通用框架 | `e2e/fixtures/assert/identity.ts` 的 `expectAccountChanged`、`expectPreferences`、`expectProfileStepTaken` |
| Minor 3 | 11.2 说 `bodyshape` 用 oapi-codegen 的运行时类型；收尾 spec 和收尾 review 说 `identity.yaml` 有 15 个操作、每个约 40 行 | 改正 | 11.2；收尾 spec 附录 B.5 第 5 条、收尾 review 第 7 节第 5 条：13 个操作，每个约 48 行 |
| 第 6 节 | P5 假定 nerve 按发出的顺序处理资料和账户的修改，Codex 的页面实验推翻了它 | 一个 store 的修改一个接一个发出，失败也放行；删除"丢弃较旧的应答" | 7.5；7.7 的"修改的顺序"；总体设计 7.7；A9 的页面版本 |
| 第 7 节 | README 和 8.5 把"逐个撤销 PAT"与 `reset-password` 并列，自助的恢复不完整 | 自助先改密码、再撤销 PAT 并重新核对；管理员用 `reset-password` | 8.5；8.7；README"部署" |
| 第 6 节（P2） | `AllowAll` 里同一个桶和键出现两次会扣两次 | 作为编程错误拒绝（panic） | 3.10；`platform/ratelimit` |
| 第 6 节 | `DateDropdown` 的键盘；时区按钮的 Tab 顺序 | 仍按期限推迟：它们是共用组件，其他调用方在 M2 到不了的页面上；交接已有关闭条件 | 不变：M4 交接第 10 节、M3 交接第 14 节 |
| 第 6 节其余 | 已知事项的其余判定 | 同意，不改 | — |

````

- [ ] **Step 2：检查**：`node tools/keywords.mjs` → `keywords: 60 rules, 3 exceptions, no hits.`；17.3 的三个相对链接（`reviews/M2-codex-adversarial-review.md`、`specs/codex-fixes.md`、`plans/codex-fixes.md`）在 `docs/v0/M2-auth/` 下存在。

- [ ] **Step 3：提交**（`docs(M2/codex-fixes): the M2 design records the fixes and Codex's review of M2 (17.3)`）

### Task 10: 上级文档、README 的恢复说明、M4 和 M8 的交接、收尾的数字

spec 2.7、2.9、2.12。总体设计 3.5、4.3、7.7；差异清单第三节"请求体"一行；README"令牌泄露后的恢复"；M4 交接第 6 节、M8 交接第 3 节；收尾 spec 附录 B.5 第 5 条和收尾 review 第 7 节第 5 条。

- [ ] **Step 1：应用块**（`… apply . 10`）：

````old docs/v0/v0-design.md
- **结构在接口边界，取值在领域**（M2 设计 3.11）：请求体不是合法 JSON、有未声明的字段、不可为空的字段传了 `null`、缺少必填字段、生成为 Go 类型的格式（`date-time`、`uuid`）写错，一律 400 `bad_request`，`errors` 一次列出全部问题；长度、其余格式、枚举、取值范围和跨字段的规则由领域层校验，一次返回 422 `validation_failed`。
- `errors` 的每一项是 `{field, code, message}`：`field` 是 JSON 路径，`code` 取自一个封闭的集合（`required`、`invalid_format`、`too_short`、`too_long`、`out_of_range`、`not_allowed`、`weak_password`、`common_password`、`must_be_future`、`contains_url`），前端按 `code` 显示文案。
````

````new docs/v0/v0-design.md
- **结构在接口边界，取值在领域**（M2 设计 3.11）：请求体不是合法 JSON、能有两种读法（同一个对象里同名的成员出现两次，名字按解码之后比较；字符串不是合法的 Unicode）、有未声明的字段、不可为空的字段传了 `null`、缺少必填字段、生成为 Go 类型的格式（`date-time`、`uuid`）写错，一律 400 `bad_request`，`errors` 一次列出全部问题；长度、其余格式、枚举、取值范围和跨字段的规则由领域层校验，一次返回 422 `validation_failed`。
- `errors` 的每一项是 `{field, code, message}`：`field` 是 JSON 路径，`code` 取自一个封闭的集合（`required`、`invalid_format`、`too_short`、`too_long`、`out_of_range`、`not_allowed`、`duplicate`、`weak_password`、`common_password`、`must_be_future`、`contains_url`），前端按 `code` 显示文案。
````

````old docs/v0/v0-design.md
- **多标签页续期**：用 `navigator.locks` 保证同一时间只有一个标签页在续期；没有它时（用 HTTP 部署，浏览器不在安全上下文中）用 localStorage 的租约。登录、注册、续期、退出、结束会话写 `nerve.auth` 都在这同一把锁或租约下；续期在锁内读出记录，写回之前再核对一次 `login_id`，变了就丢弃这次的结果，跟随新的记录（M2 设计 7.1）。
````

````new docs/v0/v0-design.md
- **多标签页续期**：用 `navigator.locks` 保证同一时间只有一个标签页在续期；没有它时（用 HTTP 部署，浏览器不在安全上下文中）用 localStorage 的租约。登录、注册、续期、退出、结束会话写 `nerve.auth` 都在这同一把锁或租约下；锁内每个 `await`（请求、等锁）回来之后，写、删记录或改标签页的状态之前都再读一次记录，它已不是这次操作的会话（`login_id` 变了，或者记录没了）就不写、不删，跟随新的记录：续期的写回、退出的删除、启动时进入"会话暂不可用"都是这样（M2 设计 7.1）。
````

````old docs/v0/v0-design.md
- **页面级的状态跟随当前的会话**：界面语言由 `StoreWrapper`（`core/lib/wrappers/store-wrapper.tsx`）按当前一代的资料设置，store 不设。主题在每个会话第一次取到资料时由 `StoreWrapper` 按资料设置一次，之后由改它的组件在 nerve 应答成功之后设置，设置之前先核对标签页仍在发出修改时的会话（`core/components/appearance/theme-switcher.tsx` 的 `inSession()`，M2/P5 spec 2.7）。store 不直接改全局状态；组件在一个 `await` 之后改页面级的状态时，都先核对会话没变。这样旧一代迟到的应答就改不到新会话的页面（M2/P5 spec 第 3 节第 16 条）。
````

````new docs/v0/v0-design.md
- **页面级的状态跟随当前的会话**：界面语言由 `StoreWrapper`（`core/lib/wrappers/store-wrapper.tsx`）按当前一代的资料设置，store 不设。主题同样由 `StoreWrapper` 按会话设置，别处不设：没有会话时（本标签页或别的标签页退出、账户停用、续期被拒）是默认的"跟随系统"，每个会话第一次取到资料时按资料设置一次；之后由改它的组件在 nerve 应答成功之后设置，设置之前先核对标签页仍在发出修改时的会话（`core/components/appearance/theme-switcher.tsx` 的 `inSession()`，M2/P5 spec 2.7）。store 不直接改全局状态；组件在一个 `await` 之后改页面级的状态时，都先核对会话没变。这样旧一代迟到的应答就改不到新会话的页面（M2/P5 spec 第 3 节第 16 条）。
- **一个 store 的修改一个接一个发出**：前一个修改有了应答或失败之后才发出下一个（`core/lib/one-at-a-time.ts`），nerve 按做出的顺序应用，最后的应答就是 nerve 保存的值。不要用"丢弃较旧的应答"代替它：丢掉的只是应答，nerve 仍可能按另一种顺序写入（M2 设计 7.7，Codex 对 M2 的评审第 6 节）。这只在一个标签页之内成立。
````

````old docs/v0/plane-diff.md
| 请求体 | DRF 的序列化器忽略未知字段 | 按契约拒绝未知字段、不合法的 `null` 和缺少的必填字段：400 `bad_request`，一次列出全部问题（M2 设计 3.11） |
````

````new docs/v0/plane-diff.md
| 请求体 | DRF 的序列化器忽略未知字段；重复的键以后一个为准（Python 的 `json`） | 按契约拒绝未知字段、不合法的 `null` 和缺少的必填字段，也拒绝能有两种读法的请求体（同一个对象里重复的成员名，字符串不是合法的 Unicode）：400 `bad_request`，一次列出全部问题（M2 设计 3.11） |
````

````old README.md
- **令牌泄露后的恢复**：刷新令牌存在浏览器的 localStorage 里，页面上的 XSS 能读出它，换来访问令牌后创建一个永不过期的 PAT（创建 PAT 不要求输入密码）。这个 PAT 不受退出、修改密码和会话 30 天期限的影响，还能再创建 PAT，所以泄露的影响不以 30 天为限（M2 设计 8.5）。怀疑泄露时：查看账户的 PAT 列表（个人设置的 api-tokens 页或 security 页，或 `GET /api/v0/me/api-tokens`），按创建时间和最后使用时间认出不认识的令牌，逐个撤销；或者由服务器管理员执行 `nerve users reset-password --email <邮箱>`，它结束该账户的全部会话、撤销全部 PAT。
````

````new README.md
- **令牌泄露后的恢复**：刷新令牌存在浏览器的 localStorage 里，页面上的 XSS 能读出它，换来访问令牌后创建一个永不过期的 PAT（创建 PAT 不要求输入密码）。这个 PAT 不受退出、修改密码和会话 30 天期限的影响，还能再创建 PAT，所以泄露的影响不以 30 天为限（M2 设计 8.5）。怀疑泄露时，按这个顺序做：
  1. **先修改密码**（个人设置的 security 页）：账户的其他会话全部结束，被盗的刷新令牌和由它换来的访问令牌在下一个请求就失效，对方不能再用会话创建 PAT；当前页面的会话保留。
  2. **再撤销不认识的 PAT**：在 api-tokens 页或 security 页的列表中（或 `GET /api/v0/me/api-tokens`），按创建时间和最后使用时间认出不认识的令牌，全部撤销，拿不准的也撤销；然后重新打开列表核对。PAT 能创建 PAT：撤销的同时，对方可能用还没撤销的 PAT 建了新的，列表里不再出现不认识的令牌才算完成。

  只撤销 PAT、不先改密码是不够的：对方还握着会话，随时能再建一个 PAT。对方不断建新令牌、撤销跟不上，或者用户已无法登录时，请服务器管理员执行 `nerve users reset-password --email <邮箱>`：它在一个事务里设新密码、结束该账户的全部会话、撤销全部 PAT。
````

````old docs/v0/M4-issue-core/handoffs/M2-closeout.md
- **关闭条件**：漏注册 worker 的变异让测试失败；事件订阅者的写法写进 M4 设计。
````

````new docs/v0/M4-issue-core/handoffs/M2-closeout.md
- **服务角色的权限**（M2 codex-fixes，M2 设计 6.1）：M4 的迁移加了表，或者 River 升级（总体设计 5.2）加了表或序列，同一个提交把授权加进 `deploy/runtime-grants.sql`。`server/internal/bootstrap/runtime_role_test.go` 的 `TestTheGrantsFileCoversEveryTableAndSequence` 在少了时失败；`TestTheRuntimeRoleServesWithTheGrantsFile` 用恰好这些权限运行 nerve（就绪、River 的清理任务、River 的索引重建），M4 的第一个业务定时任务在它读写的表上照样要有权限。
- **关闭条件**：漏注册 worker 的变异让测试失败；事件订阅者的写法写进 M4 设计；`deploy/runtime-grants.sql` 覆盖 M4 的表，上面两个测试通过。
````

````old docs/v0/M8-open-release/handoffs/M2-closeout.md
- **关闭条件**：镜像的环境里有 `NERVE_ENV=prod`；部署文件的停止宽限期不短于停机的最坏时间（或写明调小了哪些期限）；数据库重启的核对结果写进 M8 的 review。
````

````new docs/v0/M8-open-release/handoffs/M2-closeout.md
- **迁移和服务分用两个数据库角色**（M2 codex-fixes，M2 设计 6.1）：部署文件这样做时，每次 `nerve migrate up` 之后以表的所有者执行 `deploy/runtime-grants.sql`（README"部署"的"迁移"一条）。部署核对在这样的环境里看：`/readyz` 200；清理任务的日志，或 `river_job` 里 `identity.cleanup_expired_sessions` 的 `completed` 行；日志里没有 `permission denied`。自动化的证明是 `server/internal/bootstrap/runtime_role_test.go`。
- **关闭条件**：镜像的环境里有 `NERVE_ENV=prod`；部署文件的停止宽限期不短于停机的最坏时间（或写明调小了哪些期限）；数据库重启的核对结果写进 M8 的 review；分用两个角色的部署按 `deploy/runtime-grants.sql` 授权，核对结果写进 M8 的 review。
````

````old docs/v0/M2-auth/specs/closeout.md
5. **接口描述比估的短。** `identity.yaml` 623 行、15 个操作，约每个操作 40 行（含 schema）；M3 起按操作数估。
````

````new docs/v0/M2-auth/specs/closeout.md
5. **接口描述比估的短。** `identity.yaml` 623 行、13 个操作（另外 2 个操作在 `instance.yaml`），约每个操作 48 行（含 schema）；M3 起按操作数估。（codex-fixes 更正：原写 15 个操作、约 40 行。）
````

````old docs/v0/M2-auth/reviews/closeout-review.md
5. 接口描述按操作估，约每个操作 40 行；
````

````new docs/v0/M2-auth/reviews/closeout-review.md
5. 接口描述按操作估，约每个操作 48 行（`identity.yaml` 623 行、13 个操作；codex-fixes 更正，原写约 40 行）；
````

- [ ] **Step 2：检查**：`node tools/keywords.mjs` → 没有命中；`grep -rn -E "15 个操作、约|约每个操作 40|oapi-codegen 的运行时类型" docs README.md` 只剩 Codex 报告里的引文（`docs/v0/M2-auth/reviews/M2-codex-adversarial-review.md`）。

- [ ] **Step 3：提交**（`docs(M2/codex-fixes): parent docs, README recovery, M4 and M8 handoffs, closeout counts`）

### Task 11: 全部门禁

spec 第 4 节、附录 C。不改文件，没有提交；结果写进 codex-fixes 的 review。

- [ ] **Step 1：七个门禁**，依次（`make e2e` 自己构建 `bin/nerve` 和前端，起自己的 testcontainers）：

| 命令 | 预期（原型） |
|---|---|
| `make gen-check` | 退出码 0，没有"生成物与接口描述不一致" |
| `make lint-go` | 两段都是 `0 issues.` |
| `make test` | 32 行 `ok`，没有 `FAIL` |
| `make lint-web` | `keywords: 60 rules, 3 exceptions, no hits.`；`Tasks:    54 successful, 54 total` |
| `make knip` | 退出码 0，没有输出问题 |
| `make test-web` | `Tasks:    16 successful, 16 total` |
| `make e2e` | `50 passed` |

之后按 Global Constraints 的"容器"一条核对：没有这次运行建的容器，没有留下后台进程。

- [ ] **Step 2：本分支的改动就是文件表**：`git diff --stat 37b7e9c -- . ':!docs/v0/M2-auth/specs/codex-fixes.md' ':!docs/v0/M2-auth/plans/codex-fixes.md'` 列出的文件与"文件结构"的表相同（44 个文件：T0 一个，T1 十一个（含三个生成的），T2 四个，T3 两个，T4 三个，T5 两个，T6 四个，T7 六个，T8 七个，T9 一个，T10 七个；A9、A10、README、M2 设计各被两个 Task 改到，只算一次）。

---

## codex-fixes review 提交里的替换（控制者）

codex-fixes 的 review 写好之后，控制者在同一个提交里做下面两处替换：第 15 节这一行改为"已完成"并加 review 的链接；第 14 节第一项的括号随行数改。总体设计 9.4 不变，M2 在这一阶段合并时再次完成。

````old docs/v0/M2-auth/M2-design.md
| Codex 修复 | codex-fixes | 进行中 | [spec](specs/codex-fixes.md) | [plan](plans/codex-fixes.md) | — |
````

````new docs/v0/M2-auth/M2-design.md
| Codex 修复 | codex-fixes | 已完成 | [spec](specs/codex-fixes.md) | [plan](plans/codex-fixes.md) | [review](reviews/codex-fixes-review.md) |
````

````old docs/v0/M2-auth/M2-design.md
- [x] P1–P5 和收尾全部完成，每个 Phase 都有 spec、plan 和 review。（第 15 节的七行都是"已完成"，各有三个链接。）
````

````new docs/v0/M2-auth/M2-design.md
- [x] P1–P5 和收尾全部完成，每个 Phase 都有 spec、plan 和 review。（第 15 节的八行都是"已完成"，各有三个链接；Codex 修复一行的来由见 17.3。）
````

---

## 块的核对

写计划时在 `$M2TMP/codexfix/` 下做了三件事，结果记在这里；执行者开始前可以重跑第一件，得到同样的输出。

**1. 每一块依次应用时恰好一处。** `node $M2TMP/codexfix/planapply.mjs docs/v0/M2-auth/plans/codex-fixes.md check $M2TMP/codexfix/base`（`base/` 是 `37b7e9c` 的归档解开的树）。"in order"是依次应用本计划的块时这个旧文本在当时的文件里出现的次数，必须是 1；"at 37b7e9c"是它在基点的文件里出现的次数，只作参考：T7 的一块改的是 T6 写入的 `import`，review 提交的第一块改的是 Task 0 写入的一行，它们在基点上是 0 处。退出码 0：

```
T0  old   docs/v0/M2-auth/M2-design.md (plan:126): at 37b7e9c 1, in order 1
T1  new   server/internal/platform/httpserver/bodyshape/ambiguity.go: absent
T1  new   server/internal/platform/httpserver/bodyshape/ambiguity_test.go: absent
T1  old   server/internal/platform/httpserver/bodyshape/bodyshape.go (plan:381): at 37b7e9c 1, in order 1
T1  old   server/internal/platform/httpserver/bodyshape/bodyshape.go (plan:397): at 37b7e9c 1, in order 1
T1  old   server/internal/platform/httpserver/bodyshape/bodyshape.go (plan:406): at 37b7e9c 1, in order 1
T1  old   server/internal/platform/httpserver/bodyshape/bodyshape.go (plan:416): at 37b7e9c 1, in order 1
T1  old   server/internal/platform/httpserver/bodyshape/bodyshape.go (plan:430): at 37b7e9c 1, in order 1
T1  old   server/internal/shared/error.go (plan:442): at 37b7e9c 1, in order 1
T1  old   server/internal/shared/error.go (plan:451): at 37b7e9c 1, in order 1
T1  old   api/common.yaml (plan:459): at 37b7e9c 1, in order 1
T1  old   web/packages/i18n/src/locales/en/auth.json (plan:468): at 37b7e9c 1, in order 1
T1  old   web/packages/i18n/src/locales/zh-CN/auth.json (plan:477): at 37b7e9c 1, in order 1
T1  old   web/apps/web/helpers/authentication.helper.ts (plan:486): at 37b7e9c 1, in order 1
T2  old   server/internal/platform/httpserver/apitest/operations.go (plan:511): at 37b7e9c 1, in order 1
T2  old   server/internal/platform/httpserver/apitest/operations.go (plan:520): at 37b7e9c 1, in order 1
T2  old   server/internal/platform/httpserver/apitest/operations.go (plan:534): at 37b7e9c 1, in order 1
T2  old   server/internal/platform/httpserver/apitest/operations.go (plan:545): at 37b7e9c 1, in order 1
T2  old   server/internal/platform/httpserver/apitest/operations.go (plan:554): at 37b7e9c 1, in order 1
T2  old   server/internal/platform/httpserver/apitest/operations.go (plan:582): at 37b7e9c 1, in order 1
T2  old   server/internal/platform/httpserver/apitest/operations.go (plan:609): at 37b7e9c 1, in order 1
T2  old   server/internal/platform/httpserver/apitest/operations_test.go (plan:617): at 37b7e9c 1, in order 1
T2  old   server/internal/bootstrap/contract_test.go (plan:628): at 37b7e9c 1, in order 1
T2  old   e2e/stories/identity/a10-onboarding-profile.spec.ts (plan:640): at 37b7e9c 1, in order 1
T3  old   web/apps/web/core/lib/auth/token-manager.ts (plan:703): at 37b7e9c 1, in order 1
T3  old   web/apps/web/core/lib/auth/token-manager.ts (plan:713): at 37b7e9c 1, in order 1
T3  old   web/apps/web/core/lib/auth/token-manager.ts (plan:722): at 37b7e9c 1, in order 1
T3  old   web/apps/web/core/lib/auth/token-manager.ts (plan:733): at 37b7e9c 1, in order 1
T3  old   web/apps/web/core/lib/auth/token-manager.ts (plan:750): at 37b7e9c 1, in order 1
T3  old   web/apps/web/core/lib/auth/token-manager.ts (plan:765): at 37b7e9c 1, in order 1
T3  old   web/apps/web/core/lib/auth/token-manager.ts (plan:778): at 37b7e9c 1, in order 1
T3  old   web/apps/web/core/lib/auth/token-manager.session-change.test.ts (plan:805): at 37b7e9c 1, in order 1
T3  old   web/apps/web/core/lib/auth/token-manager.session-change.test.ts (plan:815): at 37b7e9c 1, in order 1
T3  old   web/apps/web/core/lib/auth/token-manager.session-change.test.ts (plan:823): at 37b7e9c 1, in order 1
T3  old   web/apps/web/core/lib/auth/token-manager.session-change.test.ts (plan:832): at 37b7e9c 1, in order 1
T4  new   deploy/runtime-grants.sql: absent
T4  new   server/internal/bootstrap/runtime_role_test.go: absent
T4  old   README.md (plan:1157): at 37b7e9c 1, in order 1
T4  old   README.md (plan:1168): at 37b7e9c 1, in order 1
T5  old   server/internal/platform/ratelimit/ratelimit.go (plan:1186): at 37b7e9c 1, in order 1
T5  old   server/internal/platform/ratelimit/ratelimit.go (plan:1195): at 37b7e9c 1, in order 1
T5  old   server/internal/platform/ratelimit/ratelimit.go (plan:1205): at 37b7e9c 1, in order 1
T5  old   server/internal/platform/ratelimit/ratelimit_test.go (plan:1216): at 37b7e9c 1, in order 1
T6  old   e2e/fixtures/assert/identity.ts (plan:1251): at 37b7e9c 1, in order 1
T6  old   e2e/fixtures/assert/identity.ts (plan:1308): at 37b7e9c 1, in order 1
T6  old   e2e/fixtures/assert/identity.ts (plan:1321): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a8-update-me.spec.ts (plan:1343): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a8-update-me.spec.ts (plan:1351): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a8-update-me.spec.ts (plan:1367): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a8-update-me.spec.ts (plan:1378): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a8-update-me.spec.ts (plan:1391): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a9-preferences.spec.ts (plan:1403): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a9-preferences.spec.ts (plan:1411): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a9-preferences.spec.ts (plan:1424): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a9-preferences.spec.ts (plan:1433): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a9-preferences.spec.ts (plan:1441): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a9-preferences.spec.ts (plan:1451): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a10-onboarding-profile.spec.ts (plan:1461): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a10-onboarding-profile.spec.ts (plan:1469): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a10-onboarding-profile.spec.ts (plan:1480): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a10-onboarding-profile.spec.ts (plan:1494): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a10-onboarding-profile.spec.ts (plan:1506): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a10-onboarding-profile.spec.ts (plan:1515): at 37b7e9c 1, in order 1
T6  old   e2e/stories/identity/a10-onboarding-profile.spec.ts (plan:1525): at 37b7e9c 1, in order 1
T7  new   web/apps/web/core/lib/one-at-a-time.ts: absent
T7  old   web/apps/web/core/store/user/profile.store.ts (plan:1566): at 37b7e9c 1, in order 1
T7  old   web/apps/web/core/store/user/profile.store.ts (plan:1576): at 37b7e9c 1, in order 1
T7  old   web/apps/web/core/store/user/profile.store.ts (plan:1587): at 37b7e9c 1, in order 1
T7  old   web/apps/web/core/store/user/profile.store.ts (plan:1601): at 37b7e9c 1, in order 1
T7  old   web/apps/web/core/store/user/profile.store.ts (plan:1615): at 37b7e9c 1, in order 1
T7  whole web/apps/web/core/store/user/profile.store.test.ts: present
T7  old   web/apps/web/core/store/user/index.ts (plan:1713): at 37b7e9c 1, in order 1
T7  old   web/apps/web/core/store/user/index.ts (plan:1722): at 37b7e9c 1, in order 1
T7  old   web/apps/web/core/store/user/index.ts (plan:1733): at 37b7e9c 1, in order 1
T7  old   web/apps/web/core/store/user/index.ts (plan:1747): at 37b7e9c 1, in order 1
T7  old   web/apps/web/core/store/user/index.ts (plan:1761): at 37b7e9c 1, in order 1
T7  whole web/apps/web/core/store/user/index.test.ts: present
T7  old   e2e/stories/identity/a9-preferences.spec.ts (plan:1861): at 37b7e9c 0, in order 1
T7  old   e2e/stories/identity/a9-preferences.spec.ts (plan:1871): at 37b7e9c 1, in order 1
T8  whole web/apps/web/core/lib/wrappers/store-wrapper.tsx: present
T8  old   web/apps/web/core/lib/store-context.tsx (plan:2063): at 37b7e9c 1, in order 1
T8  old   web/apps/web/core/lib/store-context.tsx (plan:2073): at 37b7e9c 1, in order 1
T8  old   web/apps/web/core/lib/store-context.test.ts (plan:2082): at 37b7e9c 1, in order 1
T8  old   web/apps/web/core/lib/store-context.test.ts (plan:2099): at 37b7e9c 1, in order 1
T8  old   web/apps/web/core/lib/store-context.test.ts (plan:2108): at 37b7e9c 1, in order 1
T8  old   web/apps/web/core/components/onboarding/switch-account-modal.tsx (plan:2117): at 37b7e9c 1, in order 1
T8  old   web/apps/web/core/components/onboarding/switch-account-modal.tsx (plan:2126): at 37b7e9c 1, in order 1
T8  old   web/apps/web/core/components/onboarding/switch-account-modal.tsx (plan:2136): at 37b7e9c 1, in order 1
T8  old   e2e/stories/identity/a5-refresh-reuse.spec.ts (plan:2145): at 37b7e9c 1, in order 1
T8  old   e2e/stories/identity/a5-refresh-reuse.spec.ts (plan:2153): at 37b7e9c 1, in order 1
T8  old   e2e/stories/identity/a5-refresh-reuse.spec.ts (plan:2168): at 37b7e9c 1, in order 1
T8  old   e2e/stories/identity/a5-refresh-reuse.spec.ts (plan:2178): at 37b7e9c 1, in order 1
T8  old   e2e/stories/identity/a5-refresh-reuse.spec.ts (plan:2186): at 37b7e9c 1, in order 1
T8  old   e2e/stories/identity/a6-sign-out.spec.ts (plan:2195): at 37b7e9c 1, in order 1
T8  old   e2e/stories/identity/a6-sign-out.spec.ts (plan:2210): at 37b7e9c 1, in order 1
T8  old   e2e/stories/identity/a6-sign-out.spec.ts (plan:2220): at 37b7e9c 1, in order 1
T8  old   e2e/stories/identity/a6-sign-out.spec.ts (plan:2228): at 37b7e9c 1, in order 1
T8  old   e2e/stories/identity/a12-deactivate.spec.ts (plan:2237): at 37b7e9c 1, in order 1
T8  old   e2e/stories/identity/a12-deactivate.spec.ts (plan:2251): at 37b7e9c 1, in order 1
T8  old   e2e/stories/identity/a12-deactivate.spec.ts (plan:2265): at 37b7e9c 1, in order 1
T8  old   e2e/stories/identity/a12-deactivate.spec.ts (plan:2274): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2296): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2304): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2319): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2329): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2337): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2348): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2357): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2365): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2375): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2388): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2401): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2409): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2417): at 37b7e9c 1, in order 1
T9  old   docs/v0/M2-auth/M2-design.md (plan:2425): at 37b7e9c 1, in order 1
T10  old   docs/v0/v0-design.md (plan:2462): at 37b7e9c 1, in order 1
T10  old   docs/v0/v0-design.md (plan:2472): at 37b7e9c 1, in order 1
T10  old   docs/v0/v0-design.md (plan:2480): at 37b7e9c 1, in order 1
T10  old   docs/v0/plane-diff.md (plan:2489): at 37b7e9c 1, in order 1
T10  old   README.md (plan:2497): at 37b7e9c 1, in order 1
T10  old   docs/v0/M4-issue-core/handoffs/M2-closeout.md (plan:2509): at 37b7e9c 1, in order 1
T10  old   docs/v0/M8-open-release/handoffs/M2-closeout.md (plan:2518): at 37b7e9c 1, in order 1
T10  old   docs/v0/M2-auth/specs/closeout.md (plan:2527): at 37b7e9c 1, in order 1
T10  old   docs/v0/M2-auth/reviews/closeout-review.md (plan:2535): at 37b7e9c 1, in order 1
TR  old   docs/v0/M2-auth/M2-design.md (plan:2573): at 37b7e9c 0, in order 1
TR  old   docs/v0/M2-auth/M2-design.md (plan:2581): at 37b7e9c 1, in order 1
118 replacements, 8 whole files; all ok
```

**2. 计划的树就是原型的树。** 把 `37b7e9c` 的归档解开到 `$M2TMP/codexfix/replay`，`pnpm install --frozen-lockfile`，`bash $M2TMP/codexfix/replay.sh <本计划> <spec>`：依次 `apply` Task 0–10，Task 1 之后 `make gen`，最后放入 spec 和本计划。`node $M2TMP/codexfix/treediff.mjs $M2TMP/codexfix/replay $M2TMP/codexfix/proto`：

```
2661 and 2661 files; 0 differences
```

（不比较 `node_modules`、`.git` 和构建产物；`api/dist` 比较。）

**3. 回放的树也过门禁。** `gates.sh` 在 `replay/` 上：`make gen` 之后生成物 0 处差异，`lint-go` 两段 `0 issues.`，`make test` 32 个包 `ok`，前端的 54 个检查任务成功，`knip` 没有问题，`test-web` 16 个任务成功，`e2e` 49 passed：S3 失败只因为回放的副本不是 git 仓库，`nerve` 编进的提交是 `unknown`（原型是 git 的克隆，`make e2e` 50 passed）。关键词守卫要 `git ls-files`，在原型上跑：`keywords: 60 rules, 3 exceptions, no hits.`（含 spec 和本计划）。
