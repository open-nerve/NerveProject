/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Workspace } from "@nerve/api-client";

/**
 * Where an onboarded account goes when no page is asked for (M3 design 3.14, 7.4): the workspace it opened last, while
 * that is still one of its workspaces; else the one created first, and of those created at the same moment, the
 * first in the list's order (nerve's: by name, then id); with none, the page that creates one. The last workspace is
 * only a hint the web app wrote (Profile.last_workspace_id): it counts only when the list has it.
 */
export function landingPath(workspaces: readonly Workspace[], lastWorkspaceId: string | null): string {
  const landing =
    workspaces.find((workspace) => workspace.id === lastWorkspaceId) ??
    workspaces.reduce<Workspace | undefined>(
      (first, workspace) =>
        first === undefined || Date.parse(workspace.created_at) < Date.parse(first.created_at) ? workspace : first,
      undefined
    );
  return landing ? `/${landing.slug}` : "/create-workspace";
}
