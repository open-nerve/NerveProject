/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { sortBy, cloneDeep, update, set } from "lodash-es";
import { observable, action, computed, makeObservable, runInAction } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type { TFetchStatus, TLoader } from "@nerve/types";
// helpers
import { orderProjects, shouldFilterProject } from "@nerve/utils";
// services
import type { Project } from "@nerve/api-client";
import { IssueLabelService, IssueService } from "@/services/issue";
import { ProjectService, ProjectStateService, ProjectArchiveService } from "@/services/project";
// store
import type { RootStore } from "../root.store";

export interface IProjectStore {
  // observables
  isUpdatingProject: boolean;
  loader: TLoader;
  fetchStatus: TFetchStatus;
  projectMap: Record<string, Project>; // projectId: project info
  // computed
  isInitializingProjects: boolean;
  filteredProjectIds: string[] | undefined;
  workspaceProjectIds: string[] | undefined;
  archivedProjectIds: string[] | undefined;
  totalProjectIds: string[] | undefined;
  joinedProjectIds: string[];
  currentProjectDetails: Project | undefined;
  // actions
  getProjectById: (projectId: string | undefined | null) => Project | undefined;
  getProjectIdentifierById: (projectId: string | undefined | null) => string;
  getProjectByIdentifier: (projectIdentifier: string) => Project | undefined;
  // helper actions
  processProjectAfterCreation: (workspaceSlug: string, data: Project) => void;

  // fetch actions
  fetchPartialProjects: (workspaceSlug: string) => Promise<Project[]>;
  fetchProjects: (workspaceSlug: string) => Promise<Project[]>;
  fetchProjectDetails: (workspaceSlug: string, projectId: string) => Promise<Project>;
  // project-view action
  updateProjectView: (workspaceSlug: string, projectId: string, viewProps: any) => Promise<any>;
  // CRUD actions
  createProject: (workspaceSlug: string, data: Partial<Project>) => Promise<Project>;
  updateProject: (workspaceSlug: string, projectId: string, data: Partial<Project>) => Promise<Project>;
  deleteProject: (workspaceSlug: string, projectId: string) => Promise<void>;
  // archive actions
  archiveProject: (workspaceSlug: string, projectId: string) => Promise<void>;
  restoreProject: (workspaceSlug: string, projectId: string) => Promise<void>;
}

export class ProjectStore implements IProjectStore {
  // observables
  isUpdatingProject: boolean = false;
  loader: TLoader = "init-loader";
  fetchStatus: TFetchStatus = undefined;
  projectMap: Record<string, Project> = {};

  // root store
  rootStore: RootStore;
  // service
  projectService;
  projectArchiveService;
  issueLabelService;
  issueService;
  stateService;

  constructor(_rootStore: RootStore) {
    makeObservable(this, {
      // observables
      isUpdatingProject: observable,
      loader: observable.ref,
      fetchStatus: observable.ref,
      projectMap: observable,
      // computed
      isInitializingProjects: computed,
      filteredProjectIds: computed,
      workspaceProjectIds: computed,
      archivedProjectIds: computed,
      totalProjectIds: computed,
      currentProjectDetails: computed,
      joinedProjectIds: computed,
      // helper actions
      processProjectAfterCreation: action,
      // fetch actions
      fetchPartialProjects: action,
      fetchProjects: action,
      fetchProjectDetails: action,
      // project-view action
      updateProjectView: action,
      // CRUD actions
      createProject: action,
      updateProject: action,
    });
    // root store
    this.rootStore = _rootStore;
    // services
    this.projectService = new ProjectService();
    this.projectArchiveService = new ProjectArchiveService();
    this.issueService = new IssueService();
    this.issueLabelService = new IssueLabelService();
    this.stateService = new ProjectStateService();
  }

  /**
   * @description returns true if projects are still initializing
   */
  get isInitializingProjects() {
    return this.loader === "init-loader";
  }

  /**
   * @description returns filtered projects based on filters and search query
   */
  get filteredProjectIds() {
    const workspaceDetails = this.rootStore.workspaceRoot.currentWorkspace;
    const {
      currentWorkspaceDisplayFilters: displayFilters,
      currentWorkspaceFilters: filters,
      searchQuery,
    } = this.rootStore.projectRoot.projectFilter;
    if (!workspaceDetails || !displayFilters || !filters) return;
    let workspaceProjects = Object.values(this.projectMap).filter(
      (p) =>
        p.workspace_id === workspaceDetails.id &&
        (p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          p.identifier.toLowerCase().includes(searchQuery.toLowerCase())) &&
        shouldFilterProject(p, displayFilters, filters)
    );
    workspaceProjects = orderProjects(workspaceProjects, displayFilters.order_by);
    return workspaceProjects.map((p) => p.id);
  }

