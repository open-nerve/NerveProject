/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type { Cycle } from "@nerve/api-client";
import type { IFilterOption, TFilterProperty } from "@nerve/types";
import { EQUALITY_OPERATOR, COLLECTION_OPERATOR } from "@nerve/types";
// local imports
import type { TCreateFilterConfigParams, IFilterIconConfig, TCreateFilterConfig } from "../../../rich-filters";
import { createFilterConfig, getMultiSelectConfig, createOperatorConfigEntry } from "../../../rich-filters";

/**
 * Cycle filter specific params: the cycles it offers, each by its label (its name unless getOptionLabel says) and its
 * status's icon, after the additional options (None).
 */
type TCreateCycleFilterParams = TCreateFilterConfigParams &
  IFilterIconConfig<Cycle["status"]> & {
    cycles: Cycle[];
    getOptionLabel?: (cycle: Cycle) => string;
    additionalOptions?: IFilterOption<string>[];
  };

/**
 * Helper to get the cycle multi select config
 * @param params - The filter params
 * @returns The cycle multi select config
 */
const getCycleMultiSelectConfig = (params: TCreateCycleFilterParams) =>
  getMultiSelectConfig<Cycle, string, Cycle["status"]>(
    {
      items: params.cycles,
      getId: (cycle) => cycle.id,
      getLabel: (cycle) => params.getOptionLabel?.(cycle) ?? cycle.name,
      getValue: (cycle) => cycle.id,
      getIconData: (cycle) => cycle.status,
    },
    {
      singleValueOperator: EQUALITY_OPERATOR.EXACT,
      ...params,
    },
    {
      getOptionIcon: params.getOptionIcon,
    }
  );

/**
 * Get the cycle filter config
 * @template K - The filter key
 * @param key - The filter key to use
 * @returns A function that takes parameters and returns the cycle filter config
 */
export const getCycleFilterConfig =
  <P extends TFilterProperty>(key: P): TCreateFilterConfig<P, TCreateCycleFilterParams> =>
  (params: TCreateCycleFilterParams) =>
    createFilterConfig<P>({
      id: key,
      label: "Cycle",
      ...params,
      icon: params.filterIcon,
      supportedOperatorConfigsMap: new Map([
        createOperatorConfigEntry(COLLECTION_OPERATOR.IN, params, (updatedParams) =>
          getCycleMultiSelectConfig(updatedParams)
        ),
      ]),
    });
