/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, makeObservable, observable } from "mobx";
import { computedFn } from "mobx-utils";
// types
import type { MemberUser, WorkspaceMember } from "@nerve/api-client";
// local imports
import type { IMemberFilters } from "../utils";
import { sortWorkspaceMembers } from "../utils";

export interface IWorkspaceMemberFiltersStore {
  // observables
  filters: IMemberFilters;
  // computed actions
  getFilteredMemberIds: (
    members: WorkspaceMember[],
    memberDetailsMap: Record<string, MemberUser>,
    getMemberKey: (member: WorkspaceMember) => string
  ) => string[];
  // actions
  updateFilters: (filters: Partial<IMemberFilters>) => void;
}

export class WorkspaceMemberFiltersStore implements IWorkspaceMemberFiltersStore {
  // observables
  filters: IMemberFilters = {};

  constructor() {
    makeObservable(this, {
      // observables
      filters: observable,
      // actions
      updateFilters: action,
    });
  }

  /**
   * @description get filtered and sorted member ids
   * @param members - array of workspace membership objects
   * @param memberDetailsMap - map of member details by user id
   * @param getMemberKey - function to get member key from membership object
   */
  getFilteredMemberIds = computedFn(
    (
      members: WorkspaceMember[],
      memberDetailsMap: Record<string, MemberUser>,
      getMemberKey: (member: WorkspaceMember) => string
    ): string[] => {
      if (!members || members.length === 0) return [];

      // Apply filters and sorting
      const sortedMembers = sortWorkspaceMembers(members, memberDetailsMap, getMemberKey, this.filters);

      return sortedMembers.map(getMemberKey);
    }
  );

  /**
   * @description update filters
   * @param filters - partial filters to update
   */
  updateFilters = (filters: Partial<IMemberFilters>) => {
    this.filters = { ...this.filters, ...filters };
  };
}
