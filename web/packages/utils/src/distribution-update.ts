/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { format } from "date-fns";
import { get, set } from "lodash-es";
// plane imports
import { COMPLETED_STATE_GROUPS, STATE_DISTRIBUTION } from "@plane/constants";
import type { ICycle, IModule, IState, TIssue } from "@plane/types";
// helper
import { getDate } from "./datetime";

export type DistributionObjectUpdate = {
  id: string;
  completed_issues?: number;
  pending_issues?: number;
  total_issues: number;
};

type ChartUpdates = {
  updates: {
    path: string[];
    value: number;
  }[];
  isCompleted?: boolean;
};

export type DistributionUpdates = {
  pathUpdates: { path: string[]; value: number }[];
  assigneeUpdates: DistributionObjectUpdate[];
  labelUpdates: DistributionObjectUpdate[];
};

/**
 * Get Distribution updates with the help of previous and next issue states
 * @param prevIssueState
 * @param nextIssueState
 * @param stateMap
 * @returns
 */
export const getDistributionPathsPostUpdate = (
  prevIssueState: TIssue | undefined,
  nextIssueState: TIssue | undefined,
  stateMap: Record<string, IState>
): DistributionUpdates => {
  const prevIssueDistribution = getDistributionDataOfIssue(prevIssueState, -1, stateMap);
  const nextIssueDistribution = getDistributionDataOfIssue(nextIssueState, 1, stateMap);

  const prevChartDistribution = prevIssueDistribution.chartUpdates;
  const nextChartDistribution = nextIssueDistribution.chartUpdates;

  let chartUpdates: {
    path: string[];
    value: number;
  }[];

  // if the completed status of chart updates are same the get chart updates from both the issue states
  if (prevChartDistribution.isCompleted === nextChartDistribution.isCompleted) {
    chartUpdates = [...prevChartDistribution.updates, ...nextChartDistribution.updates];
  } // if not the get chart updates from only the next update
  else {
    chartUpdates = [...nextChartDistribution.updates];
  }

  // merge the updates from both issue states into a single object
  return {
    pathUpdates: [...prevIssueDistribution.pathUpdates, ...nextIssueDistribution.pathUpdates, ...chartUpdates],
    assigneeUpdates: [...prevIssueDistribution.assigneeUpdates, ...nextIssueDistribution.assigneeUpdates],
    labelUpdates: [...prevIssueDistribution.labelUpdates, ...nextIssueDistribution.labelUpdates],
  };
};

/**
 * Get Distribution update for a single issue state
 * @param issue
 * @param multiplier
 * @param stateMap
 * @returns
 */
const getDistributionDataOfIssue = (
  issue: TIssue | undefined,
  multiplier: -1 | 1,
  stateMap: Record<string, IState>
): DistributionUpdates & { chartUpdates: ChartUpdates } => {
  const pathUpdates: { path: string[]; value: number }[] = [];

  // If issue does not exist, send a default object
  if (!issue) return { pathUpdates, assigneeUpdates: [], labelUpdates: [], chartUpdates: { updates: [] } };

  const state = stateMap[issue.state_id ?? ""];
  const stateGroup = state.group;

  // get if the state is in completed state
  const isCompleted = COMPLETED_STATE_GROUPS.indexOf(stateGroup) > -1;

  // add all the path updates that can be updated directly on the distribution object
  pathUpdates.push({ path: ["total_issues"], value: multiplier });

  // path updates for state distributions
  const stateDistribution = STATE_DISTRIBUTION[stateGroup];

  pathUpdates.push({ path: [stateDistribution.issues], value: multiplier });

  // get assignee and label distribution updates
  const assigneeUpdates = getObjectDistributionArray(issue.assignee_ids, isCompleted, multiplier);
  const labelUpdates = getObjectDistributionArray(issue.label_ids, isCompleted, multiplier);

  // chart updates based on date of completed or not completed
  const chartUpdates = getChartUpdates(isCompleted, issue.completed_at, multiplier);
  return {
    pathUpdates,
    assigneeUpdates,
    labelUpdates,
    chartUpdates,
  };
};

/**
 * This is to get distribution update array for either assignees and labels object
 * @param ids the assignee or label ids of issue
 * @param isCompleted
 * @param multiplier
 * @returns
 */
const getObjectDistributionArray = (ids: string[], isCompleted: boolean, multiplier: -1 | 1) => {
  const objectDistributionArray: DistributionObjectUpdate[] = [];

  // iterate over each id
  for (const id of ids) {
    const objectDistribution: DistributionObjectUpdate = {
      id,
      total_issues: multiplier,
    };

    // update paths for issue counts
    if (isCompleted) {
      objectDistribution["completed_issues"] = multiplier;
    } else {
      objectDistribution["pending_issues"] = multiplier;
    }

    objectDistributionArray.push(objectDistribution);
  }

  return objectDistributionArray;
};

/**
 * get chart distribution based of completed or not completed states
 * @param isCompleted
 * @param completedAt
 * @param multiplier
 * @returns
 */
const getChartUpdates = (isCompleted: boolean, completedAt: string | null, multiplier: -1 | 1) => {
  // if completed At date does not exist use current date
  let dateToUpdate = format(new Date(), "yyyy-MM-dd");
  const completedAtDate = getDate(completedAt);
  if (completedAt && completedAtDate) {
    dateToUpdate = format(completedAtDate, "yyyy-MM-dd");
  }

  // multiplier based on isCompleted state, it determines if the current count is to be added or subtracted from the list
  const completedAtMultiplier = isCompleted ? -1 : 1;

  return {
    updates: [{ path: ["distribution", "completion_chart", dateToUpdate], value: multiplier * completedAtMultiplier }],
    isCompleted,
  };
};

/**
 * Method to update distribution of either cycle or module object
 * @param distributionObject
 * @param distributionUpdates
 */
export const updateDistribution = (distributionObject: ICycle | IModule, distributionUpdates: DistributionUpdates) => {
  const { pathUpdates, assigneeUpdates, labelUpdates } = distributionUpdates;

  // iterate over path updates and directly apply changes on the distribution object
  for (const update of pathUpdates) {
    const { path, value } = update;
    const currentValue: number = get(distributionObject, path);
    if (currentValue !== undefined) set(distributionObject, path, (currentValue ?? 0) + value);
  }

  // for assignee update iterate through the assignee update and apply at the respective position
  for (const assigneeUpdate of assigneeUpdates) {
    const { id } = assigneeUpdate;

    // find and update the assignee issue counts
    if (Array.isArray(distributionObject.distribution?.assignees)) {
      const issuesAssignee = distributionObject.distribution?.assignees?.find(
        (assignee) => assignee.assignee_id === id
      );
      if (issuesAssignee) {
        issuesAssignee.completed_issues += assigneeUpdate.completed_issues ?? 0;
        issuesAssignee.pending_issues += assigneeUpdate.pending_issues ?? 0;
        issuesAssignee.total_issues += assigneeUpdate.total_issues;
      }
    }
  }

  for (const labelUpdate of labelUpdates) {
    const { id } = labelUpdate;

    // find and update the label issue counts
    if (Array.isArray(distributionObject.distribution?.labels)) {
      const issuesLabel = distributionObject.distribution?.labels?.find((label) => label.label_id === id);
      if (issuesLabel) {
        issuesLabel.completed_issues += labelUpdate.completed_issues ?? 0;
        issuesLabel.pending_issues += labelUpdate.pending_issues ?? 0;
        issuesLabel.total_issues += labelUpdate.total_issues;
      }
    }
  }
};
