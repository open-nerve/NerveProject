// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import type { WorkspaceMember } from "@nerve/api-client";
import { useEditorMention } from "./use-editor-mention";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { projectMemberOf } from "@/store/project/fake-projects";
import { json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
const state = vi.hoisted(() => ({ tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => state.tab!.app.issue.issueDetail }));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
let result: ReturnType<typeof useEditorMention>;
let props = { workspaceSlug: "ws", projectId: undefined as string | undefined, editable: true };
const Probe = observer(function Probe({ options }: { options: typeof props }) {
  result = useEditorMention({ ...options, members: state.tab!.app.memberRoot });
  return null;
});
beforeEach(async () => {
  vi.useFakeTimers();
  state.tab = await issueStoreTab();
  props = { workspaceSlug: "ws", projectId: undefined, editable: true };
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  state.tab!.app.dispose();
  vi.useRealTimers();
});
const render = () => act(async () => root.render(<Probe options={{ ...props }} />));
const member = (name: string, fields: Partial<WorkspaceMember> = {}) => membershipOf(name, fields, "ws");
async function start(query: string, project = props.projectId) {
  const nerve = state.tab!.nerve;
  const at = nerve.calls.length;
  const promise = result.fetchMentions(query);
  const outcome = track(promise);
  await until(() => nerve.calls.length >= at + (project ? 2 : 1) || outcome.settled, "member directories");
  const calls = nerve.calls.slice(at);
  expect(calls.map((call) => [call.method, call.path])).toEqual([
    ["GET", "/api/v0/workspaces/ws/members"],
    ...(project ? [["GET", `/api/v0/projects/${project}/members`]] : []),
  ]);
  return { calls, promise, outcome };
}
async function answer(read: Awaited<ReturnType<typeof start>>, rows: WorkspaceMember[], names?: string[]) {
  read.calls[0]!.answer(json(200, { data: rows }));
  read.calls[1]?.answer(json(200, { data: names?.map((name) => projectMemberOf({ id: "A" }, name)) ?? [] }));
  await until(() => read.outcome.settled, "mention result");
}
it("filters active workspace members by display name only, with five results and no email", async () => {
  await render();
  const read = await start("AL");
  await answer(read, [
    member("alice-inactive", { is_active: false }),
    member("other"),
    ...Array.from({ length: 7 }, (_, i) => member(`Alice${i}`)),
  ]);
  const sections = await read.promise;
  expect(sections.flatMap((section) => section.items.map((item) => item.id))).toEqual([
    "u-Alice0",
    "u-Alice1",
    "u-Alice2",
    "u-Alice3",
    "u-Alice4",
  ]);
  expect(JSON.stringify(sections)).not.toContain("@example.com");
  const email = await start("example.com");
  await answer(email, [member("Alice")]);
  expect(await email.promise).toEqual([]);
});
it("uses active project membership independently of display filters and excludes inactive workspace users", async () => {
  props.projectId = "A";
  state.tab!.app.memberRoot.project.filters.updateFilters("A", { roles: ["20"] });
  await render();
  const read = await start("");
  await answer(
    read,
    [member("outside"), member("inactive", { is_active: false }), member("alice"), member("guest", { role: 5 })],
    ["inactive", "alice", "guest"]
  );
  expect((await read.promise)[0]?.items.map((item) => item.id)).toEqual(["u-alice", "u-guest"]);
  expect(state.tab!.app.memberRoot.project.getFilteredProjectMemberIds("A")).toEqual([]);
});
it("does not ask for suggestions in a read-only editor", async () => {
  props.editable = false;
  await render();
  expect(await result.fetchMentions("a")).toEqual([]);
  expect(state.tab!.nerve.calls).toEqual([]);
});
it("returns the directory refusal without turning it into no results, then permits a read-only retry", async () => {
  await render();
  const read = await start("a");
  read.calls[0]!.answer(problem(403, "forbidden"));
  await until(() => read.outcome.settled, "refused directory");
  expect(read.outcome.error).toMatchObject({ status: 403, problem: { code: "forbidden" } });
  const retry = await start("a");
  await answer(retry, [member("alice")]);
  expect((await retry.promise)[0]?.items[0]?.id).toBe("u-alice");
});
it("rejects an older query and keeps a captured editor callback bound to the current scope", async () => {
  props.projectId = "A";
  await render();
  const captured = result.fetchMentions;
  const old = await start("old");
  props.projectId = undefined;
  await render();
  expect(result.fetchMentions).toBe(captured);
  const fresh = await start("new");
  await answer(fresh, [member("new")]);
  await answer(old, [member("old")], ["old"]);
  expect(old.outcome.error).toBeInstanceOf(SessionChangedError);
  expect((await fresh.promise)[0]?.items[0]?.title).toBe("new");
});
it("only returns the latest query in one editor scope", async () => {
  await render();
  const old = await start("old");
  const latest = await start("new");
  await answer(latest, [member("new")]);
  await answer(old, [member("old")]);
  expect(old.outcome.error).toBeInstanceOf(SessionChangedError);
  expect((await latest.promise)[0]?.items[0]?.title).toBe("new");
});
it.each(["permission", "retirement", "unmount"])("does not return an in-flight suggestion after %s", async (change) => {
  await render();
  const read = await start("a");
  if (change === "permission") {
    await loadWorkspaces(state.tab!.nerve, state.tab!.app.workspaceRoot, [{ ...state.tab!.workspace, role: 5 }]);
    await loadWorkspaces(state.tab!.nerve, state.tab!.app.workspaceRoot, [state.tab!.workspace]);
  } else if (change === "retirement") state.tab!.app.dispose();
  else await act(async () => root.render(null));
  await answer(read, [member("alice")]);
  expect(read.outcome.error).toBeInstanceOf(SessionChangedError);
});

it("does not fall back to workspace users when the project directory is refused", async () => {
  props.projectId = "A";
  await render();
  const read = await start("a");
  read.calls[0]!.answer(json(200, { data: [member("alice")] }));
  read.calls[1]!.answer(problem(403, "forbidden"));
  await until(() => read.outcome.settled, "project refusal");
  expect(read.outcome.error).toMatchObject({ status: 403, problem: { code: "forbidden" } });
});
it("does not read a workspace-only directory after the caller becomes a guest", async () => {
  await loadWorkspaces(state.tab!.nerve, state.tab!.app.workspaceRoot, [{ ...state.tab!.workspace, role: 5 }]);
  state.tab!.nerve.calls.length = 0;
  await render();
  await expect(result.fetchMentions("a")).rejects.toBeInstanceOf(SessionChangedError);
  expect(state.tab!.nerve.calls).toEqual([]);
});

it("keeps an overtaken current directory read retryable instead of claiming a session change", async () => {
  await render();
  const read = await start("a");
  const otherReader = state.tab!.app.memberRoot.workspace.fetchWorkspaceMembers(state.tab!.workspace);
  await until(() => state.tab!.nerve.calls.length === 2, "another directory reader");
  state.tab!.nerve.calls[1]!.answer(json(200, { data: [member("alice")] }));
  await otherReader;
  await answer(read, [member("alice")]);
  expect(read.outcome.error).toBeInstanceOf(Error);
  expect(read.outcome.error).not.toBeInstanceOf(SessionChangedError);
  const retry = await start("a");
  await answer(retry, [member("alice")]);
  expect((await retry.promise)[0]?.items[0]?.title).toBe("alice");
});

it("invalidates an already subscribed popup on host changes without replacing its stable adapter", async () => {
  await render();
  const signal = result.mentionScope;
  const current = signal.capture();
  const close = vi.fn();
  const stop = signal.subscribe(close);
  props = { ...props, projectId: "A" };
  await render();
  expect(current()).toBe(false);
  expect(close).toHaveBeenCalledTimes(1);
  expect(result.mentionScope).toBe(signal);
  expect(signal.capture()()).toBe(true);
  stop();
});
it("notifies an open popup across permission loss and recovery using the root watermark", async () => {
  await render();
  const current = result.mentionScope.capture();
  const close = vi.fn();
  const stop = result.mentionScope.subscribe(close);
  await act(async () =>
    loadWorkspaces(state.tab!.nerve, state.tab!.app.workspaceRoot, [{ ...state.tab!.workspace, role: 5 }])
  );
  expect(current()).toBe(false);
  expect(close).toHaveBeenCalled();
  stop();
});
