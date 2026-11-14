/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue } from "@nerve/api-client";
import type { IssueCatalog } from "@/lib/issue-order";
import { nextPlanningList, type PlanningList } from "@/lib/planning-directory";
import type { IPlanningStore } from "@/store/planning";
import type { IIssueRootStore } from "../root.store";
import type { IssueListScope } from "./issue-list-read";

/**
 * The planning store's lookups as a list's directory (M6 P7b spec 2.2, R17): a cycle or a module as getCycleById and
 * getModuleById give it, archived and ended ones too, of a project the project store gives; the rule is the store's.
 */
function planningLookups({ cycles, modules }: IPlanningStore): Pick<IssueCatalog, "cycles" | "modules"> {
  return {
    cycles: { get: cycles.getCycleById, has: (id) => cycles.getCycleById(id) !== undefined },
    modules: { get: modules.getModuleById, has: (id) => modules.getModuleById(id) !== undefined },
  };
}

/** The directories a list orders, groups and checks its rows by, as the root holds them now. */
export function listCatalog(root: IIssueRootStore): IssueCatalog {
  return {
    states: new Map(Object.entries(root.stateMap ?? {})),
    labels: new Map(Object.entries(root.labelMap ?? {})),
    members: new Map(Object.entries(root.memberMap ?? {})),
    ...planningLookups(root.rootStore.planning),
  };
}

/**
 * What a list ordered by a cycle's or a module's name names its rows by: the values a work item holds, the lookup that
 * names them, and a project's two lists of the domain, read by the planning store's own fetches (those of the domain's
 * pages and of usePlanningNames, into the same values: no second way to read them).
 */
type PlanningNames = {
  values: (issue: Issue) => readonly string[];
  lookup: (lookups: Pick<IssueCatalog, "cycles" | "modules">) => Pick<IssueCatalog["cycles"], "has">;
  reads: (planning: IPlanningStore) => Record<PlanningList, (projectId: string) => Promise<unknown>>;
};
const PLANNING_NAMES = new Map<string, PlanningNames>([
  [
    "cycle_name",
    {
      values: (issue) => (issue.cycle_id === null ? [] : [issue.cycle_id]),
      lookup: ({ cycles }) => cycles,
      reads: ({ cycles }) => ({ list: cycles.fetchCycles, archive: cycles.fetchArchivedCycles }),
    },
  ],
  [
    "module_name",
    {
      values: (issue) => issue.module_ids,
      lookup: ({ modules }) => modules,
      reads: ({ modules }) => ({ list: modules.fetchModules, archive: modules.fetchArchivedModules }),
    },
  ],
]);

/**
 * Reads the project's lists of the domain while the planning store cannot name a value the rows of the project hold
 * (M6 P7b spec 2.2, re-74 F2), in the order every reader of names shares (nextPlanningList): its list of what is not
 * archived, then its archive, each once at most and whether the store holds it or not, as the states and labels are
 * read again, so that a value created elsewhere after the store held both is named. A read a newer one overtook gives
 * nothing (Reconciled.fetch, re-74 F9): a value it would have named may stay unnamed, and the list then fails as when
 * nerve refused the read, to be read again (a list's read is retried, a write's confirmation marks the list
 * unavailable), as for the states and labels; no more is needed.
 */
async function readPlanningNames(root: IIssueRootStore, names: PlanningNames, project: string, issues: Issue[]) {
  const planning = root.rootStore.planning;
  const unnamed = () => {
    const lookup = names.lookup(planningLookups(planning));
    return issues.some((issue) => issue.project_id === project && names.values(issue).some((id) => !lookup.has(id)));
  };
  const read: Record<PlanningList, boolean> = { list: false, archive: false };
  for (let next = nextPlanningList(unnamed(), read); next; next = nextPlanningList(unnamed(), read)) {
    read[next] = true;
    root.assertActive();
    // oxlint-disable-next-line no-await-in-loop -- The archive is read only for what the list left unnamed.
    await names.reads(planning)[next](project);
    root.assertActive();
  }
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
  // The list's own read gates the planning names, not the project's switch (R17): nerve orders by them either way.
  const planning = order === undefined ? undefined : PLANNING_NAMES.get(order);
  if (!states && !labels && !members && !planning) return;
  if (states || labels || planning)
    await Promise.all(
      [...new Set(issues.map((issue) => issue.project_id))].map(async (project) => {
        root.assertActive();
        // The planning store names nothing of a project the project store does not give (re-74 F8).
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
          planning && readPlanningNames(root, planning, project, issues),
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
        (planning && planning.values(issue).some((id) => !planning.lookup(catalog).has(id)))
    )
  )
    throw new Error("Required issue directory is unavailable");
}
