// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { observer } from "mobx-react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueOf, issueStoreTab } from "@/store/issue/fake-issues";
import { projectOf, stateOf, loadProjects } from "@/store/project/fake-projects";
import { json, problem } from "@/lib/auth/fake-nerve";
import type { IssueSearchQuery } from "@/services/issue/issue-search.service";
import { useIssueSearch } from "./use-issue-search";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let renderer: Root;
let result: ReturnType<typeof useIssueSearch>;
const Probe = observer(function Probe({ query, open }: { query: IssueSearchQuery; open: boolean }) {
  result = useIssueSearch(tab.app.issue, "ws", query, open);
  return (
    <div>
      {result.ids.join(",")}
      {result.loading ? "loading" : ""}
      {result.error ? "error" : ""}
    </div>
  );
});
const render = (query: IssueSearchQuery = { project_id: "A" }, open = true) =>
  act(async () => {
    renderer.render(<Probe query={query} open={open} />);
  });
const searches = () => tab.nerve.to("/api/v0/workspaces/ws/issue-search");
const answer = (id: string, index = searches().length - 1, project = "A", workspace = "id-ws") =>
  act(async () => {
    searches()[index]!.answer(json(200, { data: [issueOf(id, project, workspace)] }));
  });
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
  renderer = createRoot(document.createElement("div"));
  tab.nerve.respond = (call) => {
    if (call.path.endsWith("/states")) {
      const project = tab.app.projectRoot.project.getProjectById(call.path.split("/").at(-2))!;
      call.answer(json(200, { data: [stateOf(project, "Todo", "unstarted", 1, { id: "state" })] }));
    }
  };
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  tab.app.dispose();
  vi.useRealTimers();
});
it("invalidatesCandidatesBeforeDebounceFires", async () => {
  await render();
  await answer("first");
  expect(result.ids).toEqual(["first"]);
  await render({ project_id: "A", q: "new" });
  expect(result.ids).toEqual([]);
  expect(result.loading).toBe(true);
  expect(searches()).toHaveLength(1);
  await act(async () => vi.advanceTimersByTimeAsync(499));
  expect(searches()).toHaveLength(1);
  await act(async () => vi.advanceTimersByTimeAsync(1));
  expect(searches()).toHaveLength(2);
  expect(searches()[1]!.query).toEqual({ project_id: "A", q: "new" });
  await answer("new");
  expect(result.ids).toEqual(["new"]);
});
it("a response during the new query debounce cannot enter canonical", async () => {
  await render();
  await render({ project_id: "A", q: "new" });
  await answer("stale");
  expect(result.ids).toEqual([]);
  expect(tab.app.issue.issues.held("stale")).toBeUndefined();
});
it.each(["scope", "permission", "workspace permission", "closed", "retired"])(
  "doesNotPublishAcrossScopeOrPermissionABA: %s",
  async (change) => {
    await render(change === "workspace permission" ? {} : { project_id: "A" });
    if (change === "scope") {
      await act(async () =>
        loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
          tab.project,
          projectOf("B", tab.workspace.id, { id: "B" }),
        ])
      );
      await render({ project_id: "B" });
    } else if (change.includes("permission")) {
      await act(async () => {
        tab.app.projectRoot.project.confirmProject(tab.project, (p) => ({ ...p, member_role: null, network: 0 }));
        tab.app.projectRoot.project.confirmProject(tab.project, () => tab.project);
      });
    } else if (change === "closed") await render({ project_id: "A" }, false);
    else await act(async () => tab.app.issue.retire());
    await answer("stale", 0);
    expect(result.ids).toEqual([]);
    expect(tab.app.issue.issues.held("stale")).toBeUndefined();
    expect(result.error).toBeUndefined();
  }
);
it.each([
  ["B", "id-ws"],
  ["A", "elsewhere"],
])("doesNotAcceptForeignSearchRows: %s / %s", async (project, workspace) => {
  await act(async () =>
    loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
      tab.project,
      projectOf("B", tab.workspace.id, { id: "B" }),
    ])
  );
  await render();
  await answer("foreign", 0, project, workspace);
  expect(tab.app.issue.issues.held("foreign")).toBeUndefined();
  expect(result.ids).toEqual([]);
});
it("proves cold project and state directories before publishing and retries their refusal as reads", async () => {
  await render({ for: "relation", issue_id: "source" });
  await answer("other", 0, "B");
  const cold = tab.nerve.to("/api/v0/projects/B");
  expect(cold).toHaveLength(1);
  expect(tab.app.issue.issues.held("other")).toBeUndefined();
  await act(async () => cold[0]!.answer(problem(403, "forbidden")));
  expect(result.error).toMatchObject({ status: 403 });
  expect(result.loading).toBe(false);
  await act(async () => {
    result.retry();
  });
  await answer("other", 1, "B");
  await act(async () =>
    tab.nerve.to("/api/v0/projects/B")[1]!.answer(json(200, projectOf("B", tab.workspace.id, { id: "B" })))
  );
  expect(result.ids).toEqual(["other"]);
  expect(result.error).toBeUndefined();
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
  expect(tab.app.issue.issues.getIssueById("other")?.project_id).toBe("B");
});
it("a current state directory refusal is visible and never publishes a half prepared candidate", async () => {
  tab.nerve.respond = (call) => {
    if (call.path.endsWith("/states")) call.answer(problem(403, "forbidden"));
  };
  await render();
  await answer("x");
  expect(result.error).toMatchObject({ status: 403 });
  expect(result.ids).toEqual([]);
  expect(tab.app.issue.issues.held("x")).toBeUndefined();
});

it("permission loss ends loading and a later grant starts a fresh read", async () => {
  await render();
  await act(async () =>
    tab.app.projectRoot.project.confirmProject(tab.project, (p) => ({ ...p, member_role: null, network: 0 }))
  );
  expect(result.loading).toBe(false);
  expect(result.ids).toEqual([]);
  await act(async () => tab.app.projectRoot.project.confirmProject(tab.project, () => tab.project));
  expect(result.loading).toBe(true);
  expect(searches()).toHaveLength(2);
  await answer("restored");
  expect(result.ids).toEqual(["restored"]);
});
