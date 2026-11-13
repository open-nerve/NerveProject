/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient } from "@nerve/api-client";
import type { RootStore } from "@/store/root.store";
import type { ICycleStore } from "./cycle.store";
import { CycleStore } from "./cycle.store";
import type { ICycleFilterStore } from "./cycle_filter.store";
import { CycleFilterStore } from "./cycle_filter.store";
import type { IModuleStore } from "./module.store";
import { ModuleStore } from "./module.store";

export interface IPlanningStore {
  cycles: ICycleStore;
  cycleFilters: ICycleFilterStore;
  modules: IModuleStore;
}

/**
 * The planning of a session's projects (M6): their cycles and their modules, and what each project's cycles list and
 * archive are filtered and searched by.
 */
export class PlanningStore implements IPlanningStore {
  cycles: ICycleStore;
  cycleFilters: ICycleFilterStore;
  modules: IModuleStore;

  constructor(rootStore: RootStore, api: ApiClient) {
    this.cycles = new CycleStore(rootStore, api);
    this.cycleFilters = new CycleFilterStore();
    this.modules = new ModuleStore(rootStore, api);
  }
}
