/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, computed, makeObservable } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, Project, State, StateCreate, StateGroup, StateUpdate, Workspace } from "@nerve/api-client";
import { STATE_GROUPS } from "@nerve/constants";
import { sortStates } from "@nerve/utils";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import { placeBetween } from "@/lib/place-between";
import type { Change } from "@/lib/reconciled";
import { ReconciledByKey, dropped, replaced, upserted } from "@/lib/reconciled";
// services
import { StatesService } from "@/services/project/states.service";
// store
import type { RootStore } from "@/store/root.store";
import type { IRouterStore } from "@/store/router.store";

/** A workspace as the store fetches its states: by its slug, kept under its id. */
type WorkspaceRef = Pick<Workspace, "id" | "slug">;
/** nerve's gap between a project's last state and a new one (M3 design 3.17). */
const SEQUENCE_STEP = 15000;

export interface IStateStore {
  // computed
  /** Every state the store shows, by id, which the work items' stores look up. */
  stateMap: Record<string, State>;
  /** The current workspace's states, by group, then sequence; undefined until fetched. */
  workspaceStates: State[] | undefined;
  /** The address's project's states, by group, then sequence; undefined until fetched. */
  projectStates: State[] | undefined;
  /** The address's project's states in each group, every group there; undefined until fetched. */
  groupedProjectStates: Record<string, State[]> | undefined;
  // computed actions
  getStateById: (stateId: string | null | undefined) => State | undefined;
  getProjectStates: (projectId: string | null | undefined) => State[] | undefined;
  getProjectStateIds: (projectId: string | null | undefined) => string[] | undefined;
  getStatePercentageInGroup: (stateId: string | null | undefined) => number | undefined;
  // fetch actions
  fetchProjectStates: (projectId: string) => Promise<State[] | undefined>;
  fetchWorkspaceStates: (workspace: WorkspaceRef) => Promise<State[] | undefined>;
  // changes
  createState: (projectId: string, data: StateCreate) => Promise<State>;
  updateState: (stateId: string, data: StateUpdate) => Promise<State>;
  moveState: (stateId: string, group: StateGroup, droppedOnId: string | undefined, after: boolean) => Promise<State>;
  deleteState: (stateId: string) => Promise<void>;
  markStateAsDefault: (stateId: string) => Promise<void>;
}

/**
 * The states of the projects of a session (M3 design 3.17, 7.3): each project's list, by its id, which its pages
 * fetch, and each workspace's, by its id, the states of its projects the caller is a member of. Their order is
 * sortStates' (group, then sequence), and a state's place in its group is computed from it: nerve gives no order. The
 * states of a project the project store no longer gives (deleted, left, or of a workspace no longer the caller's) do
 * not show. Its service sends with the session's client; changes go one at a time and the store writes nerve's
 * answers (v0 design 7.7); fetches do not queue.
 */
export class StateStore implements IStateStore {
  /** Each project's states, reconciled between fetches and changes (reconciled.ts). */
  private readonly projects = new ReconciledByKey<State[]>();
  /** Each workspace's states. */
  private readonly workspaces = new ReconciledByKey<State[]>();
  // services
  private readonly service: StatesService;
  /** The changes of the states, sent one at a time. */
  private readonly changes = oneAtATime();
  // stores
  private readonly router: IRouterStore;
  private readonly rootStore: RootStore;
  /** The project as the caller sees it, by the project store (ProjectStore.getProjectById). */
  private readonly projectOf: (projectId: string | undefined | null) => Project | undefined;

  constructor(_rootStore: RootStore, api: ApiClient) {
    makeObservable(this, {
      // computed
      stateMap: computed,
      workspaceStates: computed,
      projectStates: computed,
      groupedProjectStates: computed,
      // actions
      fetchProjectStates: action,
      fetchWorkspaceStates: action,
      createState: action,
      updateState: action,
      moveState: action,
      deleteState: action,
      markStateAsDefault: action,
    });
    this.service = new StatesService(api);
    this.router = _rootStore.router;
    this.rootStore = _rootStore;
    this.projectOf = _rootStore.projectRoot.project.getProjectById;
  }

  get stateMap() {
    // the projects' lists last: a state in both shows as its project's list has it, as getProjectStates gives it
    const held = [...this.workspaces.values(), ...this.projects.values()].flat();
    return Object.fromEntries(
      held.filter((state) => this.projectOf(state.project_id)).map((state) => [state.id, state])
    );
  }

  get workspaceStates() {
    const states = this.workspaces.get(this.rootStore.workspaceRoot.currentWorkspace?.id);
    return states && sortStates(states.filter((state) => this.projectOf(state.project_id)));
  }

  get projectStates() {
    return this.getProjectStates(this.router.projectId);
  }

  get groupedProjectStates() {
    const states = this.projectStates;
    if (!states) return undefined;
    return Object.fromEntries(
      Object.keys(STATE_GROUPS).map((group) => [group, states.filter((state) => state.group === group)])
    );
  }

