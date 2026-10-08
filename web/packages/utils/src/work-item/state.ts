/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import { STATE_GROUPS } from "@nerve/constants";
import type { State } from "@nerve/api-client";

/** The states by group, in STATE_GROUPS' order, then by sequence, the lowest first; the list given is left as it is. */
export const sortStates = (states: State[]): State[] =>
  states.toSorted((stateA, stateB) => {
    if (stateA.group === stateB.group) {
      return stateA.sequence - stateB.sequence;
    }
    return Object.keys(STATE_GROUPS).indexOf(stateA.group) - Object.keys(STATE_GROUPS).indexOf(stateB.group);
  });
