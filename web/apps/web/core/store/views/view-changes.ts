/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { runInAction } from "mobx";
// nerve imports
import type { View, ViewCreate, ViewUpdate } from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import type { Change, ReconciledByKey } from "@/lib/reconciled";
import { dropped, prepended, replaced } from "@/lib/reconciled";
// services
import type { ViewsService } from "@/services/views/views.service";

export interface IViewChanges {
  // changes
  create: (projectId: string, body: ViewCreate) => Promise<View>;
  update: (viewId: string, body: () => ViewUpdate) => Promise<View>;
  remove: (viewId: string) => Promise<void>;
}

/** What the changes write nerve's answers into and find their view in: the view store's (ViewStore). */
type HeldViews = {
  readonly lists: ReconciledByKey<View[]>;
  readonly details: ReconciledByKey<View | null>;
  getViewById: (viewId: string) => View | undefined;
};

/** The view's own read as nerve answered the view; a deleted one's stays deleted. */
const reread =
  (view: View): Change<View | null> =>
  (read) =>
    read && view;

/**
 * The changes of the views (P6a spec 2.1, R3): each view's go one at a time, and each project's creations (v0 design
 * 7.7); another view's change does not wait for them. A change's body is reckoned in its turn (Update view's reads the
 * page's draft then, spec 2.7). Once nerve confirms a change, its answer is written into the copies the store holds:
 * its project's views and the view's own read; the page changes nothing before (no change of its own). nerve answers a
 * change of nothing with the view as it holds it, which is written as any answer. A refusal fails the change and writes
 * nothing; so does a change of a session the tab has left, which fails before it is sent (the session's client).
 */
export class ViewChanges implements IViewChanges {
  /** The queues of the changes, one for each view's, by its id, and one for each project's creations. */
  private readonly queues = new Map<string, ReturnType<typeof oneAtATime>>();

  constructor(
    private readonly views: HeldViews,
    private readonly service: ViewsService
  ) {}

  /**
   * @description creates a view in the project: first of its views, as nerve lists them, the newest first. Fails,
   * changing nothing, when nerve refuses.
   */
  create = (projectId: string, body: ViewCreate): Promise<View> =>
    this.queueOf(`new:${projectId}`)(async () => {
      const view = await this.service.createProjectView(projectId, body);
      this.views.lists.confirm(projectId, prepended([view]));
      return view;
    });

  /**
   * @description changes the fields the body names, the body reckoned in the change's turn, once the view's changes
   * before it are answered; the store then shows nerve's answer. Fails, changing nothing, when refused.
   */
  update = (viewId: string, body: () => ViewUpdate): Promise<View> =>
    this.queueOf(viewId)(async () => {
      const view = await this.service.updateView(viewId, body());
      runInAction(() => {
        this.views.lists.confirm(view.project_id ?? undefined, replaced(view));
        this.views.details.confirm(view.id, reread(view));
      });
      return view;
    });

  /**
   * @description deletes a view, which the store then no longer gives: out of its project's views, its read null.
   * Fails, changing nothing, when nerve refuses, or asking nerve nothing when the store does not give the view in its
   * turn (one deleted before it).
   */
  remove = (viewId: string): Promise<void> =>
    this.queueOf(viewId)(async () => {
      const view = this.views.getViewById(viewId);
      if (!view) throw new Error("View not found");
      await this.service.deleteView(view.id);
      runInAction(() => {
        this.views.lists.confirm(view.project_id ?? undefined, dropped(view.id));
        this.views.details.confirm(view.id, () => null);
      });
    });

  /** The queue of the changes of key, a view's id or a project's creations. */
  private queueOf(key: string): ReturnType<typeof oneAtATime> {
    let queue = this.queues.get(key);
    if (!queue) {
      queue = oneAtATime();
      this.queues.set(key, queue);
    }
    return queue;
  }
}
