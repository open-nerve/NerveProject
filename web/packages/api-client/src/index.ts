import createFetchClient, { type ClientOptions } from "openapi-fetch";

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
  return createFetchClient<paths>(options);
}

/** A client of the Nerve API, as createClient makes it. */
export type ApiClient = ReturnType<typeof createClient>;
