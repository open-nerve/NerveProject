/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, Favorite } from "@nerve/api-client";
// lib
import type { Change } from "@/lib/reconciled";
import { ReconciledByKey } from "@/lib/reconciled";
// services
import { FavoritesService } from "@/services/favorites/favorites.service";
// store
import type { IFavoriteChanges, WorkspaceRef } from "./favorite-changes";
import { FavoriteChanges } from "./favorite-changes";
import type { FavoriteBranch, TargetKind } from "./favorite-tree";
import { favoriteIn, favoriteTree } from "./favorite-tree";

/** What a target shows as nerve confirmed it changed: its name, and a project's or a view's icon. */
type TargetShown = Pick<Favorite, "name"> | Pick<Favorite, "name" | "logo_props">;

export interface IFavoriteStore extends IFavoriteChanges {
  // computed actions
  getFavorites: (workspaceId: string) => Favorite[] | undefined;
  getFavoriteTree: (workspaceId: string) => FavoriteBranch[] | undefined;
  favoriteOf: (workspaceId: string, kind: TargetKind, entityId: string) => Favorite | undefined;
  answeredAt: (workspaceId: string) => number;
  // fetch actions
  fetchFavorites: (workspace: WorkspaceRef) => Promise<Favorite[] | undefined>;
  // following the targets
  forgetTarget: (workspaceId: string, kind: TargetKind, entityId: string) => void;
  forgetProject: (workspaceId: string, projectId: string) => void;
  renameTarget: (workspaceId: string, kind: TargetKind, entityId: string, shown: TargetShown) => void;
}

/**
 * The caller's favorites of a session (M7 design 3.8, M7/P7 spec 2.1, R1): each workspace's, by its id, read by its
 * slug, as nerve last answered them, with the changes nerve confirmed since (v0 design 7.7); the sidebar's tree and a
 * target's favorite are derived from them. Its readers take the workspace from their caller (the route's), never from
 * the address in the router store, and read only while the caller is the workspace's admin or member (spec 2.2).
 * Its changes go one at a time in each workspace, each reckoned in its turn (favorite-changes.ts). What follows a
 * target nerve confirmed changed here, deleted, archived, left or renamed (spec 2.7), is confirmed into the list without
 * asking nerve, which made the same change to the favorites in the target's own transaction: a read out meanwhile has
 * it made again on its answer.
 */
export class FavoriteStore implements IFavoriteStore {
  /** Each workspace's favorites, by its id, in nerve's order, reconciled between reads and changes. */
  readonly lists = new ReconciledByKey<Favorite[]>();
  // services
  private readonly service: FavoritesService;
  /** The changes of the favorites, which write nerve's answers into the lists. */
  private readonly changes: FavoriteChanges;

  constructor(api: ApiClient) {
    this.service = new FavoritesService(api);
    this.changes = new FavoriteChanges(this.lists, this.service);
  }

  /** @description the workspace's favorites as nerve last answered them; undefined until read */
  getFavorites = (workspaceId: string): Favorite[] | undefined => this.lists.get(workspaceId);

  /** @description the workspace's favorites as the sidebar shows them (favoriteTree); undefined until read */
  getFavoriteTree = computedFn((workspaceId: string): FavoriteBranch[] | undefined => {
    const list = this.lists.get(workspaceId);
    return list && favoriteTree(list);
  });

  /** @description the caller's favorite of the target in the workspace; undefined for none, or until read */
  favoriteOf = computedFn((workspaceId: string, kind: TargetKind, entityId: string): Favorite | undefined => {
    const list = this.lists.get(workspaceId);
    return list && favoriteIn(list, kind, entityId);
  });

  /**
   * @description when nerve answered the workspace's favorites (Reconciled.answeredAt; 0 until read), to be compared
   * with when it answered a target's copy elsewhere: the newer shows (spec 2.3)
   */
  answeredAt = (workspaceId: string): number => this.lists.answeredAt(workspaceId);

  /**
   * @description reads the workspace's favorites by its slug, held by its id, and shows them with the changes nerve
   * confirmed meanwhile; gives what it shows, or undefined for a read a newer one overtook or a change of session cut
   * (Reconciled.fetch)
   */
  fetchFavorites = (workspace: WorkspaceRef): Promise<Favorite[] | undefined> =>
    this.lists.fetch(workspace.id, () => this.service.list(workspace.slug));

  /** @description the target's favorite goes from the workspace's list: nerve deleted or archived the target */
  forgetTarget = (workspaceId: string, kind: TargetKind, entityId: string): void =>
    this.lists.confirm(workspaceId, (list) =>
      list.filter((favorite) => favorite.entity_type !== kind || favorite.entity_id !== entityId)
    );

  /**
   * @description every favorite of the project's goes from the workspace's list, its own, its cycles', modules' and
   * views': nerve archived or deleted the project, or the caller left it; folders, of none, stay
   */
  forgetProject = (workspaceId: string, projectId: string): void =>
    this.lists.confirm(workspaceId, (list) => list.filter((favorite) => favorite.project_id !== projectId));

  /**
   * @description the target's favorite shows the name nerve confirmed, and the icon when given: a rename here
   * advances no answer's time (Reconciled.confirm), so the sidebar would not tell it newer (spec 2.3, re-98 I1)
   */
  renameTarget = (workspaceId: string, kind: TargetKind, entityId: string, shown: TargetShown): void =>
    this.lists.confirm(workspaceId, renamed(kind, entityId, shown));

  // changes (favorite-changes.ts)
  add: IFavoriteStore["add"] = (workspace, kind, entityId) => this.changes.add(workspace, kind, entityId);
  remove: IFavoriteStore["remove"] = (workspace, kind, entityId) => this.changes.remove(workspace, kind, entityId);
  removeFavorite: IFavoriteStore["removeFavorite"] = (workspace, favoriteId) =>
    this.changes.removeFavorite(workspace, favoriteId);
  createFolder: IFavoriteStore["createFolder"] = (workspace, name) => this.changes.createFolder(workspace, name);
  renameFolder: IFavoriteStore["renameFolder"] = (workspace, favoriteId, name) =>
    this.changes.renameFolder(workspace, favoriteId, name);
  move: IFavoriteStore["move"] = (workspace, favoriteId, drop) => this.changes.move(workspace, favoriteId, drop);
  isPending: IFavoriteStore["isPending"] = (workspaceId, key) => this.changes.isPending(workspaceId, key);
}

/** The list with the target's favorite showing what nerve confirmed; the list itself when it holds none. */
const renamed =
  (kind: TargetKind, entityId: string, shown: TargetShown): Change<Favorite[]> =>
  (list) => {
    const favorite = favoriteIn(list, kind, entityId);
    return favorite ? list.map((held) => (held === favorite ? { ...held, ...shown } : held)) : list;
  };
