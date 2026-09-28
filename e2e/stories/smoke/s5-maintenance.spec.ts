import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { expect, test } from "../../fixtures/test";

/** The page's first request, for the instance's settings, which it needs before anything else (InstanceWrapper). */
const instance = "**/api/v0/instance";

test("S5: while nerve does not answer the instance's settings, the page says so; it carries on by itself once nerve answers", async ({
  page,
}) => {
  const watch = await watchPage(page);
  // SWR tries a failed request again 5 or 10 s after it failed (its first backoff): the story moves the page's
  // clock on instead of waiting.
  await page.clock.install();
  await page.route(instance, (route) =>
    route.fulfill({
      status: 503,
      contentType: "application/problem+json",
      body: JSON.stringify({ type: "about:blank", title: "Service Unavailable", status: 503, code: "server_busy" }),
    })
  );

  await page.goto("/");
  const title = page.getByRole("heading", { name: "Nerve cannot be reached right now" });
  await expect(title).toBeVisible();
  await expect(
    page.getByText(
      "The page could not load this server's settings. It tries again by itself and carries on once the server answers."
    )
  ).toBeVisible();
  await page.evaluate(() => {
    (window as unknown as { beforeRetry?: true }).beforeRetry = true;
  });

  // nerve answers again: the page's own retry reaches it, and the sign-in page shows, without a reload.
  await page.unroute(instance);
  const answered = page.waitForResponse((response) => new URL(response.url()).pathname === "/api/v0/instance", {
    timeout: 10_000,
  });
  await page.clock.fastForward(10_000);
  expect((await answered).status()).toBe(200);
  await expect(page.getByRole("button", { name: "Go to workspace" })).toBeVisible();
  await expect(title).toHaveCount(0);
  expect(await page.evaluate(() => (window as unknown as { beforeRetry?: true }).beforeRetry)).toBe(true);

  expect(watch.apiRequests).toEqual(["GET /api/v0/instance", "GET /api/v0/instance"]);
  expect(watch.apiFailures).toEqual(["503 GET /api/v0/instance"]);
  expect(watch.cspViolations).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(page, watch, {
    errors: ["Failed to load resource: the server responded with a status of 503 (Service Unavailable)"],
  });
});
