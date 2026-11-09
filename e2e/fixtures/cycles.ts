import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import { answer, type Api } from "./api";
import { bearer } from "./auth";

export type Cycle = components["schemas"]["Cycle"];
export type CycleCreate = components["schemas"]["CycleCreate"];
export type CycleUpdate = components["schemas"]["CycleUpdate"];
type CyclePreferences = components["schemas"]["CyclePreferences"];
type CyclePreferencesUpdate = components["schemas"]["CyclePreferencesUpdate"];

/** Creates a cycle in the project of projectId through the planning API, as the caller of token. */
export async function createCycle(api: Api, token: string, projectId: string, body: CycleCreate): Promise<Cycle> {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/cycles", {
    params: { path: { project_id: projectId } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `create the cycle ${body.name} in ${projectId}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createCycle ${body.name} answered 201 without the cycle`);
  }
  return data;
}

/** Changes the display settings of the caller of token on the page of the cycle of id, and returns them as changed. */
export async function changeCyclePreferences(
  api: Api,
  token: string,
  id: string,
  body: CyclePreferencesUpdate
): Promise<CyclePreferences> {
  const { data, error, response } = await api.PATCH("/api/v0/me/cycles/{cycle_id}/preferences", {
    params: { path: { cycle_id: id } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `change the display settings on the cycle ${id}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`changeCyclePreferences ${id} answered 200 without the settings`);
  }
  return data;
}

/**
 * The cycles of the project of projectId that the caller of token lists: the project's list, by sort order, or its
 * archived catalog, the last archived first.
 */
export async function listCycles(api: Api, token: string, projectId: string, archived = false): Promise<Cycle[]> {
  const { data, error, response } = await api.GET("/api/v0/projects/{project_id}/cycles", {
    params: { path: { project_id: projectId }, query: { archived } },
    headers: bearer(token),
  });
  expect(response.status, `list ${projectId}'s cycles (archived=${archived}): ${JSON.stringify(error)}`).toBe(200);
  return data?.data ?? [];
}

/** The cycle of id as the caller of token reads it. */
export async function readCycle(api: Api, token: string, id: string): Promise<Cycle> {
  const { data, error, response } = await api.GET("/api/v0/cycles/{cycle_id}", {
    params: { path: { cycle_id: id } },
    headers: bearer(token),
  });
  expect(response.status, `read the cycle ${id}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`readCycle ${id} answered 200 without the cycle`);
  }
  return data;
}

/**
 * The calls on cycles whose answer a story compares, each by the caller of token: its status, or the problem's code
 * and fields. The reads answer as the writes do, so a refused read is one more answer.
 */
export function cycleCalls(api: Api) {
  return {
    create: async (token: string, projectId: string, body: CycleCreate) => {
      const { error, response } = await api.POST("/api/v0/projects/{project_id}/cycles", {
        params: { path: { project_id: projectId } },
        body,
        headers: bearer(token),
      });
      return answer(response, error);
    },
    list: async (token: string, projectId: string) => {
      const { error, response } = await api.GET("/api/v0/projects/{project_id}/cycles", {
        params: { path: { project_id: projectId } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    read: async (token: string, id: string) => {
      const { error, response } = await api.GET("/api/v0/cycles/{cycle_id}", {
        params: { path: { cycle_id: id } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    update: async (token: string, id: string, body: CycleUpdate) => {
      const { error, response } = await api.PATCH("/api/v0/cycles/{cycle_id}", {
        params: { path: { cycle_id: id } },
        body,
        headers: bearer(token),
      });
      return answer(response, error);
    },
    archive: async (token: string, id: string) => {
      const { error, response } = await api.POST("/api/v0/cycles/{cycle_id}/archive", {
        params: { path: { cycle_id: id } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    unarchive: async (token: string, id: string) => {
      const { error, response } = await api.POST("/api/v0/cycles/{cycle_id}/unarchive", {
        params: { path: { cycle_id: id } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    remove: async (token: string, id: string) => {
      const { error, response } = await api.DELETE("/api/v0/cycles/{cycle_id}", {
        params: { path: { cycle_id: id } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
  };
}
