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
import { PowerKPlanningListRefusal } from "@/components/power-k/menus/planning-list-refusal";
import { PowerKModalCommandItem } from "@/components/power-k/ui/modal/command-item";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
import { usePlanningNames } from "@/hooks/use-planning-names";
import { usePlanningReads } from "@/hooks/use-planning-reads";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
// lib
import { cycleOptions } from "@/lib/planning-directory";
import { useSessionSWR } from "@/lib/use-session-swr";

type Props = {
  /** The address's workspace, whose role in the work item's project gates the reads. */
  workspaceSlug: string | undefined;
  workItem: Issue;
  /** The cycle picked, null for No cycle; the command sends nothing for the cycle the work item is in. */
  handleSelect: (cycleId: string | null) => void;
};

/** No cycle's value to cmdk: never a cycle's, whose ids are UUIDs. */
const NO_CYCLE = "no-cycle";

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
 * it lists none (PowerKPlanningListRefusal), the cycles the store may still hold too. A row is valued by its cycle's
 * id, No cycle by its own, and found by its name (powerKFilter): names need not be unique (re-77 M1).
 */
export const PowerKWorkItemCyclesMenu = observer(function PowerKWorkItemCyclesMenu(props: Props) {
  const { workspaceSlug, workItem, handleSelect } = props;
  const projectId = workItem.project_id;
  // store hooks
  const { getProjectCycleIds, getCycleById, fetchCycles } = useCycles();
  const { t } = useTranslation();
  const refusalMessage = useRefusalMessage();
  const { listed } = usePlanningReads(workspaceSlug, projectId, "cycles");
  const read = useSessionSWR(listed ? ["PROJECT_CYCLES", projectId] : null, (id) => fetchCycles(id));
  usePlanningNames(workspaceSlug, projectId, "cycles", workItem.cycle_id ? [workItem.cycle_id] : []);

  if (!listed) return <PowerKPlanningListRefusal workspaceSlug={workspaceSlug} projectId={projectId} domain="cycles" />;
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
          value={current.id}
          keywords={[current.name]}
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
            value={cycle.id}
            keywords={[cycle.name]}
            isSelected={cycle.id === workItem.cycle_id}
            onSelect={() => handleSelect(cycle.id)}
          />
        ) : (
          <PowerKModalCommandItem
            key="no-cycle"
            icon={CyclesOutline}
            label={t("cycle.no_cycle")}
            value={NO_CYCLE}
            keywords={[t("cycle.no_cycle")]}
            isSelected={workItem.cycle_id === null}
            onSelect={() => handleSelect(null)}
          />
        )
      )}
    </Command.Group>
  );
});
