/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type { IFilterOption, TFilterValue, TSupportedOperators, TBaseFilterFieldConfig } from "@nerve/types";
import { FILTER_FIELD_TYPE } from "@nerve/types";
// local imports
import type { IFilterIconConfig } from "./shared";
import { createFilterFieldConfig } from "./shared";

// ------------ Selection filters ------------

/**
 * Options transformation interface for selection filters
 */
interface TOptionTransforms<TItem, TValue extends TFilterValue = string, TIconData = undefined> {
  items: TItem[];
  getId: (item: TItem) => string;
  getLabel: (item: TItem) => string;
  getValue: (item: TItem) => TValue;
  getIconData?: (item: TItem) => TIconData;
}

/**
 * Multi-select filter configuration
 */
type TMultiSelectConfig<TValue extends TFilterValue = string> = TBaseFilterFieldConfig & {
  additionalOptions?: IFilterOption<TValue>[];
  singleValueOperator: TSupportedOperators;
};

/**
 * Helper to get the multi select config
 * @param transforms - How to transform items into options
 * @param config - Multi-select specific configuration
 * @param iconConfig - Icon configuration for options
 * @returns The multi select config
 */
export const getMultiSelectConfig = <
  TItem,
  TValue extends TFilterValue = string,
  TIconData extends string | number | boolean | object | undefined = undefined,
>(
  transforms: TOptionTransforms<TItem, TValue, TIconData>,
  config: TMultiSelectConfig<TValue>,
  iconConfig?: IFilterIconConfig<TIconData>
) =>
  createFilterFieldConfig<typeof FILTER_FIELD_TYPE.MULTI_SELECT, TValue>({
    type: FILTER_FIELD_TYPE.MULTI_SELECT,
    ...config,
    operatorLabel: config?.operatorLabel,
    getOptions: () => [
      ...(config.additionalOptions ?? []),
      ...transforms.items.map((item) => ({
        id: transforms.getId(item),
        label: transforms.getLabel(item),
        value: transforms.getValue(item),
        icon: iconConfig?.getOptionIcon?.(transforms.getIconData?.(item) as TIconData),
      })),
    ],
  });

// ------------ Date filters ------------

/**
 * Helper to get the date picker config
 * @param config - Date-specific configuration
 * @returns The date picker config
 */
export const getDatePickerConfig = (config: TBaseFilterFieldConfig) =>
  createFilterFieldConfig<typeof FILTER_FIELD_TYPE.DATE, Date>({
    type: FILTER_FIELD_TYPE.DATE,
    ...config,
  });

/**
 * Helper to get the date range picker config
 * @param config - Date range-specific configuration
 * @returns The date range picker config
 */
export const getDateRangePickerConfig = (config: TBaseFilterFieldConfig) =>
  createFilterFieldConfig<typeof FILTER_FIELD_TYPE.DATE_RANGE, Date>({
    type: FILTER_FIELD_TYPE.DATE_RANGE,
    ...config,
  });
