/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createHash } from "node:crypto";
import { readFile as readDiskFile } from "node:fs/promises";
import path from "node:path";

import { expect, type Locator, type Page, type Request, type Response, type TestInfo } from "@playwright/test";

import { intakeFileOf } from "./assert/intake";
import { bearer, signInContext, type AuthTokens } from "./auth";
import { watchPage, type PageWatch } from "./browser";
import { pasteInto, type ContentFile } from "./content-files-page";
import { deferred } from "./deferred";
import type { PageFileStoryWorld } from "./file-story";
import { expectHeaderOf, intakePath, intakeWorld, selecting, writesSeen, type ApiWrite } from "./intake-pages";
import type { Issue } from "./issues";
import { readFile } from "./storage";
import { anotherBrowser } from "./workspace-pages";

// The files of a work item of a project's intake, as its people use them on its page (M7 P5b spec 2.2, 2.3, 2.8,
// R12): intakeWorld on the nerve of a file story, [local]'s or [s3]'s (pageFileStory), each person's page, the
// description and the attachments of a work item there, an upload held on its way, and what nerve and its storage
// hold of a file. The intake page itself, its rows and its writes seen are intake-pages.ts's; the files themselves,
// their chooser, paste and download content-files-page.ts's.

/** A person's page in the story, watched, its writes (writesSeen) and its API requests followed from its first load. */
export interface PersonPage {
  readonly page: Page;
  readonly watch: PageWatch;
  readonly writes: ApiWrite[];
  /**
   * Resolves once page has had no API request in flight for half a second, and each image its descriptions show is
   * loaded: a confirmed write reads the work item's activity and comments again, and an image whose read the next step
   * cuts off fails its load (M6-M8-07). Before each move or change of selection, and at the end.
   */
  settled(): Promise<void>;
  close(): Promise<void>;
}

const sha256 = (bytes: Uint8Array) => createHash("sha256").update(bytes).digest("hex");
const apiPath = (request: Request) => new URL(request.url()).pathname.startsWith("/api/");

/**
 * The world of the intake's file stories: intakeWorld's on world's nerve, the browser's name, the page of each person
 * signed in (signIn), the intake page opened on one (open), and what nerve holds of a file (expectStored).
 */
export async function intakeFilesWorld(world: PageFileStoryWorld, info: TestInfo) {
  const people = await intakeWorld(world.api, info);
  const browser = world.page.context().browser()!;
  const browserName = browser.browserType().name();
  let opened = 0;
  /**
   * The page of the person of tokens, signed in at world's nerve, which has loaded nothing yet: the first one world's
   * own (its context the [local] wire's or the [s3] world's), each other one a browser of its own.
   */
  const signIn = async ({ tokens }: { tokens: AuthTokens }): Promise<PersonPage> => {
    const other = opened++ === 0 ? undefined : await anotherBrowser(browser, world.baseURL, tokens);
    if (!other) await signInContext(world.page.context(), world.baseURL, tokens);
    const page = other?.page ?? world.page;
    const watch = await watchPage(page);
    const writes = await writesSeen(page, browserName);
    let inFlight = 0;
    let last = Date.now();
    const touch = (request: Request, change: number) => {
      if (!apiPath(request)) return;
      inFlight += change;
      last = Date.now();
    };
    page.on("request", (request) => touch(request, 1));
    page.on("requestfinished", (request) => touch(request, -1));
    page.on("requestfailed", (request) => touch(request, -1));
    const settled = async () => {
      await expect
        .poll(() => inFlight === 0 && Date.now() - last >= 500, { timeout: 15_000, intervals: [100] })
        .toBe(true);
      await expect
        .poll(() =>
          page
            .locator("img.image-component")
            .evaluateAll((images) =>
              images.every((image) => image instanceof HTMLImageElement && image.complete && image.naturalWidth > 0)
            )
        )
        .toBe(true);
    };
    return { page, watch, writes, settled, close: other?.close ?? (async () => undefined) };
  };
  /**
   * Opens the intake page of Web on who's page, selecting the work item issue, and waits for its detail's header and
   * description, which loads the editor's code (a route set up after it then holds none of the code WebKit preloaded,
   * which it would fetch again and warn of), and for its reads (settled).
   */
  const open = async (who: PersonPage, issue: Issue) => {
    await who.page.goto(`${world.baseURL}${intakePath(people.slug, people.web.id)}?inboxIssueId=${issue.id}`);
    await expect(who.page).toHaveURL(selecting(issue));
    await expectHeaderOf(who.page, issue);
    await expect(descriptionOf(who.page, issue)).toBeVisible();
    await who.settled();
  };
  /**
   * Checks what nerve holds of the file of id, uploaded by the person of id by: its row (intakeFileOf: its bytes'
   * digest, uploaded and bound, and deleted by the person of id deletedBy, or not) and its object in the storage, the
   * local root's or the bucket's, of file's bytes. Live, the reader of token reader reads those bytes; deleted, nobody
   * reads it, while its object stays until the cleanup job removes it, which a page story cannot trigger (M5's
   * files/f15-cleanup.spec.ts covers it on a clock world).
   */
  const expectStored = async (
    id: string,
    file: ContentFile,
    { by, reader, deletedBy }: { by: string; reader: string; deletedBy?: string }
  ) => {
    const row = await intakeFileOf(world.db, id);
    expect(row).toMatchObject({
      sha: sha256(file.bytes),
      by,
      uploaded: true,
      bound: true,
      deleted: deletedBy !== undefined,
      deleted_by: deletedBy ?? null,
    });
    if (deletedBy === undefined) expect(await readFile(world.api, reader, id)).toEqual(file.bytes);
    else {
      const read = await world.api.GET("/api/v0/assets/{id}", { params: { path: { id } }, headers: bearer(reader) });
      expect(read.response.status, `${file.name} read once deleted`).toBe(404);
    }
    if (world.bucket)
      expect(await world.bucket.inspect(row.key)).toMatchObject({
        sha256: sha256(file.bytes),
        size: file.bytes.length,
        content_type: file.type,
      });
    else expect(await readDiskFile(path.join(world.nerve.storageRoot, "objects", row.key, "data"))).toEqual(file.bytes);
  };
  return { ...people, browserName, signIn, open, expectStored };
}

