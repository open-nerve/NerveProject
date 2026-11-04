import type { Page, Route } from "@playwright/test";
import { expect } from "@playwright/test";
import { deferred } from "./deferred";
import { holdAnswer } from "./settings-pages";
import { filePixel } from "./storage";

/** Pause one real boundary. PUT waits before transmission, so the browser still owns its credentials and bytes. */
export async function holdImageBoundary(page: Page, boundary: "prepare" | "put" | "complete") {
  const released = deferred();
  const finished = deferred();
  let reached = false;
  if (boundary === "put") {
    await page.route(/\/static\/local\//, async (route: Route) => {
      if (route.request().method() !== "PUT" || reached) return route.fallback();
      reached = true;
      try {
        await released.promise;
        await route.fallback();
      } finally {
        finished.resolve();
      }
    });
    return {
      wait: () => expect.poll(() => reached, { message: "the browser's raw PUT reached the gate" }).toBe(true),
      release: async () => {
        released.resolve();
        if (reached) await finished.promise;
      },
    };
  }
  const release = await holdAnswer(
    page,
    "POST",
    boundary === "prepare" ? "/api/v0/assets" : (pathname) => /^\/api\/v0\/assets\/[^/]+\/complete$/.test(pathname),
    1,
    () => {
      reached = true;
    }
  );
  return {
    wait: () => expect.poll(() => reached, { message: `nerve completed the held ${boundary}` }).toBe(true),
    release,
  };
}

/** A native File/DataTransfer drop, not a synthetic filename or mocked upload session. */
export async function dropImage(page: Page, bytes = filePixel, count = 1) {
  const data = await page.evaluateHandle(
    ({ values, count: length }) => {
      const transfer = new DataTransfer();
      for (let i = 0; i < length; i++)
        transfer.items.add(new File([new Uint8Array(values)], "same-name.png", { type: "image/png" }));
      return transfer;
    },
    { values: [...bytes], count }
  );
  try {
    await page
      .getByLabel("Drag and drop one image here", { exact: true })
      .dispatchEvent("drop", { dataTransfer: data });
  } finally {
    await data.dispose();
  }
}
