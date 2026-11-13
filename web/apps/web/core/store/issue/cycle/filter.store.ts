/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ApiClient } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { CyclePreferencesService } from "@/services/planning/cycle-preferences.service";
import { PagePreferencesFilter } from "../helpers/page-preferences-filter.store";
import type { IIssueRootStore } from "../root.store";

/**
 * The cycle page's preferences (`/me/cycles/{id}/preferences`), by cycle id (P5b spec 2.4). It follows no route: the
 * page passes the cycle of its route params on each call.
 */
export class CycleIssuesFilter extends PagePreferencesFilter {
  constructor(rootIssueStore: IIssueRootStore, api: ApiClient) {
    super(rootIssueStore, new CyclePreferencesService(api), EIssuesStoreType.CYCLE);
  }
}
