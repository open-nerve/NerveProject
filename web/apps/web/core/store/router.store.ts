/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, makeObservable, observable, computed, runInAction } from "mobx";

// the current route's parameters, as React Router gives them
type TRouteParams = Record<string, string | undefined>;

export interface IRouterStore {
  // observables
  query: TRouteParams;
  // actions
  setQuery: (query: TRouteParams) => void;
  // computed
  workspaceSlug: string | undefined;
  projectId: string | undefined;
  cycleId: string | undefined;
  moduleId: string | undefined;
  viewId: string | undefined;
  globalViewId: string | undefined;
  userId: string | undefined;
  webhookId: string | undefined;
}

export class RouterStore implements IRouterStore {
  // observables
  query: TRouteParams = {};

  constructor() {
    makeObservable(this, {
      // observables
      query: observable,
      // actions
      setQuery: action.bound,
      //computed
      workspaceSlug: computed,
      projectId: computed,
      cycleId: computed,
      moduleId: computed,
      viewId: computed,
      globalViewId: computed,
      userId: computed,
      webhookId: computed,
    });
  }

  /**
   * Sets the query
   * @param query
   */
  setQuery = (query: TRouteParams) => {
    runInAction(() => {
      this.query = query;
    });
  };

  /**
   * Returns the workspace slug from the query
   * @returns string|undefined
   */
  get workspaceSlug() {
    return this.query.workspaceSlug;
  }

  /**
   * Returns the project id from the query
   * @returns string|undefined
   */
  get projectId() {
    return this.query.projectId;
  }

  /**
   * Returns the module id from the query
   * @returns string|undefined
   */
  get moduleId() {
    return this.query.moduleId;
  }

  /**
   * Returns the cycle id from the query
   * @returns string|undefined
   */
  get cycleId() {
    return this.query.cycleId;
  }

  /**
   * Returns the view id from the query
   * @returns string|undefined
   */
  get viewId() {
    return this.query.viewId;
  }

  /**
   * Returns the global view id from the query
   * @returns string|undefined
   */
  get globalViewId() {
    return this.query.globalViewId;
  }

  /**
   * Returns the user id from the query
   * @returns string|undefined
   */
  get userId() {
    return this.query.userId;
  }

  /**
   * Returns the webhook id from the query
   * @returns string|undefined
   */
  get webhookId() {
    return this.query.webhookId;
  }
}
