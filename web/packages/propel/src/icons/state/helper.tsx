/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { StateGroup } from "@nerve/api-client";
import { EIconSize } from "@nerve/constants";

export interface IStateGroupIcon {
  className?: string;
  color?: string;
  stateGroup: StateGroup;
  size?: EIconSize;
  percentage?: number;
}

export const STATE_GROUP_COLORS: {
  [key in StateGroup]: string;
} = {
  backlog: "#60646C",
  unstarted: "#60646C",
  started: "#F59E0B",
  completed: "#46A758",
  cancelled: "#9AA4BC",
};

export const STATE_GROUP_SIZES: {
  [key in EIconSize]: string;
} = {
  [EIconSize.XS]: "10px",
  [EIconSize.SM]: "12px",
  [EIconSize.MD]: "14px",
  [EIconSize.LG]: "16px",
  [EIconSize.XL]: "18px",
};
