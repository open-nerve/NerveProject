/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useNavigate } from "react-router";
import type { Workspace } from "@nerve/api-client";
// hooks
import { useUserProfile } from "@/hooks/store/user";
// lib
import { followInSession } from "@/lib/in-session";

/**
 * Opens a workspace the caller has just joined (M3 design 3.14, 7.4): writes it in his profile as the one he opened
 * last, then goes there. The hint is best-effort: the workspace opens whether nerve saves it or not. Sent in the
 * session the page is in, and followed only while the tab stays in it (M3 design 7.1).
 */
export function useOpenWorkspace(): (workspace: Pick<Workspace, "id" | "slug">) => Promise<void> {
  const navigate = useNavigate();
  const { updateUserProfile } = useUserProfile();
  return (workspace) => {
    const open = () => void navigate(`/${workspace.slug}`);
    return followInSession(() => updateUserProfile({ last_workspace_id: workspace.id }), { done: open, failed: open });
  };
}
