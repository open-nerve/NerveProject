/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueDetail } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import type { PlanningPage } from "@/store/issue/helpers/planning-page-issues.store";
import type { RootStore } from "@/store/root.store";
import { cycleListState } from "./use-cycle-list-state";
import { moduleListState } from "./use-module-list-state";

/** The store types of the pages that list a cycle's or a module's work items (P6b spec 2.2). */
export type PlanningPageStoreType = EIssuesStoreType.CYCLE | EIssuesStoreType.MODULE;

/** A new work item's cycle and modules, as its form starts with them. */
type PlanningValues = Partial<Pick<IssueDetail, "cycle_id" | "module_ids">>;

/** What the shared hooks of those pages take of each domain: the hooks never branch on it (P6b spec 2.2). */
type PlanningPageDomain = {
  /** The route param naming the page's cycle or module. */
  param: string;
  /** The page's name in the scopes its follow-ups hold from a sending to its answer, and in its preview host's. */
  name: string;
  /** The store that reads the page's cycle's or module's progress. */
  progress: (root: RootStore) => { fetchProgress: (id: string) => Promise<unknown> };
  /** Whether a creation sent now joins the page's cycle or module, as the page shows it (its list state). */
  joinable: (root: RootStore, page: PlanningPage) => boolean;
  /**
   * A new work item's form's cycle and modules once the page's own is put beside those its caller gave (P7a spec 2.6,
   * R2, R15): a work item is in one cycle at most, the one given before the page's; it may be in many modules.
   */
  start: (given: PlanningValues, page: PlanningPage) => PlanningValues;
  /** Whether a work item, as nerve confirmed it, is in the page's cycle or module: the page's list's. */
  holds: (issue: Issue, page: PlanningPage) => boolean;
  /** Sends the work item's removal from the page's cycle or module; nerve's answer or refusal is the caller's. */
  remove: (root: RootStore, slug: string, page: PlanningPage, id: string) => Promise<unknown>;
};

/** Each domain's row; the store type picks it. */
export const PLANNING_PAGE_DOMAINS: Record<PlanningPageStoreType, PlanningPageDomain> = {
  [EIssuesStoreType.CYCLE]: {
    param: "cycleId",
    name: "cycle",
    progress: ({ planning }) => planning.cycles,
    joinable: ({ planning, projectRoot }, page) =>
      cycleListState(planning.cycles.getCycleById(page.entityID), projectRoot.project.getProjectById(page.projectID))
        .joinable,
    start: (given, page) => ({ ...given, cycle_id: given.cycle_id ?? page.entityID }),
    holds: (issue, page) => issue.cycle_id === page.entityID,
    // A work item is in one cycle at most: it leaves it for none (P5b spec 2.5).
    remove: ({ issue }, slug, page, id) => issue.cycleIssues.updateIssue(slug, page.projectID, id, { cycle_id: null }),
  },
  [EIssuesStoreType.MODULE]: {
    param: "moduleId",
    name: "module",
    progress: ({ planning }) => planning.modules,
    joinable: ({ planning, projectRoot }, page) =>
      moduleListState(planning.modules.getModuleById(page.entityID), projectRoot.project.getProjectById(page.projectID))
        .joinable,
    start: (given, page) => ({ ...given, module_ids: [...new Set([...(given.module_ids ?? []), page.entityID])] }),
    holds: (issue, page) => issue.module_ids.includes(page.entityID),
    // A work item is in a set of modules, which nerve replaces whole: it is worked out in its turn (P6b spec 2.5, R5).
    remove: ({ issue }, slug, page, id) => issue.moduleIssues.removeIssueFromModule(slug, page, id),
  },
};

/** Whether the store type is a cycle's or a module's page's, whose row the table has. */
export const isPlanningPageStoreType = (type: EIssuesStoreType): type is PlanningPageStoreType =>
  type in PLANNING_PAGE_DOMAINS;
