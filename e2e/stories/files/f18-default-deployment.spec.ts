/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createHash } from "node:crypto";
import { readFile as readDiskFile } from "node:fs/promises";
import path from "node:path";

import { expect, type Locator, type Page } from "@playwright/test";

import { createProject, createWorkspace } from "../../fixtures/api";
import { bearer, login, password } from "../../fixtures/auth";
import { submitSignIn } from "../../fixtures/auth-pages";
import { watchPage } from "../../fixtures/browser";
import {
  chooseAttachment,
  downloadedExactly,
  pasteInto,
  pngFile,
  textFile,
  type ContentFile,
} from "../../fixtures/content-files-page";
import { deploymentTest as test, loggedConfiguration } from "../../fixtures/deployment";
import { modalDialog } from "../../fixtures/dialogs";
import { createIssue } from "../../fixtures/issues";
import { expectDecoded, redPixel } from "../../fixtures/profile-images-page";
import { answerTo, bodiesSentTo } from "../../fixtures/settings-pages";
import { filePixel, readFile } from "../../fixtures/storage";

const sha256 = (bytes: Uint8Array) => createHash("sha256").update(bytes).digest("hex");
const imageIDs = (html: string) => [...html.matchAll(/src="([0-9a-f-]{36})"/g)].map((match) => match[1]!);

/** The image dialog's chooser takes the file and the dialog shows it decoded before anything is sent. */
async function selectImage(page: Page, file: ContentFile) {
  await page
    .getByLabel("Choose or replace image", { exact: true })
    .setInputFiles({ name: file.name, mimeType: file.type, buffer: file.bytes });
  await expectDecoded(page.getByAltText("Selected image preview", { exact: true }));
}

