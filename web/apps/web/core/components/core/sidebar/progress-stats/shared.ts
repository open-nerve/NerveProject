/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TWorkItemFilterCondition } from "@nerve/shared-state";
import type { TWorkItemFilterProperty } from "@nerve/types";

export const PROGRESS_STATS = [
  {
    key: "stat-states",
    i18n_title: "common.states",
  },
  {
    key: "stat-assignees",
    i18n_title: "common.assignees",
  },
  {
    key: "stat-labels",
    i18n_title: "common.labels",
  },
];

/**
 * The choice of a row of a progress's breakdown on a page whose filters it toggles (P5b spec 2.8): the row's value
 * joins the values of property chosen, or leaves them when it is one, and handleFiltersUpdate is given the condition
 * `in` of what that leaves (none: the page drops the condition, updateFilterValueFromSidebar).
 */
export const createFilterUpdateHandler =
  (
    property: TWorkItemFilterProperty,
    selectedValues: string[],
    handleFiltersUpdate: (condition: TWorkItemFilterCondition) => void
  ) =>
  (value: string) =>
    handleFiltersUpdate({
      property,
      operator: "in",
      value: selectedValues.includes(value)
        ? selectedValues.filter((selected) => selected !== value)
        : [...selectedValues, value],
    });
