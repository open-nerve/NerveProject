/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, NotificationPage, NotificationUnreadCount, Workspace } from "@nerve/api-client";
// lib
import { ReconciledByKey } from "@/lib/reconciled";
// services
import { NotificationService } from "@/services/notification.service";
// store
import type { RootStore } from "@/store/root.store";
import type { NotificationFilters } from "./filters";
import { filterKey, filtersOf, listQuery } from "./filters";

/** A workspace as the store fetches its notifications: by its slug, kept under its id. */
type WorkspaceRef = Pick<Workspace, "id" | "slug">;

/** The key of a workspace's list of the notifications some filters select: its id, and the filters' key (filterKey). */
const listKey = (workspaceId: string, key: string) => `${workspaceId}|${key}`;

/**
 * The held page, followed by the rows of the next page it does not have yet (a notification sent late lands on a page
 * read already, R7), and the next page's cursor, as nerve answered it (the contract's T4).
 */
const followed = (held: NotificationPage, next: NotificationPage): NotificationPage => ({
  data: [...held.data, ...next.data.filter((row) => !held.data.some((heldRow) => heldRow.id === row.id))],
  next_cursor: next.next_cursor,
});

export interface INotificationStore {
  // computed actions
  getList: (slug: string, filters: NotificationFilters) => NotificationPage | undefined;
  getUnreadCount: (slug: string) => NotificationUnreadCount | undefined;
  // fetch actions
  fetchList: (workspace: WorkspaceRef, filters: NotificationFilters) => Promise<NotificationPage | undefined>;
  fetchMore: (workspace: WorkspaceRef, filters: NotificationFilters) => Promise<NotificationPage | undefined>;
  fetchUnreadCount: (workspace: WorkspaceRef) => Promise<NotificationUnreadCount | undefined>;
}

/**
 * The caller's notifications of a session (M7 design 3.10, P4 spec 2.3): each workspace's lists, by its id and the
 * filters' key, the pages read so far of each as one page; each workspace's unread counts, by its id, as nerve last
 * answered them; only nerve counts them (by tab, archived, snoozed and what he sees), so the store never does. Its
 * readers name the workspace by the address's slug, as the caller's workspaces list it now, never by the router store;
 * a workspace no longer his gives nothing (v0 design 7.7). Its service sends with the session's client; fetches do not
 * queue.
 */
export class NotificationStore implements INotificationStore {
  /** Each workspace's unread counts, by its id, reconciled between fetches (reconciled.ts). */
  private readonly counts = new ReconciledByKey<NotificationUnreadCount>();
  /**
   * Each workspace's list of the notifications some filters select, by listKey: the pages read so far as one page,
   * with the cursor of the next, reconciled between its fetches, the next pages' reads and the changes (reconciled.ts).
   */
  private readonly lists = new ReconciledByKey<NotificationPage>();
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
   * @description the notifications the filters select in the caller's workspace slug names, the pages read so far as
   * one, with the next page's cursor (none past the last); undefined until fetched, or for a slug that names no
   * workspace of his now
   */
  getList = computedFn((slug: string, filters: NotificationFilters): NotificationPage | undefined => {
    const workspace = this.workspaceOf(slug);
    return workspace ? this.lists.get(listKey(workspace.id, filterKey(filters))) : undefined;
  });

  /**
   * @description fetches the first page of the workspace's notifications the filters select, in place of every page
   * held of them (a refresh, R7): a next page's read out then writes nothing. What it asks nerve is what the list's key
   * names (filtersOf). Gives the page, or undefined as fetchUnreadCount; a refusal fails it, the list as it was
   */
  fetchList = (workspace: WorkspaceRef, filters: NotificationFilters): Promise<NotificationPage | undefined> => {
    const key = filterKey(filters);
    return this.lists.fetch(listKey(workspace.id, key), () =>
      this.service.list(workspace.slug, listQuery(filtersOf(key)))
    );
  };

  /**
   * @description reads the page after the held page of the workspace's notifications the filters select, by its cursor,
   * and adds it (followed); a page of no rows that has a cursor reads on (its rows were archived or snoozed away). Asks
   * nerve nothing past the last page or for a list not fetched; gives the list, or undefined as fetchList
   */
  fetchMore = (workspace: WorkspaceRef, filters: NotificationFilters): Promise<NotificationPage | undefined> => {
    const key = filterKey(filters);
    const list = listKey(workspace.id, key);
    const cursor = this.lists.get(list)?.next_cursor;
    if (!cursor) return Promise.resolve(undefined);
    const read = () => this.service.list(workspace.slug, listQuery(filtersOf(key), cursor));
    return this.lists.fetchPart(list, cursor, read, followed);
  };

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
