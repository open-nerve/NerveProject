/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ILinkDetails } from "../issues";
import type { IIssueFilterOptions } from "../view-props";

export type TModuleStatus = "backlog" | "planned" | "in-progress" | "paused" | "completed" | "cancelled";

export type TModuleCompletionChartDistribution = {
  [key: string]: number | null;
};

type TModuleDistributionBase = {
  total_issues: number;
  pending_issues: number;
  completed_issues: number;
};

type TModuleAssigneesDistribution = {
  assignee_id: string | null;
  avatar_url: string | null;
  first_name: string | null;
  last_name: string | null;
  display_name: string | null;
};

type TModuleLabelsDistribution = {
  color: string | null;
  label_id: string | null;
  label_name: string | null;
};

export type TModuleDistribution = {
  assignees: (TModuleAssigneesDistribution & TModuleDistributionBase)[];
  completion_chart: TModuleCompletionChartDistribution;
  labels: (TModuleLabelsDistribution & TModuleDistributionBase)[];
};

export interface IModule {
  total_issues: number;
  completed_issues: number;
  backlog_issues: number;
  started_issues: number;
  unstarted_issues: number;
  cancelled_issues: number;
  distribution?: TModuleDistribution;

  id: string;
  name: string;
  description: string;
  description_text: any;
  description_html: any;
  workspace_id: string;
  project_id: string;
  lead_id: string | null;
  member_ids: string[];
  link_module?: ILinkDetails[];
  sub_issues?: number;
  is_favorite: boolean;
  sort_order: number;
  view_props: {
    filters: IIssueFilterOptions;
  };
  status?: TModuleStatus;
  archived_at: string | null;
  start_date: string | null;
  target_date: string | null;
  created_at: string;
  updated_at: string;
  created_by?: string;
  updated_by?: string;
}

export type ModuleLink = {
  title: string;
  url: string;
};
