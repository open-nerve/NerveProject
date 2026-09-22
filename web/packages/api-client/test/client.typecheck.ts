// Compile-time checks, run by `pnpm check:types`: tsc fails when the generated
// types stop matching how clients call the API. Nothing here is executed.
import { createClient, type components } from "../src/index";

type InstanceInfo = components["schemas"]["InstanceInfo"];

export async function describeInstance(baseUrl: string): Promise<string> {
  const client = createClient({ baseUrl });
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
}
