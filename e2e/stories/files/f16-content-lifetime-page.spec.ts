/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { randomUUID } from "node:crypto";

import type { Locator, Page, Request } from "@playwright/test";

import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import { assetRows } from "../../fixtures/assert/storage";
import { createIssueComment } from "../../fixtures/comments";
import { bearer, password } from "../../fixtures/auth";
import { submitSignIn } from "../../fixtures/auth-pages";
import { chooseAttachment, contentFilesPage, textFile, type ContentFile } from "../../fixtures/content-files-page";
import { deferred } from "../../fixtures/deferred";
import { modalDialog } from "../../fixtures/dialogs";
import { pageFileStory, type PageFileStoryWorld } from "../../fixtures/file-story";
import { createIssue, issueArchived, issueRead, updateIssue } from "../../fixtures/issues";
import { openCommands } from "../../fixtures/power-k";
import { test } from "../../fixtures/profile-image-wire";
import { expectDecoded } from "../../fixtures/profile-images-page";
import { moveWithinApp } from "../../fixtures/settings-pages";
import { stateIds } from "../../fixtures/states";
import { filePixel, storedFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";

type Host = "detail" | "peek";
type Issue = { id: string; sequence_id: number };
type Files = Awaited<ReturnType<typeof contentFilesPage>>;

/** A problem answer as nerve writes one. */
const problem = (status: number, code: string) => ({
  status,
  contentType: "application/problem+json",
  body: JSON.stringify({ type: "about:blank", title: code, status, code }),
});

/** The host's own subtree: the detail page, or the peek above the project's list. */
async function openHost(page: Page, baseURL: string, slug: string, projectID: string, issue: Issue, kind: Host) {
  if (kind === "detail") {
    await page.goto(`${baseURL}/${slug}/browse/FILE-${issue.sequence_id}`);
    return page.locator("body");
  }
  await page.goto(`${baseURL}/${slug}/projects/${projectID}/issues`);
  await page.locator(`#issue-${issue.id}`).click();
  const peek = page.locator("#full-screen-portal");
  await expect(peek.getByRole("button", { name: "Close the peek view", exact: true })).toBeVisible();
  return peek;
}

/** Every control of the work item that a canonical re-read unmounts and remounts. */
function controls(host: Locator, issueID: string, commentID = "") {
  const card = host.locator(`#comment-${commentID}`);
  return {
    title: host.getByRole("textbox", { name: "Work item title", exact: true }),
    /** The body as it shows, editable or not. */
    document: host.locator(`[id="editor-container-${issueID}"] .ProseMirror`),
    body: host.locator(`[id="editor-container-${issueID}"] [contenteditable="true"]`),
    composer: host.locator(`[id^="editor-container-add_comment_${issueID}_"] [contenteditable="true"]`),
    card,
    edit: card.getByRole("textbox", { name: "Comment", exact: true }),
    trigger: host.getByRole("button", { name: /^Attachments \d+$/ }),
    widget: host.getByRole("button", { name: /^Attachments \d+$/ }).locator("xpath=../following-sibling::*[1]"),
    status: (text: string) => host.getByRole("status").filter({ hasText: text }),
  };
}
type Controls = ReturnType<typeof controls>;

/** The stored body: a paragraph and an image, each with its block identity, so opening it writes nothing. */
async function storeBody(world: PageFileStoryWorld, token: string, issueID: string) {
  const image = await storedFile(
    world.api,
    token,
    "ISSUE_DESCRIPTION",
    { kind: "issue", id: issueID },
    filePixel,
    "image/png",
    "body.png",
    world.uploadStatus
  );
  await updateIssue(world.api, token, issueID, {
    description_html: `<p data-id="${randomUUID()}">Stored body</p><image-component data-id="${randomUUID()}" src="${image.id}"></image-component>`,
  });
  return image;
}

/** Every body the page saves, each refused with a known 422 until accept. */
async function refuseBodies(page: Page, issueID: string) {
  const saves: string[] = [];
  let refusing = true;
  await page.route(
    (url) => url.pathname === `/api/v0/issues/${issueID}`,
    async (route) => {
      const request = route.request();
      const sent = request.method() === "PATCH" ? (request.postDataJSON() as { description_html?: string }) : {};
      if (sent.description_html === undefined) return route.fallback();
      saves.push(sent.description_html);
      return refusing ? route.fulfill(problem(422, "validation_failed")) : route.fallback();
    }
  );
  return {
    saves,
    accept: () => {
      refusing = false;
    },
  };
}

/** The person edits the stored body; nerve refuses the save it sends, which leaves the edit unsaved in the editor. */
async function refusedEdit(page: Page, host: Locator, c: Controls, saves: readonly string[]) {
  await c.body.getByText("Stored body", { exact: true }).click();
  await page.keyboard.press("End");
  await page.keyboard.type(" and a refused edit");
  await expect.poll(() => saves.length, { timeout: 10_000 }).toBe(1);
  expect(saves[0]).toContain("Stored body and a refused edit");
  await expect(host.getByRole("alert")).toBeVisible();
}

/** Holds the work item's next read until answer says how it ends: a refusal, or nerve's own answer. */
async function holdNextRead(page: Page, issueID: string) {
  const reached = deferred();
  const ending = deferred<{ status: number; code: string } | undefined>();
  let armed = true;
  await page.route(
    (url) => url.pathname === `/api/v0/issues/${issueID}`,
    async (route) => {
      if (route.request().method() !== "GET" || !armed) return route.fallback();
      armed = false;
      reached.resolve();
      const refusal = await ending.promise;
      return refusal ? route.fulfill(problem(refusal.status, refusal.code)) : route.fallback();
    }
  );
  return { reached: reached.promise, answer: ending.resolve };
}

/** The work item's attachment widget, expanded if it is collapsed. */
async function openFiles(c: Controls) {
  if ((await c.trigger.getAttribute("aria-expanded")) !== "true") await c.trigger.click();
}

/** A confirmed upload of file into the work item's own attachment widget. */
async function upload(page: Page, files: Files, c: Controls, issueID: string, file: ContentFile, ordinal = 0) {
  await openFiles(c);
  await chooseAttachment(page, c.widget, file);
  return files.transfer(ordinal, "ISSUE_ATTACHMENT", { kind: "issue", id: issueID }, file);
}

/** Nothing of the old input shows: the stored body, an empty composer, no preserved input and no confirmation. */
async function expectNoOldInput(c: Controls) {
  await expect(c.document).toContainText("Stored body");
  await expect(c.document).not.toContainText("refused edit");
  await expect(c.composer).toBeEmpty();
  await expect(c.status("Your unsaved input")).toHaveCount(0);
  await expect(c.status("This content changed")).toHaveCount(0);
  await expect(c.status("File uploaded.")).toHaveCount(0);
}

for (const kind of ["detail", "peek"] as const)
  pageFileStory(
    `F16 (page, ${kind}): an unsent comment, a refused body with its image and an unknown comment edit outlive a confirmed upload's and deletion's canonical re-read, its failure and explicit retry; a changed body is never overwritten and nothing is sent again by itself`,
    async (world, info) => {
      info.setTimeout(240_000);
      const { api, page, baseURL, db } = world;
      const files = await contentFilesPage(world, info);
      const { owner, other, workspace, project, issue, comment } = files;
      await inviteAndAccept(api, owner.token, workspace.slug, other, 15);
      await addProjectMembers(api, owner.token, project.id, [{ member_id: other.id, role: 15 }]);
      const image = await storeBody(world, owner.token, issue.id);
      const issuePath = `/api/v0/issues/${issue.id}`;
      const commentPath = `/api/v0/issue-comments/${comment.id}`;
      const bodies = await refuseBodies(page, issue.id);
      const commentEdits: string[] = [];
      let loseEdit = true;
      await page.route(
        (url) => url.pathname === commentPath,
        async (route) => {
          if (route.request().method() !== "PATCH") return route.fallback();
          commentEdits.push((route.request().postDataJSON() as { comment_html: string }).comment_html);
          // The answer is lost before nerve has it: the page cannot know whether the edit was applied.
          return loseEdit ? route.abort("failed") : route.fallback();
        }
      );

      const host = await openHost(page, baseURL, workspace.slug, project.id, issue, kind);
      const c = controls(host, issue.id, comment.id);
      await expect(c.title).toHaveValue(issue.name);
      await expectDecoded(c.body.locator("img.image-component"));
      const contentReads = () => files.calls("GET", `/api/v0/assets/${image.id}/content`).length;

      // The body edit is refused (a known 422): its text and image stay in the editor, unsaved.
      await refusedEdit(page, host, c, bodies.saves);
      expect(bodies.saves[0]).toContain(image.id);
      // The comment edit's answer is lost: unknown, never resent by itself.
      await c.card.getByRole("button", { name: /^Options for .+’s comment$/ }).click();
      await page.getByRole("menuitem", { name: "Edit", exact: true }).click();
      await c.edit.click();
      await c.edit.press("ControlOrMeta+A");
      await page.keyboard.insertText("Edited comment whose answer is lost");
      await c.card.getByRole("button", { name: "Save", exact: true }).click();
      await expect(c.card.getByText(/^The comment may have been saved\./)).toBeVisible();
      expect(commentEdits).toHaveLength(1);
      // A new comment the person has not sent.
      await c.composer.click();
      await page.keyboard.insertText("Unsent comment");
      await expect(c.composer).toHaveText("Unsent comment");

      // A confirmed upload into the same work item: its canonical re-read withdraws every control above it.
      const read = await holdNextRead(page, issue.id);
      const file = textFile("lifetime.txt", "bytes of an attachment whose confirmation re-reads its work item");
      const fileID = await upload(page, files, c, issue.id, file);
      await read.reached;
      await expect(c.status("File uploaded.")).toBeVisible();
      await expect(c.title).toHaveCount(0);
      await expect(c.body).toHaveCount(0);
      await expect(c.composer).toHaveCount(0);
      await expect(c.card).toHaveCount(0);
      // The read fails for now: only a read retry is offered, and nothing of the input is sent.
      read.answer({ status: 503, code: "server_busy" });
      const retry = host.getByRole("alert").filter({ hasText: "The server is busy." });
      await expect(retry.getByRole("button", { name: "Retry", exact: true })).toBeVisible();
      await expect(c.status("File uploaded.")).toBeVisible();
      await files.persisted(fileID, file);
      // Meanwhile another member changes the body the refused edit started from.
      await updateIssue(api, other.token, issue.id, {
        description_html: `<p data-id="${randomUUID()}">Body from the other member</p>`,
      });
      const readsBefore = contentReads();
      await retry.getByRole("button", { name: "Retry", exact: true }).click();

      // The fresh owner: every input and the confirmation are back; the changed body waits for the person's choice.
      await expect(c.body).toContainText("Stored body and a refused edit");
      await expect(c.body).not.toContainText("Body from the other member");
      await expectDecoded(c.body.locator("img.image-component"));
      expect(contentReads()).toBeGreaterThan(readsBefore);
      await expect(c.status("This content changed while your input was preserved.")).toBeVisible();
      await expect(c.composer).toHaveText("Unsent comment");
      await expect(c.edit).toHaveText("Edited comment whose answer is lost");
      await expect(c.card.getByText(/^The comment may have been saved\./)).toBeVisible();
      await expect(c.status("File uploaded.")).toBeVisible();
      await expect(c.title).toHaveValue(issue.name);
      await page.waitForTimeout(2500);
      expect(bodies.saves).toHaveLength(1);
      expect(commentEdits).toHaveLength(1);
      expect(files.calls("POST", `/api/v0/issues/${issue.id}/comments`)).toHaveLength(0);
      expect(files.calls("POST", "/api/v0/assets")).toHaveLength(1);
      expect(files.calls("POST", `/api/v0/assets/${fileID}/complete`)).toHaveLength(1);
      const stored = await issueRead(api, owner.token, issue.id);
      expect("issue" in stored && stored.issue.description_html).toContain("Body from the other member");

      // A confirmed deletion: the same re-read, read at once this time; the inputs stay, the deletion is announced.
      await c.widget.getByRole("button", { name: `Delete file ${file.name}`, exact: true }).click();
      await modalDialog(page).getByRole("button", { name: "Delete file", exact: true }).click();
      await expect(c.status("File deleted.")).toBeVisible();
      await expect(c.widget.getByRole("button", { name: `Delete file ${file.name}`, exact: true })).toHaveCount(0);
      await expect(c.body).toContainText("Stored body and a refused edit");
      await expect(c.status("This content changed while your input was preserved.")).toBeVisible();
      await expect(c.composer).toHaveText("Unsent comment");
      await expect(c.edit).toHaveText("Edited comment whose answer is lost");
      expect(files.calls("DELETE", `/api/v0/assets/${fileID}`)).toHaveLength(1);
      expect(await assetRows(db, [fileID])).toMatchObject([{ is_deleted: true }]);
      expect(bodies.saves).toHaveLength(1);
      expect(commentEdits).toHaveLength(1);

      // Only the person's explicit choices send anything: the kept body, the checked edit and the new comment.
      bodies.accept();
      loseEdit = false;
      await host.getByRole("button", { name: "Save my changes", exact: true }).click();
      await expect.poll(() => bodies.saves.length).toBe(2);
      expect(bodies.saves[1]).toContain("Stored body and a refused edit");
      expect(bodies.saves[1]).toContain(image.id);
      await expect(c.status("This content changed while your input was preserved.")).toHaveCount(0);
      await expect
        .poll(async () => {
          const saved = await issueRead(api, owner.token, issue.id);
          return "issue" in saved ? saved.issue.description_html : "";
        })
        .toContain("Stored body and a refused edit");
      await c.card.getByRole("button", { name: "Read comments to check", exact: true }).click();
      await c.card.getByRole("button", { name: "I checked; allow another submission", exact: true }).click();
      await c.card.getByRole("button", { name: "Save", exact: true }).click();
      await expect(c.edit).toHaveCount(0);
      await expect(c.card).toContainText("Edited comment whose answer is lost");
      // The explicit resend is the very edit whose answer was lost.
      expect(commentEdits).toHaveLength(2);
      expect(commentEdits[1]).toBe(commentEdits[0]);
      expect(commentEdits[0]).toContain("Edited comment whose answer is lost");
      await expect(c.composer).toHaveText("Unsent comment");
      await host.getByRole("button", { name: "Comment", exact: true }).click();
      await expect(c.composer).toBeEmpty();
      await expect(host.getByText("Unsent comment", { exact: true })).toBeVisible();
      expect(files.calls("POST", `/api/v0/issues/${issue.id}/comments`)).toHaveLength(1);
      const [edited] = await db.query<{ comment_html: string }>("SELECT comment_html FROM issue_comments WHERE id=$1", [
        comment.id,
      ]);
      expect(edited?.comment_html).toContain("Edited comment whose answer is lost");
      await files.finish(
        (failure) =>
          failure === `422 PATCH ${issuePath}` ||
          failure === `503 GET ${issuePath}` ||
          failure.endsWith(` PATCH ${commentPath}`)
      );
    }
  );

for (const [status, kind] of [
  [403, "detail"],
  [404, "peek"],
] as const)
  test(`F16 (page, ${kind}): a canonical ${status} retires the retained input and the confirmation; the same work item read again revives neither [local]`, async ({
    imageWorld: world,
  }, info) => {
    info.setTimeout(120_000);
    const { page, baseURL } = world;
    const files = await contentFilesPage(world, info);
    const { owner, workspace, project, issue } = files;
    await storeBody(world, owner.token, issue.id);
    const bodies = await refuseBodies(page, issue.id);
    const host = await openHost(page, baseURL, workspace.slug, project.id, issue, kind);
    const c = controls(host, issue.id);
    await refusedEdit(page, host, c, bodies.saves);
    await c.composer.click();
    await page.keyboard.insertText("Unsent comment");

    const read = await holdNextRead(page, issue.id);
    await upload(
      page,
      files,
      c,
      issue.id,
      textFile("refused-owner.txt", "bytes confirmed before the owner is refused")
    );
    await read.reached;
    await expect(c.body).toHaveCount(0);
    await expect(c.status("File uploaded.")).toBeVisible();
    read.answer(status === 404 ? { status, code: "issue.not_found" } : { status, code: "forbidden" });
    // The owner's refusal is a conclusion, not a passing failure: the confirmation and every input end with it.
    await expect(c.status("File uploaded.")).toHaveCount(0);
    await expect.poll(() => files.watch.apiFailures).toContain(`${status} GET /api/v0/issues/${issue.id}`);

    // The same work item, visible again: a new owner holding none of the old input. The detail's reader asks for it;
    // the peek's 404 has the page read again the list it came from, which still holds it, and it shows by itself.
    if (kind === "detail") await host.getByRole("alert").getByRole("button", { name: "Retry", exact: true }).click();
    await expect(c.title).toHaveValue(issue.name);
    await expectNoOldInput(c);
    await page.waitForTimeout(2500);
    expect(bodies.saves).toHaveLength(1);
    expect(files.calls("POST", `/api/v0/issues/${issue.id}/comments`)).toHaveLength(0);
    await files.finish(
      (failure) =>
        failure === `422 PATCH /api/v0/issues/${issue.id}` || failure === `${status} GET /api/v0/issues/${issue.id}`
    );
  });

test("F16 (page): a canonical re-read refused once with 401 is renewed and sent again; a lifted refusal is no conclusion, and every input carries on [local]", async ({
  imageWorld: world,
}, info) => {
  info.setTimeout(120_000);
  const { page, baseURL } = world;
  const files = await contentFilesPage(world, info);
  const { owner, workspace, project, issue } = files;
  await storeBody(world, owner.token, issue.id);
  const bodies = await refuseBodies(page, issue.id);
  const host = await openHost(page, baseURL, workspace.slug, project.id, issue, "detail");
  const c = controls(host, issue.id);
  await refusedEdit(page, host, c, bodies.saves);
  await c.composer.click();
  await page.keyboard.insertText("Unsent comment");

  // nerve refuses the confirmation's re-read once as unauthenticated; the page renews its token and sends it again.
  const tokens: (string | null)[] = [];
  let refused = false;
  await page.route(
    (url) => url.pathname === `/api/v0/issues/${issue.id}`,
    async (route) => {
      if (route.request().method() !== "GET") return route.fallback();
      tokens.push(await route.request().headerValue("authorization"));
      if (refused) return route.fallback();
      refused = true;
      return route.fulfill(problem(401, "unauthorized"));
    }
  );
  const renewal = page.waitForResponse((r) => new URL(r.url()).pathname === "/api/v0/auth/refresh");
  await upload(page, files, c, issue.id, textFile("renewed-read.txt", "bytes confirmed before a renewed read"));
  expect((await renewal).status()).toBe(200);
  await expect(c.body).toContainText("Stored body and a refused edit");
  await expect(c.composer).toHaveText("Unsent comment");
  await expect(c.status("File uploaded.")).toBeVisible();
  await expect(c.status("Your unsaved input has been preserved.")).toBeVisible();
  expect(tokens).toHaveLength(2);
  expect(tokens[1]).toMatch(/^Bearer /);
  expect(tokens[1]).not.toBe(tokens[0]);
  await page.waitForTimeout(2500);
  expect(bodies.saves).toHaveLength(1);
  await files.finish(
    (failure) => failure === `422 PATCH /api/v0/issues/${issue.id}` || failure === `401 GET /api/v0/issues/${issue.id}`
  );
});

test("F16 (page): a canonical 401 after another session changed the password ends this session; its root retires with every input, and another account signing in on the same tab finds none of it [local]", async ({
  imageWorld: world,
}, info) => {
  info.setTimeout(120_000);
  const { api, page, baseURL } = world;
  const files = await contentFilesPage(world, info);
  const { owner, other, workspace, project, issue } = files;
  // The account that signs in next on the same tab: a member of the same project.
  await inviteAndAccept(api, owner.token, workspace.slug, other, 15);
  await addProjectMembers(api, owner.token, project.id, [{ member_id: other.id, role: 15 }]);
  expect(
    (await api.PATCH("/api/v0/me/profile", { headers: bearer(other.token), body: { is_onboarded: true } })).response
      .status
  ).toBe(200);
  await storeBody(world, owner.token, issue.id);
  const bodies = await refuseBodies(page, issue.id);
  const host = await openHost(page, baseURL, workspace.slug, project.id, issue, "detail");
  const c = controls(host, issue.id);
  await refusedEdit(page, host, c, bodies.saves);
  await c.composer.click();
  await page.keyboard.insertText("Unsent comment");

  // While the confirmation's re-read is on its way, the owner's token changes the password: every session ends.
  const read = await holdNextRead(page, issue.id);
  await upload(page, files, c, issue.id, textFile("ended-session.txt", "bytes confirmed before the session ends"));
  await read.reached;
  const renewed = "N3w-Passw0rd!";
  const changed = await api.POST("/api/v0/me/change-password", {
    body: { current_password: password, new_password: renewed },
    headers: bearer(owner.token),
  });
  expect(changed.response.status).toBe(204);
  read.answer(undefined);
  // nerve refuses the read, and the renewal too: the session ends, and the sign-in page shows.
  await expect(page.getByRole("button", { name: "Go to workspace", exact: true })).toBeVisible();
  expect(files.watch.apiFailures).toContain(`401 GET /api/v0/issues/${issue.id}`);

  // Another account signs in on the same tab: its work item holds none of the old session's input.
  expect(await submitSignIn(page, other.email, password)).toBe(200);
  await expect(page.getByRole("button", { name: "Go to workspace", exact: true })).toHaveCount(0);
  if (!page.url().includes(`/browse/FILE-${issue.sequence_id}`))
    await moveWithinApp(page, `/${workspace.slug}/browse/FILE-${issue.sequence_id}`);
  await expect(c.title).toHaveValue(issue.name);
  await expectNoOldInput(c);
  await page.waitForTimeout(2500);
  expect(bodies.saves).toHaveLength(1);
  expect(files.calls("POST", `/api/v0/issues/${issue.id}/comments`)).toHaveLength(0);
  await files.finish((failure) => failure === `422 PATCH /api/v0/issues/${issue.id}` || failure.startsWith("401 "));
});

test("F16 (page): Back and Forward keep a left transfer's late end out of the work items they move between [local]", async ({
  imageWorld: world,
}, info) => {
  info.setTimeout(120_000);
  const { api, page, baseURL, db } = world;
  const files = await contentFilesPage(world, info);
  const { owner, workspace, project, issue } = files;
  const next = await createIssue(api, owner.token, project.id, { name: "Next item" });
  // The raw upload waits here until the story lets it go on.
  const putReached = deferred();
  const putReleased = deferred();
  let held = false;
  await page.route(
    () => true,
    async (route) => {
      if (route.request().method() !== "PUT" || held) return route.fallback();
      held = true;
      putReached.resolve();
      await putReleased.promise;
      // A request the page has dropped sends nothing on.
      await route.continue().catch(() => undefined);
    }
  );
  const body = page.locator("body");
  const detail = controls(body, issue.id);
  const away = controls(body, next.id);
  await page.goto(`${baseURL}/${workspace.slug}/browse/FILE-${issue.sequence_id}`);
  await expect(detail.title).toHaveValue(issue.name);
  await openFiles(detail);
  const file = textFile("left-behind.txt", "bytes whose upload the person leaves behind");
  await chooseAttachment(page, detail.widget, file);
  await putReached.promise;

  // The person goes on to another work item while the bytes are on their way, then Back once it has shown its files.
  const awayFiles = page.waitForResponse(
    (response) => new URL(response.url()).pathname === `/api/v0/issues/${next.id}/attachments`
  );
  await moveWithinApp(page, `/${workspace.slug}/browse/FILE-${next.sequence_id}`);
  await expect(away.title).toHaveValue(next.name);
  await awayFiles;
  await page.goBack();
  await expect(detail.title).toHaveValue(issue.name);
  await openFiles(detail);
  await expect(detail.widget.getByText("Choose a file.", { exact: true })).toBeVisible();
  putReleased.resolve();
  await page.waitForTimeout(1500);
  // The left transfer is confirmed nowhere and shows nowhere: no file, no confirmation, the count unchanged.
  const [row] = await db.query<{ id: string; uploaded: boolean }>(
    "SELECT id, is_uploaded AS uploaded FROM file_assets WHERE attributes->>'name'=$1",
    [file.name]
  );
  expect(row).toMatchObject({ uploaded: false });
  expect(files.calls("POST", `/api/v0/assets/${row!.id}/complete`)).toHaveLength(0);
  await expect(detail.status("File uploaded.")).toHaveCount(0);
  await expect(detail.widget.getByRole("button", { name: `Download ${file.name}`, exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Attachments 0", exact: true })).toBeVisible();
  // Forward again: the other work item is as it was.
  await page.goForward();
  await expect(away.title).toHaveValue(next.name);
  await openFiles(away);
  await expect(away.widget.getByText("Choose a file.", { exact: true })).toBeVisible();
  await expect(away.status("File uploaded.")).toHaveCount(0);
  await files.finish();
});

test("F16 (page): a transfer left behind in one workspace ends there; in another workspace its late end confirms nothing and shows nowhere [local]", async ({
  imageWorld: world,
}, info) => {
  info.setTimeout(120_000);
  const { api, page, baseURL, db } = world;
  const files = await contentFilesPage(world, info);
  const { owner, workspace, issue } = files;
  // A second workspace of the same account, with a work item of its own.
  const elsewhere = await createWorkspace(api, owner.token, { name: "Elsewhere", slug: slugFor(info, "elsewhere") });
  const there = await createProject(api, owner.token, elsewhere.slug, {
    name: "Elsewhere",
    identifier: "ELSE",
    network: 0,
  });
  const away = await createIssue(api, owner.token, there.id, { name: "Elsewhere item" });
  // The raw upload waits here until the story lets it go on.
  const putReached = deferred();
  const putReleased = deferred();
  let held = false;
  await page.route(
    () => true,
    async (route) => {
      if (route.request().method() !== "PUT" || held) return route.fallback();
      held = true;
      putReached.resolve();
      await putReleased.promise;
      // A request the page has dropped sends nothing on.
      await route.continue().catch(() => undefined);
    }
  );
  const body = page.locator("body");
  const here = controls(body, issue.id);
  const thereControls = controls(body, away.id);
  await page.goto(`${baseURL}/${workspace.slug}/browse/FILE-${issue.sequence_id}`);
  await expect(here.title).toHaveValue(issue.name);
  await openFiles(here);
  const file = textFile("left-in-files.txt", "bytes whose upload the person leaves in the first workspace");
  await chooseAttachment(page, here.widget, file);
  await putReached.promise;

  // The person goes to the other workspace's work item while the bytes are on their way.
  const awayFiles = page.waitForResponse(
    (response) => new URL(response.url()).pathname === `/api/v0/issues/${away.id}/attachments`
  );
  await moveWithinApp(page, `/${elsewhere.slug}/browse/ELSE-${away.sequence_id}`);
  await expect(thereControls.title).toHaveValue(away.name);
  await awayFiles;
  putReleased.resolve();
  await page.waitForTimeout(1500);
  // The left transfer is confirmed nowhere and shows nowhere.
  const [row] = await db.query<{ id: string; uploaded: boolean }>(
    "SELECT id, is_uploaded AS uploaded FROM file_assets WHERE attributes->>'name'=$1",
    [file.name]
  );
  expect(row).toMatchObject({ uploaded: false });
  expect(files.calls("POST", `/api/v0/assets/${row!.id}/complete`)).toHaveLength(0);
  await expect(page.getByRole("status").filter({ hasText: "File uploaded." })).toHaveCount(0);
  await openFiles(thereControls);
  await expect(thereControls.widget.getByText("Choose a file.", { exact: true })).toBeVisible();
  await expect(thereControls.widget.getByRole("button", { name: /^Download / })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Attachments 0", exact: true })).toBeVisible();
  // Back in the first workspace, its work item holds no file either.
  const hereFiles = page.waitForResponse(
    (response) => new URL(response.url()).pathname === `/api/v0/issues/${issue.id}/attachments`
  );
  await moveWithinApp(page, `/${workspace.slug}/browse/FILE-${issue.sequence_id}`);
  await expect(here.title).toHaveValue(issue.name);
  await hereFiles;
  await expect(page.getByRole("button", { name: "Attachments 0", exact: true })).toBeVisible();
  await files.finish();
});

test("F16 (page): a reconnect does not read the proven browse route again, so its detail host keeps the unsent comment and the upload's confirmation [local]", async ({
  imageWorld: world,
}, info) => {
  info.setTimeout(120_000);
  const { page, baseURL } = world;
  const files = await contentFilesPage(world, info);
  const { workspace, issue } = files;
  const route = `/api/v0/workspaces/${workspace.slug}/issues/FILE-${issue.sequence_id}`;
  const c = controls(page.locator("body"), issue.id);
  await page.goto(`${baseURL}/${workspace.slug}/browse/FILE-${issue.sequence_id}`);
  await expect(c.title).toHaveValue(issue.name);
  await upload(page, files, c, issue.id, textFile("kept.txt", "bytes confirmed before the network went away"));
  await expect(c.status("File uploaded.")).toBeVisible();
  await c.composer.click();
  await page.keyboard.type("Unsent through the outage");
  const title = await c.title.elementHandle();
  const lookups = files.calls("GET", route).length;

  // The network goes away and comes back: the browser announces it to the page.
  await page.context().setOffline(true);
  await page.waitForTimeout(300);
  await page.context().setOffline(false);
  await page.waitForTimeout(1500);
  // The route is not read again and its detail stays the same mount, with what it keeps.
  expect(files.calls("GET", route)).toHaveLength(lookups);
  expect(await title!.evaluate((node) => node.isConnected)).toBe(true);
  await expect(c.composer).toContainText("Unsent through the outage");
  await expect(c.status("File uploaded.")).toBeVisible();
  await files.finish();
});

test("F16 (page): another work item's peek above the detail keeps input of its own; a reopened peek and a revisited detail start empty [local]", async ({
  imageWorld: world,
}, info) => {
  info.setTimeout(120_000);
  const { api, page, baseURL } = world;
  const files = await contentFilesPage(world, info);
  const { owner, workspace, issue, project } = files;
  const sub = await createIssue(api, owner.token, project.id, { name: "Sub item", parent_id: issue.id });
  const detail = controls(page.locator("body"), issue.id);
  await page.goto(`${baseURL}/${workspace.slug}/browse/FILE-${issue.sequence_id}`);
  await expect(detail.title).toHaveValue(issue.name);
  await detail.composer.click();
  await page.keyboard.insertText("Kept by the detail");

  // The sub-item's peek is a second host above the first: neither sees the other's input.
  const subs = page.getByRole("button", { name: /^Sub-work items/ });
  if ((await subs.getAttribute("aria-expanded")) !== "true") await subs.click();
  const peekHost = page.locator("#full-screen-portal");
  const peek = controls(peekHost, sub.id);
  const openPeek = async () => {
    await page.locator(`#issue-${sub.id}`).click();
    await expect(peek.title).toHaveValue(sub.name);
  };
  await openPeek();
  await expect(peek.composer).toBeEmpty();
  await peek.composer.click();
  await page.keyboard.insertText("Kept by the peek");
  await expect(detail.composer).toHaveText("Kept by the detail");
  // Closing the peek ends its host; opening it again starts a new one.
  await peekHost.getByRole("button", { name: "Close the peek view", exact: true }).click();
  await expect(peek.title).toHaveCount(0);
  await openPeek();
  await expect(peek.composer).toBeEmpty();
  await expect(detail.composer).toHaveText("Kept by the detail");
  await peekHost.getByRole("button", { name: "Close the peek view", exact: true }).click();

  // Leaving the detail for another work item ends its host too: coming back starts empty.
  const subDetail = controls(page.locator("body"), sub.id);
  await moveWithinApp(page, `/${workspace.slug}/browse/FILE-${sub.sequence_id}`);
  await expect(subDetail.title).toHaveValue(sub.name);
  await expect(subDetail.composer).toBeEmpty();
  await moveWithinApp(page, `/${workspace.slug}/browse/FILE-${issue.sequence_id}`);
  await expect(detail.title).toHaveValue(issue.name);
  await expect(detail.composer).toBeEmpty();
  expect(files.calls("POST", /\/comments$/)).toHaveLength(0);
  await files.finish();
});

test("F16 (page): a project role that changes and changes back retires the member's retained input; as a guest the work item's files are read-only, and back as a member nothing revives [local]", async ({
  imageWorld: world,
}, info) => {
  info.setTimeout(120_000);
  const { api, page, baseURL } = world;
  const files = await contentFilesPage(world, info, "member");
  const { owner, other, workspace, project, issue } = files;
  // Guests see every work item of this project, so the guest still reads this one.
  expect(await changeProject(api, owner.token, project.id, { guest_view_all_features: true })).toMatchObject({
    status: 200,
  });
  await storeBody(world, owner.token, issue.id);
  // The owner's comment carries a file, which neither a member nor a guest that did not write it may remove.
  const ownerComment = await createIssueComment(api, owner.token, issue.id, "<p>The owner's note</p>");
  const ownerFile = textFile("owner-note.txt", "bytes of the owner's comment");
  const kind = { kind: "comment", id: ownerComment.id } as const;
  await storedFile(api, owner.token, "COMMENT_ATTACHMENT", kind, ownerFile.bytes, ownerFile.type, ownerFile.name);
  const bodies = await refuseBodies(page, issue.id);
  const host = await openHost(page, baseURL, workspace.slug, project.id, issue, "detail");
  const c = controls(host, issue.id);
  const ownerSection = host
    .locator(`[id="comment-${ownerComment.id}"]`)
    .getByRole("button", { name: "Attachments", exact: true });
  const ownerFileReadOnly = async () => {
    if ((await ownerSection.getAttribute("aria-expanded")) !== "true") await ownerSection.click();
    const section = ownerSection.locator("xpath=..");
    await expect(section.getByRole("button", { name: `Download ${ownerFile.name}`, exact: true })).toBeVisible();
    await expect(section.getByLabel("Choose or replace file", { exact: true })).toHaveCount(0);
    await expect(section.getByRole("button", { name: `Delete file ${ownerFile.name}`, exact: true })).toHaveCount(0);
  };
  await ownerFileReadOnly();
  const file = textFile("member-file.txt", "bytes the member attached");
  await upload(page, files, c, issue.id, file);
  await expect(c.status("File uploaded.")).toBeVisible();
  await expect(c.widget.getByRole("button", { name: `Delete file ${file.name}`, exact: true })).toBeVisible();
  await refusedEdit(page, host, c, bodies.saves);
  await c.composer.click();
  await page.keyboard.insertText("Unsent comment");
  const membership = await projectMembershipOf(api, owner.token, project.id, other.id);
  const roles = projectMemberWrites(api, project.id);
  // Every read of the workspace's projects, with when the page sent it.
  const projectsPath = `/api/v0/workspaces/${workspace.slug}/projects`;
  const directoryReads: { sent: number; request: Request }[] = [];
  page.on("request", (request) => {
    if (request.method() === "GET" && new URL(request.url()).pathname === projectsPath)
      directoryReads.push({ sent: Date.now(), request });
  });
  // The command palette reads the workspace's projects as it opens, unless SWR deduplicates a read begun in the
  // last 2 s: it is opened until a read sent after the role changed has answered, and the page knows the role then.
  const learnRole = async (changed: number) =>
    expect(async () => {
      await openCommands(page);
      await page.keyboard.press("Escape");
      const answers = await Promise.all(
        directoryReads
          .filter((read) => read.sent > changed)
          .map(async (read) => (await read.request.response())?.status())
      );
      expect(answers).toContain(200);
    }).toPass({ intervals: [2_100], timeout: 20_000 });

  // A guest: the same work item, its stored body and its files, none of them writable and none of the old input.
  expect(await roles.change(owner.token, membership, 5)).toEqual({ status: 200 });
  await learnRole(Date.now());
  // A guest reads the title as text and the body without an editor; a guest may still comment, from empty.
  await expect(c.document).toContainText("Stored body");
  await expect(c.document).not.toContainText("refused edit");
  await expect(c.title).toHaveCount(0);
  await expect(c.body).toHaveCount(0);
  await expect(c.composer).toBeEmpty();
  await expect(c.status("File uploaded.")).toHaveCount(0);
  await openFiles(c);
  await expect(c.widget.getByRole("button", { name: `Download ${file.name}`, exact: true })).toBeVisible();
  await expect(c.widget.getByLabel("Choose or replace file", { exact: true })).toHaveCount(0);
  await expect(c.widget.getByRole("button", { name: `Delete file ${file.name}`, exact: true })).toHaveCount(0);
  await ownerFileReadOnly();

  // A member again: the work item is writable once more, and nothing of the old input comes back.
  expect(await roles.change(owner.token, membership, 15)).toEqual({ status: 200 });
  await learnRole(Date.now());
  await expect(c.title).toHaveValue(issue.name);
  await expect(c.body).toBeVisible();
  await expectNoOldInput(c);
  await page.waitForTimeout(2500);
  expect(bodies.saves).toHaveLength(1);
  expect(files.calls("POST", `/api/v0/issues/${issue.id}/comments`)).toHaveLength(0);
  await files.finish(
    (failure) =>
      failure === `422 PATCH /api/v0/issues/${issue.id}` ||
      // A role change ends the member's content view, and with it the view's reads still in flight.
      (/ GET \/api\/v0\/assets\/[^/]+(\/content)?$/.test(failure) && !/^\d{3} /.test(failure))
  );
});

test("F16 (page): archived during the confirmation's re-read, the fresh owner is read-only and shows its saved body, not the refused edit it keeps [local]", async ({
  imageWorld: world,
}, info) => {
  info.setTimeout(120_000);
  const { api, page, baseURL } = world;
  const files = await contentFilesPage(world, info);
  const { owner, workspace, project, issue } = files;
  await updateIssue(api, owner.token, issue.id, { state_id: (await stateIds(api, owner.token, project.id)).Done });
  await storeBody(world, owner.token, issue.id);
  const bodies = await refuseBodies(page, issue.id);
  const host = await openHost(page, baseURL, workspace.slug, project.id, issue, "detail");
  const c = controls(host, issue.id);
  await refusedEdit(page, host, c, bodies.saves);

  const read = await holdNextRead(page, issue.id);
  await upload(page, files, c, issue.id, textFile("archived-owner.txt", "bytes confirmed while another tab archives"));
  await read.reached;
  expect(await issueArchived(api, owner.token, issue.id)).toMatchObject({ status: 200 });
  read.answer(undefined);
  await expect(c.status("Your unsaved input is preserved, but this work item is now read-only.")).toBeVisible();
  await expect(c.document).toContainText("Stored body");
  await expect(c.document).not.toContainText("refused edit");
  await expect(c.body).toHaveCount(0);
  await page.waitForTimeout(2500);
  expect(bodies.saves).toHaveLength(1);
  const stored = await issueRead(api, owner.token, issue.id);
  expect("issue" in stored && stored.issue.description_html).not.toContain("refused edit");
  await files.finish((failure) => failure === `422 PATCH /api/v0/issues/${issue.id}`);
});

test("F16 (page): an image's content grant answered as about to expire is renewed once as it expires, and the image stays shown [local]", async ({
  imageWorld: world,
}, info) => {
  info.setTimeout(120_000);
  const { page, baseURL } = world;
  const files = await contentFilesPage(world, info);
  const { owner, workspace, project, issue } = files;
  const image = await storeBody(world, owner.token, issue.id);
  const contentPath = `/api/v0/assets/${image.id}/content`;
  // Only the page's copy of the first grant says it ends early: nerve's own TTL is not what this changes or proves.
  let shortened = false;
  await page.route(
    (url) => url.pathname === contentPath,
    async (route) => {
      if (shortened) return route.fallback();
      shortened = true;
      const response = await route.fetch();
      const grant = (await response.json()) as { url: string; expires_at: string };
      await route.fulfill({ response, json: { ...grant, expires_at: new Date(Date.now() + 3000).toISOString() } });
    }
  );
  await openHost(page, baseURL, workspace.slug, project.id, issue, "detail");
  const shown = controls(page.locator("body"), issue.id).body.locator("img.image-component");
  await expectDecoded(shown);
  expect(files.calls("GET", contentPath)).toHaveLength(1);
  await expect.poll(() => files.calls("GET", contentPath).length, { timeout: 15_000 }).toBe(2);
  await expectDecoded(shown);
  // The renewed grant is nerve's own: nothing is read again before it ends.
  await page.waitForTimeout(5000);
  expect(files.calls("GET", contentPath)).toHaveLength(2);
  await files.finish();
});
