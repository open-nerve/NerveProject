import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import type { Api } from "./api";
import { bearer } from "./auth";

// A project's states as the stories read and make them (M3 design 3.17).

export type State = components["schemas"]["State"];
export type StateCreate = components["schemas"]["StateCreate"];
export type StateUpdate = components["schemas"]["StateUpdate"];

/** Creates a state in the project of projectId with the bearer token given, an admin's of the project, and returns it. */
export async function createState(api: Api, token: string, projectId: string, body: StateCreate): Promise<State> {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/states", {
    params: { path: { project_id: projectId } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `create the state ${body.name} in ${projectId}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createState ${body.name} answered 201 without the state`);
  }
  return data;
}

/** The states of the project of projectId that the caller of token lists, in their order. */
export async function listStates(api: Api, token: string, projectId: string): Promise<State[]> {
  const { data, error, response } = await api.GET("/api/v0/projects/{project_id}/states", {
    params: { path: { project_id: projectId } },
    headers: bearer(token),
  });
  expect(response.status, `list ${projectId}'s states: ${JSON.stringify(error)}`).toBe(200);
  return data?.data ?? [];
}

/** The ids of the states of the project of projectId, by name, as the caller of token lists them. */
export async function stateIds(api: Api, token: string, projectId: string): Promise<Record<string, string>> {
  return Object.fromEntries((await listStates(api, token, projectId)).map((s) => [s.name, s.id]));
}
