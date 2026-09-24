/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type { TIssuePriorities } from "@nerve/constants";
import { ISSUE_PRIORITIES } from "@nerve/constants";
import type { TFilterProperty, TSupportedOperators } from "@nerve/types";
import { EQUALITY_OPERATOR, COLLECTION_OPERATOR } from "@nerve/types";
// local imports
import type { TCreateFilterConfigParams, IFilterIconConfig, TCreateFilterConfig } from "../../../rich-filters";
import { createFilterConfig, getMultiSelectConfig, createOperatorConfigEntry } from "../../../rich-filters";

// ------------ Priority filter ------------

/**
 * Priority filter specific params
 */
type TCreatePriorityFilterParams = TCreateFilterConfigParams & IFilterIconConfig<TIssuePriorities>;

/**
 * Helper to get the priority multi select config
 * @param params - The filter params
 * @returns The priority multi select config
 */
const getPriorityMultiSelectConfig = (params: TCreatePriorityFilterParams, singleValueOperator: TSupportedOperators) =>
  getMultiSelectConfig<{ key: TIssuePriorities; title: string }, TIssuePriorities, TIssuePriorities>(
    {
      items: ISSUE_PRIORITIES,
      getId: (priority) => priority.key,
      getLabel: (priority) => priority.title,
      getValue: (priority) => priority.key,
      getIconData: (priority) => priority.key,
    },
    {
      singleValueOperator,
      ...params,
    },
    {
      getOptionIcon: params.getOptionIcon,
    }
  );

/**
 * Get the priority filter config
 * @template K - The filter key
 * @param key - The filter key to use
 * @returns A function that takes parameters and returns the priority filter config
 */
export const getPriorityFilterConfig =
  <P extends TFilterProperty>(key: P): TCreateFilterConfig<P, TCreatePriorityFilterParams> =>
  (params: TCreatePriorityFilterParams) =>
    createFilterConfig<P>({
      id: key,
      label: "Priority",
      ...params,
      icon: params.filterIcon,
      supportedOperatorConfigsMap: new Map([
        createOperatorConfigEntry(COLLECTION_OPERATOR.IN, params, (updatedParams) =>
          getPriorityMultiSelectConfig(updatedParams, EQUALITY_OPERATOR.EXACT)
        ),
      ]),
    });
