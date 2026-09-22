import createFetchClient, { type ClientOptions } from "openapi-fetch";

import type { paths } from "./schema.gen";

export type { components, paths } from "./schema.gen";

/**
 * Creates a client for the Nerve API. Paths, parameters, request bodies and
 * responses are typed from api/dist/openapi.yaml; error bodies are
 * problem+json, components["schemas"]["Problem"].
 */
export function createClient(options?: ClientOptions) {
  return createFetchClient<paths>(options);
}
