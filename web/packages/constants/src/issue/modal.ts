/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

// nerve imports

export const DEFAULT_WORK_ITEM_FORM_VALUES: Partial<IssueDetail> = {
  project_id: "",
  name: "",
  description_html: "",
  state_id: "",
  parent_id: null,
  priority: "none",
  assignee_ids: [],
  label_ids: [],
  start_date: null,
  target_date: null,
};
