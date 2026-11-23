/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Project } from "@nerve/api-client";
// hooks
import { useIntake } from "@/hooks/store/use-intake";
import { useProject } from "@/hooks/store/use-project";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

/**
 * Whether nerve counts the pending work items of the project's intake for the caller (getIntakeCounts): he is its
 * member, an admin, a member or a guest (member_role is not null), it is not archived, and its intake is turned on.
 */
const counted = (project: Project | undefined): boolean =>
  project !== undefined && project.member_role !== null && project.archived_at === null && project.intake_view;

/**
 * The one fetch of the pending work items of the caller's intakes in the route's workspace (P5a spec 2.5, R7), which
 * the sidebar's projects list makes once for every project's intake item, the items reading the store (pendingCount).
 * Its gate: the route names a workspace of the caller's, and at least one of its projects the project store holds
 * (knownProjectIds) is one nerve counts for him (counted); none otherwise. Its key names those projects, their ids
 * sorted and joined, so that turning a project's intake on or off, archiving one or leaving one reads again. SWR
 * fetches it as a page mounts and as its key changes, never as the tab regains focus (R8).
 */
export function useIntakeCounts(): void {
  const workspace = useRouteWorkspace();
  const { knownProjectIds, getProjectById } = useProject();
  const { fetchCounts } = useIntake();
  const projectIds = workspace
    ? knownProjectIds(workspace.id)
        .filter((id) => counted(getProjectById(id)))
        .toSorted()
        .join(",")
    : "";
  useSessionSWR(
    workspace && projectIds ? ["INTAKE_COUNTS", workspace.id, workspace.slug, projectIds] : null,
    (id, slug) => fetchCounts({ id, slug })
  );
}
