/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { AxiosHeaders, type AxiosResponse } from "axios";
import { runInAction } from "mobx";
import type { TIssueSubIssues } from "@nerve/types";
import { FakeNerve, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { APIService } from "@/services/api.service";
import { issueOf } from "@/store/issue/fake-issues";
import { labelOf, loadProjects, projectMemberOf, projectOf, stateOf } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
const { RootStore } = await import("@/store/root.store");
let app: InstanceType<typeof RootStore>;
let nerve: FakeNerve;
const ws = workspaceOf("ws");
const here = projectOf("A", ws.id);
const elsewhere = projectOf("B", ws.id);
const state = stateOf(elsewhere, "Todo", "unstarted", 1000);
const label = labelOf(elsewhere, "Bug", 1000);
const member = projectMemberOf(elsewhere, "ann");

function reply<T>(data: T): AxiosResponse<T> {
  return { data, status: 200, statusText: "OK", headers: {}, config: { headers: new AxiosHeaders() } };
}

beforeEach(async () => {
  vi.restoreAllMocks();
  vi.useFakeTimers();
  nerve = new FakeNerve();
  app = new RootStore(nerve.client());
  binding.app = app;
  runInAction(() => {
    app.issue.workspaceSlug = ws.slug;
    app.issue.projectId = here.id;
    app.router.setQuery({ workspaceSlug: ws.slug, projectId: here.id });
  });
  await loadWorkspaces(nerve, app.workspaceRoot, [ws]);
  await loadProjects(nerve, app.projectRoot.project, ws, [here, elsewhere]);
  nerve.calls.length = 0;
});
afterEach(() => {
  app.dispose();
  vi.useRealTimers();
});

// Restoring the unused cycle/module prefetches must fail at the real legacy HTTP boundary.
it.each(["fetch", "create"].flatMap((operation) => [false, true].map((crossProject) => ({ operation, crossProject }))))(
  "$operation sub-issues loads only the unique external project's state/member/label directories ($crossProject)",
  async ({ operation, crossProject }) => {
    const parent = issueOf("parent", here.id, ws.id);
    const own = { ...issueOf("own", here.id, ws.id), parent_id: parent.id };
    const external = ["external-1", "external-2"].map((id) =>
      Object.assign(issueOf(id, elsewhere.id, ws.id), {
        parent_id: parent.id,
        state_id: state.id,
        label_ids: [label.id],
        assignee_ids: [member.member_id],
      })
    );
    const children = crossProject ? [own, ...external] : [own];
    const response: TIssueSubIssues = {
      sub_issues: children,
      state_distribution: {
        backlog: [],
        unstarted: children.map((child) => child.id),
        started: [],
        completed: [],
        cancelled: [],
      },
    };
    app.issue.issues.addIssue([parent]);
    const endpoint = `/api/workspaces/${ws.slug}/projects/${here.id}/issues/${parent.id}/sub-issues/`;
    const get = vi.spyOn(APIService.prototype, "get").mockImplementation(async (url) => {
      if (url === endpoint) return reply(response);
      // Old directory APIs are outside P10a; an empty real wire list lets their stores settle on the red run.
      return reply([]);
    });
    const post = vi.spyOn(APIService.prototype, "post").mockResolvedValue(reply(response));
    const subIssues = app.issue.issueDetail.subIssues;
    if (operation === "fetch") await subIssues.fetchSubIssues(ws.slug, here.id, parent.id);
    else
      await subIssues.createSubIssues(
        ws.slug,
        here.id,
        parent.id,
        children.map((child) => child.id)
      );

    if (crossProject) {
      await until(() => nerve.calls.length === 3, "external directories requested");
      expect(nerve.calls.map(({ method, path }) => [method, path]).toSorted()).toEqual([
        ["GET", `/api/v0/projects/${elsewhere.id}/labels`],
        ["GET", `/api/v0/projects/${elsewhere.id}/members`],
        ["GET", `/api/v0/projects/${elsewhere.id}/states`],
      ]);
      for (const call of nerve.calls) {
        const data = call.path.endsWith("/states") ? [state] : call.path.endsWith("/members") ? [member] : [label];
        call.answer(json(200, { data }));
      }
      await until(
        () =>
          app.state.getStateById(state.id) !== undefined &&
          app.label.getLabelById(label.id) !== undefined &&
          app.memberRoot.project.getProjectMemberIds(elsewhere.id, true)?.includes(member.member_id) === true,
        "external directories available to sub-issue properties"
      );
    } else {
      await vi.advanceTimersByTimeAsync(0);
      expect(nerve.calls).toEqual([]);
    }
    expect(subIssues.subIssuesByIssueId(parent.id)).toEqual(children.map((child) => child.id));
    expect(subIssues.stateDistributionByIssueId(parent.id)).toEqual(response.state_distribution);
    expect(app.issue.issues.getIssueById(parent.id)?.sub_issues_count).toBe(children.length);
    for (const child of children) expect(app.issue.issues.getIssueById(child.id)).toEqual(child);
    expect(get.mock.calls.map(([url]) => url)).toEqual(operation === "fetch" ? [endpoint] : []);
    expect(post.mock.calls.map(([url, body]) => [url, body])).toEqual(
      operation === "create" ? [[endpoint, { sub_issue_ids: children.map((child) => child.id) }]] : []
    );
  }
);
