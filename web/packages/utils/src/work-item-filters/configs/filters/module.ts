/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type { Module } from "@nerve/api-client";
import type { IFilterOption, TFilterProperty } from "@nerve/types";
import { EQUALITY_OPERATOR, COLLECTION_OPERATOR } from "@nerve/types";
// local imports
import type { TCreateFilterConfigParams, IFilterIconConfig, TCreateFilterConfig } from "../../../rich-filters";
import { createFilterConfig, getMultiSelectConfig, createOperatorConfigEntry } from "../../../rich-filters";

/**
 * Module filter specific params: the modules it offers, each by its label (its name unless getOptionLabel says) and the
 * one icon of a module, after the additional options (None).
 */
type TCreateModuleFilterParams = TCreateFilterConfigParams &
  IFilterIconConfig<undefined> & {
    modules: Module[];
    getOptionLabel?: (module: Module) => string;
    additionalOptions?: IFilterOption<string>[];
  };

/**
 * Helper to get the module multi select config
 * @param params - The filter params
 * @returns The module multi select config
 */
const getModuleMultiSelectConfig = (params: TCreateModuleFilterParams) =>
  getMultiSelectConfig<Module, string, undefined>(
    {
      items: params.modules,
      getId: (module) => module.id,
      getLabel: (module) => params.getOptionLabel?.(module) ?? module.name,
      getValue: (module) => module.id,
      getIconData: () => undefined,
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
 * Get the module filter config
 * @template K - The filter key
 * @param key - The filter key to use
 * @returns A function that takes parameters and returns the module filter config
 */
export const getModuleFilterConfig =
  <P extends TFilterProperty>(key: P): TCreateFilterConfig<P, TCreateModuleFilterParams> =>
  (params: TCreateModuleFilterParams) =>
    createFilterConfig<P>({
      id: key,
      label: "Module",
      ...params,
      icon: params.filterIcon,
      supportedOperatorConfigsMap: new Map([
        createOperatorConfigEntry(COLLECTION_OPERATOR.IN, params, (updatedParams) =>
          getModuleMultiSelectConfig(updatedParams)
        ),
      ]),
    });
