/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { maxBy } from "lodash-es";
import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, Project, View } from "@nerve/api-client";
// lib
import { ReconciledByKey } from "@/lib/reconciled";
// services
import { ViewsService } from "@/services/views/views.service";
// store
import type { RootStore } from "@/store/root.store";
import type { IViewChanges } from "./view-changes";
import { ViewChanges } from "./view-changes";

/** A copy of a view the store holds, and when nerve answered it (Reconciled.answeredAt). */
type Copy = { view: View; answeredAt: number };

export interface IViewStore extends IViewChanges {
  // computed actions
  getProjectViews: (projectId: string | null | undefined) => View[] | undefined;
  getViewById: (viewId: string | null | undefined) => View | undefined;
  // fetch actions
  fetchProjectViews: (projectId: string) => Promise<View[] | undefined>;
  fetchView: (viewId: string) => Promise<View | null | undefined>;
}

/**
 * The views of the projects of a session (M7 design 3.7, P6a spec 2.1): each project's views, by its id, in nerve's
 * order, the newest first; each view's own read, by its id. A view shows as nerve last answered it, of the copies the
 * store holds (v0 design 7.7). Every reader takes its project or view from its caller (the route's), never from the
 * address in the router store; the views of a project the project store no longer gives (deleted, left, or of a
 * workspace no longer the caller's) do not show. The store reads only where a page uses the views (R20): it holds what
 * its readers fetched. Its service sends with the session's client; fetches do not queue. Its changes go one at a
 * time for each view and write nerve's answers into every copy that holds the view (view-changes.ts, v0 design 7.7).
 */
export class ViewStore implements IViewStore {
  /** Each project's views, in nerve's order, reconciled between fetches and changes. */
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

  constructor(rootStore: RootStore, api: ApiClient) {
    this.service = new ViewsService(api);
    this.projectOf = rootStore.projectRoot.project.getProjectById;
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
   * @description the view as nerve last answered it, of its own read and its project's views; nothing until one of
   * them has it, once it is deleted here, or once the project store no longer gives its project
   */
  getViewById = computedFn((viewId: string | null | undefined): View | undefined => {
    if (!viewId) return undefined;
    const read = this.details.get(viewId);
    const copies: Copy[] = [
      ...this.rowsOf(viewId),
      ...(read ? [{ view: read, answeredAt: this.details.answeredAt(viewId) }] : []),
    ];
    const newest = maxBy(copies, "answeredAt")?.view;
    return newest && this.projectOf(newest.project_id) ? newest : undefined;
  });

  /**
   * @description fetches the project's views and shows them with the changes nerve confirmed meanwhile; gives what it
   * shows, or undefined for a fetch a newer one overtook or a change of session cut (Reconciled.fetch)
   */
  fetchProjectViews = (projectId: string): Promise<View[] | undefined> =>
    this.lists.fetch(projectId, () => this.service.listProjectViews(projectId));

  /** @description reads the view alone, as fetchProjectViews; null for one deleted here meanwhile */
  fetchView = (viewId: string): Promise<View | null | undefined> =>
    this.details.fetch(viewId, () => this.service.getView(viewId));

  // changes (view-changes.ts)
  create: IViewStore["create"] = (projectId, body) => this.changes.create(projectId, body);
  update: IViewStore["update"] = (viewId, body) => this.changes.update(viewId, body);
  remove: IViewStore["remove"] = (viewId) => this.changes.remove(viewId);

  /**
   * The view's rows in the projects' views the store holds, each with when nerve answered that list, which is its
   * project's: a project's view has its project's id (only a workspace's view has none, which no project lists).
   */
  private rowsOf(viewId: string): Copy[] {
    return this.lists
      .values()
      .flatMap((rows) => rows.filter((row) => row.id === viewId))
      .flatMap((row) => (row.project_id ? [{ view: row, answeredAt: this.lists.answeredAt(row.project_id) }] : []));
  }
}
