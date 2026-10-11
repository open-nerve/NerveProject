/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import type { State, StateGroup } from "@nerve/api-client";
// components
import { StateItem } from "@/components/project-states";

type TStateList = {
  groupKey: StateGroup;
  groupedStates: Record<string, State[]>;
  states: State[];
  disabled?: boolean;
};

export const StateList = observer(function StateList(props: TStateList) {
  const { groupKey, groupedStates, states, disabled = false } = props;

  return (
    <>
      {states.map((state: State) => (
        <StateItem
          key={state.id}
          groupKey={groupKey}
          groupedStates={groupedStates}
          totalStates={states.length || 0}
          state={state}
          disabled={disabled}
        />
      ))}
    </>
  );
});
