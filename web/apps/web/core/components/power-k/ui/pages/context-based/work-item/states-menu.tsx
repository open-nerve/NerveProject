/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Command } from "cmdk";
import { observer } from "mobx-react";
// plane types
import type { TIssue } from "@nerve/types";
import { Spinner } from "@nerve/ui";
// hooks
import { useProjectState } from "@/hooks/store/use-project-state";
// local imports
import { PowerKProjectStatesMenuItems } from "./state-menu-item";

type Props = {
  handleSelect: (stateId: string) => void;
  workItemDetails: TIssue;
};

export const PowerKProjectStatesMenu = observer(function PowerKProjectStatesMenu(props: Props) {
  const { workItemDetails } = props;
  // store hooks
  const { getProjectStateIds, getStateById } = useProjectState();
  // derived values
  const projectStateIds = workItemDetails.project_id ? getProjectStateIds(workItemDetails.project_id) : undefined;
  const projectStates = projectStateIds ? projectStateIds.map((stateId) => getStateById(stateId)) : undefined;
  const filteredProjectStates = projectStates ? projectStates.filter((state) => !!state) : undefined;

  if (!filteredProjectStates) return <Spinner />;

  return (
    <Command.Group>
      <PowerKProjectStatesMenuItems
        {...props}
        selectedStateId={workItemDetails.state_id ?? undefined}
        states={filteredProjectStates}
      />
    </Command.Group>
  );
});
