/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
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
  const [stateLoader, setStateLoader] = useState(false);
  // store hooks
  const { fetchProjectStates, getProjectStateIds, getStateById, getStatePercentageInGroup } = useProjectState();
  // derived values
  const stateIds = getProjectStateIds(projectId);

  // fetch states if not provided
  const onDropdownOpen = async () => {
    if ((stateIds === undefined || stateIds.length === 0) && projectId) {
      setStateLoader(true);
      await fetchProjectStates(projectId);
      setStateLoader(false);
    }
  };

  return (
    <WorkItemStateDropdownBase
      {...props}
      getStateById={getStateById}
      getStatePercentageInGroup={getStatePercentageInGroup}
      isInitializing={stateLoader}
      stateIds={stateIds ?? []}
      onDropdownOpen={onDropdownOpen}
    />
  );
});
