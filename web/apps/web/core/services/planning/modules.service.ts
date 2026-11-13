/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type {
  ApiClient,
  Module,
  ModuleCreate,
  ModuleLink,
  ModuleLinkCreate,
  ModuleLinkPatch,
  ModuleListItem,
  ModuleUpdate,
  Progress,
} from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/**
 * A project's modules (M6 design 3.2, 3.5–3.8): work items planned together, with a status, dates, a lead, members
 * and links; never frozen, archived once completed or cancelled.
 */
export class ModulesService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The project's modules that are not archived, in nerve's order (sort order, then creation, then id), with counts. */
  async list(projectId: string): Promise<ModuleListItem[]> {
    return unwrap(
      await this.api.GET("/api/v0/projects/{project_id}/modules", { params: { path: { project_id: projectId } } })
    ).data;
  }

  /** The project's archived modules, the latest archived first, with counts. */
  async listArchived(projectId: string): Promise<ModuleListItem[]> {
    return unwrap(
      await this.api.GET("/api/v0/projects/{project_id}/modules", {
        params: { path: { project_id: projectId }, query: { archived: true } },
      })
    ).data;
  }

  /** The module as nerve holds it, an archived one's too, and while the project's modules are turned off. */
  async get(moduleId: string): Promise<Module> {
    return unwrap(await this.api.GET("/api/v0/modules/{module_id}", { params: { path: { module_id: moduleId } } }));
  }

  /** Creates a module, first of its project's. */
  async create(projectId: string, body: ModuleCreate): Promise<Module> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/modules", {
        params: { path: { project_id: projectId } },
        body,
      })
    );
  }

  /** Changes the fields body names (member_ids, the whole set); the answer is the module as nerve holds it. */
  async update(moduleId: string, body: ModuleUpdate): Promise<Module> {
    return unwrap(
      await this.api.PATCH("/api/v0/modules/{module_id}", { params: { path: { module_id: moduleId } }, body })
    );
  }

  /** Deletes a module, and its links. */
  async remove(moduleId: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/modules/{module_id}", { params: { path: { module_id: moduleId } } }));
  }

  /** Archives a completed or cancelled module; one archived already is answered as it is. */
  async archive(moduleId: string): Promise<Module> {
    return unwrap(
      await this.api.POST("/api/v0/modules/{module_id}/archive", { params: { path: { module_id: moduleId } } })
    );
  }

  /** Unarchives a module; one not archived is answered as it is. */
  async unarchive(moduleId: string): Promise<Module> {
    return unwrap(
      await this.api.POST("/api/v0/modules/{module_id}/unarchive", { params: { path: { module_id: moduleId } } })
    );
  }

  /** The module's live progress, of the work items the caller sees in it, and its burndown. */
  async progress(moduleId: string): Promise<Progress> {
    return unwrap(
      await this.api.GET("/api/v0/modules/{module_id}/progress", { params: { path: { module_id: moduleId } } })
    );
  }

  /** The module's links, the latest added first. */
  async links(moduleId: string): Promise<ModuleLink[]> {
    return unwrap(
      await this.api.GET("/api/v0/modules/{module_id}/links", { params: { path: { module_id: moduleId } } })
    ).data;
  }

  /** Adds a link to the module; nerve answers it with its preview metadata empty, which it reads later. */
  async addLink(moduleId: string, body: ModuleLinkCreate): Promise<ModuleLink> {
    return unwrap(
      await this.api.POST("/api/v0/modules/{module_id}/links", { params: { path: { module_id: moduleId } }, body })
    );
  }

  /** Changes the link's URL or title; a changed URL empties its preview metadata. */
  async changeLink(linkId: string, body: ModuleLinkPatch): Promise<ModuleLink> {
    return unwrap(
      await this.api.PATCH("/api/v0/module-links/{module_link_id}", {
        params: { path: { module_link_id: linkId } },
        body,
      })
    );
  }

  /** Removes the link from its module. */
  async removeLink(linkId: string): Promise<void> {
    unwrap(
      await this.api.DELETE("/api/v0/module-links/{module_link_id}", { params: { path: { module_link_id: linkId } } })
    );
  }
}
