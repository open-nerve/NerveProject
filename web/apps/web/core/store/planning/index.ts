/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient } from "@nerve/api-client";
import type { RootStore } from "@/store/root.store";
import type { ICycleStore } from "./cycle.store";
import { CycleStore } from "./cycle.store";

export interface IPlanningStore {
  cycles: ICycleStore;
}

/** The planning of a session's projects (M6): their cycles; P6 adds their modules here. */
export class PlanningStore implements IPlanningStore {
  cycles: ICycleStore;

  constructor(rootStore: RootStore, api: ApiClient) {
    this.cycles = new CycleStore(rootStore, api);
  }
}
