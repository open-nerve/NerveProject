/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve types
import type { Cycle } from "@nerve/api-client";
import { Spinner } from "@nerve/ui";
// components
import type { TPowerKContext } from "@/components/power-k/core/types";
import { PowerKCyclesMenu } from "@/components/power-k/menus/cycles";
import { PowerKMenuEmptyState } from "@/components/power-k/menus/empty-state";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

type Props = {
  context: TPowerKContext;
  handleSelect: (cycle: Cycle) => void;
};

/**
 * The cycles of the route's project, to open one of (P5a spec 2.7): nerve's list of its cycles not archived, by the
 * read the project's pages make (its wrapper's, use-project-fetch.ts, whose key this one shares: the archives' pages
 * have no wrapper), a member's of a project not archived whose cycles are on. It waits for nerve's first answer, or
 * says why nerve could not give it.
 */
export const PowerKOpenProjectCyclesMenu = observer(function PowerKOpenProjectCyclesMenu(props: Props) {
  const { context, handleSelect } = props;
  const { workspaceSlug, projectId } = context.params;
  // store hooks
  const { getProjectCycleIds, getCycleById, fetchCycles } = useCycles();
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const refusalMessage = useRefusalMessage();
  // a member's read, of a project not archived whose cycles are on: as the project's wrapper reads them
  const project = getProjectById(projectId);
  const role = workspaceSlug ? getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) : undefined;
  const listed = role !== undefined && project?.archived_at === null && project.cycle_view;
  const read = useSessionSWR(listed ? ["PROJECT_CYCLES", projectId] : null, (id) => fetchCycles(id));
  const cycleIds = getProjectCycleIds(projectId);

  if (cycleIds === undefined)
    return read.error === undefined ? <Spinner /> : <PowerKMenuEmptyState emptyText={refusalMessage(read.error)} />;

  const cycles = cycleIds.map((cycleId) => getCycleById(cycleId)).filter((cycle) => cycle !== undefined);
  return <PowerKCyclesMenu cycles={cycles} onSelect={handleSelect} />;
});
