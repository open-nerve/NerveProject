import { setTimeout as sleep } from "node:timers/promises";

import type { Page } from "@playwright/test";

import { deferred } from "./deferred";

/**
 * Runs act, and resolves with what act resolved with and whether page began a download while act ran or within a
 * second of its end: what page must not download, such as the secret of a token answered after its page was left,
 * stays undownloaded the whole time, where a look made at once could come before page has begun it.
 */
export async function downloadedWithin<T>(
  page: Page,
  act: () => Promise<T>
): Promise<{ result: T; downloaded: boolean }> {
  const begun = deferred<true>();
  const onDownload = () => begun.resolve(true);
  page.on("download", onDownload);
  try {
    const result = await act();
    return { result, downloaded: await Promise.race([begun.promise, sleep(1_000, false)]) };
  } finally {
    page.off("download", onDownload);
  }
}
