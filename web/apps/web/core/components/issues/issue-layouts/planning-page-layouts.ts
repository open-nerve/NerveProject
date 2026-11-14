/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
import { EIssuesStoreType } from "@nerve/types";
import type { TWorkItemFilterProperty } from "@nerve/types";
import type { PlanningPageStoreType } from "@/hooks/planning-page-domains";
import { useCycleListState } from "@/hooks/use-cycle-list-state";
import { useModuleListState } from "@/hooks/use-module-list-state";
import type { PlanningDomain } from "@/lib/planning-switch";

/** An empty state's texts, by their keys. */
type EmptyTexts = { title: string; description: string };

/**
 * What the shared layout root, the four layouts, the rows' quick actions and the empty state of a planning page take of
 * its domain (P6b spec 2.2, re-70 F4): they never branch on it. The page's route param, its name and its removal of a
 * work item are the shared hooks' table's (PLANNING_PAGE_DOMAINS).
 */
type PlanningPageLayout = {
  /** The domain whose switch puts the page's list on, and whose turned-off state the page is while it is off. */
  domain: PlanningDomain;
  /** The name that leads the key of the read of the page's preferences. */
  preferences: string;
  /** The filters the page offers: its own display options' (P5b R9). */
  filters: TWorkItemFilterProperty[];
  /**
   * The page's list state, as the page shows its cycle or module and its project: whether the page has it (of the
   * route's project, read-only or not), read-only, joinable by a new one.
   */
  useListState: () => { held: boolean; readOnly: boolean; joinable: boolean };
  /** What a row's removal of its work item from the page's cycle or module is named, and says once nerve confirms it. */
  remove: { label: string; success: string };
  /** The empty state's texts: once read-only, why there are none; else the page's own, with its new work item's. */
  empty: { readOnly: EmptyTexts; none: EmptyTexts & { create: string } };
};

/** Each domain's row; the store type picks it. */
export const PLANNING_PAGE_LAYOUTS: Record<PlanningPageStoreType, PlanningPageLayout> = {
  [EIssuesStoreType.CYCLE]: {
    domain: "cycles",
    preferences: "CYCLE_ISSUE_FILTERS",
    filters: ISSUE_DISPLAY_FILTERS_BY_PAGE.cycle_issues.filters,
    // read-only once the cycle has ended or is archived (P5b R5)
    useListState: useCycleListState,
    remove: { label: "issue.remove.cycle.label", success: "issue.remove.cycle.success" },
    empty: {
      readOnly: {
        title: "project_cycles.empty_state.completed_no_issues.title",
        description: "project_cycles.empty_state.completed_no_issues.description",
      },
      none: {
        title: "project_empty_state.cycle_work_items.title",
        description: "project_empty_state.cycle_work_items.description",
        create: "project_empty_state.cycle_work_items.cta_primary",
      },
    },
  },
  [EIssuesStoreType.MODULE]: {
    domain: "modules",
    preferences: "MODULE_ISSUE_FILTERS",
    filters: ISSUE_DISPLAY_FILTERS_BY_PAGE.module_issues.filters,
    // read-only once the module is archived, whatever its status (P6b R9)
    useListState: useModuleListState,
    remove: { label: "issue.remove.module.label", success: "issue.remove.module.success" },
    empty: {
      readOnly: {
        title: "project_module.empty_state.archived.title",
        description: "project_module.empty_state.archived.description",
      },
      none: {
        title: "project_empty_state.module_work_items.title",
        description: "project_empty_state.module_work_items.description",
        create: "project_empty_state.module_work_items.cta_primary",
      },
    },
  },
};
