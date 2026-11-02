/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json } from "@/lib/auth/fake-nerve";
import { loadArchivedProjects, loadProject, loadProjects, projectOf } from "@/store/project/fake-projects";
import { issueOf, issuePageOf, issueStoreTab, requestedIssue } from "./fake-issues";
import { IssueListBinding } from "./helpers/issue-list-binding";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

it.each([false, true].flatMap((archived) => [false, true].map((empty) => ({ archived, empty }))))(
  "R39 an old directory omission preserves issue reads but a new absence revokes them archived=$archived empty=$empty",
  async ({ archived, empty }) => {
    const store = tab.app.projectRoot.project;
    const project = {
      ...tab.project,
      network: 0 as const,
      archived_at: archived ? "2026-10-30T00:00:00Z" : null,
    };
    await loadProjects(tab.nerve, store, tab.workspace, archived ? [] : [project]);
    if (archived) await loadArchivedProjects(tab.nerve, store, tab.workspace, [project]);
    const binding = new IssueListBinding(tab.app.issue, {
      scope: "project",
      workspaceSlug: tab.workspace.slug,
      workspaceID: tab.workspace.id,
      projectID: project.id,
      query: { order_by: "sort_order" },
    });
    const endpoint: [string, string] = ["GET", `/api/v0/projects/${project.id}/issues`];
    const first = await requestedIssue(tab.nerve, () => binding.fetchFirst(), endpoint);
    first.call.answer(json(200, issuePageOf([issueOf("held")])));
    await first.promise;
    const identity = tab.app.issue.permissionReads.identity(tab.workspace.id, [project.id]);
    const old = await requestedIssue(
      tab.nerve,
      () => (archived ? store.fetchArchivedProjects(tab.workspace) : store.fetchProjects(tab.workspace)),
      ["GET", `/api/v0/workspaces/${tab.workspace.slug}/projects`]
    );
    await loadProject(tab.nerve, store, project, { ...project, name: "new detail proof" });
    const current = await requestedIssue(tab.nerve, () => binding.fetchFirst(), endpoint);
    const peer = projectOf("PEER", tab.workspace.id, { archived_at: project.archived_at });
    old.call.answer(json(200, { data: empty ? [] : [peer] }));
    await old.promise;
    expect.soft(tab.app.issue.permissionReads.identity(tab.workspace.id, [project.id])).toBe(identity);
    expect.soft(tab.app.issue.issues.getIssueById("held")?.id).toBe("held");
    expect.soft(binding.group()?.rows.map((row) => row.id)).toEqual(["held"]);
    current.call.answer(json(200, issuePageOf([issueOf("current")])));
    await current.promise;
    expect.soft(tab.app.issue.issues.getIssueById("current")?.id).toBe("current");
    expect.soft(binding.group()?.rows.map((row) => row.id)).toEqual(["current"]);
    const late = await requestedIssue(tab.nerve, () => binding.fetchFirst(), endpoint);
    if (archived) await loadArchivedProjects(tab.nerve, store, tab.workspace, []);
    else await loadProjects(tab.nerve, store, tab.workspace, []);
    expect(tab.app.issue.permissionReads.identity(tab.workspace.id, [project.id])).not.toBe(identity);
    expect(tab.app.issue.issues.getIssueById("held")).toBeUndefined();
    expect(tab.app.issue.issues.getIssueById("current")).toBeUndefined();
    late.call.answer(json(200, issuePageOf([issueOf("revoked")])));
    await late.promise;
    expect(tab.app.issue.issues.getIssueById("revoked")).toBeUndefined();
    expect(binding.group()?.rows ?? []).toEqual([]);
  }
);
