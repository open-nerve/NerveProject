/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

// nerve imports
import { DEFAULT_WORK_ITEM_FORM_VALUES } from "@nerve/constants";
import type { Project, State } from "@nerve/api-client";
import type { ISearchIssueResponse } from "@nerve/types";

export const getUpdateFormDataForReset = (projectId: string | undefined, formData: Partial<IssueDetail>) => ({
  ...DEFAULT_WORK_ITEM_FORM_VALUES,
  project_id: projectId,
  name: formData.name,
  description_html: formData.description_html,
  priority: formData.priority,
  start_date: formData.start_date,
  target_date: formData.target_date,
});

export const convertWorkItemDataToSearchResponse = (
  workspaceSlug: string,
  workItem: Issue,
  project: Project | undefined,
  state: State | undefined
): ISearchIssueResponse => ({
  id: workItem.id,
  name: workItem.name,
  project_id: workItem.project_id ?? "",
  project__identifier: project?.identifier ?? "",
  project__name: project?.name ?? "",
  sequence_id: workItem.sequence_id,
  state__color: state?.color ?? "",
  start_date: workItem.start_date,
  state__group: state?.group,
  state__name: state?.name ?? "",
  workspace__slug: workspaceSlug,
});

export function getChangedIssuefields<T extends object>(formData: T, dirtyFields: Partial<Record<keyof T, boolean>>) {
  const changed: Partial<T> = {};
  for (const key of Object.keys(dirtyFields) as (keyof T)[]) {
    if (dirtyFields[key]) changed[key] = formData[key];
  }
  return changed;
}
