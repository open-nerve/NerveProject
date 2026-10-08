/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type {
  IWorkspaceSearchResults,
  IWorkspaceView,
  TIssuesResponse,
  TSearchResponse,
  TSearchEntityRequestPayload,
} from "@nerve/types";
// services
import { APIService } from "@/services/api.service";

export class WorkspaceService extends APIService {
  async searchWorkspace(
    workspaceSlug: string,
    params: {
      project_id?: string;
      search: string;
      workspace_search: boolean;
    }
  ): Promise<IWorkspaceSearchResults> {
    return this.get(`/api/workspaces/${workspaceSlug}/search/`, {
      params,
    })
      .then((res) => res?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
  async createView(workspaceSlug: string, data: Partial<IWorkspaceView>): Promise<IWorkspaceView> {
    return this.post(`/api/workspaces/${workspaceSlug}/views/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async updateView(workspaceSlug: string, viewId: string, data: Partial<IWorkspaceView>): Promise<IWorkspaceView> {
    return this.patch(`/api/workspaces/${workspaceSlug}/views/${viewId}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async deleteView(workspaceSlug: string, viewId: string): Promise<any> {
    return this.delete(`/api/workspaces/${workspaceSlug}/views/${viewId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getAllViews(workspaceSlug: string): Promise<IWorkspaceView[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/views/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getViewDetails(workspaceSlug: string, viewId: string): Promise<IWorkspaceView> {
    return this.get(`/api/workspaces/${workspaceSlug}/views/${viewId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getViewIssues(workspaceSlug: string, params: any, config = {}): Promise<TIssuesResponse> {
    const path = params.expand?.includes("issue_relation")
      ? `/api/workspaces/${workspaceSlug}/issues-detail/`
      : `/api/workspaces/${workspaceSlug}/issues/`;
    return this.get(
      path,
      {
        params,
      },
      config
    )
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async searchEntity(workspaceSlug: string, params: TSearchEntityRequestPayload): Promise<TSearchResponse> {
    return this.get(`/api/workspaces/${workspaceSlug}/entity-search/`, {
      params: {
        ...params,
        query_type: params.query_type.join(","),
      },
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}
