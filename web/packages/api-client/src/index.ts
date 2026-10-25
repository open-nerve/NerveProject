import createFetchClient, { createQuerySerializer, type ClientOptions } from "openapi-fetch";

import type { paths } from "./schema.gen";

// The schemas' own names (User, Profile, Problem …): openapi-typescript's --root-types (M2 design 3.12).
export type * from "./schema.gen";
export type { Middleware } from "openapi-fetch";

/**
 * Creates a client for the Nerve API. Paths, parameters, request bodies and
 * responses are typed from api/dist/openapi.yaml; error bodies are
 * problem+json, Problem.
 */
export function createClient(options?: ClientOptions) {
  const client = createFetchClient<paths>(options);
  if (options?.querySerializer !== undefined) return client;
  const get = client.GET;
  client.GET = (path, ...args) => {
    const init = args[0];
    if (init !== undefined) args[0] = listQueryOptions(path, init);
    return get(path, ...args);
  };
  const request = client.request;
  client.request = (method, path, ...args) => {
    const init = args[0];
    if (method === "get" && init !== undefined) args[0] = listQueryOptions(path, init);
    return request(method, path, ...args);
  };
  return client;
}

// These two operations mix CSV alternatives with repeated date conditions.
// OpenAPI types do not configure openapi-fetch's runtime serialization.
function listQueryOptions<Init extends { querySerializer?: unknown }>(path: string, init: Init): Init {
  if (
    (path !== "/api/v0/projects/{project_id}/issues" &&
      path !== "/api/v0/workspaces/{slug}/members/{user_id}/issues") ||
    init?.querySerializer !== undefined
  ) {
    return init;
  }
  return Object.assign({}, init, { querySerializer: listQueryString });
}

function listQueryString(query: Record<string, unknown>): string {
  const csvKeys = new Set([
    "state_id",
    "state_group",
    "priority",
    "label_id",
    "assignee_id",
    "created_by_id",
    "mention_id",
    "subscriber_id",
    "project_id",
  ]);
  const normal = createQuerySerializer();
  const csv = createQuerySerializer({ array: { style: "form", explode: false } });
  return Object.entries(query)
    .map(([key, value]) => (csvKeys.has(key) ? csv : normal)({ [key]: value }))
    .filter(Boolean)
    .join("&");
}

/** A client of the Nerve API, as createClient makes it. */
export type ApiClient = ReturnType<typeof createClient>;