  /**
   * Returns project IDs belong to the current workspace
   */
  get workspaceProjectIds() {
    const workspaceDetails = this.rootStore.workspaceRoot.currentWorkspace;
    if (!workspaceDetails) return;
    const workspaceProjects = Object.values(this.projectMap).filter(
      (p) => p.workspace_id === workspaceDetails.id && !p.archived_at
    );
    const projectIds = workspaceProjects.map((p) => p.id);
    return projectIds ?? null;
  }

  /**
   * Returns archived project IDs belong to current workspace.
   */
  get archivedProjectIds() {
    const currentWorkspace = this.rootStore.workspaceRoot.currentWorkspace;
    if (!currentWorkspace) return;

    let projects = Object.values(this.projectMap ?? {});
    projects = sortBy(projects, "archived_at");

    const projectIds = projects
      .filter((project) => project.workspace_id === currentWorkspace.id && !!project.archived_at)
      .map((project) => project.id);
    return projectIds;
  }

  /**
   * Returns total project IDs belong to the current workspace
   */
  // workspaceProjectIds + archivedProjectIds
  get totalProjectIds() {
    const currentWorkspace = this.rootStore.workspaceRoot.currentWorkspace;
    if (!currentWorkspace) return;

    const workspaceProjects = this.workspaceProjectIds ?? [];
    const archivedProjects = this.archivedProjectIds ?? [];
    return [...workspaceProjects, ...archivedProjects];
  }

  /**
   * Returns current project details
   */
  get currentProjectDetails() {
    if (!this.rootStore.router.projectId) return;
    return this.projectMap?.[this.rootStore.router.projectId];
  }

  /**
   * Returns joined project IDs belong to the current workspace
   */
  get joinedProjectIds() {
    const currentWorkspace = this.rootStore.workspaceRoot.currentWorkspace;
    if (!currentWorkspace) return [];

    let projects = Object.values(this.projectMap ?? {});
    projects = sortBy(projects, "sort_order");

    const projectIds = projects
      .filter(
        (project) => project.workspace_id === currentWorkspace.id && !!project.member_role && !project.archived_at
      )
      .map((project) => project.id);
    return projectIds;
  }

  /**
   * @description process project after creation
   * @param workspaceSlug
   * @param data
   */
  processProjectAfterCreation = (workspaceSlug: string, data: Project) => {
    runInAction(() => {
      set(this.projectMap, [data.id], data);
      // updating the user project role in workspaceProjectsPermissions
      set(this.rootStore.user.permission.workspaceProjectsPermissions, [workspaceSlug, data.id], data.member_role);
    });
  };

  /**
   * get Workspace projects partial data using workspace slug
   * @param workspaceSlug
   * @returns Promise<Project[]>
   *
   */
  fetchPartialProjects = async (workspaceSlug: string) => {
    try {
      this.loader = "init-loader";
      const projectsResponse = await this.projectService.getProjectsLite(workspaceSlug);
      runInAction(() => {
        projectsResponse.forEach((project) => {
          update(this.projectMap, [project.id], (p) => ({ ...p, ...project }));
        });
        this.loader = "loaded";
        if (!this.fetchStatus) this.fetchStatus = "partial";
      });
      return projectsResponse;
    } catch (error) {
      console.log("Failed to fetch project from workspace store");
      this.loader = "loaded";
      throw error;
    }
  };

  /**
   * get Workspace projects using workspace slug
   * @param workspaceSlug
   * @returns Promise<Project[]>
   *
   */
  fetchProjects = async (workspaceSlug: string) => {
    try {
      if (this.workspaceProjectIds && this.workspaceProjectIds.length > 0) {
        this.loader = "mutation";
      } else {
        this.loader = "init-loader";
      }
      const projectsResponse = await this.projectService.getProjects(workspaceSlug);
      runInAction(() => {
        projectsResponse.forEach((project) => {
          update(this.projectMap, [project.id], (p) => ({ ...p, ...project }));
        });
        this.loader = "loaded";
        this.fetchStatus = "complete";
      });
      return projectsResponse;
    } catch (error) {
      console.log("Failed to fetch project from workspace store");
      this.loader = "loaded";
      throw error;
    }
  };

  /**
   * Fetches project details using workspace slug and project id
   * @param workspaceSlug
   * @param projectId
   * @returns Promise<Project>
   */
  fetchProjectDetails = async (workspaceSlug: string, projectId: string) => {
    try {
      const response = await this.projectService.getProject(workspaceSlug, projectId);
      runInAction(() => {
        update(this.projectMap, [projectId], (p) => ({ ...p, ...response }));
      });
      return response;
    } catch (error) {
      console.log("Error while fetching project details", error);
      throw error;
    }
  };

