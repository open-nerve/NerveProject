import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import { answer, type Api } from "./api";
import { bearer } from "./auth";
import type { Database } from "./db";

type Module = components["schemas"]["Module"];
type ModuleCreate = components["schemas"]["ModuleCreate"];
type ModuleUpdate = components["schemas"]["ModuleUpdate"];
type ModuleLink = components["schemas"]["ModuleLink"];
type ModuleLinkCreate = components["schemas"]["ModuleLinkCreate"];
type ModuleLinkPatch = components["schemas"]["ModuleLinkPatch"];
type ModulePreferences = components["schemas"]["ModulePreferences"];
type ModulePreferencesUpdate = components["schemas"]["ModulePreferencesUpdate"];

/** Creates a module in the project of projectId through the planning API, as the caller of token. */
export async function createModule(api: Api, token: string, projectId: string, body: ModuleCreate): Promise<Module> {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/modules", {
    params: { path: { project_id: projectId } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `create the module ${body.name} in ${projectId}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createModule ${body.name} answered 201 without the module`);
  }
  return data;
}

/** Changes the display settings of the caller of token on the page of the module of id, and returns them as changed. */
export async function changeModulePreferences(
  api: Api,
  token: string,
  id: string,
  body: ModulePreferencesUpdate
): Promise<ModulePreferences> {
  const { data, error, response } = await api.PATCH("/api/v0/me/modules/{module_id}/preferences", {
    params: { path: { module_id: id } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `change the display settings on the module ${id}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`changeModulePreferences ${id} answered 200 without the settings`);
  }
  return data;
}

/**
 * The modules of the project of projectId that the caller of token lists: the project's list, by sort order, or its
 * archived catalog, the last archived first.
 */
export async function listModules(api: Api, token: string, projectId: string, archived = false): Promise<Module[]> {
  const { data, error, response } = await api.GET("/api/v0/projects/{project_id}/modules", {
    params: { path: { project_id: projectId }, query: { archived } },
    headers: bearer(token),
  });
  expect(response.status, `list ${projectId}'s modules (archived=${archived}): ${JSON.stringify(error)}`).toBe(200);
  return data?.data ?? [];
}

/** The module of id as the caller of token reads it. */
export async function readModule(api: Api, token: string, id: string): Promise<Module> {
  const { data, error, response } = await api.GET("/api/v0/modules/{module_id}", {
    params: { path: { module_id: id } },
    headers: bearer(token),
  });
  expect(response.status, `read the module ${id}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`readModule ${id} answered 200 without the module`);
  }
  return data;
}

/** Adds a link to the module of moduleId through the planning API, as the caller of token. */
export async function createModuleLink(
  api: Api,
  token: string,
  moduleId: string,
  body: ModuleLinkCreate
): Promise<ModuleLink> {
  const { data, error, response } = await api.POST("/api/v0/modules/{module_id}/links", {
    params: { path: { module_id: moduleId } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `add the link ${body.url} to ${moduleId}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createModuleLink ${body.url} answered 201 without the link`);
  }
  return data;
}

/** Changes the module link of id through the planning API, as the caller of token, and returns it as changed. */
export async function updateModuleLink(
  api: Api,
  token: string,
  id: string,
  body: ModuleLinkPatch
): Promise<ModuleLink> {
  const { data, error, response } = await api.PATCH("/api/v0/module-links/{module_link_id}", {
    params: { path: { module_link_id: id } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `change the module link ${id}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`updateModuleLink ${id} answered 200 without the link`);
  }
  return data;
}

/** The links of the module of moduleId as the caller of token lists them, the last added first. */
export async function listModuleLinks(api: Api, token: string, moduleId: string): Promise<ModuleLink[]> {
  const { data, error, response } = await api.GET("/api/v0/modules/{module_id}/links", {
    params: { path: { module_id: moduleId } },
    headers: bearer(token),
  });
  expect(response.status, `list the links of ${moduleId}: ${JSON.stringify(error)}`).toBe(200);
  return data?.data ?? [];
}

/**
 * The calls on modules and their links whose answer a story compares, each by the caller of token: its status, or the
 * problem's code and fields. The reads answer as the writes do, so a refused read is one more answer.
 */
export function moduleCalls(api: Api) {
  return {
    create: async (token: string, projectId: string, body: ModuleCreate) => {
      const { error, response } = await api.POST("/api/v0/projects/{project_id}/modules", {
        params: { path: { project_id: projectId } },
        body,
        headers: bearer(token),
      });
      return answer(response, error);
    },
    read: async (token: string, id: string) => {
      const { error, response } = await api.GET("/api/v0/modules/{module_id}", {
        params: { path: { module_id: id } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    update: async (token: string, id: string, body: ModuleUpdate) => {
      const { error, response } = await api.PATCH("/api/v0/modules/{module_id}", {
        params: { path: { module_id: id } },
        body,
        headers: bearer(token),
      });
      return answer(response, error);
    },
    archive: async (token: string, id: string) => {
      const { error, response } = await api.POST("/api/v0/modules/{module_id}/archive", {
        params: { path: { module_id: id } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    unarchive: async (token: string, id: string) => {
      const { error, response } = await api.POST("/api/v0/modules/{module_id}/unarchive", {
        params: { path: { module_id: id } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    remove: async (token: string, id: string) => {
      const { error, response } = await api.DELETE("/api/v0/modules/{module_id}", {
        params: { path: { module_id: id } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    links: async (token: string, moduleId: string) => {
      const { error, response } = await api.GET("/api/v0/modules/{module_id}/links", {
        params: { path: { module_id: moduleId } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    addLink: async (token: string, moduleId: string, body: ModuleLinkCreate) => {
      const { error, response } = await api.POST("/api/v0/modules/{module_id}/links", {
        params: { path: { module_id: moduleId } },
        body,
        headers: bearer(token),
      });
      return answer(response, error);
    },
    changeLink: async (token: string, id: string, body: ModuleLinkPatch) => {
      const { error, response } = await api.PATCH("/api/v0/module-links/{module_link_id}", {
        params: { path: { module_link_id: id } },
        body,
        headers: bearer(token),
      });
      return answer(response, error);
    },
    removeLink: async (token: string, id: string) => {
      const { error, response } = await api.DELETE("/api/v0/module-links/{module_link_id}", {
        params: { path: { module_link_id: id } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
  };
}

/** The crawls of the module link of linkId, its planning.crawl_link jobs, the first first: each one's address and state. */
export function moduleLinkCrawls(db: Database, linkId: string) {
  return db.query<{ url: string; state: string }>(
    `SELECT args->>'url' AS url, state::text AS state FROM river_job
      WHERE kind = 'planning.crawl_link' AND args->>'link_id' = $1 ORDER BY id`,
    [linkId]
  );
}

/**
 * Waits, a bounded while, until the module link of linkId has had exactly a crawl of each of urls, in order, and each
 * has completed: its address blocked, as a link-local one is, the crawl is done.
 */
export async function expectModuleLinkCrawled(db: Database, linkId: string, urls: string[]): Promise<void> {
  await expect
    .poll(() => moduleLinkCrawls(db, linkId), {
      timeout: 10_000,
      intervals: [20, 50, 100],
      message: `the crawls of the module link ${linkId}`,
    })
    .toEqual(urls.map((url) => ({ url, state: "completed" })));
}
