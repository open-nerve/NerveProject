/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { maxBy, sortBy } from "lodash-es";
import { action, comparer, computed, makeObservable, observable, reaction, runInAction } from "mobx";
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
import { placeAt } from "@/lib/place-between";
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
type DirectoryRead = { source: ReconciledByKey<Project[]>; workspaceId: string; answeredAt: number };
type ProjectRead = { directories: DirectoryRead[]; positiveWatermark: number };
/** A field of a project that is on or off, as nerve's ProjectUpdate changes it: a feature, the guests' view. */
export type ProjectToggleField = {
  [K in keyof ProjectUpdate]-?: Exclude<ProjectUpdate[K], undefined> extends boolean ? K : never;
}[keyof ProjectUpdate];
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
  knownProjectIds: (workspaceId: string) => string[];
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
  toggleProject: (projectId: string, field: ProjectToggleField) => Promise<Project>;
  toggleAutoArchive: (projectId: string) => Promise<Project>;
  deleteProject: (project: ProjectRef) => Promise<void>;
  archiveProject: (projectId: string) => Promise<Project>;
  restoreProject: (projectId: string) => Promise<Project>;
  joinProject: (projectId: string) => Promise<Project>;
  leaveProject: (project: ProjectRef) => Promise<void>;
  updateProjectSortOrder: (project: ProjectRef, droppedOnId: string | undefined, dropAtEnd: boolean) => Promise<void>;
  /** Confirm an asynchronous change only while its original project facts remain current, including across recovery. */
  withCurrentProject: <T>(projectId: string, send: () => Promise<T>, confirm: (result: T) => void) => Promise<T>;
  confirmProject: (project: ProjectRef, change: Change<Project>) => void;
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
  /** Directory keys survive temporary workspace absence, so dispatch snapshots include earlier accepted reads. */
  private readonly directoryWorkspaces = new Set<string>();
  /**
   * Each project as nerve last answered it alone, by a read or a full write response; null once deleted or left: then only a list gives it (a fetch out
   * meanwhile drops it from its answer).
   */
  private readonly details = new ReconciledByKey<Project | null>();
  /** Only provenance: after a newer presence disappears, an older copy in the other class cannot take its place. */
  private readonly latestPresence = observable.map<
    string,
    { workspaceId: string; archived: boolean; answeredAt: number }
  >();
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
      toggleProject: action,
      toggleAutoArchive: action,
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
    return {
      unarchived: this.visibleDirectory(workspace?.id, false),
      archived: this.visibleDirectory(workspace?.id, true),
    };
  }

  /** Every public directory projection uses the same current facts as permissions and project details. */
  private visibleDirectory(workspaceId: string | undefined, archived: boolean): Project[] | undefined {
    return (archived ? this.archived : this.unarchived).get(workspaceId)?.flatMap(({ id }) => {
      const project = this.getProjectById(id);
      return project && project.workspace_id === workspaceId && (project.archived_at !== null) === archived
        ? [project]
        : [];
    });
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
   * The project as nerve last answered it, of the copies the store holds: its own read's and its workspace's lists'
   * (a list fetched after the read shows a change made meanwhile, a role for one; P8b's F-3). The changes nerve
   * confirmed are made on every copy that holds the project; an update of the project is not made on the archived
   * list, since nerve changes no archived project (v0 design 7.7). Nothing once deleted or left until a list has it
   * again (a public project, to one no longer its member), or when its workspace is no longer among the caller's.
   */
  getProjectById = computedFn((projectId: string | undefined | null): Project | undefined => {
    if (!projectId) return undefined;
    const workspaces = this.rootStore.workspaceRoot.workspaces ?? [];
    const copies: { project: Project; answeredAt: number }[] = [];
    const read = this.details.get(projectId);
    if (read) copies.push({ project: read, answeredAt: this.details.answeredAt(projectId) });
    for (const { id } of workspaces) {
      for (const lists of [this.unarchived, this.archived]) {
        const project = lists.get(id)?.find((held) => held.id === projectId);
        if (project) copies.push({ project, answeredAt: lists.answeredAt(id) });
      }
    }
    const newest = maxBy(copies, "answeredAt");
    if (!newest || !workspaces.some((workspace) => workspace.id === newest.project.workspace_id)) return undefined;
    const { project, answeredAt } = newest;
    if ((this.latestPresence.get(projectId)?.answeredAt ?? 0) > answeredAt) return undefined;
    const directory = project.archived_at ? this.archived : this.unarchived;
    // A complete directory also answers absence, but only for its workspace and archive class.
    if (
      directory.answeredAt(project.workspace_id) > answeredAt &&
      !directory.get(project.workspace_id)?.some((listed) => listed.id === projectId)
    )
      return undefined;
    return project;
  });

  /** IDs represented in the workspace's lists or individual reads, for permission-scope observation. */
  knownProjectIds = (workspaceId: string): string[] => [
    ...new Set([
      ...(this.unarchived.get(workspaceId) ?? []).map((project) => project.id),
      ...(this.archived.get(workspaceId) ?? []).map((project) => project.id),
      ...this.details.values().flatMap((project) => (project?.workspace_id === workspaceId ? [project.id] : [])),
      ...[...this.latestPresence].flatMap(([id, presence]) => (presence.workspaceId === workspaceId ? [id] : [])),
    ]),
  ];

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
  fetchProjects = (workspace: WorkspaceRef): Promise<Project[] | undefined> => this.fetchDirectory(workspace, false);

  /** @description fetches the workspace's archived projects, as fetchProjects */
  fetchArchivedProjects = (workspace: WorkspaceRef): Promise<Project[] | undefined> =>
    this.fetchDirectory(workspace, true);

  /** @description reads the project alone, as the caller sees it (a page of the project), as fetchProjects */
  fetchProject = (projectId: string): Promise<Project | null | undefined> =>
    this.rootStore.observeProfileRead({ kind: "project", id: projectId }, async () => {
      const directories = this.captureFacts();
      const project = await this.details.fetch(projectId, async () => {
        const answer = await this.service.get(projectId);
        return this.afterFacts(answer, directories).project;
      });
      if (project && this.details.get(projectId) === project)
        this.notePresence(project, this.details.answeredAt(projectId));
      return project;
    });

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
      this.unarchived.confirm(project.workspace_id, this.inDirectory(project, false, prepended([project])));
      return project;
    });

  /** @description changes a project; the store then shows nerve's answer. Fails, changing nothing, when refused. */
  updateProject = (projectId: string, data: ProjectUpdate): Promise<Project> =>
    this.changes(() => this.send(projectId, data));

  /**
   * @description turns a field of a project that is on or off the other way, from its value as nerve last answered it,
   * in the change's turn: two quick turns end where they began. Fails, changing nothing, when nerve refuses, or asking
   * nerve nothing when the store does not give the project.
   */
  toggleProject = (projectId: string, field: ProjectToggleField): Promise<Project> =>
    this.changes(async () => {
      // set by its key, which checks the value against ProjectUpdate (a computed key in a literal would not)
      const data: ProjectUpdate = {};
      data[field] = !this.held(projectId)[field];
      return this.send(projectId, data);
    });

  /**
   * @description turns the archiving of a project's closed work items on, after a month, or off, as toggleProject:
   * from the value nerve last answered, in the change's turn
   */
  toggleAutoArchive = (projectId: string): Promise<Project> =>
    this.changes(() => this.send(projectId, { archive_in: this.held(projectId).archive_in === 0 ? 1 : 0 }));

  /** @description deletes a project, which the store then no longer gives; fails, changing nothing, when refused */
  deleteProject = (project: ProjectRef): Promise<void> =>
    this.changes(async () => {
      await this.service.delete(project.id);
      this.forget(project);
    });

  /** @description archives a project: it moves to the end of its workspace's archived list until the next fetch */
  archiveProject = (projectId: string): Promise<Project> =>
    this.changes(() =>
      this.changeProject(
        projectId,
        () => this.service.archive(projectId),
        (project) => {
          this.unarchived.confirm(project.workspace_id, this.inDirectory(project, false, dropped(project.id)));
          this.archived.confirm(project.workspace_id, this.inDirectory(project, true, upserted(project)));
        }
      )
    );

  /** @description unarchives a project: it moves to the end of its workspace's list until the next fetch */
  restoreProject = (projectId: string): Promise<Project> =>
    this.changes(() =>
      this.changeProject(
        projectId,
        () => this.service.unarchive(projectId),
        (project) => {
          this.archived.confirm(project.workspace_id, this.inDirectory(project, true, dropped(project.id)));
          this.unarchived.confirm(project.workspace_id, this.inDirectory(project, false, upserted(project)));
        }
      )
    );

  /**
   * @description makes the caller a member of a project; the store then shows it as he now sees it, in its place in
   * its list, else last. Fails, changing nothing, when nerve refuses.
   */
  joinProject = (projectId: string): Promise<Project> =>
    this.changes(() =>
      this.changeProject(
        projectId,
        () => this.service.join(projectId),
        (project) => {
          (project.archived_at ? this.archived : this.unarchived).confirm(
            project.workspace_id,
            this.inDirectory(project, project.archived_at !== null, upserted(project))
          );
        }
      )
    );

  /**
   * @description ends the caller's membership of a project, which the store then no longer gives (nerve may still
   * list a public one to him: the next fetch shows it). Fails, changing nothing, when nerve refuses (its only admin).
   */
  leaveProject = (project: ProjectRef): Promise<void> =>
    this.changes(() =>
      this.withCurrentProject(
        project.id,
        () => this.service.leave(project.id),
        () => this.forget(project)
      )
    );

  /**
   * @description moves a project in the caller's sidebar where he dropped it: before the project droppedOnId names, or
   * last for none or at the end. Its place is reckoned in the change's turn, from his projects as nerve last answered
   * them; the store then shows the place nerve gives it. Fails, changing nothing, when nerve refuses, or asking nerve
   * nothing when his projects no longer have it in its turn (left, archived or deleted before it).
   */
  updateProjectSortOrder = (project: ProjectRef, droppedOnId: string | undefined, dropAtEnd: boolean): Promise<void> =>
    this.changes(async () => {
      const joined = this.joinedIn(project.workspace_id);
      if (!joined.some((held) => held.id === project.id)) throw new Error("Project not found");
      const sortOrder = placeAt(joined, "sort_order", droppedOnId, dropAtEnd ? "end" : "before", SIDEBAR_STEP);
      if (sortOrder === undefined) return;
      await this.withCurrentProject(
        project.id,
        () => this.preferences.update(project.id, { sort_order: sortOrder }),
        ({ sort_order }) => {
          this.confirmProject(project, (held) => ({ ...held, sort_order }));
        }
      );
    });

  /**
   * @description a change nerve confirmed to what the caller sees of a project, by another store (its members, his
   * role in it) or this one: it is made wherever the store shows the project, and on the answers of the fetches out
   * (reconciled.ts)
   */
  confirmProject = (project: ProjectRef, change: Change<Project>): void => {
    const inPlace: Change<Project[]> = (list) => list.map((held) => (held.id === project.id ? change(held) : held));
    const held = this.getProjectById(project.id);
    if (!held) return;
    this.unarchived.confirm(project.workspace_id, this.inDirectory(held, false, inPlace));
    this.archived.confirm(project.workspace_id, this.inDirectory(held, true, inPlace));
    this.details.confirm(
      project.id,
      this.untilFact(
        held,
        (value) => value && change(value),
        (_value, current) => current
      )
    );
  };

  /** Sends a change of a project, its turn come, and makes nerve's answer on its list and its own read. */
  private async send(projectId: string, data: ProjectUpdate): Promise<Project> {
    return this.changeProject(
      projectId,
      () => this.service.update(projectId, data),
      (project) => {
        this.unarchived.confirm(project.workspace_id, this.inDirectory(project, false, replaced(project)));
      }
    );
  }

  /** Provenance at dispatch, for every source: positive answers and each complete directory's coverage. */
  private captureFacts(): ProjectRead {
    const directories = [this.unarchived, this.archived].flatMap((source) =>
      [...this.directoryWorkspaces].map((id) => ({ source, workspaceId: id, answeredAt: source.answeredAt(id) }))
    );
    // Reconciled publishes an answer before the fetch continuation records its provenance. A reaction may
    // dispatch another request in that interval; its snapshot must already include the accepted answer.
    const accepted = [
      ...directories.map(({ answeredAt }) => answeredAt),
      ...this.details.values().flatMap((project) => (project ? [this.details.answeredAt(project.id)] : [])),
      ...[...this.latestPresence.values()].map(({ answeredAt }) => answeredAt),
    ];
    return {
      positiveWatermark: accepted.reduce((latest, answeredAt) => Math.max(latest, answeredAt), 0),
      directories,
    };
  }

  private async fetchDirectory(workspace: WorkspaceRef, archived: boolean): Promise<Project[] | undefined> {
    this.directoryWorkspaces.add(workspace.id);
    const source = archived ? this.archived : this.unarchived;
    const reads = this.captureFacts();
    const reading = source.fetch(workspace.id, () => this.service.list(workspace.slug, archived));
    // Reconcile cross-source facts during the existing read replay, immediately before publication.
    // Doing this in the async read callback leaves another microtask in which a detail can be accepted.
    source.confirm(workspace.id, (value, answered) => {
      if (answered === undefined) return value;
      const currentRows = value.flatMap((answer) => {
        const current = this.afterFacts(answer, reads).project;
        return current && (current.archived_at !== null) === archived ? [current] : [];
      });
      const projects = new Map(currentRows.map((project) => [project.id, project]));
      // Absence in this older answer cannot revoke a detail accepted while the directory was out.
      // Keep only currently visible proof in this exact workspace and archive class.
      for (const detail of this.details.values()) {
        if (!detail || this.details.answeredAt(detail.id) <= reads.positiveWatermark) continue;
        const current = this.getProjectById(detail.id);
        if (current?.workspace_id === workspace.id && (current.archived_at !== null) === archived)
          projects.set(current.id, current);
      }
      return [...projects.values()];
    });
    const projects = await reading;
    if (projects && source.get(workspace.id) === projects) {
      runInAction(() => projects.forEach((project) => this.notePresence(project, source.answeredAt(workspace.id))));
    }
    return projects;
  }

  private notePresence(project: Project, answeredAt: number): void {
    if ((this.latestPresence.get(project.id)?.answeredAt ?? 0) > answeredAt) return;
    runInAction(() =>
      this.latestPresence.set(project.id, {
        workspaceId: project.workspace_id,
        archived: project.archived_at !== null,
        answeredAt,
      })
    );
  }

  /** Newer positive proof or a complete directory supersedes an older request, including absence in its class. */
  private afterFacts(answer: Project, reads: ProjectRead): { newer: boolean; project: Project | null } {
    if ((this.latestPresence.get(answer.id)?.answeredAt ?? 0) > reads.positiveWatermark) {
      return { newer: true, project: this.getProjectById(answer.id) ?? null };
    }
    const newer = [this.unarchived, this.archived].flatMap((source) => {
      const workspaceId = answer.workspace_id;
      const before =
        reads.directories.find((read) => read.source === source && read.workspaceId === workspaceId)?.answeredAt ?? 0;
      return source.answeredAt(workspaceId) > before ? [{ source, workspaceId }] : [];
    });
    const copies = newer.flatMap(({ source, workspaceId }) => {
      const project = source.get(workspaceId)?.find((listed) => listed.id === answer.id);
      return project ? [{ project, answeredAt: source.answeredAt(workspaceId) }] : [];
    });
    const project = maxBy(copies, "answeredAt")?.project;
    if (project) return { newer: true, project };
    const knownClass = this.latestPresence.get(answer.id)?.archived ?? answer.archived_at !== null;
    const directory = knownClass ? this.archived : this.unarchived;
    const absent = newer.some(({ source }) => source === directory);
    return { newer: absent, project: absent ? null : answer };
  }

  /** Reconcile old confirmations with newer directory facts, never leave a revoked row in an old raw response. */
  private untilFact<V>(
    project: Project,
    change: Change<V>,
    resolved: (value: V, project: Project | null) => V
  ): Change<V> {
    const reads = this.captureFacts();
    return (value, answered) => {
      const fact = this.afterFacts(project, reads);
      return fact.newer ? resolved(value, fact.project) : change(value, answered);
    };
  }

  private inDirectory(project: Project, archived: boolean, change: Change<Project[]>): Change<Project[]> {
    return this.untilFact(project, change, (value, current) =>
      current && (current.archived_at !== null) === archived
        ? upserted(current)(value)
        : dropped<Project>(project.id)(value)
    );
  }

  /**
   * A write may confirm its data while the facts it started with still stand. An intervening change of fact,
   * including disappearance followed by recovery, makes its old response unsuitable for updating the store.
   * The transient observation ends on success or failure; identical directory refreshes keep the write valid.
   */
  withCurrentProject = async <T>(
    projectId: string,
    send: () => Promise<T>,
    confirm: (result: T) => void
  ): Promise<T> => {
    let current = true;
    const stop = reaction(
      () => this.getProjectById(projectId),
      () => {
        current = false;
      },
      { equals: comparer.structural }
    );
    let result: T;
    try {
      result = await send();
    } finally {
      stop();
    }
    if (current) runInAction(() => confirm(result));
    return result;
  };

  private changeProject(
    projectId: string,
    send: () => Promise<Project>,
    confirm: (project: Project) => void
  ): Promise<Project> {
    const reads = this.captureFacts();
    return this.withCurrentProject(projectId, send, (project) => {
      if (this.afterFacts(project, reads).project === null) return;
      confirm(project);
      // A confirmed full response is independent positive proof, including when no detail was loaded before it.
      const choose = this.untilFact<Project | null>(
        project,
        () => project,
        (_value, current) => current
      );
      this.details.receive(project.id, project, (held) => choose(held ?? null));
      this.notePresence(project, this.details.answeredAt(project.id));
    });
  }

  /** The project as the store gives it; fails when it gives none. */
  private held(projectId: string): Project {
    const project = this.getProjectById(projectId);
    if (!project) throw new Error("Project not found");
    return project;
  }

  /** The workspace's projects the caller is a member of, not archived, by their place in his sidebar. */
  private joinedIn(workspaceId: string | undefined): Project[] {
    const joined = (this.visibleDirectory(workspaceId, false) ?? []).filter((project) => project.member_role !== null);
    return sortBy(joined, "sort_order");
  }

  /** The project leaves both lists, and its own read is gone. */
  private forget(project: ProjectRef) {
    this.unarchived.confirm(project.workspace_id, dropped(project.id));
    this.archived.confirm(project.workspace_id, dropped(project.id));
    this.details.confirm(project.id, () => null);
  }
}
