/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue } from "@nerve/api-client";
import type { IssueCatalog } from "@/lib/issue-order";
import type { IIssueRootStore } from "../root.store";
import type { IssueListScope } from "./issue-list-read";

/** The directories a list orders, groups and checks its rows by, as the root holds them now. */
export function listCatalog(root: IIssueRootStore): IssueCatalog {
  return {
    states: new Map(Object.entries(root.stateMap ?? {})),
    labels: new Map(Object.entries(root.labelMap ?? {})),
    members: new Map(Object.entries(root.memberMap ?? {})),
    // No cycle or module directory before P7 (M6 P3b spec R9): prepareCatalog refuses an order that needs one.
    cycles: new Map(),
    modules: new Map(),
  };
}

/** Reads what the list's order and groups need to know of these rows; fails loudly when a directory still lacks one. */
export async function prepareListCatalog(root: IIssueRootStore, scope: IssueListScope, issues: Issue[]): Promise<void> {
  // A proven empty scope needs no sort or membership facts, even during write confirmation.
  if (scope.scope === "project" && scope.empty) return;
  const stores = root.rootStore;
  const order = scope.query.order_by?.replace(/^-/, "");
  const states =
    order === "state_name" ||
    scope.query.group_by === "state_group" ||
    ("sub_group_by" in scope.query && scope.query.sub_group_by === "state_group") ||
    !!scope.query.state_group?.length;
  const labels = order === "label_name";
  const members = order === "assignee_name";
  const cycles = order === "cycle_name";
  const modules = order === "module_name";
  if (!states && !labels && !members && !cycles && !modules) return;
  if (states || labels)
    await Promise.all(
      [...new Set(issues.map((issue) => issue.project_id))].map(async (project) => {
        root.assertActive();
        if (!stores.projectRoot.project.getProjectById(project)) await stores.projectRoot.project.fetchProject(project);
        root.assertActive();
        await Promise.all([
          states &&
          (stores.state.getProjectStates(project) === undefined ||
            issues.some((issue) => issue.project_id === project && !stores.state.getStateById(issue.state_id)))
            ? stores.state.fetchProjectStates(project)
            : undefined,
          labels &&
          (stores.label.getProjectLabels(project) === undefined ||
            issues.some(
              (issue) => issue.project_id === project && issue.label_ids.some((id) => !stores.label.getLabelById(id))
            ))
            ? stores.label.fetchProjectLabels(project)
            : undefined,
        ]);
        root.assertActive();
      })
    );
  if (members && issues.some((issue) => issue.assignee_ids.length)) {
    root.assertActive();
    if (
      !stores.memberRoot.workspace.getMemberships(scope.workspaceSlug) ||
      issues.some((issue) => issue.assignee_ids.some((id) => !stores.memberRoot.getUserDetails(id)))
    )
      await stores.memberRoot.workspace.fetchWorkspaceMembers({
        id: scope.workspaceID,
        slug: scope.workspaceSlug,
      });
    root.assertActive();
  }
  const catalog = listCatalog(root);
  if (
    issues.some(
      (issue) =>
        (states && !catalog.states.has(issue.state_id)) ||
        (labels && issue.label_ids.some((id) => !catalog.labels.has(id))) ||
        (members && issue.assignee_ids.some((id) => !catalog.members.has(id))) ||
        (cycles && issue.cycle_id !== null && !catalog.cycles.has(issue.cycle_id)) ||
        (modules && issue.module_ids.some((id) => !catalog.modules.has(id)))
    )
  )
    throw new Error("Required issue directory is unavailable");
}
