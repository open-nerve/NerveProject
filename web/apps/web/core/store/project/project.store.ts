/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { sortBy } from "lodash-es";
import { action, computed, makeObservable } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type {
  ApiClient,
  IdentifierAvailability,
  Project,
  ProjectCreate,
  ProjectUpdate,
  Workspace,
} from "@nerve/api-client";
import type { TLoader } from "@nerve/types";
import { orderProjects, shouldFilterProject } from "@nerve/utils";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import { placeBetween } from "@/lib/place-between";
import type { Change } from "@/lib/reconciled";
import { ReconciledByKey, dropped, prepended, replaced, upserted } from "@/lib/reconciled";
// services
import { ProjectPreferencesService } from "@/services/project/project-preferences.service";
import { ProjectsService } from "@/services/project/projects.service";
// store
import type { RootStore } from "../root.store";
import type { IProjectFilterStore } from "./project_filter.store";

/** A workspace as the store fetches its projects: by its slug, kept under its id. */
type WorkspaceRef = Pick<Workspace, "id" | "slug">;
/** A project as the store changes it where no answer of nerve names its workspace. */
type ProjectRef = Pick<Project, "id" | "workspace_id">;
/**
 * nerve's gap between the places of the caller's sidebar: a project he is made a member of goes this far before his
 * first (M3 design 3.18).
 */
const SIDEBAR_STEP = 10000;

export interface IProjectStore {
  // computed
  /** "init-loader" until the current workspace's projects are fetched, then "loaded". */
  loader: TLoader;
  /** The current workspace's projects that are not archived, in nerve's order; undefined until fetched. */
  workspaceProjectIds: string[] | undefined;
  /** The current workspace's projects, the archived ones last, of the lists fetched. */
  totalProjectIds: string[] | undefined;
  /** The projects page's projects, by its filters and its order; undefined until both lists are fetched. */
  filteredProjectIds: string[] | undefined;
  /** The current workspace's projects the caller is a member of, not archived, by their place in his sidebar. */
  joinedProjectIds: string[];
  currentProjectDetails: Project | undefined;
  // computed actions
  getProjectById: (projectId: string | undefined | null) => Project | undefined;
  getProjectIdentifierById: (projectId: string | undefined | null) => string | undefined;
  getProjectByIdentifier: (projectIdentifier: string) => Project | undefined;
  // fetch actions
  fetchProjects: (workspace: WorkspaceRef) => Promise<Project[] | undefined>;
  fetchArchivedProjects: (workspace: WorkspaceRef) => Promise<Project[] | undefined>;
  fetchProject: (projectId: string) => Promise<Project | null | undefined>;
  checkProjectIdentifier: (workspaceSlug: string, identifier: string) => Promise<IdentifierAvailability>;
  // changes
  createProject: (workspaceSlug: string, data: ProjectCreate) => Promise<Project>;
  updateProject: (projectId: string, data: ProjectUpdate) => Promise<Project>;
  deleteProject: (project: ProjectRef) => Promise<void>;
  archiveProject: (projectId: string) => Promise<Project>;
  restoreProject: (projectId: string) => Promise<Project>;
  joinProject: (projectId: string) => Promise<Project>;
  leaveProject: (project: ProjectRef) => Promise<void>;
  updateProjectSortOrder: (project: ProjectRef, droppedOnId: string | undefined, dropAtEnd: boolean) => Promise<void>;
}

/**
 * The projects the caller sees, each as he sees it (M3 design 3.19, 7.3): two lists for each workspace, by its id,
 * the archived projects and the others, and each project's own read, by its id, which the project's pages fetch.
 * What the store holds of a workspace he is no longer a member of, it no longer gives (v0 design 7.7). Its services
 * send with the session's client; changes go one at a time and the store writes nerve's answers (7.7); fetches do not
 * queue.
 */
export class ProjectStore implements IProjectStore {
  /** Each workspace's projects that are not archived, reconciled between fetches and changes (reconciled.ts). */
  private readonly unarchived = new ReconciledByKey<Project[]>();
  /** Each workspace's archived projects. */
  private readonly archived = new ReconciledByKey<Project[]>();
  /** Each project as nerve last read it alone; null once deleted or left. */
  private readonly details = new ReconciledByKey<Project | null>();
  // services
  private readonly service: ProjectsService;
  private readonly preferences: ProjectPreferencesService;
  /** The changes of the projects, sent one at a time. */
  private readonly changes = oneAtATime();
  // stores
  private readonly rootStore: RootStore;
  /** The projects page's filters, by which filteredProjectIds picks and orders. */
  private readonly filters: IProjectFilterStore;

