/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { WorkspaceRole } from "@nerve/api-client";

/**
 * A workspace's roles, the lowest first, as its role selects offer them: a select gives the picked role's number, the
 * WorkspaceRole nerve's WorkspaceMemberUpdate, InvitationCreate and WorkspaceInvitationUpdate take (a string is
 * refused).
 */
export const WORKSPACE_ROLES: WorkspaceRole[] = [5, 15, 20];
