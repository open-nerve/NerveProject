/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Workspace } from "@nerve/api-client";
import { useParams } from "react-router";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";

/**
 * The workspace the route names (its workspaceSlug), as the caller's list holds it, or null: read from the route, not
 * from the router store, which StoreWrapper's effect writes after the render, so that it follows a change of
 * workspace one commit late (the keyword rule workspace-address-route; P10's project-address-route for projects).
 */
export function useRouteWorkspace(): Workspace | null {
  const { workspaceSlug } = useParams();
  const { getWorkspaceBySlug } = useWorkspace();
  return workspaceSlug ? getWorkspaceBySlug(workspaceSlug) : null;
}
