/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TStateGroups } from "../state";

export enum EUserProjectRoles {
  ADMIN = 20,
  MEMBER = 15,
  GUEST = 5,
}

export type TProjectIssuesSearchParams = {
  search: string;
  parent?: boolean;
  issue_relation?: boolean;
  cycle?: boolean;
  module?: string;
  sub_issue?: boolean;
  issue_id?: string;
  workspace_search: boolean;
  target_date?: string;
};

export interface ISearchIssueResponse {
  id: string;
  name: string;
  project_id: string;
  project__identifier: string;
  project__name: string;
  sequence_id: number;
  start_date: string | null;
  state__color: string;
  state__group: TStateGroups;
  state__name: string;
  workspace__slug: string;
}
