import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import type { Api } from "./api";
import { bearer } from "./auth";

type RecentVisitCreate = components["schemas"]["RecentVisitCreate"];

/**
 * Records the caller of token's visit of a work item or a project through the navigation API (204, new or revisited):
 * an admin, a member or a guest of the target's workspace who sees the target; its workspace and project are the
 * target's (M7 P3d spec 2.4, 2.5).
 */
export async function recordVisit(api: Api, token: string, body: RecentVisitCreate): Promise<void> {
  const { error, response } = await api.POST("/api/v0/me/recent-visits", { body, headers: bearer(token) });
  expect(
    response.status,
    `record the visit of the ${body.entity_type} ${body.entity_id}: ${JSON.stringify(error)}`
  ).toBe(204);
}
