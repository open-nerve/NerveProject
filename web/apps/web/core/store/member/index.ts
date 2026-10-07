/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { makeObservable, observable } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, MemberUser } from "@nerve/api-client";
// store
import type { IProjectMemberStore } from "@/store/member/project/project-member.store";
import { ProjectMemberStore } from "@/store/member/project/project-member.store";
import type { RootStore } from "@/store/root.store";
// local imports
import type { IWorkspaceMemberStore } from "./workspace/workspace-member.store";
import { WorkspaceMemberStore } from "./workspace/workspace-member.store";

export interface IMemberRootStore {
  // observables
  memberMap: Record<string, MemberUser>;
  // computed actions
  getUserDetails: (userId: string) => MemberUser | undefined;
  // sub-stores
  workspace: IWorkspaceMemberStore;
  project: IProjectMemberStore;
}

export class MemberRootStore implements IMemberRootStore {
  // observables
  memberMap: Record<string, MemberUser> = {};
  // sub-stores
  workspace: IWorkspaceMemberStore;
  project: IProjectMemberStore;

  constructor(_rootStore: RootStore, api: ApiClient) {
    makeObservable(this, {
      // observables
      memberMap: observable,
    });
    // sub-stores
    this.workspace = new WorkspaceMemberStore(this, _rootStore, api);
    this.project = new ProjectMemberStore(this, _rootStore);
  }

  /**
   * @description get user details from userId
   * @param userId
   */
  getUserDetails = computedFn((userId: string): MemberUser | undefined => this.memberMap?.[userId] ?? undefined);
}
