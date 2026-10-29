/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ISearchIssueResponse, TProjectIssuesSearchParams } from "@nerve/types";
// services
import { APIService } from "@/services/api.service";

/**
 * The old search address, still used by M4, M6 and M7; its later migration removes this service.
 */
export class ProjectService extends APIService {
  async projectIssuesSearch(
    workspaceSlug: string,
    projectId: string,
    params: TProjectIssuesSearchParams
  ): Promise<ISearchIssueResponse[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/search-issues/`, {
      params,
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}
