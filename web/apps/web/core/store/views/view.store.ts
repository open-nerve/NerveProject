/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { maxBy } from "lodash-es";
import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, Project, View, Workspace } from "@nerve/api-client";
// lib
import { ReconciledByKey } from "@/lib/reconciled";
import { workspaceViewsOn } from "@/lib/views-entry";
// services
import { ViewsService } from "@/services/views/views.service";
// store
import type { RootStore } from "@/store/root.store";
import type { IViewChanges } from "./view-changes";
import { ViewChanges } from "./view-changes";

/** A copy of a view the store holds, and when nerve answered it (Reconciled.answeredAt). */
type Copy = { view: View; answeredAt: number };
/** A workspace as the caller's list gives it: its id holds its views, its slug names it to nerve. */
type WorkspaceRef = Pick<Workspace, "id" | "slug">;

export interface IViewStore extends IViewChanges {
  // computed actions
  getProjectViews: (projectId: string | null | undefined) => View[] | undefined;
  getWorkspaceViews: (workspaceSlug: string) => View[] | undefined;
  getViewById: (viewId: string | null | undefined) => View | undefined;
  // fetch actions
  fetchProjectViews: (projectId: string) => Promise<View[] | undefined>;
  fetchWorkspaceViews: (workspace: WorkspaceRef) => Promise<View[] | undefined>;
  fetchView: (viewId: string) => Promise<View | null | undefined>;
}

/**
 * The views of a session (M7 design 3.7, P6a spec 2.1; M7/P6b spec 2.6, R9): each project's views, by its id, and each
 * workspace's own (of no project), by its id, in nerve's order, the newest first; each view's own read, by its id. A
 * view shows as nerve last answered it, of the copies the store holds (v0 design 7.7). Every reader takes its project,
 * workspace or view from its caller (the route's), never from the address in the router store. A project's views do
 * not show once the project store no longer gives the project (deleted, left, or of a workspace no longer the
 * caller's); a workspace's own show only while the workspace store gives the workspace and the caller's role there is
 * an admin's or a member's (workspaceViewsOn): never to its guest. The store reads only where a page uses the views
 * (R20): it holds what its readers fetched. Its service sends with the session's client; fetches do not queue. Its
 * changes go one at a time for each view and write nerve's answers into every copy that holds the view
 * (view-changes.ts, v0 design 7.7).
 */
export class ViewStore implements IViewStore {
  /**
   * Each container's views, a project's or a workspace's own, by its id, in nerve's order, reconciled between fetches
   * and changes.
   */
  readonly lists = new ReconciledByKey<View[]>();
  /** Each view as nerve last answered it alone; null once deleted here (a fetch out meanwhile answers null too). */
  readonly details = new ReconciledByKey<View | null>();
  // services
  private readonly service: ViewsService;
  /** The changes of the views, which write nerve's answers into the values above. */
  private readonly changes: ViewChanges;
  // stores
  /** The project as the caller sees it, by the project store (ProjectStore.getProjectById). */
  private readonly projectOf: (projectId: string | undefined | null) => Project | undefined;
  /** The caller's workspace of the id, with his role there, by the workspace store (it has no getter by id). */
  private readonly workspaceOf: (workspaceId: string) => Workspace | undefined;
  /** The caller's workspace of the slug, with his role there, by the workspace store. */
  private readonly workspaceBySlug: (workspaceSlug: string) => Workspace | null;

  constructor(rootStore: RootStore, api: ApiClient) {
    this.service = new ViewsService(api);
    this.projectOf = rootStore.projectRoot.project.getProjectById;
    this.workspaceOf = (workspaceId) =>
      rootStore.workspaceRoot.workspaces?.find((workspace) => workspace.id === workspaceId);
    this.workspaceBySlug = (workspaceSlug) => rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
    this.changes = new ViewChanges(this, this.service);
  }

