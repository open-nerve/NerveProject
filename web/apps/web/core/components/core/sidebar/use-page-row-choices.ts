/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import type { IWorkItemFilterInstance, TWorkItemFilterCondition } from "@nerve/shared-state";
import type { TWorkItemFilterProperty } from "@nerve/types";
import { toFilterArray } from "@nerve/utils";
// hooks
import type { PlanningPageStoreType } from "@/hooks/planning-page-domains";
import { useWorkItemFilters } from "@/hooks/store/work-item-filters/use-work-item-filters";
// local imports
import type { RowChoice } from "./distribution-stats";
import { createFilterUpdateHandler } from "./progress-stats/shared";

/** The values of property the filters' condition `in` holds (findFirstConditionByPropertyAndOperator's, observed). */
const selectedOf = (filter: IWorkItemFilterInstance, property: TWorkItemFilterProperty) =>
  toFilterArray(
    filter.allConditions.find((condition) => condition.property === property && condition.operator === "in")?.value
  ).map(String);

/**
 * What choosing a row of a planning page's progress does (ProgressTabs' choices; P5b spec 2.8, R8; P6b spec 2.7, R14):
 * while the page of the store type and id shows its work items by its filters (the instance its layout root makes),
 * a row of a state group, an assignee or a label toggles its value in them, saved as the page's own change of them, and
 * says whether it is chosen. Without that instance the rows are plain: before the page has read its preferences, with
 * its domain turned off, and in the list page's preview (?peekCycle, ?peekModule), which never has one. Called in the
 * render of an observer, which then follows the filters.
 */
export function usePageRowChoices(storeType: PlanningPageStoreType, id: string) {
  const { getFilter, updateFilterValueFromSidebar } = useWorkItemFilters();
  const filter = getFilter(storeType, id);
  const toggle = (property: TWorkItemFilterProperty): RowChoice => {
    if (!filter) return {};
    const values = selectedOf(filter, property);
    const update = (condition: TWorkItemFilterCondition) => updateFilterValueFromSidebar(storeType, id, condition);
    return { selected: values, choose: createFilterUpdateHandler(property, values, update) };
  };
  return { states: toggle("state_group"), assignees: toggle("assignee_id"), labels: toggle("label_id") };
}