  /**
   * Returns project details using project id
   * @param projectId
   * @returns Project | null
   */
  getProjectById = computedFn((projectId: string | undefined | null) => {
    const projectInfo = this.projectMap[projectId ?? ""] || undefined;
    return projectInfo;
  });

  /**
   * Returns project details using project identifier
   * @param projectIdentifier
   * @returns Project | undefined
   */
  getProjectByIdentifier = computedFn((projectIdentifier: string) =>
    Object.values(this.projectMap).find((project) => project.identifier === projectIdentifier)
  );

  /**
   * Returns project identifier using project id
   * @param projectId
   * @returns string
   */
  getProjectIdentifierById = computedFn((projectId: string | undefined | null) => {
    const projectInfo = this.projectMap?.[projectId ?? ""];
    return projectInfo?.identifier;
  });

  /**
   * Updates the project view
   * @param workspaceSlug
   * @param projectId
   * @param viewProps
   * @returns
   */
  updateProjectView = async (workspaceSlug: string, projectId: string, viewProps: { sort_order: number }) => {
    const currentProjectSortOrder = this.getProjectById(projectId)?.sort_order;
    try {
      runInAction(() => {
        set(this.projectMap, [projectId, "sort_order"], viewProps?.sort_order);
      });
      const response = await this.projectService.updateProjectUserProperties(workspaceSlug, projectId, viewProps);
      return response;
    } catch (error) {
      runInAction(() => {
        set(this.projectMap, [projectId, "sort_order"], currentProjectSortOrder);
      });
      console.log("Failed to update sort order of the projects");
      throw error;
    }
  };

  /**
   * Creates a project in the workspace and adds it to the store
   * @param workspaceSlug
   * @param data
   * @returns Promise<Project>
   */
  createProject = async (workspaceSlug: string, data: any) => {
    try {
      const response = await this.projectService.createProject(workspaceSlug, data);
      this.processProjectAfterCreation(workspaceSlug, response);
      return response;
    } catch (error) {
      console.log("Failed to create project from project store");
      throw error;
    }
  };

  /**
   * Updates a details of a project and updates it in the store
   * @param workspaceSlug
   * @param projectId
   * @param data
   * @returns Promise<Project>
   */
  updateProject = async (workspaceSlug: string, projectId: string, data: Partial<Project>) => {
    const projectDetails = cloneDeep(this.getProjectById(projectId));
    try {
      runInAction(() => {
        set(this.projectMap, [projectId], { ...projectDetails, ...data });
        this.isUpdatingProject = true;
      });
      const response = await this.projectService.updateProject(workspaceSlug, projectId, data);
      runInAction(() => {
        this.isUpdatingProject = false;
      });
      return response;
    } catch (error) {
      console.log("Failed to create project from project store");
      runInAction(() => {
        set(this.projectMap, [projectId], projectDetails);
        this.isUpdatingProject = false;
      });
      throw error;
    }
  };

  /**
   * Deletes a project from specific workspace and deletes it from the store
   * @param workspaceSlug
   * @param projectId
   * @returns Promise<void>
   */
  deleteProject = async (workspaceSlug: string, projectId: string) => {
    try {
      if (!this.projectMap?.[projectId]) return;
      await this.projectService.deleteProject(workspaceSlug, projectId);
      runInAction(() => {
        delete this.projectMap[projectId];
        if (this.rootStore.favorite.entityMap[projectId]) this.rootStore.favorite.removeFavoriteFromStore(projectId);
        delete this.rootStore.user.permission.workspaceProjectsPermissions[workspaceSlug][projectId];
      });
    } catch (error) {
      console.log("Failed to delete project from project store");
      throw error;
    }
  };

  /**
   * Archives a project from specific workspace and updates it in the store
   * @param workspaceSlug
   * @param projectId
   * @returns Promise<void>
   */
  archiveProject = async (workspaceSlug: string, projectId: string) => {
    const response = await this.projectArchiveService.archiveProject(workspaceSlug, projectId);
    runInAction(() => {
      set(this.projectMap, [projectId, "archived_at"], response.archived_at);
      this.rootStore.favorite.removeFavoriteFromStore(projectId);
    });
  };

  /**
   * Restores a project from specific workspace and updates it in the store
   * @param workspaceSlug
   * @param projectId
   * @returns Promise<void>
   */
  restoreProject = async (workspaceSlug: string, projectId: string) => {
    await this.projectArchiveService.restoreProject(workspaceSlug, projectId);
    runInAction(() => {
      set(this.projectMap, [projectId, "archived_at"], null);
    });
  };
}
