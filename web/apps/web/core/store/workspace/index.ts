/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, computed, observable, makeObservable, runInAction } from "mobx";
// nerve imports
import type { ApiClient, SlugAvailability, Workspace, WorkspaceCreate, WorkspaceUpdate } from "@nerve/api-client";
// lib
import { SessionChangedError } from "@/lib/auth/token-manager";
import { oneAtATime } from "@/lib/one-at-a-time";
// services
import { WorkspacesService } from "@/services/workspace/workspaces.service";
// store
import type { RootStore } from "@/store/root.store";
// sub-stores
import type { IWorkspacePreferencesStore } from "./preferences.store";
import { WorkspacePreferencesStore } from "./preferences.store";
import type { IWebhookStore } from "./webhook.store";
import { WebhookStore } from "./webhook.store";

export interface IWorkspaceRootStore {
  /** The caller's workspaces, in nerve's order (by name, then id); undefined until fetched. */
  workspaces: Workspace[] | undefined;
  // computed
  currentWorkspace: Workspace | null;
  // computed actions
  getWorkspaceBySlug: (workspaceSlug: string) => Workspace | null;
  // fetch actions
  fetchWorkspaces: () => Promise<Workspace[] | undefined>;
  checkWorkspaceSlug: (slug: string) => Promise<SlugAvailability>;
  // crud actions
  createWorkspace: (data: WorkspaceCreate) => Promise<Workspace>;
  updateWorkspace: (workspaceSlug: string, data: WorkspaceUpdate) => Promise<Workspace>;
  deleteWorkspace: (workspaceSlug: string) => Promise<void>;
  leaveWorkspace: (workspaceSlug: string) => Promise<void>;
  acceptInvitation: (invitationId: string, token: string) => Promise<Workspace>;
  declineInvitation: (invitationId: string, token: string) => Promise<void>;
  // sub-stores
  preferences: IWorkspacePreferencesStore;
  webhook: IWebhookStore;
}

/**
 * The workspaces of the account of a session (M3 design 7.3): its service sends with the session's client, which
 * the RootStore of the session hands down. Changes go one at a time (v0 design 7.7); fetches do not queue.
 */
export class WorkspaceRootStore implements IWorkspaceRootStore {
  workspaces: Workspace[] | undefined = undefined;
  // services
  private readonly service: WorkspacesService;
  /** The changes of the workspaces, sent one at a time. */
  private readonly changes = oneAtATime();
  // root store
  router;
  // sub-stores
  preferences: IWorkspacePreferencesStore;
  webhook: IWebhookStore;

  constructor(_rootStore: RootStore, api: ApiClient) {
    makeObservable(this, {
      // observables
      workspaces: observable.ref,
      // computed
      currentWorkspace: computed,
      // actions
      fetchWorkspaces: action,
      createWorkspace: action,
      updateWorkspace: action,
      deleteWorkspace: action,
      leaveWorkspace: action,
      acceptInvitation: action,
      declineInvitation: action,
    });

    // services
    this.service = new WorkspacesService(api);
    // root store
    this.router = _rootStore.router;
    // sub-stores
    this.preferences = new WorkspacePreferencesStore(api);
    this.webhook = new WebhookStore(_rootStore);
  }

  /** The workspace the address names, when the caller is a member of it. */
  get currentWorkspace() {
    const workspaceSlug = this.router.workspaceSlug;
    return workspaceSlug ? this.getWorkspaceBySlug(workspaceSlug) : null;
  }

  /** The workspace of the caller's that slug names, or null. */
  getWorkspaceBySlug = (workspaceSlug: string) =>
    this.workspaces?.find((workspace) => workspace.slug === workspaceSlug) ?? null;

  /**
   * @description fetches the caller's workspaces and gives them; a change of session while they load is no failure:
   * the new session's store fetches its own (store-context.tsx), and this one gives undefined
   * @returns {Promise<Workspace[] | undefined>}
   */
  fetchWorkspaces = async (): Promise<Workspace[] | undefined> => {
    try {
      const workspaces = await this.service.list();
      runInAction(() => {
        this.workspaces = workspaces;
      });
      return workspaces;
    } catch (error) {
      if (error instanceof SessionChangedError) return undefined;
      throw error;
    }
  };

  /**
   * @description whether slug can name a new workspace, or why not (reserved, taken, invalid)
   * @returns {Promise<SlugAvailability>}
   */
  checkWorkspaceSlug = (slug: string): Promise<SlugAvailability> => this.service.checkSlug(slug);

  /**
   * @description creates a workspace, with the caller as its admin; once the list is fetched it has the new one
   * last, until the next fetch puts it in nerve's order. Fails, changing nothing, when nerve refuses.
   * @returns {Promise<Workspace>}
   */
  createWorkspace = (data: WorkspaceCreate): Promise<Workspace> =>
    this.changes(async () => {
      const workspace = await this.service.create(data);
      runInAction(() => {
        if (this.workspaces) this.workspaces = [...this.workspaces, workspace];
      });
      return workspace;
    });

  /**
   * @description changes a workspace's name, organization size or time zone; the list then has nerve's answer.
   * Fails, changing nothing, when nerve refuses.
   * @returns {Promise<Workspace>}
   */
  updateWorkspace = (workspaceSlug: string, data: WorkspaceUpdate): Promise<Workspace> =>
    this.changes(async () => {
      const workspace = await this.service.update(workspaceSlug, data);
      runInAction(() => {
        this.workspaces = this.workspaces?.map((w) => (w.id === workspace.id ? workspace : w));
      });
      return workspace;
    });

  /**
   * @description deletes a workspace, which then leaves the list; fails, changing nothing, when nerve refuses
   * @returns {Promise<void>}
   */
  deleteWorkspace = (workspaceSlug: string): Promise<void> =>
    this.changes(async () => {
      await this.service.delete(workspaceSlug);
      this.drop(workspaceSlug);
    });

  /**
   * @description ends the caller's membership of a workspace, which then leaves the list; fails, changing nothing,
   * when nerve refuses (the only admin of the workspace or of one of its projects, 409)
   * @returns {Promise<void>}
   */
  leaveWorkspace = (workspaceSlug: string): Promise<void> =>
    this.changes(async () => {
      await this.service.leave(workspaceSlug);
      this.drop(workspaceSlug);
    });

  /**
   * @description accepts an invitation sent to the caller's address: its workspace joins the list, or takes its own
   * place there when the caller was a member already (his role stays). Fails, changing nothing, when nerve refuses
   * (another address's invitation, 403; one declined, or no longer there).
   * @returns {Promise<Workspace>}
   */
  acceptInvitation = (invitationId: string, token: string): Promise<Workspace> =>
    this.changes(async () => {
      const workspace = await this.service.accept(invitationId, token);
      runInAction(() => {
        const listed = this.workspaces?.some((w) => w.id === workspace.id);
        if (listed) this.workspaces = this.workspaces?.map((w) => (w.id === workspace.id ? workspace : w));
        else if (this.workspaces) this.workspaces = [...this.workspaces, workspace];
      });
      return workspace;
    });

  /**
   * @description declines an invitation sent to the caller's address; his workspaces stay as they are. Fails when
   * nerve refuses.
   * @returns {Promise<void>}
   */
  declineInvitation = (invitationId: string, token: string): Promise<void> =>
    this.changes(() => this.service.decline(invitationId, token));

  /** The workspace leaves the caller's list: it was deleted, or he is no longer a member. */
  private drop(workspaceSlug: string): void {
    runInAction(() => {
      this.workspaces = this.workspaces?.filter((workspace) => workspace.slug !== workspaceSlug);
    });
  }
}
