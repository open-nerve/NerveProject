/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { State, StateGroup } from "@nerve/api-client";
// components
import { GroupItem } from "@/components/project-states";

type TGroupList = {
  projectId: string;
  groupedStates: Record<string, State[]>;
  isEditable: boolean;
};

export const GroupList = observer(function GroupList(props: TGroupList) {
  const { projectId, groupedStates, isEditable } = props;
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
    <div className="space-y-5">
      {Object.entries(groupedStates).map(([key, value]) => {
        const groupKey = key as StateGroup;
        const groupStates = value;
        return (
          <GroupItem
            key={groupKey}
            projectId={projectId}
            groupKey={groupKey}
            states={groupStates}
            groupedStates={groupedStates}
            groupsExpanded={groupsExpanded}
            isEditable={isEditable}
            handleGroupCollapse={handleGroupCollapse}
            handleExpand={handleExpand}
          />
        );
      })}
    </div>
  );
});
