import type { operations } from "@nerve/api-client";
import { expect } from "@playwright/test";

import type { Api } from "./api";
import { bearer } from "./auth";

type CycleCreate = operations["createCycle"]["requestBody"]["content"]["application/json"];
type Cycle = operations["createCycle"]["responses"][201]["content"]["application/json"];

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