  constructor(_rootStore: RootStore, filters: IProjectFilterStore, api: ApiClient) {
    makeObservable(this, {
      // computed
      loader: computed,
      workspaceProjectIds: computed,
      totalProjectIds: computed,
      filteredProjectIds: computed,
      joinedProjectIds: computed,
      currentProjectDetails: computed,
      // actions
      fetchProjects: action,
      fetchArchivedProjects: action,
      fetchProject: action,
      createProject: action,
      updateProject: action,
      deleteProject: action,
      archiveProject: action,
      restoreProject: action,
      joinProject: action,
      leaveProject: action,
      updateProjectSortOrder: action,
    });
    this.rootStore = _rootStore;
    this.filters = filters;
    this.service = new ProjectsService(api);
    this.preferences = new ProjectPreferencesService(api);
  }

  /** The current workspace's two lists, each undefined until fetched. */
  private get currentLists() {
    const workspace = this.rootStore.workspaceRoot.currentWorkspace;
    return { unarchived: this.unarchived.get(workspace?.id), archived: this.archived.get(workspace?.id) };
  }

  get loader(): TLoader {
    return this.workspaceProjectIds === undefined ? "init-loader" : "loaded";
  }

  get workspaceProjectIds() {
    return this.currentLists.unarchived?.map((project) => project.id);
  }

  get totalProjectIds() {
    if (!this.rootStore.workspaceRoot.currentWorkspace) return undefined;
    const { unarchived, archived } = this.currentLists;
    return [...(unarchived ?? []), ...(archived ?? [])].map((project) => project.id);
  }

  get filteredProjectIds() {
    const {
      currentWorkspaceDisplayFilters: displayFilters,
      currentWorkspaceFilters: filters,
      searchQuery,
    } = this.filters;
    const { unarchived, archived } = this.currentLists;
    if (!displayFilters || !filters || !unarchived || !archived) return undefined;
    const query = searchQuery.toLowerCase();
    const found = [...unarchived, ...archived].filter(
      (project) =>
        (project.name.toLowerCase().includes(query) || project.identifier.toLowerCase().includes(query)) &&
        shouldFilterProject(project, displayFilters, filters)
    );
    return orderProjects(found, displayFilters.order_by).map((project) => project.id);
  }

  get joinedProjectIds() {
    return this.joinedIn(this.rootStore.workspaceRoot.currentWorkspace?.id).map((project) => project.id);
  }

  get currentProjectDetails() {
    return this.getProjectById(this.rootStore.router.projectId);
  }

  /**
   * The project as the store last had it from nerve, its own read first, else from its workspace's lists; nothing
   * once deleted or left, or when its workspace is no longer among the caller's.
   */
  getProjectById = computedFn((projectId: string | undefined | null): Project | undefined => {
    if (!projectId) return undefined;
    const workspaces = this.rootStore.workspaceRoot.workspaces ?? [];
    const read = this.details.get(projectId);
    if (read !== undefined) {
      return read && workspaces.some((workspace) => workspace.id === read.workspace_id) ? read : undefined;
    }
    for (const { id } of workspaces) {
      const listed = [...(this.unarchived.get(id) ?? []), ...(this.archived.get(id) ?? [])];
      const project = listed.find((held) => held.id === projectId);
      if (project) return project;
    }
    return undefined;
  });

  getProjectIdentifierById = computedFn(
    (projectId: string | undefined | null): string | undefined => this.getProjectById(projectId)?.identifier
  );

  /** The current workspace's project that identifier names. */
  getProjectByIdentifier = computedFn((projectIdentifier: string): Project | undefined => {
    const { unarchived, archived } = this.currentLists;
    return [...(unarchived ?? []), ...(archived ?? [])].find((project) => project.identifier === projectIdentifier);
  });

  /**
   * @description fetches the workspace's projects that are not archived, and shows them with the changes nerve
   * confirmed meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or a change of session cut
   * (Reconciled.fetch)
   */
  fetchProjects = (workspace: WorkspaceRef): Promise<Project[] | undefined> =>
    this.unarchived.fetch(workspace.id, () => this.service.list(workspace.slug, false));

  /** @description fetches the workspace's archived projects, as fetchProjects */
  fetchArchivedProjects = (workspace: WorkspaceRef): Promise<Project[] | undefined> =>
    this.archived.fetch(workspace.id, () => this.service.list(workspace.slug, true));

  /** @description reads the project alone, as the caller sees it (a page of the project), as fetchProjects */
  fetchProject = (projectId: string): Promise<Project | null | undefined> =>
    this.details.fetch(projectId, () => this.service.get(projectId));

  /** @description whether identifier can name a new project of the workspace */
  checkProjectIdentifier = (workspaceSlug: string, identifier: string): Promise<IdentifierAvailability> =>
    this.service.checkIdentifier(workspaceSlug, identifier);

