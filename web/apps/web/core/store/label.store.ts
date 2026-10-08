/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, computed, makeObservable } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, Label, LabelCreate, LabelUpdate, Project } from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import { placeAt } from "@/lib/place-between";
import type { Change } from "@/lib/reconciled";
import { ReconciledByKey, replaced, upserted } from "@/lib/reconciled";
// services
import { LabelsService } from "@/services/project/labels.service";
// store
import type { RootStore } from "@/store/root.store";
import type { IRouterStore } from "@/store/router.store";

/** A label at the top with the labels under it, in their project's order: labels have two levels (M3 design 3.16). */
type LabelTree = Label & { children: Label[] };
/** nerve's gap between a project's last label and a new one (M3 design 4.10). */
const SORT_ORDER_STEP = 10000;

export interface ILabelStore {
  // computed
  /** Every label the store shows, by id, which the work items' stores look up. */
  labelMap: Record<string, Label>;
  /** The address's project's labels, by sort order, then id; undefined until fetched. */
  projectLabels: Label[] | undefined;
  /** The address's project's labels at the top, each with the labels under it; undefined until fetched. */
  projectLabelsTree: LabelTree[] | undefined;
  // computed actions
  getLabelById: (labelId: string | null | undefined) => Label | undefined;
  getProjectLabels: (projectId: string | null | undefined) => Label[] | undefined;
  getProjectLabelIds: (projectId: string | null | undefined) => string[] | undefined;
  // fetch actions
  fetchProjectLabels: (projectId: string) => Promise<Label[] | undefined>;
  // changes
  createLabel: (projectId: string, data: LabelCreate) => Promise<Label>;
  updateLabel: (labelId: string, data: LabelUpdate) => Promise<Label>;
  updateLabelPosition: (
    labelId: string,
    parentId: string | null,
    droppedOnId: string | undefined,
    dropAtEndOfList: boolean
  ) => Promise<Label | undefined>;
  deleteLabel: (labelId: string) => Promise<void>;
}

/** nerve's order of a project's labels: by sort order, the lowest first, then by id. */
const inOrder = (labels: Label[]): Label[] =>
  labels.toSorted((a, b) => a.sort_order - b.sort_order || Number(a.id > b.id) - Number(a.id < b.id));

/** A label at the top, a copy, with the labels of the list under it, in the list's order. */
const treeOf = (top: Label, labels: Label[]): LabelTree => ({
  ...top,
  children: labels.filter((label) => label.parent_id === top.id),
});

/** The list without the label id names and the labels under it, as nerve deletes them. */
const deleted =
  (id: string): Change<Label[]> =>
  (list) =>
    list.filter((label) => label.id !== id && label.parent_id !== id);

/**
 * The labels of the projects of a session (M3 design 3.16, 7.3): each project's list, by its id, which its pages
 * fetch; labels are a project's only, the workspace's across its projects are M7's. Their order is nerve's (sort
 * order, then id), and the labels under a label at the top come from parent_id. The labels of a project the project
 * store no longer gives (deleted, left, or of a workspace no longer the caller's) do not show. Its service sends with
 * the session's client; changes go one at a time and the store writes nerve's answers (v0 design 7.7); fetches do
 * not queue.
 */
export class LabelStore implements ILabelStore {
  /** Each project's labels, reconciled between fetches and changes (reconciled.ts). */
  private readonly projects = new ReconciledByKey<Label[]>();
  // services
  private readonly service: LabelsService;
  /** The changes of the labels, sent one at a time. */
  private readonly changes = oneAtATime();
  // stores
  private readonly router: IRouterStore;
  /** The project as the caller sees it, by the project store (ProjectStore.getProjectById). */
  private readonly projectOf: (projectId: string | undefined | null) => Project | undefined;

  constructor(_rootStore: RootStore, api: ApiClient) {
    makeObservable(this, {
      // computed
      labelMap: computed,
      projectLabels: computed,
      projectLabelsTree: computed,
      // actions
      fetchProjectLabels: action,
      createLabel: action,
      updateLabel: action,
      updateLabelPosition: action,
      deleteLabel: action,
    });
    this.service = new LabelsService(api);
    this.router = _rootStore.router;
    this.projectOf = _rootStore.projectRoot.project.getProjectById;
  }

