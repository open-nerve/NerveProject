/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { State, StateGroup } from "@nerve/api-client";
import type { TStateOperationsCallbacks } from "@nerve/types";
import { cn } from "@nerve/utils";
// components
import { GroupItem } from "@/components/project-states";

type TGroupList = {
  groupedStates: Record<string, State[]>;
  stateOperationsCallbacks: TStateOperationsCallbacks;
  isEditable: boolean;
  groupListClassName?: string;
  groupItemClassName?: string;
  stateItemClassName?: string;
};

export const GroupList = observer(function GroupList(props: TGroupList) {
  const {
    groupedStates,
    stateOperationsCallbacks,
    isEditable,
    groupListClassName,
    groupItemClassName,
    stateItemClassName,
  } = props;
  // states
  const [groupsExpanded, setGroupsExpanded] = useState<Partial<StateGroup>[]>([
    "backlog",
    "unstarted",
    "started",
    "completed",
    "cancelled",
  ]);

  const handleGroupCollapse = (groupKey: StateGroup) => {
    setGroupsExpanded((prev) => {
      if (prev.includes(groupKey)) {
        return prev.filter((key) => key !== groupKey);
      }
      return prev;
    });
  };

  const handleExpand = (groupKey: StateGroup) => {
    setGroupsExpanded((prev) => {
      if (prev.includes(groupKey)) {
        return prev;
      }
      return [...prev, groupKey];
    });
  };
  return (
    <div className={cn("space-y-5", groupListClassName)}>
      {Object.entries(groupedStates).map(([key, value]) => {
        const groupKey = key as StateGroup;
        const groupStates = value;
        return (
          <GroupItem
            key={groupKey}
            groupKey={groupKey}
            states={groupStates}
            groupedStates={groupedStates}
            groupsExpanded={groupsExpanded}
            stateOperationsCallbacks={stateOperationsCallbacks}
            isEditable={isEditable}
            handleGroupCollapse={handleGroupCollapse}
            handleExpand={handleExpand}
            groupItemClassName={groupItemClassName}
            stateItemClassName={stateItemClassName}
          />
        );
      })}
    </div>
  );
});
