/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

// nerve imports
import { DEFAULT_WORK_ITEM_FORM_VALUES } from "@nerve/constants";

export const getUpdateFormDataForReset = (projectId: string | undefined, formData: Partial<IssueDetail>) => ({
  ...DEFAULT_WORK_ITEM_FORM_VALUES,
  project_id: projectId,
  name: formData.name,
  description_html: formData.description_html,
  priority: formData.priority,
  start_date: formData.start_date,
  target_date: formData.target_date,
});

export function getChangedIssuefields<T extends object>(formData: T, dirtyFields: Partial<Record<keyof T, boolean>>) {
  const changed: Partial<T> = {};
  for (const key of Object.keys(dirtyFields) as (keyof T)[]) {
    if (dirtyFields[key]) changed[key] = formData[key];
  }
  return changed;
}
