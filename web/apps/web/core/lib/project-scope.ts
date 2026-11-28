/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Project } from "@nerve/api-client";

/**
 * Whether the caller is the project's member (member_role, null when he is not an active one) and it is not archived:
 * the projects a workspace's work items and its label catalog list (listWorkspaceIssues, listWorkspaceLabels; M7/P6b
 * spec 2.2, 2.3), and so the projects the workspace's pages offer and count. The project's workspace is the caller's
 * to check.
 */
export const joinedAndUnarchived = (project: Project): boolean =>
  project.member_role !== null && project.archived_at === null;
