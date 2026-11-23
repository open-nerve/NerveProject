/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { IntakeIssuePage } from "@nerve/api-client";
// hooks
import { useIntake } from "@/hooks/store/use-intake";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// lib
import { intakeEntryOn } from "@/lib/intake-entry";
import { useSessionSWR } from "@/lib/use-session-swr";
// store
import type { IntakeFilters } from "@/store/issue/intake/filters";
import { filterKey, filtersOf } from "@/store/issue/intake/filters";

/** The intake page's list: the pages the store holds of it, and SWR's read of it. */
export type IntakeList = {
  /** The work items the filters select, the pages read so far as one; undefined until read, or ungated. */
  page: IntakeIssuePage | undefined;
  /** The read of its first page, whose error is why nerve could not read it. */
  read: ReturnType<typeof useSessionSWR<IntakeIssuePage | undefined>>;
};

/**
 * The read of the project's intake list the intake page's filters select (P5a spec 2.5), by the session's SWR keyed
 * by the caller's permissions over the route's workspace and the project (identity, so that a change of his role reads
 * it again: a guest sees his own submissions alone), the workspace's id and slug, the project and the filters' key
 * (filterKey), so that a change of filters reads the list they name; the fetcher is given the key's strings alone and
 * fetches the filters they name (filtersOf), never those of the render that made it. Its gate: the route names a
 * workspace of the caller's and the project's intake is one he is offered (intakeEntryOn: turned on, and a role of his
 * in it, an archived project's too); none otherwise. SWR reads it as the page mounts and as the key changes, however
 * soon after its last read of that key (no deduping interval: a confirmation joins no list, R31, which only this read
 * shows), and again as the browser reconnects, which is a refresh: the first page in place of every page held.
 */
export function useIntakeList(projectId: string, filters: IntakeFilters): IntakeList {
  const workspace = useRouteWorkspace();
  const { rootIssueStore: root } = useIssueDetail();
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { fetchList, getList } = useIntake();
  const gated =
    workspace !== null &&
    intakeEntryOn(getProjectById(projectId), getProjectRoleByWorkspaceSlugAndProjectId(workspace.slug, projectId));
  const identity = root.permissionReads.identity(workspace?.id, [projectId], true);
  const read = useSessionSWR(
    gated ? ["INTAKE_ISSUES", identity, workspace.id, workspace.slug, projectId, filterKey(filters)] : null,
    (_identity, _workspaceId, _slug, project, key) => fetchList(project, filtersOf(key)),
    { dedupingInterval: 0 }
  );
  return { page: gated ? getList(projectId, filters) : undefined, read };
}