test("F18 (page): a default production deployment keeps every kind of file through a restart", async ({
  deployment,
  page,
}, info) => {
  info.setTimeout(240_000);
  const { db, email } = deployment;
  let { api, nerve } = deployment.current;
  const baseURL = nerve.baseURL;
  // Production, the base configuration's files and limits: only the deployment's own variables differ.
  expect(loggedConfiguration(deployment.current.log)).toMatchObject({
    env: "prod",
    auth: { signup_enabled: false },
    database: { auto_migrate: false },
    clock: { offset: 0 },
    files: {
      driver: "local",
      transfer_rate: { per_minute: 60, burst: 20 },
      size_limit: 5242880,
    },
  });

  // Hosts come from the owner endpoints as the first account; the browser makes every file.
  const tokens = await login(api, email);
  const token = tokens.access_token;
  expect(
    (await api.PATCH("/api/v0/me/profile", { headers: bearer(token), body: { is_onboarded: true } })).response.status
  ).toBe(200);
  const workspace = await createWorkspace(api, token, { name: "Deployed", slug: "deployed" });
  const project = await createProject(api, token, workspace.slug, { name: "Deployed", identifier: "DEPL", network: 0 });
  const issue = await createIssue(api, token, project.id, { name: "Deployed files" });
  const watch = await watchPage(page);
  await page.goto(baseURL);
  expect(await submitSignIn(page, email, password)).toBe(200);
  await expect(page).not.toHaveURL(`${baseURL}/`);
  // The person moves on once a page has finished loading: a navigation never cuts off the last page's reads.
  const visit = async (address: string) => {
    await page.waitForLoadState("networkidle");
    await page.goto(address);
  };

  // Avatar.
  const avatar = pngFile("avatar.png", filePixel);
  await visit(`${baseURL}/settings/profile/general`);
  await page.getByRole("button", { name: "Change avatar", exact: true }).click();
  await expect(modalDialog(page)).toHaveAccessibleName("Change avatar");
  await selectImage(page, avatar);
  const me = await answerTo(page, "PATCH", "/api/v0/me", () =>
    page.getByRole("button", { name: "Upload and save", exact: true }).click()
  );
  expect(me.status()).toBe(200);
  await expect(modalDialog(page)).toHaveCount(0);
  const avatarControl = () => page.getByRole("button", { name: "Change avatar", exact: true }).locator("img");
  await expectDecoded(avatarControl());

  // Workspace icon.
  const logo = pngFile("logo.png", redPixel);
  await visit(`${baseURL}/${workspace.slug}/settings`);
  await page.getByRole("button", { name: "Upload workspace icon", exact: true }).first().click();
  await selectImage(page, logo);
  const iconSaved = await answerTo(page, "PATCH", `/api/v0/workspaces/${workspace.slug}`, () =>
    page.getByRole("button", { name: "Upload and save", exact: true }).click()
  );
  expect(iconSaved.status()).toBe(200);
  await expect(modalDialog(page)).toHaveCount(0);
  const switcherIcon = () => page.getByRole("button", { name: "Open workspace switcher" }).locator("img");
  await expectDecoded(switcherIcon());

  // Project cover: a preset first, then an upload.
  const projectPath = `/api/v0/projects/${project.id}`;
  await visit(`${baseURL}/${workspace.slug}/settings/projects/${project.id}`);
  await page.getByRole("button", { name: "Change cover", exact: true }).click();
  await page.getByRole("button", { name: "Cover 29", exact: true }).click();
  const preset = await answerTo(page, "PATCH", projectPath, () =>
    page.getByRole("button", { name: "Update project", exact: true }).click()
  );
  expect(await preset.json()).toMatchObject({ cover_builtin_id: "cover-29", cover_image_asset_id: null });
  const cover = pngFile("cover.png", filePixel);
  await page.getByRole("button", { name: "Change cover", exact: true }).click();
  await page.getByRole("button", { name: "Upload image", exact: true }).click();
  await selectImage(page, cover);
  await page.getByRole("button", { name: "Upload and save", exact: true }).click();
  await expect(modalDialog(page)).toHaveCount(0);
  const uploadedCover = await answerTo(page, "PATCH", projectPath, () =>
    page.getByRole("button", { name: "Update project", exact: true }).click()
  );
  expect(uploadedCover.status()).toBe(200);
  expect(await uploadedCover.json()).toMatchObject({
    cover_builtin_id: null,
    cover_image_asset_id: expect.any(String),
  });

  // Attachment, downloaded by the browser.
  const issuePage = `${baseURL}/${workspace.slug}/browse/DEPL-${issue.sequence_id}`;
  const report = textFile("report.txt", "bytes of a file attached on a default deployment");
  await visit(issuePage);
  await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue(issue.name);
  const attachments = (count: number) => page.getByRole("button", { name: `Attachments ${count}`, exact: true });
  const widget = page.getByRole("button", { name: /^Attachments \d+$/ }).locator("xpath=../following-sibling::*[1]");
  await attachments(0).click();
  await chooseAttachment(page, widget, report);
  await expect(page.getByRole("status").filter({ hasText: "File uploaded." })).toBeVisible();
  await expect(attachments(1)).toBeVisible();
  const download = (where: Locator, file: ContentFile) =>
    downloadedExactly(page, where.getByRole("button", { name: `Download ${file.name}`, exact: true }), file);
  await download(widget, report);

  // A pasted image in the description.
  const picture = pngFile("pasted.png", redPixel);
  const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
  await editor.click();
  await page.keyboard.press("ControlOrMeta+End");
  const saves = (await bodiesSentTo(page, "PATCH", `/api/v0/issues/${issue.id}`)) as { description_html?: string }[];
  expect(await pasteInto(editor, {}, [picture])).toBe(true);
  await expectDecoded(editor.locator("img.image-component"));
  await expect
    .poll(() => saves.map((body) => imageIDs(body.description_html ?? "").length), { timeout: 20_000 })
    .toContain(1);

  // A comment with a file.
  const note = textFile("comment-note.txt", "bytes of a comment's file on a default deployment");
  const commentEditor = page.locator(`[id^="editor-container-add_comment_${issue.id}_"] [contenteditable="true"]`);
  const composer = page.getByRole("region", { name: "Files for this comment", exact: true });
  await commentEditor.click();
  await page.keyboard.insertText("A comment with a file");
  await chooseAttachment(page, composer, note);
  await expect(composer.getByText(note.name, { exact: true })).toBeVisible();
  const commented = page.waitForResponse(
    (r) => r.request().method() === "POST" && new URL(r.url()).pathname === `/api/v0/issues/${issue.id}/comments`
  );
  await page.getByRole("button", { name: "Comment", exact: true }).click();
  expect((await commented).status()).toBe(201);
  const comment = (await (await commented).json()) as { id: string };
  const commentFiles = page.locator(`[id="comment-${comment.id}"]`);
  await commentFiles.getByRole("button", { name: "Attachments", exact: true }).click();
  await download(commentFiles, note);

  // What the server holds for each: the bound row, the object under the exclusive root, an authorized read.
  const [user] = await db.query<{ avatar: string }>("SELECT avatar_asset_id AS avatar FROM users WHERE email=$1", [
    email,
  ]);
  const [space] = await db.query<{ logo: string }>("SELECT logo_asset_id AS logo FROM workspaces WHERE id=$1", [
    workspace.id,
  ]);
  const [projected] = await db.query<{ cover: string }>(
    "SELECT cover_image_asset_id AS cover FROM projects WHERE id=$1",
    [project.id]
  );
  const [described_] = await db.query<{ html: string }>("SELECT description_html AS html FROM issues WHERE id=$1", [
    issue.id,
  ]);
  const files = new Map<string, ContentFile>([
    [user!.avatar, avatar],
    [space!.logo, logo],
    [projected!.cover, cover],
    [imageIDs(described_!.html)[0]!, picture],
  ]);
  for (const row of await db.query<{ id: string; name: string }>(
    "SELECT id, attributes->>'name' AS name FROM file_assets WHERE issue_id=$1 AND entity_type IN ('ISSUE_ATTACHMENT','COMMENT_ATTACHMENT')",
    [issue.id]
  ))
    files.set(row.id, row.name === report.name ? report : note);
  expect(files.size).toBe(6);
  const stored = async () => {
    const rows = await db.query<{ id: string; key: string; sha: string; bound: boolean; deleted: boolean }>(
      `SELECT id, storage_metadata->'final'->>'key' AS key, storage_metadata->'final'->>'sha256' AS sha,
      bound_at IS NOT NULL AS bound, is_deleted AS deleted FROM file_assets WHERE id = ANY($1::uuid[]) ORDER BY id`,
      [[...files.keys()]]
    );
    expect(rows).toHaveLength(files.size);
    const root = deployment.variables.NERVE_FILES__LOCAL__ROOT!;
    await Promise.all(
      rows.map(async (row) => {
        const file = files.get(row.id)!;
        expect(row).toMatchObject({ sha: sha256(file.bytes), bound: true, deleted: false });
        expect(await readDiskFile(path.join(root, "objects", row.key, "data"))).toEqual(file.bytes);
        expect(await readFile(api, token, row.id)).toEqual(file.bytes);
      })
    );
    return rows;
  };
  const before = await stored();

  // The process restarts on the same address, database, key and storage root, while no page of it is open; the
  // browser's session continues.
  await visit("about:blank");
  ({ api, nerve } = await deployment.restart());
  expect(nerve.baseURL).toBe(baseURL);
  expect(await stored()).toEqual(before);
  await visit(`${baseURL}/settings/profile/general`);
  await expectDecoded(avatarControl());
  await visit(`${baseURL}/${workspace.slug}/projects`);
  await expectDecoded(switcherIcon());
  await visit(`${baseURL}/${workspace.slug}/settings/projects/${project.id}`);
  await expectDecoded(page.locator(`img[src*="/static/local/"]`).first());
  await visit(issuePage);
  await expectDecoded(editor.locator("img.image-component"));
  await attachments(1).click();
  await download(widget, report);
  await commentFiles.getByRole("button", { name: "Attachments", exact: true }).click();
  await download(commentFiles, note);

  expect(watch.oldApiRequests).toEqual([]);
  expect(watch.cspViolations).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  expect(watch.apiFailures).toEqual([]);
  await info.attach("deployment", {
    body: JSON.stringify({
      variables: Object.keys(deployment.variables),
      fixture: ["NERVE_DATABASE__URL", "NERVE_SERVER__ADDR", "NERVE_SERVER__ADDR_FILE"],
      files: before.map(({ id, key, sha }) => ({ id, key, sha, name: files.get(id)!.name })),
      apiRequests: watch.apiRequests.length,
    }),
    contentType: "application/json",
  });
});
