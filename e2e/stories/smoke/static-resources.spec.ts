import { expect, test, type Page } from "@playwright/test";
import { createServer } from "node:http";
import { once } from "node:events";
import { watchStaticResources } from "../../fixtures/static-resources";

const origin = "http://127.0.0.1:40501";
const picture = `${origin}/probe.svg`;
const svg = '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="8"><rect width="12" height="8"/></svg>';

async function openDocument(page: Page) {
  // The transport is controlled; the browser's request events and image decoder are real.
  await page.route(`${origin}/`, (route) =>
    route.fulfill({ contentType: "text/html", body: "<html><body></body></html>" })
  );
  await page.goto(origin);
  return watchStaticResources(page);
}

async function loadImage(page: Page, targetURL = picture): Promise<boolean> {
  const sent = page.waitForRequest(targetURL);
  await page.evaluate((url) => {
    const image = new Image();
    image.src = url;
    document.body.replaceChildren(image);
  }, targetURL);
  // WebKit need not dispatch an image error event for an explicitly aborted transport.
  const response = await (await sent).response();
  if (!response || response.status() !== 200 || (await response.finished()) !== null) return false;
  return page.locator("img").evaluate((image: HTMLImageElement) =>
    image.decode().then(
      () => true,
      () => false
    )
  );
}

test("static resource guard accepts a Firefox image restart only after a decoded replacement", async ({
  page,
  browserName,
}) => {
  if (browserName === "webkit") {
    // This engine does not expose replacing a pending src as a failed request. Its explicit
    // transport cancellation must remain an error even if a later image succeeds.
    const watch = await openDocument(page);
    await page.route(picture, (route) => route.abort("aborted"));
    expect(await loadImage(page)).toBe(false);
    const original = await watch.failures();
    expect(original).toHaveLength(1);
    await page.unroute(picture);
    await page.route(picture, (route) => route.fulfill({ contentType: "image/svg+xml", body: svg }));
    expect(await loadImage(page)).toBe(true);
    expect(await watch.failures()).toEqual(original);
    return;
  }
  let received!: () => void;
  const pending = new Promise<void>((resolve) => {
    received = resolve;
  });
  let images = 0;
  const server = createServer((request, response) => {
    if (request.url === "/probe.svg") {
      if (++images === 1) {
        received();
        return;
      }
      response.writeHead(200, { "Content-Type": "image/svg+xml", "Cache-Control": "no-store" });
      response.end(svg);
    } else {
      response.writeHead(200, { "Content-Type": "text/html" });
      response.end("<html><body></body></html>");
    }
  }).listen(0, "127.0.0.1");
  try {
    await once(server, "listening");
    const address = server.address();
    if (!address || typeof address === "string") throw new Error("No probe address");
    const url = `http://127.0.0.1:${address.port}/probe.svg`;
    await page.goto(`http://127.0.0.1:${address.port}`);
    const watch = watchStaticResources(page);
    await page.evaluate((src) => {
      const img = new Image();
      img.src = src;
      document.body.append(img);
    }, url);
    await pending;
    const ended = page.waitForEvent("requestfailed", (request) => request.url() === url);
    await page.locator("img").evaluate((img: HTMLImageElement) => {
      img.src = "data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg'/>";
    });
    const failed = await ended;
    if (browserName === "firefox") expect(failed.failure()?.errorText).toBe("NS_BINDING_ABORTED");
    const original = await watch.failures();
    expect(original).toHaveLength(1);
    const completed = page.waitForEvent("requestfinished", (request) => request.url() === url);
    expect(await loadImage(page, url)).toBe(true);
    await completed;
    expect(await watch.failures()).toEqual(browserName === "firefox" ? [] : original);
  } finally {
    server.closeAllConnections();
    await new Promise<void>((resolve) => server.close(() => resolve()));
  }
});

test("static resource guard retains an aborted image with no replacement", async ({ page }) => {
  const watch = await openDocument(page);
  await page.route(picture, (route) => route.abort("aborted"));
  expect(await loadImage(page)).toBe(false);
  expect(await watch.failures()).toHaveLength(1);
});

test("static resource guard retains HTTP failures even after a successful image", async ({ page }) => {
  const watch = await openDocument(page);
  await page.route(picture, (route) => route.fulfill({ status: 503, body: "unavailable" }));
  expect(await loadImage(page)).toBe(false);
  await page.unroute(picture);
  await page.route(picture, (route) => route.fulfill({ contentType: "image/svg+xml", body: svg }));
  expect(await loadImage(page)).toBe(true);
  expect(await watch.failures()).toContain(`503 ${picture}`);
});

test("static resource guard retains an abort when the replacement cannot decode", async ({ page }) => {
  const watch = await openDocument(page);
  await page.route(picture, (route) => route.abort("aborted"));
  expect(await loadImage(page)).toBe(false);
  await page.unroute(picture);
  await page.route(picture, (route) => route.fulfill({ contentType: "image/svg+xml", body: "not an image" }));
  expect(await loadImage(page)).toBe(false);
  expect(await watch.failures()).toHaveLength(1);
});

test("static resource guard retains a cancelled script even when its URL later decodes as an image", async ({
  page,
}) => {
  const watch = await openDocument(page);
  await page.route(picture, (route) => route.abort("aborted"));
  await page.evaluate(
    (url) =>
      new Promise<void>((resolve) => {
        const script = document.createElement("script");
        script.addEventListener("error", () => resolve(), { once: true });
        script.src = url;
        document.body.append(script);
      }),
    picture
  );
  await page.unroute(picture);
  await page.route(picture, (route) => route.fulfill({ contentType: "image/svg+xml", body: svg }));
  expect(await loadImage(page)).toBe(true);
  expect(await watch.failures()).toHaveLength(1);
});