  /** @description the state, of a project the caller sees, as the store last had it from nerve */
  getStateById = computedFn((stateId: string | null | undefined): State | undefined =>
    stateId ? this.stateMap[stateId] : undefined
  );

  /**
   * @description the project's states, by group, then sequence: its own list, else its workspace's; undefined until
   * fetched, or once the project store no longer gives the project
   */
  getProjectStates = computedFn((projectId: string | null | undefined): State[] | undefined => {
    const project = this.projectOf(projectId);
    if (!project) return undefined;
    const listed =
      this.projects.get(project.id) ??
      this.workspaces.get(project.workspace_id)?.filter((state) => state.project_id === project.id);
    return listed && sortStates(listed);
  });

  getProjectStateIds = computedFn((projectId: string | null | undefined): string[] | undefined =>
    this.getProjectStates(projectId)?.map((state) => state.id)
  );

  /** @description the state's place in its group, in its project's order: 100 for the group's last, as a percentage */
  getStatePercentageInGroup = computedFn((stateId: string | null | undefined): number | undefined => {
    const state = this.getStateById(stateId);
    if (!state) return undefined;
    const group = this.getProjectStates(state.project_id)?.filter((held) => held.group === state.group) ?? [];
    const place = group.findIndex((held) => held.id === state.id);
    return place === -1 ? undefined : ((place + 1) / group.length) * 100;
  });

  /**
   * @description fetches a project's states, a member's to fetch as nerve refuses anyone else, and shows them with
   * the changes nerve confirmed meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or a
   * change of session cut (Reconciled.fetch)
   */
  fetchProjectStates = (projectId: string): Promise<State[] | undefined> =>
    this.projects.fetch(projectId, () => this.service.list(projectId));

  /** @description fetches the states of the workspace's projects the caller is a member of, as fetchProjectStates */
  fetchWorkspaceStates = (workspace: WorkspaceRef): Promise<State[] | undefined> =>
    this.workspaces.fetch(workspace.id, () => this.service.listInWorkspace(workspace.slug));

  /** @description creates a state, last of its project's; fails, changing nothing, when nerve refuses */
  createState = (projectId: string, data: StateCreate): Promise<State> =>
    this.changes(async () => {
      const state = await this.service.create(projectId, data);
      this.confirm(state, upserted(state));
      return state;
    });

  /**
   * @description changes a state, its place among its project's (group, sequence) too; the store then shows nerve's
   * answer. Fails, changing nothing, when nerve refuses.
   */
  updateState = (stateId: string, data: StateUpdate): Promise<State> => this.changes(() => this.send(stateId, data));

  /**
   * @description moves a state where it was dropped: into group, before the state droppedOnId names (after it, for
   * after), or last of the group for none. Its sequence is reckoned in the change's turn, from the states as nerve last
   * answered them; the store then shows nerve's answer. Fails, changing nothing, when nerve refuses or the store does
   * not have it.
   */
  moveState = (stateId: string, group: StateGroup, droppedOnId: string | undefined, after: boolean): Promise<State> =>
    this.changes(async () => {
      const state = this.held(stateId);
      const siblings = (this.getProjectStates(state.project_id) ?? []).filter((held) => held.group === group);
      const droppedOn = siblings.findIndex((held) => held.id === droppedOnId);
      const at = droppedOn === -1 ? siblings.length : droppedOn + (after ? 1 : 0);
      const sequence = placeBetween(siblings[at - 1]?.sequence, siblings[at]?.sequence, SEQUENCE_STEP);
      return this.send(state.id, sequence === undefined ? { group } : { group, sequence });
    });

  /** @description deletes a state; fails, changing nothing, when nerve refuses or the store does not have it */
  deleteState = (stateId: string): Promise<void> =>
    this.changes(async () => {
      const state = this.held(stateId);
      await this.service.delete(state.id);
      this.confirm(state, dropped(state.id));
    });

  /**
   * @description makes a state its project's default, and the one that was no longer; fails, changing nothing, when
   * nerve refuses or the store does not have it
   */
  markStateAsDefault = (stateId: string): Promise<void> =>
    this.changes(async () => {
      const state = this.held(stateId);
      await this.service.markDefault(state.id);
      this.confirm(state, (list) =>
        list.map((held) => (held.project_id === state.project_id ? { ...held, default: held.id === state.id } : held))
      );
    });

  /** Sends a change of a state, its turn come, and makes nerve's answer on its project's list and its workspace's. */
  private async send(stateId: string, data: StateUpdate): Promise<State> {
    const state = await this.service.update(stateId, data);
    this.confirm(state, replaced(state));
    return state;
  }

  /** The state as the store has it; fails when it has none. */
  private held(stateId: string): State {
    const state = this.getStateById(stateId);
    if (!state) throw new Error("State not found");
    return state;
  }

  /** A change nerve confirmed to a state's project: made on its project's list and on its workspace's. */
  private confirm(state: Pick<State, "project_id" | "workspace_id">, change: Change<State[]>): void {
    this.projects.confirm(state.project_id, change);
    this.workspaces.confirm(state.workspace_id, change);
  }
}
