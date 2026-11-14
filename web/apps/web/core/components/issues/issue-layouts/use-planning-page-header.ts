/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import type { PlanningPageStoreType } from "@/hooks/planning-page-domains";
import { useIssues } from "@/hooks/store/use-issues";
import { useUserPermissions } from "@/hooks/store/user";
import { usePlanningReads } from "@/hooks/use-planning-reads";
import { PLANNING_PAGE_LAYOUTS } from "./planning-page-layouts";

/** The page whose header it is, as its route names it: the project, and the page's cycle or module. */
type PlanningPageAddress = { workspaceSlug: string; projectId: string; entityId: string };

/** What a planning page's header shows of the page's work items. */
type PlanningPageHeader = {
  /** Their count as nerve gives it, while the page's current list is its own; else none. */
  count: number | undefined;
  /** Whether the page's layouts, the toggle of its filters and its display show: once its preferences are read. */
  preferencesShown: boolean;
  /** Whether the header offers a new work item of the page's cycle or module. */
  canCreate: boolean;
};

/**
 * What a cycle's or a module's page's header shows of the page's work items (P5b spec 2.6; P6b spec 2.2, 2.7, 2.8, R3,
 * R6), by the page's store type, whose domain's row of the layouts' table gives the switch and the list state: for
 * those whom the page lists them (the layout root's own condition: a member of the project while the domain is on),
 * while the page has its cycle or module (held: of the route's project, read-only or not; one of another project the
 * page says is not there, review re-71 M2, fix F9), their count, only while the current list is the page's own; the
 * page's layouts, filters and display once nerve has given the page's preferences, not before its first answer nor
 * while that first read failed (re-67 M8); and a new work item of the page for the project's admins and members while
 * the page shows it taking them (joinable, which holds it). Each header lays them out itself.
 */
export function usePlanningPageHeader(
  storeType: PlanningPageStoreType,
  { workspaceSlug, projectId, entityId }: PlanningPageAddress
): PlanningPageHeader {
  const { domain, useListState } = PLANNING_PAGE_LAYOUTS[storeType];
  const { issues, issuesFilter } = useIssues(storeType);
  const { allowPermissions } = useUserPermissions();
  const { held, joinable } = useListState();
  const { listed } = usePlanningReads(workspaceSlug, projectId, domain);
  const shown = listed && held;
  return {
    count: shown ? issues.pageTotal({ projectID: projectId, entityID: entityId }) : undefined,
    preferencesShown: shown && issuesFilter.getIssueFilters(entityId) !== undefined,
    canCreate:
      joinable &&
      allowPermissions(
        [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
        EUserPermissionsLevel.PROJECT,
        workspaceSlug,
        projectId
      ),
  };
}
