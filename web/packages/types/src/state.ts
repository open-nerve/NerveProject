/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { State, StateCreate, StateGroup, StateUpdate } from "@nerve/api-client";

export type TStateOperationsCallbacks = {
  createState: (data: StateCreate) => Promise<State>;
  updateState: (stateId: string, data: StateUpdate) => Promise<State>;
  moveState: (stateId: string, group: StateGroup, droppedOnId: string, after: boolean) => Promise<State>;
  deleteState: (stateId: string) => Promise<void>;
  markStateAsDefault: (stateId: string) => Promise<void>;
};
