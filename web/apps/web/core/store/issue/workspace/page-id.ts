/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { DEFAULT_GLOBAL_VIEWS_LIST, ISSUE_STORE_TO_FILTERS_MAP, type DefaultWorkspaceViewKey } from "@nerve/constants";
import { EIssuesStoreType, type TWorkItemFilterProperty } from "@nerve/types";
import type { WorkspaceIssueQuery } from "@/services/issue/issue-list.service";

/** The default view a key names (M7/P6b spec 2.5); none for a saved view's id. */
const defaultViewOf = (globalViewId: string) => DEFAULT_GLOBAL_VIEWS_LIST.find((view) => view.key === globalViewId);

/**
 * The id of a workspace's view's page's preferences (M7/P6b spec 2.4, R2, R3): a saved view's is its id; a default
 * view's is its key in its workspace, for the four keys repeat in every workspace and the preference services take an
 * id alone. The route's globalViewId is never the id of a default view's page.
 */
export function workspacePageId(slug: string, globalViewId: string): string {
  return defaultViewOf(globalViewId) ? JSON.stringify([slug, globalViewId]) : globalViewId;
}

/** The default view a page id names, in whichever workspace; none for a saved view's page or any other string. */
export function defaultViewOfPage(id: string): DefaultWorkspaceViewKey | undefined {
  if (!id.startsWith("[")) return undefined;
  let value: unknown;
  try {
    value = JSON.parse(id);
  } catch {
    return undefined;
  }
  if (!Array.isArray(value) || value.length !== 2 || typeof value[0] !== "string" || typeof value[1] !== "string")
    return undefined;
  return defaultViewOf(value[1])?.key;
}

/** A default view's own term, `me` in it, fixed last in its page's query; a saved view has none (§2.5, R4). */
export function ownTerm(
  globalViewId: string
): Pick<WorkspaceIssueQuery, "assignee_id" | "created_by_id" | "subscriber_id"> {
  const own = defaultViewOf(globalViewId)?.own;
  return own ? { [own]: ["me"] } : {};
}

/** The words a workspace's view's page filters by: the contract's 11, a default view's own key left out (§2.5). */
export function workspacePageWords(globalViewId: string): TWorkItemFilterProperty[] {
  const own = defaultViewOf(globalViewId)?.own;
  return (ISSUE_STORE_TO_FILTERS_MAP[EIssuesStoreType.GLOBAL]?.filters ?? []).filter((word) => word !== own);
}
