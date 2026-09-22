import { expect, test } from "../../fixtures/test";

// The typed client cannot express a path the API does not have, so the
// request goes out directly.
test("S4: a caller requests an API path that does not exist", async ({ request }) => {
  const res = await request.get("/api/v0/nope");

  expect(res.status()).toBe(404);
  expect(res.headers()["content-type"]).toBe("application/problem+json");
  expect(await res.json()).toEqual({
    status: 404,
    code: "not_found",
    title: "Not Found",
    detail: "no API endpoint for GET /api/v0/nope",
  });
});
