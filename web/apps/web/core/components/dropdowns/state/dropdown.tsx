/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "react-router";
import { DirectoryReadError, useDirectoryRead } from "@/components/issues/issue-detail-widgets/directory-read";
import { observer } from "mobx-react";
// hooks
import { useProjectState } from "@/hooks/store/use-project-state";
// local imports
import type { TWorkItemStateDropdownBaseProps } from "./base";
import { WorkItemStateDropdownBase } from "./base";

type TWorkItemStateDropdownProps = Omit<
  TWorkItemStateDropdownBaseProps,
  "stateIds" | "getStateById" | "getStatePercentageInGroup" | "onDropdownOpen" | "isInitializing"
>;

export const StateDropdown = observer(function StateDropdown(props: TWorkItemStateDropdownProps) {
  const { projectId } = props;
  // states
  const { workspaceSlug } = useParams();
  // store hooks
  const { fetchProjectStates, getProjectStateIds, getStateById, getStatePercentageInGroup } = useProjectState();
  // derived values
  const stateIds = getProjectStateIds(projectId);

  const read = useDirectoryRead(workspaceSlug, projectId, projectId ? () => fetchProjectStates(projectId) : undefined);

  return (
    <WorkItemStateDropdownBase
      {...props}
      getStateById={getStateById}
      getStatePercentageInGroup={getStatePercentageInGroup}
      isInitializing={read.loading}
      readStatus={read.error ? <DirectoryReadError error={read.error} retry={read.load} /> : undefined}
      stateIds={stateIds ?? []}
      onDropdownOpen={read.load}
    />
  );
});
