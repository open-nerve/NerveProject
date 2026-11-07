import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import type { Api } from "./api";
import { bearer } from "./auth";

type Module = components["schemas"]["Module"];
type ModuleCreate = components["schemas"]["ModuleCreate"];
type ModuleLink = components["schemas"]["ModuleLink"];
type ModuleLinkCreate = components["schemas"]["ModuleLinkCreate"];

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
