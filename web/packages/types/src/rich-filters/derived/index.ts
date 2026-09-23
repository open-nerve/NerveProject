/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TFilterValue } from "../expression";
import type {
  TDateFilterFieldConfig,
  TDateRangeFilterFieldConfig,
  TSingleSelectFilterFieldConfig,
  TMultiSelectFilterFieldConfig,
} from "../field-types";
import type { TOperatorSpecificConfigs } from "../operator-configs";
import type { TFilterOperatorHelper } from "./shared";

// -------- DATE FILTER OPERATORS --------

/**
 * Union type representing all operators that support single date filter types.
 */
export type TSupportedSingleDateFilterOperators<V extends TFilterValue = TFilterValue> = {
  [K in keyof TOperatorSpecificConfigs]: TFilterOperatorHelper<TOperatorSpecificConfigs, K, TDateFilterFieldConfig<V>>;
}[keyof TOperatorSpecificConfigs];

/**
 * Union type representing all operators that support range date filter types.
 */
export type TSupportedRangeDateFilterOperators<V extends TFilterValue = TFilterValue> = {
  [K in keyof TOperatorSpecificConfigs]: TFilterOperatorHelper<
    TOperatorSpecificConfigs,
    K,
    TDateRangeFilterFieldConfig<V>
  >;
}[keyof TOperatorSpecificConfigs];

/**
 * Union type representing all operators that support date filter types.
 */
export type TSupportedDateFilterOperators<V extends TFilterValue = TFilterValue> =
  | TSupportedSingleDateFilterOperators<V>
  | TSupportedRangeDateFilterOperators<V>;

export type TAllAvailableDateFilterOperatorsForDisplay<V extends TFilterValue = TFilterValue> =
  TSupportedDateFilterOperators<V>;

// -------- SELECT FILTER OPERATORS --------

/**
 * Union type representing all operators that support single select filter types.
 */
export type TSupportedSingleSelectFilterOperators<V extends TFilterValue = TFilterValue> = {
  [K in keyof TOperatorSpecificConfigs]: TFilterOperatorHelper<
    TOperatorSpecificConfigs,
    K,
    TSingleSelectFilterFieldConfig<V>
  >;
}[keyof TOperatorSpecificConfigs];

/**
 * Union type representing all operators that support multi select filter types.
 */
export type TSupportedMultiSelectFilterOperators<V extends TFilterValue = TFilterValue> = {
  [K in keyof TOperatorSpecificConfigs]: TFilterOperatorHelper<
    TOperatorSpecificConfigs,
    K,
    TMultiSelectFilterFieldConfig<V>
  >;
}[keyof TOperatorSpecificConfigs];

/**
 * Union type representing all operators that support any select filter types.
 */
export type TSupportedSelectFilterOperators<V extends TFilterValue = TFilterValue> =
  | TSupportedSingleSelectFilterOperators<V>
  | TSupportedMultiSelectFilterOperators<V>;

export type TAllAvailableSelectFilterOperatorsForDisplay<V extends TFilterValue = TFilterValue> =
  TSupportedSelectFilterOperators<V>;

// -------- RE-EXPORTS --------

export * from "./shared";
