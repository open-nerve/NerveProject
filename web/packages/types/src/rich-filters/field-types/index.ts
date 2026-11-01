/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TFilterValue } from "../expression";
import type { TSupportedOperators } from "../operators";
import type { TBaseFilterFieldConfig, IFilterOption } from "./shared";

/**
 * Filter types
 */
export const FILTER_FIELD_TYPE = {
  DATE: "date",
  DATE_RANGE: "date_range",
  SINGLE_SELECT: "single_select",
  MULTI_SELECT: "multi_select",
} as const;

// -------- DATE FILTER CONFIGURATIONS --------

type TBaseDateFilterFieldConfig = TBaseFilterFieldConfig & {
  min?: Date;
  max?: Date;
};

/**
 * Date filter configuration - for temporal filtering.
 * - min: Minimum allowed date
 * - max: Maximum allowed date
 */
export type TDateFilterFieldConfig = TBaseDateFilterFieldConfig & {
  type: typeof FILTER_FIELD_TYPE.DATE;
};

/**
 * Date range filter configuration - for temporal filtering.
 * - min: Minimum allowed date
 * - max: Maximum allowed date
 */
export type TDateRangeFilterFieldConfig = TBaseDateFilterFieldConfig & {
  type: typeof FILTER_FIELD_TYPE.DATE_RANGE;
};

// -------- SELECT FILTER CONFIGURATIONS --------

/**
 * Single-select filter configuration - dropdown with one selectable option.
 * - getOptions: Options as static array or async function
 */
export type TSingleSelectFilterFieldConfig<V extends TFilterValue> = TBaseFilterFieldConfig & {
  type: typeof FILTER_FIELD_TYPE.SINGLE_SELECT;
  getOptions: IFilterOption<V>[] | (() => IFilterOption<V>[] | Promise<IFilterOption<V>[]>);
};

/**
 * Multi-select filter configuration - allows selecting multiple options.
 * - getOptions: Options as static array or async function
 * - singleValueOperator: Operator to show when single value is selected
 */
export type TMultiSelectFilterFieldConfig<V extends TFilterValue> = TBaseFilterFieldConfig & {
  type: typeof FILTER_FIELD_TYPE.MULTI_SELECT;
  getOptions: IFilterOption<V>[] | (() => IFilterOption<V>[] | Promise<IFilterOption<V>[]>);
  singleValueOperator: TSupportedOperators;
};

// -------- UNION TYPES --------

/**
 * All supported filter configurations
 */
export type TSupportedFilterFieldConfigs<V extends TFilterValue = TFilterValue> =
  | TDateFilterFieldConfig
  | TDateRangeFilterFieldConfig
  | TSingleSelectFilterFieldConfig<V>
  | TMultiSelectFilterFieldConfig<V>;

export type TFilterFieldType = (typeof FILTER_FIELD_TYPE)[keyof typeof FILTER_FIELD_TYPE];

// -------- RE-EXPORTS --------

export * from "./shared";
