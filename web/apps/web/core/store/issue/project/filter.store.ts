/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ApiClient } from "@nerve/api-client";
import { computed, makeObservable } from "mobx";
import type { IIssueFilters } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { ProjectPreferencesService } from "@/services/project/project-preferences.service";
import type { IPagePreferencesFilter } from "../helpers/page-preferences-filter.store";
import { PagePreferencesFilter } from "../helpers/page-preferences-filter.store";
import type { IIssueRootStore } from "../root.store";

export interface IProjectIssuesFilter extends IPagePreferencesFilter {
  /** The filters of the project in the route. */
  issueFilters: IIssueFilters | undefined;
}

/** The project page's preferences (`/me/projects/{id}/preferences`), by project id. */
export class ProjectIssuesFilter extends PagePreferencesFilter implements IProjectIssuesFilter {
  constructor(rootIssueStore: IIssueRootStore, api: ApiClient) {
    super(rootIssueStore, new ProjectPreferencesService(api), EIssuesStoreType.PROJECT);
    makeObservable(this, { issueFilters: computed });
  }

  get issueFilters() {
    const projectId = this.rootIssueStore.projectId;
    if (!projectId) return undefined;

    return this.getIssueFilters(projectId);
  }
}