  get labelMap() {
    const shown = this.projects
      .values()
      .flat()
      .filter((label) => this.projectOf(label.project_id));
    return Object.fromEntries(shown.map((label) => [label.id, label]));
  }

  get projectLabels() {
    return this.getProjectLabels(this.router.projectId);
  }

  get projectLabelsTree() {
    const labels = this.projectLabels;
    return labels?.filter((label) => label.parent_id === null).map((top) => treeOf(top, labels));
  }

  /** @description the label, of a project the caller sees, as the store last had it from nerve */
  getLabelById = computedFn((labelId: string | null | undefined): Label | undefined =>
    labelId ? this.labelMap[labelId] : undefined
  );

  /**
   * @description the project's labels, by sort order, then id; undefined until fetched, or once the project store
   * no longer gives the project
   */
  getProjectLabels = computedFn((projectId: string | null | undefined): Label[] | undefined => {
    const project = this.projectOf(projectId);
    const labels = project && this.projects.get(project.id);
    return labels && inOrder(labels);
  });

  getProjectLabelIds = computedFn((projectId: string | null | undefined): string[] | undefined =>
    this.getProjectLabels(projectId)?.map((label) => label.id)
  );

  /**
   * @description fetches a project's labels, a member's to fetch as nerve refuses anyone else, and shows them with
   * the changes nerve confirmed meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or a
   * change of session cut (Reconciled.fetch)
   */
  fetchProjectLabels = (projectId: string): Promise<Label[] | undefined> =>
    this.projects.fetch(projectId, () => this.service.list(projectId));

  /**
   * @description creates a label, last of its project's: at the top unless data names a parent; fails, changing
   * nothing, when nerve refuses
   */
  createLabel = (projectId: string, data: LabelCreate): Promise<Label> =>
    this.changes(async () => {
      const label = await this.service.create(projectId, data);
      this.projects.confirm(label.project_id, upserted(label));
      return label;
    });

  /** @description changes a label; the store then shows nerve's answer. Fails, changing nothing, when nerve refuses. */
  updateLabel = (labelId: string, data: LabelUpdate): Promise<Label> => this.changes(() => this.send(labelId, data));

  /**
   * @description moves a label where it was dropped: under parentId (null for the top), before the label droppedOnId
   * names, or last there for none or at the end of the list. Its place among the labels under its new parent is
   * reckoned when the change goes out, from the labels as nerve last answered them (none among none: it takes the
   * parent alone); dropped under its own parent on no label, it stays and nothing is sent. Fails, changing nothing,
   * when nerve refuses or the store does not have it.
   */
  updateLabelPosition = (
    labelId: string,
    parentId: string | null,
    droppedOnId: string | undefined,
    dropAtEndOfList: boolean
  ): Promise<Label | undefined> =>
    this.changes(async () => {
      const label = this.held(labelId);
      if (label.parent_id === parentId && !droppedOnId) return undefined;
      const siblings = (this.getProjectLabels(label.project_id) ?? []).filter((held) => held.parent_id === parentId);
      const sortOrder = placeAt(
        siblings,
        "sort_order",
        droppedOnId,
        dropAtEndOfList ? "end" : "before",
        SORT_ORDER_STEP
      );
      return this.send(
        label.id,
        sortOrder === undefined ? { parent_id: parentId } : { parent_id: parentId, sort_order: sortOrder }
      );
    });

  /**
   * @description deletes a label and the labels under it; fails, changing nothing, when nerve refuses or the store
   * does not have it
   */
  deleteLabel = (labelId: string): Promise<void> =>
    this.changes(async () => {
      const label = this.held(labelId);
      await this.service.delete(label.id);
      this.projects.confirm(label.project_id, deleted(label.id));
    });

  /** Sends a change of a label, its turn come, and makes nerve's answer on its project's list. */
  private async send(labelId: string, data: LabelUpdate): Promise<Label> {
    const label = await this.service.update(labelId, data);
    this.projects.confirm(label.project_id, replaced(label));
    return label;
  }

  /** The label as the store has it; fails when it has none. */
  private held(labelId: string): Label {
    const label = this.getLabelById(labelId);
    if (!label) throw new Error("Label not found");
    return label;
  }
}
