/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TSupportedOperators, TSupportedDateFilterOperators } from "@nerve/types";
import { EQUALITY_OPERATOR, COLLECTION_OPERATOR, COMPARISON_OPERATOR } from "@nerve/types";

/**
 * Empty operator label for unselected state
 */
export const EMPTY_OPERATOR_LABEL = "--";

/**
 * Operator labels
 */
export const OPERATOR_LABELS_MAP: Record<TSupportedOperators, string> = {
  [EQUALITY_OPERATOR.EXACT]: "is",
  [COLLECTION_OPERATOR.IN]: "is any of",
  [COMPARISON_OPERATOR.RANGE]: "between",
} as const;

/**
 * Date-specific operator labels
 */
export const DATE_OPERATOR_LABELS_MAP: Record<TSupportedDateFilterOperators, string> = {
  [EQUALITY_OPERATOR.EXACT]: "is",
  [COMPARISON_OPERATOR.RANGE]: "between",
} as const;
