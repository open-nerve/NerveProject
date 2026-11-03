import type { TestInfo } from "@playwright/test";

import type { Api } from "./api";
import type { Database } from "./db";
import { s3Test } from "./s3-world";
import { test } from "./test";

interface FileStoryWorld {
  api: Api;
  db: Database;
  uploadStatus: 200 | 204;
}

/** Register identical business assertions on both existing, independently cleaned-up fixtures. */
export function fileStory(name: string, run: (world: FileStoryWorld, info: TestInfo) => Promise<void>) {
  test(`${name} [local]`, async ({ api, db }, info) => {
    await run({ api, db, uploadStatus: 204 }, info);
  });
  s3Test(`${name} [s3]`, async ({ s3World: { api, db } }, info) => {
    await run({ api, db, uploadStatus: 200 }, info);
  });
}
