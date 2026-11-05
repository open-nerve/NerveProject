/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Locator, Page, Request } from "@playwright/test";

import { assetRows, imageHTML } from "../../fixtures/assert/storage";
import { chooseAttachment, contentFilesPage, dropFiles, pngFile, textFile } from "../../fixtures/content-files-page";
import { modalDialog } from "../../fixtures/dialogs";
import { pageFileStory } from "../../fixtures/file-story";
import { test } from "../../fixtures/profile-image-wire";
import { deferred } from "../../fixtures/deferred";
import { createDraft, draftRead, draftUpdated, updateIssue } from "../../fixtures/issues";
import { expectDecoded } from "../../fixtures/profile-images-page";
import { filePixel, prepareFile, storedFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";

/** The real drafts page row menu opens the draft's own modal. */
async function openDraft(page: Page, baseURL: string, slug: string, draftID: string, images: number, action = "Edit") {
  await page.goto(`${baseURL}/${slug}/drafts`);
  const row = page.locator(`[id="issue-${draftID}"]`);
  await row.locator('button[aria-haspopup="menu"]:visible').last().press("Enter");
  await page.getByRole("menuitem", { name: action, exact: true }).press("Enter");
  const modal = modalDialog(page);
  await expect(modal.getByRole("textbox", { name: "Title", exact: true })).toBeVisible();
  // The form refuses to leave while its description, images included, is still loading.
  await expect(modal.locator(".ProseMirror")).toBeVisible();
  const shown = modal.locator("img.image-component");
  await expect(shown).toHaveCount(images);
  for (const index of Array.from({ length: images }, (_, i) => i))
    // eslint-disable-next-line no-await-in-loop -- Each image decodes before the form can be submitted.
    await expectDecoded(shown.nth(index));
  return modal;
}
const filesOf = (modal: Locator) => modal.getByRole("region", { name: "Attachments", exact: true });
/** The modal's own project dropdown, by keyboard. */
/** The first image identity of a stored body. */
const imageOf = (html: string) => /src="([0-9a-f-]{36})"/.exec(html)?.[1];

async function chooseProject(page: Page, modal: Locator, name: string) {
  await modal.getByRole("button", { name: /^Project: /, exact: false }).focus();
  await page.keyboard.press("Enter");
  await page.getByRole("option", { name, exact: true }).click();
  await expect(modal.getByRole("button", { name: `Project: ${name}`, exact: true })).toBeVisible();
}
const editorOf = (modal: Locator) => modal.locator('.ProseMirror[contenteditable="true"]').first();

pageFileStory(
  "F10 (page): a draft without a project takes an attachment and an image in its modal, keeps both through a move, refuses publication with an unfinished file without changing anything, and publishes once from two tabs",
  async (world, info) => {
    info.setTimeout(240_000);
    const { api, page, baseURL, db } = world;
    const files = await contentFilesPage(world, info);
    const { owner, workspace, project } = files;
    const draft = await createDraft(api, owner.token, workspace.slug, {
      project_id: null,
      payload: { name: "Draft without a project", description_html: "<p>Draft body</p>" },
    });
    const scope = { kind: "draft", id: draft.id } as const;
    const note = textFile("draft-note.txt", "bytes of the draft's attachment");
    const picture = pngFile("draft-picture.png");

    let modal = await openDraft(page, baseURL, workspace.slug, draft.id, 0);
    await chooseAttachment(page, filesOf(modal), note);
    const noteID = await files.transfer(0, "DRAFT_ISSUE_ATTACHMENT", scope, note);
    await expect(filesOf(modal).getByRole("button", { name: `Download ${note.name}`, exact: true })).toBeVisible();
    await files.persisted(noteID, note);
    await dropFiles(editorOf(modal), [picture]);
    const pictureID = await files.transfer(1, "DRAFT_ISSUE_DESCRIPTION", scope, picture);
    await expectDecoded(modal.locator("img.image-component"));
    await files.persisted(pictureID, picture, false);
    const saved = page.waitForResponse(
      (r) => r.request().method() === "PATCH" && new URL(r.url()).pathname === `/api/v0/drafts/${draft.id}`
    );
    await modal.getByRole("button", { name: "Update", exact: true }).click();
    expect((await saved).status()).toBe(200);
    await expect(modal).toHaveCount(0);
    const stored = await draftRead(api, owner.token, draft.id);
    expect(stored).toMatchObject({ status: 200 });
    const html = "draft" in stored ? (stored.draft.payload.description_html ?? "") : "";
    expect(html).toContain(`src="${pictureID}"`);
    expect(html).not.toMatch(/grant=|blob:|\/static\//);

    // Reopened, both come back from their stored identities.
    modal = await openDraft(page, baseURL, workspace.slug, draft.id, 1);
    await expect(filesOf(modal).getByRole("button", { name: `Download ${note.name}`, exact: true })).toBeVisible();
    await files.downloaded(filesOf(modal).getByRole("button", { name: `Download ${note.name}`, exact: true }), note);
    const ids = [noteID, pictureID];

    // A move carries every file of the draft to the new project, under a fresh content lifetime.
    await chooseProject(page, modal, project.name);
    const moved = page.waitForResponse(
      (r) => r.request().method() === "PATCH" && new URL(r.url()).pathname === `/api/v0/drafts/${draft.id}`
    );
    await modal.getByRole("button", { name: "Update", exact: true }).click();
    expect((await moved).status()).toBe(200);
    await expect(modal).toHaveCount(0);
    for (const row of await assetRows(db, ids))
      expect(row).toMatchObject({ draft_issue_id: draft.id, project_id: project.id, issue_id: null });

    // An unfinished file refuses publication: the files, the draft and the project's work items are unchanged,
    // and the page keeps its form and points to the review.
    await prepareFile(api, owner.token, {
      purpose: "DRAFT_ISSUE_ATTACHMENT",
      scope,
      name: "later.txt",
      size: 5,
      content_type: "text/plain",
    });
    const pendingRows = async () =>
      (
        await db.query<{ id: string }>(
          "SELECT id FROM file_assets WHERE draft_issue_id=$1 AND NOT is_deleted ORDER BY id",
          [draft.id]
        )
      ).map(({ id }) => id);
    const everyID = await pendingRows();
    expect(everyID).toHaveLength(3);
    const issues = () =>
      db.query<{ n: number }>("SELECT count(*)::int AS n FROM issues WHERE project_id=$1", [project.id]);
    const issuesBefore = await issues();
    modal = await openDraft(page, baseURL, workspace.slug, draft.id, 1, "Move to project");
    const before = await assetRows(db, everyID);
    const publishPath = `/api/v0/drafts/${draft.id}/publish`;
    const refused = page.waitForResponse(
      (r) => r.request().method() === "POST" && new URL(r.url()).pathname === publishPath,
      { timeout: 15_000 }
    );
    await modal.getByRole("button", { name: "Add to project", exact: true }).click();
    expect((await refused).status()).toBe(409);
    expect((await (await refused).json()).code).toBe("asset.not_uploaded");
    await expect(
      modal.getByRole("alert").filter({ hasText: "Some files of this draft are unfinished." })
    ).toBeVisible();
    await expect(modal.getByRole("textbox", { name: "Title", exact: true })).toHaveValue(draft.payload.name!);
    expect(await assetRows(db, everyID)).toEqual(before);
    expect(await issues()).toEqual(issuesBefore);
    expect(await draftRead(api, owner.token, draft.id)).toMatchObject({ status: 200 });

    // The review lists the unfinished file; cancelling keeps it, an explicit discard deletes exactly it.
    await filesOf(modal).getByRole("button", { name: "Review unfinished draft files", exact: true }).click();
    const discard = filesOf(modal).getByRole("button", { name: "Discard file later.txt", exact: true });
    await discard.click();
    const confirm = page.getByRole("dialog").filter({ hasText: "Discard draft file" });
    await confirm.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(confirm).toHaveCount(0);
    expect(files.calls("DELETE", /^\/api\/v0\/assets\//)).toHaveLength(0);
    await discard.click();
    await page
      .getByRole("dialog")
      .filter({ hasText: "Discard draft file" })
      .getByRole("button", { name: "Discard file", exact: true })
      .click();
    await expect(discard).toHaveCount(0);
    const discarded = files.calls("DELETE", /^\/api\/v0\/assets\//);
    expect(discarded).toHaveLength(1);
    expect((await discarded[0]!.response())?.status()).toBe(204);
    expect(await pendingRows()).toEqual(ids.toSorted());

    // Two tabs publish the same draft at once: one work item, its files moved once, and a readable refusal.
    const other = await page.context().newPage();
    const otherRequests: Request[] = [];
    other.on("request", (request) => otherRequests.push(request));
    const otherModal = await openDraft(other, baseURL, workspace.slug, draft.id, 1, "Move to project");
    // Each tab saves before it publishes; the tab that loses the race is refused at whichever step meets the
    // consumed draft, and never sends again.
    const draftPath = `/api/v0/drafts/${draft.id}`;
    const outcome = (target: Page) =>
      target.waitForResponse(
        (r) => {
          const pathname = new URL(r.url()).pathname;
          return (
            (r.request().method() === "POST" && pathname === publishPath) ||
            (r.request().method() === "PATCH" && pathname === draftPath && r.status() >= 400)
          );
        },
        { timeout: 15_000 }
      );
    const [mine, theirs] = [outcome(page), outcome(other)];
    await Promise.all([
      modal.getByRole("button", { name: "Add to project", exact: true }).click(),
      otherModal.getByRole("button", { name: "Add to project", exact: true }).click(),
    ]);
    const results = [await mine, await theirs];
    const winner = results.find((r) => r.status() === 201);
    const loser = results.find((r) => r !== winner);
    expect(winner?.request().method()).toBe("POST");
    expect(loser?.status()).toBe(404);
    expect(await issues()).toEqual([{ n: issuesBefore[0]!.n + 1 }]);
    const published = await winner!.json();
    for (const row of await assetRows(db, ids))
      expect(row).toMatchObject({ draft_issue_id: null, issue_id: published.id, project_id: project.id });
    expect(published.attachment_count).toBe(1);
    expect(published.description_html).toContain(`src="${pictureID}"`);
    const refusedModal = loser === results[0] ? modal : otherModal;
    await expect(refusedModal.getByRole("alert").first()).toBeVisible();
    // Neither tab sends a save or a publication again by itself.
    const writes = () =>
      [...files.requests, ...otherRequests].filter(
        (r) => ["PATCH", "POST"].includes(r.method()) && new URL(r.url()).pathname.startsWith(draftPath)
      ).length;
    const settledWrites = writes();
    await page.waitForTimeout(1000);
    expect(writes()).toBe(settledWrites);
    await other.close();
    // The refused publication, and whichever of this tab's reads met the consumed draft.
    await files.finish(
      (failure) => failure === `409 POST ${publishPath}` || (failure.startsWith("404 ") && failure.includes(draftPath))
    );
  }
);

test("F10 (page): a lost prepare answer leaves a file only the review finds; another tab's upload in progress is listed and kept, an explicit discard deletes exactly the lost one, and publication then succeeds once [local]", async ({
  imageWorld: world,
}, info) => {
  info.setTimeout(120_000);
  const { page, baseURL, db } = world;
  const files = await contentFilesPage(world, info);
  const { workspace, project, draft } = files;
  const scope = { kind: "draft", id: draft.id } as const;
  const lost = textFile("lost-answer.txt", "bytes whose prepare answer is lost");
  const elsewhere = textFile("other-tab.txt", "bytes uploading in another tab");
  const publishPath = `/api/v0/drafts/${draft.id}/publish`;
  const modal = await openDraft(page, baseURL, workspace.slug, draft.id, 0, "Move to project");

  // The server keeps the pending row; its answer never reaches the page, which sends nothing else for it.
  await page.route(
    (url) => url.pathname === "/api/v0/assets",
    async (route) => {
      if (route.request().method() !== "POST") return route.fallback();
      await route.fetch();
      await route.abort("failed");
    },
    { times: 1 }
  );
  await chooseAttachment(page, filesOf(modal), lost);
  await expect(filesOf(modal).getByRole("alert")).toBeVisible();
  await expect(filesOf(modal).getByRole("button", { name: "Retry upload", exact: true })).toBeVisible();
  const [lostRow] = await db.query<{ id: string; uploaded: boolean }>(
    "SELECT id, is_uploaded AS uploaded FROM file_assets WHERE draft_issue_id=$1 AND attributes->>'name'=$2",
    [draft.id, lost.name]
  );
  expect(lostRow).toMatchObject({ uploaded: false });
  expect(files.calls("POST", "/api/v0/assets").map((r) => r.postDataJSON())).toEqual([
    { purpose: "DRAFT_ISSUE_ATTACHMENT", scope, name: lost.name, content_type: lost.type, size: lost.bytes.length },
  ]);
  expect(files.requests.filter((r) => r.method() === "PUT")).toHaveLength(0);

  // Publication is refused with the review pointed out; the form stays.
  const refused = page.waitForResponse(
    (r) => r.request().method() === "POST" && new URL(r.url()).pathname === publishPath,
    { timeout: 15_000 }
  );
  await modal.getByRole("button", { name: "Add to project", exact: true }).click();
  expect((await refused).status()).toBe(409);
  await expect(modal.getByRole("alert").filter({ hasText: "Some files of this draft are unfinished." })).toBeVisible();

  // Another tab of the same person is uploading to the same draft; its file is unfinished until its answer arrives.
  const other = await page.context().newPage();
  // Its confirmation waits before it leaves the tab (route.fetch would resend WebKit's PUT without its File body).
  const confirmation = deferred();
  await other.route(
    (url) => /^\/api\/v0\/assets\/[^/]+\/complete$/.test(url.pathname),
    async (route) => {
      await confirmation.promise;
      await route.continue();
    }
  );
  const release = async () => confirmation.resolve();
  const otherModal = await openDraft(other, baseURL, workspace.slug, draft.id, 0);
  await chooseAttachment(other, filesOf(otherModal), elsewhere);
  await expect
    .poll(
      async () =>
        (
          await db.query("SELECT 1 FROM file_assets WHERE draft_issue_id=$1 AND attributes->>'name'=$2", [
            draft.id,
            elsewhere.name,
          ])
        ).length
    )
    .toBe(1);

  // The review lists both with its warning; the person keeps the other tab's file and discards only the lost one.
  await filesOf(modal).getByRole("button", { name: "Review unfinished draft files", exact: true }).click();
  await expect(filesOf(modal).getByText(/They may be in use in another tab/)).toBeVisible();
  const discardOf = (name: string) => filesOf(modal).getByRole("button", { name: `Discard file ${name}`, exact: true });
  await expect(discardOf(lost.name)).toBeVisible();
  await expect(discardOf(elsewhere.name)).toBeVisible();
  await discardOf(elsewhere.name).click();
  const confirm = page.getByRole("dialog").filter({ hasText: "Discard draft file" });
  await expect(confirm).toContainText("This file may be uploading in another tab");
  await confirm.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(confirm).toHaveCount(0);
  await discardOf(lost.name).click();
  await confirm.getByRole("button", { name: "Discard file", exact: true }).click();
  await expect(discardOf(lost.name)).toHaveCount(0);
  const deletions = files.calls("DELETE", /^\/api\/v0\/assets\//);
  expect(deletions.map((r) => new URL(r.url()).pathname)).toEqual([`/api/v0/assets/${lostRow!.id}`]);
  expect((await deletions[0]!.response())?.status()).toBe(204);

  // The other tab's answer arrives: its file is attached, and the publication carries it.
  await release();
  const elsewhereID = await (async () => {
    await expect(
      filesOf(otherModal).getByRole("button", { name: `Download ${elsewhere.name}`, exact: true })
    ).toBeVisible();
    const [row] = await db.query<{ id: string }>(
      "SELECT id FROM file_assets WHERE draft_issue_id=$1 AND attributes->>'name'=$2",
      [draft.id, elsewhere.name]
    );
    return row!.id;
  })();
  await files.persisted(elsewhereID, elsewhere);
  await other.close();
  const published = page.waitForResponse(
    (r) => r.request().method() === "POST" && new URL(r.url()).pathname === publishPath,
    { timeout: 15_000 }
  );
  await modal.getByRole("button", { name: "Add to project", exact: true }).click();
  expect((await published).status()).toBe(201);
  const issue = await (await published).json();
  expect(issue).toMatchObject({ project_id: project.id, attachment_count: 1 });
  expect(await assetRows(db, [elsewhereID])).toMatchObject([{ issue_id: issue.id, draft_issue_id: null }]);
  expect(files.calls("POST", publishPath)).toHaveLength(2);
  expect(files.calls("POST", "/api/v0/assets")).toHaveLength(1);
  await files.finish((failure) => failure === `409 POST ${publishPath}` || failure.endsWith("POST /api/v0/assets"));
});

pageFileStory(
  "F10 (page): an ordinary copy with an image ends as an Issue holding its own server copy; a draft copy whose first image copy fails keeps its one target, ends as a Draft on the explicit retry, and lists the lost copy for an explicit discard",
  async (world, info) => {
    info.setTimeout(150_000);
    const { api, page, baseURL, db, uploadStatus } = world;
    const files = await contentFilesPage(world, info);
    const { owner, workspace, issue, draft } = files;
    const draftsPath = `/api/v0/workspaces/${workspace.slug}/drafts`;

    // The ordinary copy: one target draft, one server copy of the image, one publication, an Issue.
    const source = await storedFile(
      api,
      owner.token,
      "ISSUE_DESCRIPTION",
      { kind: "issue", id: issue.id },
      filePixel,
      "image/png",
      "source.png",
      uploadStatus
    );
    await updateIssue(api, owner.token, issue.id, { description_html: `<p>Copied body</p>${imageHTML(source.id)}` });
    const sourceBefore = await assetRows(db, [source.id]);
    await page.goto(`${baseURL}/${workspace.slug}/browse/FILE-${issue.sequence_id}`);
    await expectDecoded(page.locator("img.image-component").first());
    await page.getByRole("button", { name: `Options: ${issue.name}`, exact: true }).click();
    await page.getByRole("menuitem", { name: "Make a copy", exact: true }).click();
    let modal = modalDialog(page);
    // A new work item has no content host: the copied image stays inert until the server copies it.
    await expect(modal.getByRole("status").filter({ hasText: "Image unavailable" })).toBeVisible();
    const publication = page.waitForResponse(
      (r) => r.request().method() === "POST" && /^\/api\/v0\/drafts\/[^/]+\/publish$/.test(new URL(r.url()).pathname),
      { timeout: 20_000 }
    );
    await modal.getByRole("button", { name: "Save", exact: true }).click();
    expect((await publication).status()).toBe(201);
    const copied = await (await publication).json();
    const copiedImage = imageOf(copied.description_html);
    expect(copiedImage).toBeDefined();
    expect(copiedImage).not.toBe(source.id);
    await files.persisted(copiedImage!, pngFile("source.png"));
    expect(await assetRows(db, [source.id])).toEqual(sourceBefore);
    expect(files.calls("POST", draftsPath)).toHaveLength(1);
    expect(files.calls("POST", `/api/v0/assets/${source.id}/duplicate`)).toHaveLength(1);
    await expect(modal).toHaveCount(0);

    // The draft copy: the first image copy is made but its answer fails; the target is kept and retried, never twice.
    const draftImage = await storedFile(
      api,
      owner.token,
      "DRAFT_ISSUE_DESCRIPTION",
      { kind: "draft", id: draft.id },
      filePixel,
      "image/png",
      "draft-source.png",
      uploadStatus
    );
    const sourcePayload = {
      name: "Draft to copy",
      description_html: `<p>Draft copy body</p>${imageHTML(draftImage.id)}`,
    };
    expect(await draftUpdated(api, owner.token, draft.id, { payload: sourcePayload })).toMatchObject({ status: 200 });
    const duplicatePath = `/api/v0/assets/${draftImage.id}/duplicate`;
    await page.route(
      (url) => url.pathname === duplicatePath,
      async (route) => {
        await route.fetch();
        await route.fulfill({
          status: 503,
          contentType: "application/problem+json",
          body: JSON.stringify({ type: "about:blank", title: "Busy", status: 503, code: "server_busy" }),
        });
      },
      { times: 1 }
    );
    modal = await openDraft(page, baseURL, workspace.slug, draft.id, 0, "Make a copy");
    await expect(modal.getByRole("status").filter({ hasText: "Image unavailable" })).toBeVisible();
    await modal.getByRole("button", { name: "Save to Drafts", exact: true }).click();
    const kept = modal
      .getByRole("status")
      .filter({ hasText: "A draft copy is kept in your drafts. Trying again continues with the same draft." });
    await expect(kept).toBeVisible();
    await expect(kept.getByRole("link", { name: "Open drafts", exact: true })).toHaveAttribute(
      "href",
      `/${workspace.slug}/drafts`
    );
    expect(files.calls("POST", draftsPath)).toHaveLength(2);
    expect(files.calls("POST", duplicatePath)).toHaveLength(1);
    const saved = page.waitForResponse(
      (r) => r.request().method() === "PATCH" && /^\/api\/v0\/drafts\/[^/]+$/.test(new URL(r.url()).pathname),
      { timeout: 15_000 }
    );
    await modal.getByRole("button", { name: "Save to Drafts", exact: true }).click();
    expect((await saved).status()).toBe(200);
    const target = await (await saved).json();
    await expect(modal).toHaveCount(0);
    expect(target.id).not.toBe(draft.id);
    expect(files.calls("POST", draftsPath)).toHaveLength(2);
    expect(files.calls("POST", duplicatePath)).toHaveLength(2);
    expect(files.calls("POST", /\/publish$/)).toHaveLength(1);
    expect(await draftRead(api, owner.token, draft.id)).toMatchObject({
      status: 200,
      draft: { payload: sourcePayload },
    });
    const targetImage = imageOf(target.payload.description_html);
    const [lostCopy] = await db.query<{ id: string }>(
      "SELECT id FROM file_assets WHERE draft_issue_id=$1 AND id<>$2 AND NOT is_deleted",
      [target.id, targetImage]
    );
    expect(lostCopy).toBeDefined();

    // The new draft lists the copy whose answer was lost; discarding it leaves the draft's own image.
    modal = await openDraft(page, baseURL, workspace.slug, target.id, 1);
    await filesOf(modal).getByRole("button", { name: "Review unfinished draft files", exact: true }).click();
    const discard = filesOf(modal).getByRole("button", { name: "Discard file draft-source.png", exact: true });
    await expect(discard).toHaveCount(1);
    await discard.click();
    await page
      .getByRole("dialog")
      .filter({ hasText: "Discard draft file" })
      .getByRole("button", { name: "Discard file", exact: true })
      .click();
    await expect(discard).toHaveCount(0);
    expect(files.calls("DELETE", `/api/v0/assets/${lostCopy!.id}`)).toHaveLength(1);
    await expectDecoded(modal.locator("img.image-component"));
    expect(await assetRows(db, [draftImage.id])).toMatchObject([{ draft_issue_id: draft.id, is_deleted: false }]);
    await files.finish((failure) => failure === `503 POST ${duplicatePath}`);
  }
);