/** The description of the work item issue as page shows it, on its intake's page or its own: its editor's document. */
export function descriptionOf(page: Page, issue: Issue): Locator {
  return page.locator(`[id="editor-container-${issue.id}"] .ProseMirror`);
}

/**
 * Pastes the image file at the end of the editable description of the work item issue on page, and resolves with the
 * id nerve gave the file and nerve's answer to the description's save, a PATCH of savePath, that binds it: once the
 * transfer is confirmed, the description is saved with the image.
 */
export async function pasteImage(page: Page, issue: Issue, file: ContentFile, savePath: string) {
  const editor = descriptionOf(page, issue);
  await editor.focus();
  await page.keyboard.press("ControlOrMeta+End");
  const saved = page.waitForResponse(
    (response) => response.request().method() === "PATCH" && new URL(response.url()).pathname === savePath,
    { timeout: 20_000 }
  );
  const confirmed = confirmation(page);
  expect(await pasteInto(editor, {}, [file]), `the editor takes the paste of ${file.name}`).toBe(true);
  return { id: assetOf(await confirmed), save: await saved };
}

/** Resolves with nerve's answer to the next confirmation of a transfer (completeAsset) that page sends. */
export function confirmation(page: Page): Promise<Response> {
  return page.waitForResponse(
    (response) =>
      response.request().method() === "POST" &&
      /^\/api\/v0\/assets\/[^/]+\/complete$/.test(new URL(response.url()).pathname),
    { timeout: 20_000 }
  );
}

/** The id of the file of a confirmation's answer (confirmation's). */
export function assetOf(answer: Response): string {
  return new URL(answer.url()).pathname.split("/")[4]!;
}

/**
 * The attachments of the work item that region shows: their button, named by their count (count), which opens and
 * closes them, and the panel it opens, with its picker, its list and their buttons.
 */
export function attachmentsOf(region: Locator) {
  return {
    count: (count: number) => region.getByRole("button", { name: `Attachments ${count}`, exact: true }),
    panel: region.getByRole("button", { name: /^Attachments \d+$/ }).locator("xpath=../following-sibling::*[1]"),
  };
}

/**
 * Holds the bytes of each file that page sends to the storage from now on (an upload's PUT, to nerve's local root or
 * the bucket) before they leave, until the story releases them, as M5's page stories do (F8, F16): holdAnswer would
 * send them again from the test, and WebKit gives a route no File's bytes. Each hold is in the array as it begins;
 * once released, the upload goes on, unless page dropped it meanwhile, which then sends nothing, and settled resolves.
 */
export async function holdUploads(page: Page): Promise<{ release: () => void; settled: Promise<void> }[]> {
  const held: { release: () => void; settled: Promise<void> }[] = [];
  await page.route(
    () => true,
    async (route) => {
      if (route.request().method() !== "PUT" || apiPath(route.request())) return route.fallback();
      const released = deferred();
      const settled = deferred();
      held.push({ release: released.resolve, settled: settled.promise });
      await released.promise;
      await route.continue().catch(() => undefined);
      settled.resolve();
    }
  );
  return held;
}
