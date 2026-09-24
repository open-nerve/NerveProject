/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TIssuePriorities } from "./issues";

export type TDeDupeIssue = {
  id: string;
  project_id: string;
  sequence_id: number;
  name: string;
  priority: TIssuePriorities;
  state_id: string;
  created_by: string;
};
