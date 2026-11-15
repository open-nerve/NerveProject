/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import type { Issue, Module } from "@nerve/api-client";
import { Spinner } from "@nerve/ui";
// components
import { PowerKMenuEmptyState } from "@/components/power-k/menus/empty-state";
import { PowerKModulesMenu } from "@/components/power-k/menus/modules";
import { PowerKPlanningListRefusal } from "@/components/power-k/menus/planning-list-refusal";
// hooks
import { useModules } from "@/hooks/store/use-modules";
import { usePlanningNames } from "@/hooks/use-planning-names";
import { usePlanningReads } from "@/hooks/use-planning-reads";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
// lib
import { moduleOptions } from "@/lib/planning-directory";
import { useSessionSWR } from "@/lib/use-session-swr";

type Props = {
  /** The address's workspace, whose role in the work item's project gates the reads. */
  workspaceSlug: string | undefined;
  workItem: Issue;
  /** The module picked: the command adds the work item to it, or takes it out of it if it is in it. */
  handleSelect: (moduleId: string) => void;
};

/**
 * The page of the work item's command that changes its modules (P7c spec 2.5, R6): the options of its module dropdown
 * (moduleOptions: the project's modules not archived, whatever their status, in nerve's order), then the archived
 * modules it is in, captioned, there only to be taken out, as the dropdown lists them (module/base.tsx), named as the
 * planning store has them (usePlanningNames); those it is in ticked as nerve confirmed them (the work item the store
 * holds), never ahead of its answer. The project's modules are read as the page opens, by the read the open menus make
 * (project-modules-menu.tsx: its key, its gate; the page has no project's wrapper on /browse); it waits for nerve's
 * first answer, or says why nerve could not give it, and while the gate is closed (the project archived, its modules
 * turned off, no role) why it lists none (PowerKPlanningListRefusal), the modules the store may still hold too.
 */
export const PowerKWorkItemModulesMenu = observer(function PowerKWorkItemModulesMenu(props: Props) {
  const { workspaceSlug, workItem, handleSelect } = props;
  const projectId = workItem.project_id;
  // store hooks
  const { getProjectModuleIds, getModuleById, fetchModules } = useModules();
  const refusalMessage = useRefusalMessage();
  const { listed } = usePlanningReads(workspaceSlug, projectId, "modules");
  const read = useSessionSWR(listed ? ["PROJECT_MODULES", projectId] : null, (id) => fetchModules(id));
  usePlanningNames(workspaceSlug, projectId, "modules", workItem.module_ids);

  if (!listed)
    return <PowerKPlanningListRefusal workspaceSlug={workspaceSlug} projectId={projectId} domain="modules" />;
  const options = moduleOptions(getProjectModuleIds(projectId), getModuleById);
  if (options === undefined)
    return read.error === undefined ? <Spinner /> : <PowerKMenuEmptyState emptyText={refusalMessage(read.error)} />;

  // The archived modules it is in, after the options, which never have them: picked, they only take it out.
  const archived = workItem.module_ids
    .map((moduleId) => getModuleById(moduleId))
    .filter((item): item is Module => !!item?.archived_at);
  return (
    <PowerKModulesMenu
      modules={[...options, ...archived]}
      value={workItem.module_ids}
      onSelect={(item) => handleSelect(item.id)}
    />
  );
});
