import type { Page, Request } from "@playwright/test";

function isStatic(request: Request): boolean {
  return !new URL(request.url()).pathname.startsWith("/api/");
}

/** Tracks static resources separately from the API, whose stricter watcher is unchanged. */
export function watchStaticResources(page: Page) {
  const loaded: string[] = [];
  const failed: { request: Request; message: string; order: number }[] = [];
  const finished = new Map<Request, number>();
  const statuses = new WeakMap<Request, number>();
  let order = 0;
  page.on("response", (response) => {
    const request = response.request();
    if (!isStatic(request)) return;
    statuses.set(request, response.status());
    const message = `${response.status()} ${response.url()}`;
    if (response.status() < 400) loaded.push(message);
    else failed.push({ request, message, order: ++order });
  });
  page.on("requestfailed", (request) => {
    if (isStatic(request))
      failed.push({ request, message: `${request.failure()?.errorText} ${request.url()}`, order: ++order });
  });
  page.on("requestfinished", (request) => {
    if (isStatic(request)) finished.set(request, ++order);
  });
  return {
    loaded,
    async failures(): Promise<string[]> {
      const decoded = await page
        .locator("img")
        .evaluateAll((images: HTMLImageElement[]) =>
          images.filter((image) => image.complete && image.naturalWidth > 0).map((image) => image.currentSrc)
        );
      // React assigns img.src during initial properties and again at commit. Firefox may restart the
      // pending image request. Accept only that precise cancellation, a later fully received 200 for
      // the same URL, and the actual current image decoded from it. HTTP/API/script failures stay errors.
      return failed
        .filter(({ request, order: failedAt }) => {
          if (
            statuses.has(request) ||
            request.resourceType() !== "image" ||
            request.failure()?.errorText !== "NS_BINDING_ABORTED"
          )
            return true;
          return (
            !decoded.includes(request.url()) ||
            ![...finished].some(
              ([replacement, ended]) =>
                ended > failedAt &&
                replacement !== request &&
                replacement.resourceType() === "image" &&
                replacement.url() === request.url() &&
                statuses.get(replacement) === 200
            )
          );
        })
        .map(({ message }) => message);
    },
  };
}
