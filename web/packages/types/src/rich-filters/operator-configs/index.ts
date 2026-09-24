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
import type { EQUALITY_OPERATOR, COLLECTION_OPERATOR, COMPARISON_OPERATOR } from "../operators";

// ----------------------------- EXACT Operator -----------------------------
type TExactOperatorConfigs = TSingleSelectFilterFieldConfig<TFilterValue> | TDateFilterFieldConfig<TFilterValue>;

// ----------------------------- IN Operator -----------------------------
type TInOperatorConfigs = TMultiSelectFilterFieldConfig<TFilterValue>;

// ----------------------------- RANGE Operator -----------------------------
type TRangeOperatorConfigs = TDateRangeFilterFieldConfig<TFilterValue>;

// ----------------------------- Operator Specific Configs -----------------------------

/**
 * Type-safe mapping of specific operators to their supported filter type configurations.
 */
export type TOperatorSpecificConfigs = {
  [EQUALITY_OPERATOR.EXACT]: TExactOperatorConfigs;
  [COLLECTION_OPERATOR.IN]: TInOperatorConfigs;
  [COMPARISON_OPERATOR.RANGE]: TRangeOperatorConfigs;
};

/**
 * Operator filter configuration mapping - for different operators.
 * Provides type-safe mapping of operators to their specific supported configurations.
 */
export type TOperatorConfigMap = Map<
  keyof TOperatorSpecificConfigs,
  TOperatorSpecificConfigs[keyof TOperatorSpecificConfigs]
>;
