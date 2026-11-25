/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Project, View } from "@nerve/api-client";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
import { viewEntryOn } from "@/lib/views-entry";
// store
import type { RootStore } from "@/store/root.store";

/** The views of the search's project whose names have its query, each with its project. */
type ViewResults = { view: View; project: Project }[];

/**
 * The views the command palette's search names (P6a spec 2.8, R13): of the search's project alone, projectId, none at
 * the workspace's scope (a search of views there would need nerve to search them). While the caller is offered the ways
 * into the project's views (viewEntryOn: its views turned on and his role an admin's, a member's or a guest's, by the
 * address's workspace and project, never the router's), of the store's list of the project's views, in nerve's order,
 * those whose names have needle, the lowercase query: filtered here, as the search's projects are, for the palette
 * filters nothing while it searches (wrapper.tsx). While the store holds no such list, the open palette reads it by
 * the key the views list reads it by, once each time it opens, searching or not (R20: the project's shell reads no
 * views); once the store holds it, it is the store's, with the changes nerve confirmed meanwhile. Until nerve answers,
 * or once it refused, the search names no view, and holds up none of its other results.
 */
export function useViewSearch(
  rootStore: RootStore,
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  needle: string,
  open: boolean
): ViewResults {
  const { views } = rootStore;
  const project = rootStore.projectRoot.project.getProjectById(projectId);
  const role = workspaceSlug
    ? rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)
    : undefined;
  // with no way in, the search names no view and reads none; while the store lacks the list, none yet
  const listed = viewEntryOn(project, role) ? views.getProjectViews(projectId) : [];
  useSessionSWR(open && !listed ? ["PROJECT_VIEWS", projectId] : null, (id) => views.fetchProjectViews(id));

  if (!project) return [];
  return (listed ?? []).filter((view) => view.name.toLowerCase().includes(needle)).map((view) => ({ view, project }));
}
