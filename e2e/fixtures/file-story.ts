import type { Page, TestInfo } from "@playwright/test";

import type { Api } from "./api";
import type { Database } from "./db";
import { s3Test } from "./s3-world";
import { test } from "./test";
import { test as pageTest, type ImageWire } from "./profile-image-wire";
import type { Nerve } from "./server";
import type { S3Bucket } from "./s3";

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

export interface PageFileStoryWorld extends FileStoryWorld {
  page: Page;
  nerve: Nerve;
  baseURL: string;
  bucket?: S3Bucket;
  wire?: ImageWire;
}

/** The actual application page, unchanged upload grants and owned storage from the existing fixtures. */
export function pageFileStory(name: string, run: (world: PageFileStoryWorld, info: TestInfo) => Promise<void>) {
  pageTest(`${name} [local]`, async ({ imageWorld }, info) => {
    await run(imageWorld, info);
  });
  s3Test(`${name} [s3]`, async ({ s3World, page }, info) => {
    await run({ ...s3World, page, baseURL: s3World.nerve.baseURL, uploadStatus: 200 }, info);
  });
}
