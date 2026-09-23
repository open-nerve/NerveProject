/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FC, ReactNode, SVGAttributes } from "react";
import type { TFilterProperty } from "../expression";
import type { TOperatorConfigMap } from "../operator-configs";

/**
 * Main filter configuration type for different properties.
 * This is the primary configuration type used throughout the application.
 *
 * @template P - Property key type (e.g., 'state_id', 'priority', 'assignee')
 * @template V - Value type for the filter
 */
export type TFilterConfig<P extends TFilterProperty> = {
  id: P;
  label: string;
  icon?: FC<SVGAttributes<SVGElement>>;
  isEnabled: boolean;
  allowMultipleFilters?: boolean;
  supportedOperatorConfigsMap: TOperatorConfigMap;
  rightContent?: ReactNode; // content to display on the right side of the filter option in the dropdown
  tooltipContent?: ReactNode; // content to display when hovering over the applied filter item in the filter list
};