  /**
   * @description the project's views, in nerve's order, each as nerve last answered it (getViewById); undefined until
   * fetched, or once the project store no longer gives the project
   */
  getProjectViews = computedFn((projectId: string | null | undefined): View[] | undefined => {
    const project = this.projectOf(projectId);
    return project && this.lists.get(project.id)?.map((row) => this.getViewById(row.id) ?? row);
  });

  /**
   * @description the workspace's own views, of no project, in nerve's order, each as nerve last answered it
   * (getViewById); undefined until fetched, or while the workspace store does not give the workspace, or the caller's
   * role there is not an admin's or a member's (workspaceViewsOn): never to its guest
   */
  getWorkspaceViews = computedFn((workspaceSlug: string): View[] | undefined => {
    const workspace = this.workspaceBySlug(workspaceSlug);
    if (!workspace || !workspaceViewsOn(workspace.role)) return undefined;
    return this.lists.get(workspace.id)?.map((row) => this.getViewById(row.id) ?? row);
  });

  /**
   * @description the view as nerve last answered it, of its own read and its container's views; nothing until one of
   * them has it, once it is deleted here, or once the caller no longer sees it (shown)
   */
  getViewById = computedFn((viewId: string | null | undefined): View | undefined => {
    if (!viewId) return undefined;
    const read = this.details.get(viewId);
    const copies: Copy[] = [
      ...this.rowsOf(viewId),
      ...(read ? [{ view: read, answeredAt: this.details.answeredAt(viewId) }] : []),
    ];
    const newest = maxBy(copies, "answeredAt")?.view;
    return newest && this.shown(newest) ? newest : undefined;
  });

  /**
   * @description fetches the project's views and shows them with the changes nerve confirmed meanwhile; gives what it
   * shows, or undefined for a fetch a newer one overtook or a change of session cut (Reconciled.fetch)
   */
  fetchProjectViews = (projectId: string): Promise<View[] | undefined> =>
    this.lists.fetch(projectId, () => this.service.listProjectViews(projectId));

  /**
   * @description fetches the workspace's own views, by its slug, held by its id, as fetchProjectViews does a project's
   * (M7/P6b spec 2.6)
   */
  fetchWorkspaceViews = (workspace: WorkspaceRef): Promise<View[] | undefined> =>
    this.lists.fetch(workspace.id, () => this.service.listWorkspaceViews(workspace.slug));

  /** @description reads the view alone, as fetchProjectViews; null for one deleted here meanwhile */
  fetchView = (viewId: string): Promise<View | null | undefined> =>
    this.details.fetch(viewId, () => this.service.getView(viewId));

  // changes (view-changes.ts)
  create: IViewStore["create"] = (container, body) => this.changes.create(container, body);
  update: IViewStore["update"] = (viewId, body) => this.changes.update(viewId, body);
  remove: IViewStore["remove"] = (viewId) => this.changes.remove(viewId);
  isUpdating: IViewStore["isUpdating"] = (viewId) => this.changes.isUpdating(viewId);

  /**
   * Whether the caller sees the view, by its kind: a project's while the project store gives its project; a
   * workspace's own (of no project) while the workspace store gives its workspace, of which his role is an admin's or a
   * member's (workspaceViewsOn): never its guest's.
   */
  private shown(view: View): boolean {
    if (view.project_id !== null) return this.projectOf(view.project_id) !== undefined;
    return workspaceViewsOn(this.workspaceOf(view.workspace_id)?.role);
  }

  /**
   * The view's rows in the containers' views the store holds, each with when nerve answered that list, which is its
   * container's: its project's, else its workspace's (a workspace's own view, of no project).
   */
  private rowsOf(viewId: string): Copy[] {
    return this.lists
      .values()
      .flatMap((rows) => rows.filter((row) => row.id === viewId))
      .map((row) => ({ view: row, answeredAt: this.lists.answeredAt(row.project_id ?? row.workspace_id) }));
  }
}
