// Compile-time checks, run by `pnpm check:types`: tsc fails when the generated
// types stop matching how clients call the API. Nothing here is executed.
import { createClient, type components } from "../src/index";

type InstanceInfo = components["schemas"]["InstanceInfo"];

export async function describeInstance(baseUrl: string): Promise<string> {
  const client = createClient({ baseUrl });
  // Both public GET entry points retain the no-options call shape.
  await client.request("get", "/api/v0/instance");
  const { data, error } = await client.GET("/api/v0/instance");
  if (error) {
    // Error bodies are problem+json; clients branch on code.
    const code: string = error.code;
    throw new Error(`${error.status} ${code}`);
  }
  const info: InstanceInfo = data;
  const apiVersion: "v0" = info.api_version;
  return `${info.product} ${info.version} (${info.commit}), API ${apiVersion}`;
}

export async function rejectsUndocumentedCalls(baseUrl: string): Promise<void> {
  const client = createClient({ baseUrl });
  // @ts-expect-error: /api/v0/nope is not in the contract
  await client.GET("/api/v0/nope");
  // @ts-expect-error: /api/v0/instance has no POST
  await client.POST("/api/v0/instance");
  await client.GET("/api/v0/projects/{project_id}/issues", {
    params: {
      path: { project_id: "project" },
      query: { priority: ["high", "low"], start_date: ["2026-10-01..", "..2026-10-31"] },
    },
  });
  await client.GET("/api/v0/workspaces/{slug}/members/{user_id}/issues", {
    // @ts-expect-error: member list still requires relation after wrapping the client
    params: { path: { slug: "acme", user_id: "member" } },
  });
  // @ts-expect-error: generic request still refuses undocumented routes
  await client.request("get", "/api/v0/nope");
}
