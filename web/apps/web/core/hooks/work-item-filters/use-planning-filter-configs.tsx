/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useMemo } from "react";
import { CyclesOutline, ModuleOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import { CycleGroupIcon } from "@nerve/propel/icons";
import type { TFilterConditionNode, TFilterConfig, TFilterValue, TWorkItemFilterProperty } from "@nerve/types";
import { getCycleFilterConfig, getModuleFilterConfig, toFilterArray } from "@nerve/utils";
import { useCycles } from "@/hooks/store/use-cycles";
import { useModules } from "@/hooks/store/use-modules";
import { useProject } from "@/hooks/store/use-project";
import { usePlanningNames } from "@/hooks/use-planning-names";
import { listWordOn } from "@/lib/planning-switch";

/** A condition of a list's filter: the values it holds of a cycle or a module stay options, by their names. */
export type TPlanningFilterCondition = TFilterConditionNode<TWorkItemFilterProperty, TFilterValue>;

type Props = {
  workspaceSlug: string;
  /** The page's project, which the filters' wrapper gives (the route's); none on a page of many projects. */
  projectId: string | undefined;
  /** Whether the page's options have a filter: never the page's own domain's (P7b spec 2.5). */
  isFilterEnabled: (key: TWorkItemFilterProperty) => boolean;
  /** The filter's conditions: those saved and those applied since. */
  conditions: readonly TPlanningFilterCondition[];
};

/**
 * The ids a property's conditions hold, None aside, one array while they stay the same: read as the caller renders,
 * which observes the conditions (a value picked or taken out renders it again), and compared by their text.
 */
function useHeld(conditions: readonly TPlanningFilterCondition[], property: TWorkItemFilterProperty): string[] {
  const text = conditions
    .filter((condition) => condition.property === property)
    .flatMap((condition) => toFilterArray(condition.value).map(String))
    .filter((value) => value !== "none")
    .join(",");
  return useMemo(() => (text ? text.split(",") : []), [text]);
}

/**
 * A filter's values of a domain (P7b spec 2.6, R7): the project's that are not archived, ended ones too (a filter may
 * name any of them, unlike a work item's dropdown), in nerve's order; then those the conditions hold that its list does
 * not give, as the store names them: the archived ones in its archive's order, then any other (created since). Held,
 * they are picked; no other archived one is offered. A held value the store cannot name is none: its condition shows
 * it unavailable.
 */
function filterValues<T>(
  listed: readonly string[] | undefined,
  archived: readonly string[] | undefined,
  held: readonly string[],
  valueOf: (id: string) => T | undefined
): T[] {
  const kept = held.filter((id) => !listed?.includes(id));
  const ids = new Set([...(listed ?? []), ...(archived ?? []).filter((id) => kept.includes(id)), ...kept]);
  return [...ids].flatMap((id) => {
    const value = valueOf(id);
    return value ? [value] : [];
  });
}

/** A value's label: an archived one's name with the caption after it, as the cycle dropdown names one. */
const named = (name: string, archivedAt: string | null, caption: string) => (archivedAt ? `${name}, ${caption}` : name);

/**
 * The filters of a project's list by a work item's cycle and by its modules (P7b spec 2.6, R7, R8): each offered
 * where the page's options have it, while the project has the domain turned on (listWordOn) and the store holds its
 * list; None first, then the domain's values (filterValues). The list is the one the project's wrapper reads, by the
 * key every page shares, while the domain is on (use-project-fetch.ts), as the labels', states' and members' filters
 * take theirs: the filters read none of their own as a page mounts. The values the conditions hold are named through
 * usePlanningNames, which reads the project's archive for an archived one (its list first, by that key, while the
 * store has none), nothing while the domain is off. An applied condition shows whatever the page offers and the switch
 * says (they decide "add filter" alone): its values by the names the store has, else as unavailable, and removable;
 * nerve answers the list by it either way (R8).
 */
export function usePlanningFilterConfigs({
  workspaceSlug,
  projectId,
  isFilterEnabled,
  conditions,
}: Props): TFilterConfig<TWorkItemFilterProperty>[] {
  const { t } = useTranslation();
  const cycleLabel = t("common.filter_options.fields.cycle_id");
  const moduleLabel = t("common.filter_options.fields.module_id");
  const noCycle = t("cycle.no_cycle");
  const noModule = t("module.no_module");
  const caption = t("common.archived");
  // store hooks
  const { getProjectById } = useProject();
  const { getProjectCycleIds, getArchivedCycleIds, getCycleById } = useCycles();
  const { getProjectModuleIds, getArchivedModuleIds, getModuleById } = useModules();
  // the names of the values held
  const heldCycles = useHeld(conditions, "cycle_id");
  const heldModules = useHeld(conditions, "module_id");
  usePlanningNames(workspaceSlug, projectId, "cycles", heldCycles);
  usePlanningNames(workspaceSlug, projectId, "modules", heldModules);
  // derived values
  const project = getProjectById(projectId);
  const listedCycles = getProjectCycleIds(projectId);
  const archivedCycles = getArchivedCycleIds(projectId);
  const listedModules = getProjectModuleIds(projectId);
  const archivedModules = getArchivedModuleIds(projectId);
  const cyclesOffered = isFilterEnabled("cycle_id") && listWordOn(project, "cycle_id") && listedCycles !== undefined;
  const modulesOffered =
    isFilterEnabled("module_id") && listWordOn(project, "module_id") && listedModules !== undefined;

  const cycleFilterConfig = useMemo(
    () =>
      getCycleFilterConfig<TWorkItemFilterProperty>("cycle_id")({
        label: cycleLabel,
        isEnabled: cyclesOffered,
        filterIcon: CyclesOutline,
        cycles: filterValues(listedCycles, archivedCycles, heldCycles, getCycleById),
        getOptionLabel: (cycle) => named(cycle.name, cycle.archived_at, caption),
        additionalOptions: [{ id: "none", value: "none", label: noCycle }],
        getOptionIcon: (status) => <CycleGroupIcon cycleGroup={status} className="size-3.5 flex-shrink-0" />,
      }),
    [cycleLabel, cyclesOffered, listedCycles, archivedCycles, heldCycles, getCycleById, caption, noCycle]
  );

  const moduleFilterConfig = useMemo(
    () =>
      getModuleFilterConfig<TWorkItemFilterProperty>("module_id")({
        label: moduleLabel,
        isEnabled: modulesOffered,
        filterIcon: ModuleOutline,
        modules: filterValues(listedModules, archivedModules, heldModules, getModuleById),
        getOptionLabel: (item) => named(item.name, item.archived_at, caption),
        additionalOptions: [{ id: "none", value: "none", label: noModule }],
        getOptionIcon: () => <ModuleOutline className="size-3.5 flex-shrink-0" />,
      }),
    [moduleLabel, modulesOffered, listedModules, archivedModules, heldModules, getModuleById, caption, noModule]
  );

  return [cycleFilterConfig, moduleFilterConfig];
}
