import { expect, test } from "../../fixtures/test";

test("S3: a caller reads the instance information with the typed client", async ({ api }) => {
  const version = process.env.NERVE_VERSION;
  expect(version, "NERVE_VERSION, the version make build stamped into bin/nerve (make e2e sets it)").toBeTruthy();

  const { data, error, response } = await api.GET("/api/v0/instance");

  expect(response.status).toBe(200);
  expect(error).toBeUndefined();
  expect(data).toEqual({
    product: "Nerve",
    version,
    commit: expect.stringMatching(/^[0-9a-f]{40}$/),
    api_version: "v0",
  });
});
