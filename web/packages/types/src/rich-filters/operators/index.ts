/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

/**
 * Logical operators
 */
export const LOGICAL_OPERATOR = {
  AND: "and",
} as const;

/**
 * Equality operators
 */
export const EQUALITY_OPERATOR = {
  EXACT: "exact",
} as const;

/**
 * Collection operators
 */
export const COLLECTION_OPERATOR = {
  IN: "in",
} as const;

/**
 * Comparison operators
 */
export const COMPARISON_OPERATOR = {
  RANGE: "range",
} as const;

/**
 * Operators that support multiple values
 */
export const MULTI_VALUE_OPERATORS: ReadonlyArray<TSupportedOperators> = [
  COLLECTION_OPERATOR.IN,
  COMPARISON_OPERATOR.RANGE,
] as const;

/**
 * All operators that can be used in filter conditions
 */
export const OPERATORS = {
  ...EQUALITY_OPERATOR,
  ...COLLECTION_OPERATOR,
  ...COMPARISON_OPERATOR,
} as const;

// -------- TYPES --------

export type TLogicalOperator = (typeof LOGICAL_OPERATOR)[keyof typeof LOGICAL_OPERATOR];

/**
 * Union type representing all operators that can be used in a filter condition.
 */
export type TSupportedOperators = (typeof OPERATORS)[keyof typeof OPERATORS];

/**
 * All operators available for use in rich filters UI, including negated versions.
 */
export type TAllAvailableOperatorsForDisplay = TSupportedOperators;
