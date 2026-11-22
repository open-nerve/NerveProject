/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, NotificationUnreadCount, Workspace } from "@nerve/api-client";
// lib
import { ReconciledByKey } from "@/lib/reconciled";
// services
import { NotificationService } from "@/services/notification.service";
// store
import type { RootStore } from "@/store/root.store";

/** A workspace as the store fetches its notifications: by its slug, kept under its id. */
type WorkspaceRef = Pick<Workspace, "id" | "slug">;

export interface INotificationStore {
  // computed actions
  getUnreadCount: (slug: string) => NotificationUnreadCount | undefined;
  // fetch actions
  fetchUnreadCount: (workspace: WorkspaceRef) => Promise<NotificationUnreadCount | undefined>;
}

/**
 * The caller's notifications of a session (M7 design 3.10, P4 spec 2.3): each workspace's unread counts, by its id, as
 * nerve last answered them; only nerve counts them (by tab, archived, snoozed and what he sees), so the store never
 * does. Its readers name the workspace by the address's slug, as the caller's workspaces list it now, never by the
 * router store; a workspace no longer his gives nothing (v0 design 7.7). Its service sends with the session's client;
 * fetches do not queue.
 */
export class NotificationStore implements INotificationStore {
  /** Each workspace's unread counts, by its id, reconciled between fetches (reconciled.ts). */
  private readonly counts = new ReconciledByKey<NotificationUnreadCount>();
  // services
  private readonly service: NotificationService;
  // stores
  /** The caller's workspace the slug names, by his workspaces now (WorkspaceRootStore.getWorkspaceBySlug). */
  private readonly workspaceOf: (slug: string) => Workspace | null;

  constructor(rootStore: RootStore, api: ApiClient) {
    this.service = new NotificationService(api);
    this.workspaceOf = rootStore.workspaceRoot.getWorkspaceBySlug;
  }

  /**
   * @description the unread counts of the caller's workspace slug names, as nerve last answered them; undefined until
   * fetched, or for a slug that names no workspace of his now
   */
  getUnreadCount = computedFn((slug: string): NotificationUnreadCount | undefined => {
    const workspace = this.workspaceOf(slug);
    return workspace ? this.counts.get(workspace.id) : undefined;
  });

  /**
   * @description fetches the workspace's unread counts; gives them, or undefined for a fetch a newer one overtook or a
   * change of session cut (Reconciled.fetch); a refusal fails it, the counts as they were
   */
  fetchUnreadCount = (workspace: WorkspaceRef): Promise<NotificationUnreadCount | undefined> =>
    this.counts.fetch(workspace.id, () => this.service.unreadCount(workspace.slug));
}
