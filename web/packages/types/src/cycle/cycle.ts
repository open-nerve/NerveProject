/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IIssueFilterOptions } from "../view-props";

export type TCycleGroups = "current" | "upcoming" | "completed" | "draft";

type TCycleCompletionChartDistribution = {
  [key: string]: number | null;
};

type TCycleDistributionBase = {
  total_issues: number;
  pending_issues: number;
  completed_issues: number;
};

type TCycleAssigneesDistribution = {
  assignee_id: string | null;
  avatar_url: string | null;
  first_name: string | null;
  last_name: string | null;
  display_name: string | null;
};

type TCycleLabelsDistribution = {
  color: string | null;
  label_id: string | null;
  label_name: string | null;
};

export type TCycleDistribution = {
  assignees: (TCycleAssigneesDistribution & TCycleDistributionBase)[];
  completion_chart: TCycleCompletionChartDistribution;
  labels: (TCycleLabelsDistribution & TCycleDistributionBase)[];
};

export type TProgressSnapshot = {
  total_issues: number;
  completed_issues: number;
  backlog_issues: number;
  started_issues: number;
  unstarted_issues: number;
  cancelled_issues: number;
  distribution?: TCycleDistribution;
};

interface IProjectDetails {
  id: string;
}

export interface ICycle extends TProgressSnapshot {
  progress_snapshot: TProgressSnapshot | undefined;

  created_at?: string;
  created_by?: string;
  description: string;
  end_date: string | null;
  id: string;
  is_favorite?: boolean;
  name: string;
  owned_by_id: string;
  project_id: string;
  status?: TCycleGroups;
  sort_order: number;
  start_date: string | null;
  sub_issues?: number;
  updated_at?: string;
  updated_by?: string;
  archived_at: string | null;
  assignee_ids?: string[];
  view_props: {
    filters: IIssueFilterOptions;
  };
  workspace_id: string;
  project_detail: IProjectDetails;
}

export type CycleDateCheckData = {
  start_date: string;
  end_date: string;
  cycle_id?: string;
};
