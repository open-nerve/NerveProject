/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observable, runInAction } from "mobx";
// nerve imports
import type { Favorite, FavoriteKind, FavoriteUpdate, Workspace } from "@nerve/api-client";
// lib
import { problemCode } from "@/lib/api-error";
import { oneAtATime } from "@/lib/one-at-a-time";
import type { Change, ReconciledByKey } from "@/lib/reconciled";
import { replaced, upserted } from "@/lib/reconciled";
// services
import type { FavoritesService } from "@/services/favorites/favorites.service";
// store
import type { FavoriteDrop } from "./favorite-place";
import { moveOf } from "./favorite-place";
import type { TargetKind } from "./favorite-tree";
import { favoriteIn } from "./favorite-tree";

/** A workspace as the caller's list gives it: its id holds its favorites, its slug names it to nerve. */
export type WorkspaceRef = Pick<Workspace, "id" | "slug">;

export interface IFavoriteChanges {
  // changes
  add: (workspace: WorkspaceRef, kind: TargetKind, entityId: string) => Promise<Favorite>;
  remove: (workspace: WorkspaceRef, kind: TargetKind, entityId: string) => Promise<void>;
  removeFavorite: (workspace: WorkspaceRef, favoriteId: string) => Promise<void>;
  createFolder: (workspace: WorkspaceRef, name: string) => Promise<Favorite>;
  renameFolder: (workspace: WorkspaceRef, favoriteId: string, name: string) => Promise<void>;
  move: (workspace: WorkspaceRef, favoriteId: string, drop: FavoriteDrop) => Promise<void>;
  /**
   * Whether a change of the workspace's favorites about key is queued or out (observable): a target's,
   * "{kind}:{entityId}", or a favorite's id, whichever entry made the change.
   */
  isPending: (workspaceId: string, key: string) => boolean;
}

/** nerve's answer that the favorite is not there: deleted, or of a target the caller no longer sees. */
const NOT_FOUND = "navigation.favorite_not_found";

/** A target's key in isPending. */
const targetKey = (kind: FavoriteKind, entityId: string | null) => `${kind}:${entityId}`;
/** The key of a change's count: a key in its workspace. */
const pendingKey = (workspaceId: string, key: string) => JSON.stringify([workspaceId, key]);

/** The list without the favorite, a folder with every favorite it holds (deleteFavorite). */
const without =
  (favoriteId: string): Change<Favorite[]> =>
  (list) =>
    list.filter((favorite) => favorite.id !== favoriteId && favorite.parent_id !== favoriteId);

/**
 * The changes of one's favorites (M7/P7 spec 2.1, R3; v0 design 7.7): one queue for each workspace, as a move's place
 * depends on its siblings. Each change is an intent whose body is reckoned in its turn from the list nerve answered
 * last, so that an intent made twice sends once: favoring a target favored already, or removing one gone, sends
 * nothing. Nothing is written before nerve confirms a change, its answer then confirmed into the list (a read out
 * meanwhile has it made again on its answer); a refusal fails the change and writes nothing, as does a change of a
 * session the tab has left, which fails before it is sent; but a favorite nerve says is not there (NOT_FOUND) goes from
 * the list, a folder with what it holds, for nothing could remove it else (re-98 m2): a removal so answered is done, a
 * rename or a move still fails. Whether a change about a target or a favorite is queued or out is observable by both
 * its keys from its sending to its answer (re-98 o3): the star of a target whose row is being removed waits too.
 */
export class FavoriteChanges implements IFavoriteChanges {
  /** The queues of the changes, one for each workspace, by its id. */
  private readonly queues = new Map<string, ReturnType<typeof oneAtATime>>();
  /** How many changes about each key are queued or out, by [workspace's id, key]; none, no entry. */
  private readonly pending = observable.map<string, number>();

  constructor(
    private readonly lists: ReconciledByKey<Favorite[]>,
    private readonly service: FavoritesService
  ) {}

  /**
   * @description favors the target: nothing sent when its favorite is held in the change's turn, which it gives;
   * else nerve's answer, its new favorite or the one it had (200), written in the list once
   */
  add = (workspace: WorkspaceRef, kind: TargetKind, entityId: string): Promise<Favorite> =>
    this.queued(workspace.id, this.targetKeys(workspace.id, kind, entityId), async () => {
      const held = this.heldOf(workspace.id, kind, entityId);
      if (held) return held;
      const favorite = await this.service.create(workspace.slug, { entity_type: kind, entity_id: entityId });
      this.lists.confirm(workspace.id, upserted(favorite));
      return favorite;
    });

  /** @description removes the target's favorite: nothing sent when none is held in the change's turn */
  remove = (workspace: WorkspaceRef, kind: TargetKind, entityId: string): Promise<void> =>
    this.queued(workspace.id, this.targetKeys(workspace.id, kind, entityId), async () => {
      const favorite = this.heldOf(workspace.id, kind, entityId);
      if (favorite) await this.deleted(workspace.id, favorite.id);
    });

