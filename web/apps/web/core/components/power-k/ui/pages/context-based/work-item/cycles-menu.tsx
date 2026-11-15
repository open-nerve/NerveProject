/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { Command } from "cmdk";
import { observer } from "mobx-react";
import { CyclesOutline } from "@makeplane/propel/icons";
import type { Cycle, Issue } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { CycleGroupIcon } from "@nerve/propel/icons";
import { Spinner } from "@nerve/ui";
// components
import { PowerKMenuEmptyState } from "@/components/power-k/menus/empty-state";
import { PowerKModalCommandItem } from "@/components/power-k/ui/modal/command-item";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import { usePlanningNames } from "@/hooks/use-planning-names";
import { usePlanningReads } from "@/hooks/use-planning-reads";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
// lib
import { problemMessageKey } from "@/lib/error-messages";
import { cycleOptions } from "@/lib/planning-directory";
import { planningListRefusal } from "@/lib/planning-switch";
import { useSessionSWR } from "@/lib/use-session-swr";

type Props = {
  /** The address's workspace, whose role in the work item's project gates the reads. */
  workspaceSlug: string | undefined;
  workItem: Issue;
  /** The cycle picked, null for No cycle; the command sends nothing for the cycle the work item is in. */
  handleSelect: (cycleId: string | null) => void;
};

/** A cycle's icon on the page: its status's, as the cycle dropdown's options show it. */
const icon = (cycle: Cycle) => <CycleGroupIcon cycleGroup={cycle.status} className="size-3.5 shrink-0" />;

/**
 * The page of the work item's command that changes its cycle (P7c spec 2.5, R6): the options of its cycle dropdown
 * (cycleOptions: No cycle first, then the project's cycles it may join, none ended, none archived, in nerve's order),
 * the cycle it is in marked. That cycle, ended or archived, is no option: it shows first by its name, an archived one
 * captioned, as the dropdown's button shows it, named as the planning store has it (usePlanningNames), never to be
 * picked. The project's cycles are read as the page opens, by the read the open menus make (project-cycles-menu.tsx:
 * its key, its gate; the page has no project's wrapper on /browse); it waits for nerve's first answer, or says why
 * nerve could not give it, and while the gate is closed (the project archived, its cycles turned off, no role) why
 * it lists none (planningListRefusal), the cycles the store may still hold too.
 */
export const PowerKWorkItemCyclesMenu = observer(function PowerKWorkItemCyclesMenu(props: Props) {
  const { workspaceSlug, workItem, handleSelect } = props;
  const projectId = workItem.project_id;
  // store hooks
  const { getProjectCycleIds, getCycleById, fetchCycles } = useCycles();
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { t } = useTranslation();
  const refusalMessage = useRefusalMessage();
  const { listed } = usePlanningReads(workspaceSlug, projectId, "cycles");
  const read = useSessionSWR(listed ? ["PROJECT_CYCLES", projectId] : null, (id) => fetchCycles(id));
  usePlanningNames(workspaceSlug, projectId, "cycles", workItem.cycle_id ? [workItem.cycle_id] : []);

  if (!listed) {
    const role = workspaceSlug ? getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) : undefined;
    const refusal = planningListRefusal(getProjectById(projectId), role, "cycles");
    return <PowerKMenuEmptyState emptyText={t(problemMessageKey(refusal))} />;
  }
  const options = cycleOptions(getProjectCycleIds(projectId), getCycleById);
  if (options === undefined)
    return read.error === undefined ? <Spinner /> : <PowerKMenuEmptyState emptyText={refusalMessage(read.error)} />;

  const held = getCycleById(workItem.cycle_id);
  const current = held && !options.some((cycle) => cycle?.id === held.id) ? held : undefined;
  return (
    <Command.Group>
      {current && (
        <PowerKModalCommandItem
          iconNode={icon(current)}
          label={
            <>
              <span className="truncate">{current.name}</span>
              {current.archived_at && (
                <span className="shrink-0 text-caption-sm-regular text-placeholder">
                  <span className="sr-only">, </span>
                  {t("common.archived")}
                </span>
              )}
            </>
          }
          value={current.name}
          isSelected
          isDisabled
          onSelect={() => handleSelect(current.id)}
        />
      )}
      {options.map((cycle) =>
        cycle ? (
          <PowerKModalCommandItem
            key={cycle.id}
            iconNode={icon(cycle)}
            label={cycle.name}
            value={cycle.name}
            isSelected={cycle.id === workItem.cycle_id}
            onSelect={() => handleSelect(cycle.id)}
          />
        ) : (
          <PowerKModalCommandItem
            key="no-cycle"
            icon={CyclesOutline}
            label={t("cycle.no_cycle")}
            value={t("cycle.no_cycle")}
            isSelected={workItem.cycle_id === null}
            onSelect={() => handleSelect(null)}
          />
        )
      )}
    </Command.Group>
  );
});
