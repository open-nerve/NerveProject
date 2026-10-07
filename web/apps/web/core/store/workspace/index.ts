/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, computed, makeObservable } from "mobx";
// nerve imports
import type { ApiClient, SlugAvailability, Workspace, WorkspaceCreate, WorkspaceUpdate } from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import { Reconciled, dropped, replaced, upserted } from "@/lib/reconciled";
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
  /**
   * The caller's workspaces, in nerve's order (by name, then id) but for those created or joined since nerve listed
   * them, which come last until the next fetch; undefined until fetched.
   */
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
  deleteWorkspace: (workspace: Pick<Workspace, "id" | "slug">) => Promise<void>;
  leaveWorkspace: (workspace: Pick<Workspace, "id" | "slug">) => Promise<void>;
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
  /** The list, reconciled between its fetches and the changes nerve confirmed (reconciled.ts). */
  private readonly list = new Reconciled<Workspace[]>();
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
    this.preferences = new WorkspacePreferencesStore(this.getWorkspaceBySlug, api);
    this.webhook = new WebhookStore(_rootStore);
  }

  get workspaces(): Workspace[] | undefined {
    return this.list.value;
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
   * @description fetches the caller's workspaces and shows them with the changes nerve confirmed meanwhile; gives what
   * it shows, or undefined for a fetch a newer one overtook or a change of session cut (Reconciled.fetch)
   * @returns {Promise<Workspace[] | undefined>}
   */
  fetchWorkspaces = (): Promise<Workspace[] | undefined> => this.list.fetch(() => this.service.list());

  /**
   * @description whether slug can name a new workspace, or why not (reserved, taken, invalid)
   * @returns {Promise<SlugAvailability>}
   */
  checkWorkspaceSlug = (slug: string): Promise<SlugAvailability> => this.service.checkSlug(slug);

  /**
   * @description creates a workspace, with the caller as its admin: the list has it last, until the next fetch puts
   * it in nerve's order. Fails, changing nothing, when nerve refuses.
   * @returns {Promise<Workspace>}
   */
  createWorkspace = (data: WorkspaceCreate): Promise<Workspace> =>
    this.changes(async () => {
      const workspace = await this.service.create(data);
      this.list.confirm(upserted(workspace));
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
      this.list.confirm(replaced(workspace));
      return workspace;
    });

  /**
   * @description deletes a workspace, which then leaves the list, and with it what the stores keep of it, by its id
   * (its members, invitations and the caller's settings there); fails, changing nothing, when nerve refuses
   * @returns {Promise<void>}
   */
  deleteWorkspace = (workspace: Pick<Workspace, "id" | "slug">): Promise<void> =>
    this.changes(async () => {
      await this.service.delete(workspace.slug);
      this.list.confirm(dropped(workspace.id));
    });

  /**
   * @description ends the caller's membership of a workspace, which then leaves the list, as deleteWorkspace; fails,
   * changing nothing, when nerve refuses (the only admin of the workspace or of one of its projects, 409)
   * @returns {Promise<void>}
   */
  leaveWorkspace = (workspace: Pick<Workspace, "id" | "slug">): Promise<void> =>
    this.changes(async () => {
      await this.service.leave(workspace.slug);
      this.list.confirm(dropped(workspace.id));
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
      this.list.confirm(upserted(workspace));
      return workspace;
    });

  /**
   * @description declines an invitation sent to the caller's address; his workspaces stay as they are. Fails when
   * nerve refuses.
   * @returns {Promise<void>}
   */
  declineInvitation = (invitationId: string, token: string): Promise<void> =>
    this.changes(() => this.service.decline(invitationId, token));
}