  /**
   * @description removes the favorite from its row's menu, a folder with every favorite it holds: nothing sent when
   * it is not held in the change's turn
   */
  removeFavorite = (workspace: WorkspaceRef, favoriteId: string): Promise<void> =>
    this.queued(workspace.id, this.favoriteKeys(workspace.id, favoriteId), async () => {
      if (this.heldById(workspace.id, favoriteId)) await this.deleted(workspace.id, favoriteId);
    });

  /**
   * @description makes a folder of the name, which the caller has trimmed; fails, writing nothing, when nerve
   * refuses, a name another of the caller's folders has among them (navigation.favorite_folder_name_taken)
   */
  createFolder = (workspace: WorkspaceRef, name: string): Promise<Favorite> =>
    this.queued(workspace.id, [], async () => {
      const folder = await this.service.create(workspace.slug, { entity_type: "folder", name });
      this.lists.confirm(workspace.id, upserted(folder));
      return folder;
    });

  /** @description renames the folder: nothing sent when it is not held in the change's turn, or has the name then */
  renameFolder = (workspace: WorkspaceRef, favoriteId: string, name: string): Promise<void> =>
    this.queued(workspace.id, this.favoriteKeys(workspace.id, favoriteId), async () => {
      const folder = this.heldById(workspace.id, favoriteId);
      if (folder && folder.name !== name) await this.changed(workspace.id, favoriteId, { name });
    });

  /**
   * @description moves the favorite where it is dropped: one PATCH of what changes, reckoned in the change's turn
   * (moveOf), or nothing sent when nothing would move then
   */
  move = (workspace: WorkspaceRef, favoriteId: string, drop: FavoriteDrop): Promise<void> =>
    this.queued(workspace.id, this.favoriteKeys(workspace.id, favoriteId), async () => {
      const list = this.lists.get(workspace.id);
      const body = list && moveOf(list, favoriteId, drop);
      if (body) await this.changed(workspace.id, favoriteId, body);
    });

  /** @description whether a change about key is queued or out in the workspace; observable */
  isPending = (workspaceId: string, key: string): boolean => (this.pending.get(pendingKey(workspaceId, key)) ?? 0) > 0;

  /** Deletes the favorite; nerve's answer that it is not there is as good. */
  private async deleted(workspaceId: string, favoriteId: string): Promise<void> {
    try {
      await this.service.remove(favoriteId);
    } catch (error) {
      if (problemCode(error) !== NOT_FOUND) throw error;
    }
    this.lists.confirm(workspaceId, without(favoriteId));
  }

  /** Changes the favorite as body says; nerve's answer that it is not there takes it from the list, and fails. */
  private async changed(workspaceId: string, favoriteId: string, body: FavoriteUpdate): Promise<void> {
    try {
      this.lists.confirm(workspaceId, replaced(await this.service.update(favoriteId, body)));
    } catch (error) {
      if (problemCode(error) === NOT_FOUND) this.lists.confirm(workspaceId, without(favoriteId));
      throw error;
    }
  }

  private heldOf(workspaceId: string, kind: TargetKind, entityId: string): Favorite | undefined {
    const list = this.lists.get(workspaceId);
    return list && favoriteIn(list, kind, entityId);
  }

  private heldById(workspaceId: string, favoriteId: string): Favorite | undefined {
    return this.lists.get(workspaceId)?.find((favorite) => favorite.id === favoriteId);
  }

  /** A target's keys: its own, and its favorite's id when the list holds one now. */
  private targetKeys(workspaceId: string, kind: TargetKind, entityId: string): string[] {
    const held = this.heldOf(workspaceId, kind, entityId);
    return [targetKey(kind, entityId), ...(held ? [held.id] : [])];
  }

  /** A favorite's keys: its id, and its target's key when the list holds it now and it is of a target. */
  private favoriteKeys(workspaceId: string, favoriteId: string): string[] {
    const held = this.heldById(workspaceId, favoriteId);
    const target = held && held.entity_type !== "folder" ? [targetKey(held.entity_type, held.entity_id)] : [];
    return [favoriteId, ...target];
  }

  /**
   * Queues a change in the workspace's queue, counted under its keys from now until it settles: answered or refused
   * (its count drops before the caller's own follow-up runs, which waits on the same settling).
   */
  private queued<T>(workspaceId: string, keys: string[], task: () => Promise<T>): Promise<T> {
    const counted = keys.map((key) => pendingKey(workspaceId, key));
    this.count(counted, 1);
    const run = this.queueOf(workspaceId)(task);
    void run.then(
      () => this.count(counted, -1),
      () => this.count(counted, -1)
    );
    return run;
  }

  private count(keys: string[], step: 1 | -1): void {
    runInAction(() => {
      for (const key of keys) {
        const left = (this.pending.get(key) ?? 0) + step;
        if (left > 0) this.pending.set(key, left);
        else this.pending.delete(key);
      }
    });
  }

  /** The queue of the workspace's changes. */
  private queueOf(workspaceId: string): ReturnType<typeof oneAtATime> {
    let queue = this.queues.get(workspaceId);
    if (!queue) {
      queue = oneAtATime();
      this.queues.set(workspaceId, queue);
    }
    return queue;
  }
}
