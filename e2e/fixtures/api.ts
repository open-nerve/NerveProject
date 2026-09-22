import { createClient } from "@nerve/api-client";

/** The typed Nerve API client, generated from api/dist/openapi.yaml. */
export type Api = ReturnType<typeof createClient>;

/** Returns a client for the nerve at baseURL. */
export function createApi(baseURL: string): Api {
  return createClient({ baseUrl: baseURL });
}