  /**
   * @description creates a project: it comes first in its workspace's list, as nerve lists it, the caller's sidebar
   * having it first. Fails, changing nothing, when nerve refuses.
   */
  createProject = (workspaceSlug: string, data: ProjectCreate): Promise<Project> =>
    this.changes(async () => {
      const project = await this.service.create(workspaceSlug, data);
      this.unarchived.confirm(project.workspace_id, prepended([project]));
      return project;
    });

  /** @description changes a project; the store then shows nerve's answer. Fails, changing nothing, when refused. */
  updateProject = (projectId: string, data: ProjectUpdate): Promise<Project> =>
    this.changes(async () => {
      const project = await this.service.update(projectId, data);
      this.unarchived.confirm(project.workspace_id, replaced(project));
      this.details.confirm(project.id, () => project);
      return project;
    });

  /** @description deletes a project, which the store then no longer gives; fails, changing nothing, when refused */
  deleteProject = (project: ProjectRef): Promise<void> =>
    this.changes(async () => {
      await this.service.delete(project.id);
      this.forget(project);
    });

  /** @description archives a project: it moves to the end of its workspace's archived list until the next fetch */
  archiveProject = (projectId: string): Promise<Project> =>
    this.changes(async () => {
      const project = await this.service.archive(projectId);
      this.unarchived.confirm(project.workspace_id, dropped(project.id));
      this.archived.confirm(project.workspace_id, upserted(project));
      this.details.confirm(project.id, () => project);
      return project;
    });

  /** @description unarchives a project: it moves to the end of its workspace's list until the next fetch */
  restoreProject = (projectId: string): Promise<Project> =>
    this.changes(async () => {
      const project = await this.service.unarchive(projectId);
      this.archived.confirm(project.workspace_id, dropped(project.id));
      this.unarchived.confirm(project.workspace_id, upserted(project));
      this.details.confirm(project.id, () => project);
      return project;
    });

  /**
   * @description makes the caller a member of a project; the store then shows it as he now sees it, in its place in
   * its list, else last. Fails, changing nothing, when nerve refuses.
   */
  joinProject = (projectId: string): Promise<Project> =>
    this.changes(async () => {
      const project = await this.service.join(projectId);
      (project.archived_at ? this.archived : this.unarchived).confirm(project.workspace_id, upserted(project));
      this.details.confirm(project.id, () => project);
      return project;
    });

  /**
   * @description ends the caller's membership of a project, which the store then no longer gives (nerve may still
   * list a public one to him: the next fetch shows it). Fails, changing nothing, when nerve refuses (its only admin).
   */
  leaveProject = (project: ProjectRef): Promise<void> =>
    this.changes(async () => {
      await this.service.leave(project.id);
      this.forget(project);
    });

  /**
   * @description moves a project in the caller's sidebar where he dropped it: before the project droppedOnId names, or
   * last for none or at the end. Its place is reckoned in the change's turn, from his projects as nerve last answered
   * them; the store then shows the place nerve gives it. Fails, changing nothing, when nerve refuses.
   */
  updateProjectSortOrder = (project: ProjectRef, droppedOnId: string | undefined, dropAtEnd: boolean): Promise<void> =>
    this.changes(async () => {
      const joined = this.joinedIn(project.workspace_id);
      const droppedOn = joined.findIndex((held) => held.id === droppedOnId);
      const at = dropAtEnd || droppedOn === -1 ? joined.length : droppedOn;
      const sortOrder = placeBetween(joined[at - 1]?.sort_order, joined[at]?.sort_order, SIDEBAR_STEP);
      if (sortOrder === undefined) return;
      const { sort_order } = await this.preferences.update(project.id, { sort_order: sortOrder });
      const placed: Change<Project[]> = (list) =>
        list.map((held) => (held.id === project.id ? { ...held, sort_order } : held));
      this.unarchived.confirm(project.workspace_id, placed);
      this.details.confirm(project.id, (held) => held && { ...held, sort_order });
    });

  /** The workspace's projects the caller is a member of, not archived, by their place in his sidebar. */
  private joinedIn(workspaceId: string | undefined): Project[] {
    const joined = (this.unarchived.get(workspaceId) ?? []).filter((project) => project.member_role !== null);
    return sortBy(joined, "sort_order");
  }

  /** The project leaves both lists, and its own read is gone. */
  private forget(project: ProjectRef) {
    this.unarchived.confirm(project.workspace_id, dropped(project.id));
    this.archived.confirm(project.workspace_id, dropped(project.id));
    this.details.confirm(project.id, () => null);
  }
}
