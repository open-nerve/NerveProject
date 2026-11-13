/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve types
import type { Module } from "@nerve/api-client";
import { Spinner } from "@nerve/ui";
// components
import type { TPowerKContext } from "@/components/power-k/core/types";
import { PowerKMenuEmptyState } from "@/components/power-k/menus/empty-state";
import { PowerKModulesMenu } from "@/components/power-k/menus/modules";
// hooks
import { useModules } from "@/hooks/store/use-modules";
import { usePlanningReads } from "@/hooks/use-planning-reads";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

type Props = {
  context: TPowerKContext;
  handleSelect: (module: Module) => void;
};

/**
 * The modules of the route's project, to open one of (P6a spec 2.3, 2.9): nerve's list of its modules not archived, by
 * the read the project's pages make (its wrapper's, use-project-fetch.ts, whose key this one shares: the archives' pages
 * have no wrapper), a member's of a project not archived whose modules are on. It waits for nerve's first answer, or
 * says why nerve could not give it.
 */
export const PowerKOpenProjectModulesMenu = observer(function PowerKOpenProjectModulesMenu(props: Props) {
  const { context, handleSelect } = props;
  const { workspaceSlug, projectId } = context.params;
  // store hooks
  const { getProjectModuleIds, getModuleById, fetchModules } = useModules();
  const refusalMessage = useRefusalMessage();
  // a member's read, of a project not archived whose modules are on: as the project's wrapper reads them
  const { listed } = usePlanningReads(workspaceSlug, projectId, "modules");
  const read = useSessionSWR(listed ? ["PROJECT_MODULES", projectId] : null, (id) => fetchModules(id));
  const moduleIds = getProjectModuleIds(projectId);

  if (moduleIds === undefined)
    return read.error === undefined ? <Spinner /> : <PowerKMenuEmptyState emptyText={refusalMessage(read.error)} />;

  const modules = moduleIds.map((moduleId) => getModuleById(moduleId)).filter((module) => module !== undefined);
  return <PowerKModulesMenu modules={modules} onSelect={handleSelect} />;
});
