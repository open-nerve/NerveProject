/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { randomUUID } from "node:crypto";

import type { Locator, Page } from "@playwright/test";

import { createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { assetRows, imageHTML } from "../../fixtures/assert/storage";
import { contentFilesPage, copyFrom, dropFiles, pasteInto, pngFile, textFile } from "../../fixtures/content-files-page";
import { pageFileStory } from "../../fixtures/file-story";
import { createIssue, issueArchived, updateIssue } from "../../fixtures/issues";
import { expectDecoded } from "../../fixtures/profile-images-page";
import { stateIds } from "../../fixtures/states";
import { filePixel, storedFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";

const editorOf = (page: Page, issueID: string) =>
  page.locator(`[id="editor-container-${issueID}"] [contenteditable="true"]`);
/** The caret at the end of the editor, as a click and the keyboard put it there. */
async function caretAtEnd(page: Page, editor: Locator) {
  await editor.click();
  await page.keyboard.press("ControlOrMeta+End");
}
const imageIDs = (html: string) => [...html.matchAll(/src="([0-9a-f-]{36})"/g)].map((match) => match[1]!);

pageFileStory(
  "F11 (page): a copied image becomes the target's own server copy that Undo and Redo reuse; forged and unreadable sources and a refused target insert nothing; plain files are left to the browser's drop and to ProseMirror's paste",
  async (world, info) => {
    info.setTimeout(180_000);
    const { api, page, baseURL, db, uploadStatus } = world;
    const files = await contentFilesPage(world, info);
    const { owner, other, workspace, project, issue: target } = files;
    const source = await createIssue(api, owner.token, project.id, { name: "Copy source" });
    const original = await storedFile(
      api,
      owner.token,
      "ISSUE_DESCRIPTION",
      { kind: "issue", id: source.id },
      filePixel,
      "image/png",
      "original.png",
      uploadStatus
    );
    await updateIssue(api, owner.token, source.id, { description_html: `<p>Source text</p>${imageHTML(original.id)}` });
    const originalBefore = await assetRows(db, [original.id]);
    // An image in a workspace the owner cannot read.
    const elsewhere = await createWorkspace(api, other.token, { name: "Elsewhere", slug: slugFor(info, "elsewhere") });
    const hidden = await createProject(api, other.token, elsewhere.slug, { name: "Hidden", identifier: "HIDE" });
    const hiddenIssue = await createIssue(api, other.token, hidden.id, { name: "Hidden source" });
    const unreadable = await storedFile(
      api,
      other.token,
      "ISSUE_DESCRIPTION",
      { kind: "issue", id: hiddenIssue.id },
      filePixel,
      "image/png",
      "unreadable.png",
      uploadStatus
    );
    const saves = (issueID: string) =>
      files
        .calls("PATCH", `/api/v0/issues/${issueID}`)
        .map((request) => (request.postDataJSON() as { description_html?: string }).description_html)
        .filter((html): html is string => html !== undefined);
    const savedWith = (issueID: string, accept: (html: string) => boolean) =>
      page.waitForResponse(
        (r) =>
          r.request().method() === "PATCH" &&
          new URL(r.url()).pathname === `/api/v0/issues/${issueID}` &&
          accept((r.request().postDataJSON() as { description_html?: string }).description_html ?? ""),
        { timeout: 20_000 }
      );

    // The editor's own copy of the source body, as a clipboard carries it.
    await page.goto(`${baseURL}/${workspace.slug}/browse/FILE-${source.sequence_id}`);
    const sourceEditor = editorOf(page, source.id);
    await expectDecoded(sourceEditor.locator("img.image-component"));
    await sourceEditor.click();
    await page.keyboard.press("ControlOrMeta+A");
    const clipboard = await copyFrom(sourceEditor);
    expect(clipboard["text/nerve-editor-html"]).toContain(original.id);

    // Pasted into the target: one server copy into the target's own scope; only its new identity is saved.
    await page.goto(`${baseURL}/${workspace.slug}/browse/FILE-${target.sequence_id}`);
    const editor = editorOf(page, target.id);
    await caretAtEnd(page, editor);
    const duplicatePath = `/api/v0/assets/${original.id}/duplicate`;
    let saved = savedWith(target.id, (html) => imageIDs(html).length === 1);
    expect(await pasteInto(editor, clipboard)).toBe(true);
    const savedHTML = ((await saved).request().postDataJSON() as { description_html: string }).description_html;
    // What the target saves is never the source's identity, but its own copy, of which exactly one is made.
    expect(imageIDs(savedHTML)).not.toContain(original.id);
    const duplicates = files.calls("POST", duplicatePath);
    expect(duplicates).toHaveLength(1);
    const copied = (await duplicates[0]!.response())!;
    expect(copied.status()).toBe(201);
    expect(copied.request().postDataJSON()).toEqual({
      purpose: "ISSUE_DESCRIPTION",
      scope: { kind: "issue", id: target.id },
    });
    const copy = ((await copied.json()) as { id: string }).id;
    expect(imageIDs(savedHTML)).toEqual([copy]);
    await expectDecoded(editor.locator("img.image-component"));
    await files.persisted(copy, pngFile("original.png"));
    expect(await assetRows(db, [original.id])).toEqual(originalBefore);

    // Undo and Redo move the same identity in and out: no second copy, no deletion.
    saved = savedWith(target.id, (html) => imageIDs(html).length === 0);
    await page.keyboard.press("ControlOrMeta+A");
    await page.keyboard.press("Delete");
    await saved;
    saved = savedWith(target.id, (html) => imageIDs(html).join() === copy);
    await page.keyboard.press("ControlOrMeta+Z");
    await expectDecoded(editor.locator("img.image-component"));
    await saved;
    saved = savedWith(target.id, (html) => imageIDs(html).length === 0);
    await page.keyboard.press("ControlOrMeta+Shift+Z");
    await expect(editor.locator("img.image-component")).toHaveCount(0);
    await saved;
    saved = savedWith(target.id, (html) => imageIDs(html).join() === copy);
    await page.keyboard.press("ControlOrMeta+Z");
    await saved;
    expect(files.calls("POST", /\/duplicate$/)).toHaveLength(1);
    expect(files.calls("DELETE", /^\/api\/v0\/assets\//)).toHaveLength(0);
    expect(await assetRows(db, [copy])).toMatchObject([{ is_deleted: false, issue_id: target.id }]);

    // A forged identity and one from a workspace the owner cannot read meet the same refusal; nothing is inserted.
    const refuse = async (id: string) => {
      await caretAtEnd(page, editor);
      const refused = page.waitForResponse(
        (r) => r.request().method() === "POST" && new URL(r.url()).pathname === `/api/v0/assets/${id}/duplicate`,
        { timeout: 15_000 }
      );
      expect(
        await pasteInto(editor, { "text/nerve-editor-html": `<image-component src="${id}"></image-component>` })
      ).toBe(true);
      const answer = await refused;
      await expect(editor.getByRole("alert")).toBeVisible();
      await page.keyboard.press("ControlOrMeta+Z");
      await expect(editor.getByRole("alert")).toHaveCount(0);
      return answer;
    };
    const refusals = [await refuse(randomUUID()), await refuse(unreadable.id)];
    expect(refusals.map((r) => r.status())).toEqual([404, 404]);
    expect(await refusals[1]!.json()).toEqual(await refusals[0]!.json());
    expect(saves(target.id).some((html) => html.includes(unreadable.id))).toBe(false);
    expect(await assetRows(db, [unreadable.id])).toMatchObject([{ issue_id: hiddenIssue.id, is_deleted: false }]);

    // A plain file is not the editor's to take. Its drop stays the browser's; its paste goes to ProseMirror, which
    // pastes the text that came with it (a file manager puts the file's name there), uploads nothing and says nothing.
    const note = textFile("plain.txt", "not an image");
    expect(await dropFiles(editor, [note])).toBe(false);
    saved = savedWith(target.id, (html) => html.includes("plain.txt"));
    await pasteInto(editor, { "text/plain": "plain.txt" }, [note]);
    await saved;
    await expect(editor).toContainText("plain.txt");
    await expect(editor.getByRole("alert")).toHaveCount(0);
    expect(files.calls("POST", "/api/v0/assets")).toHaveLength(0);
    await files.quiet();

    // The target is archived elsewhere: the server refuses the copy, and nothing of it is saved.
    const done = (await stateIds(api, owner.token, project.id)).Done;
    const archived = await createIssue(api, owner.token, project.id, { name: "Archived target", state_id: done });
    await page.goto(`${baseURL}/${workspace.slug}/browse/FILE-${archived.sequence_id}`);
    const archivedEditor = editorOf(page, archived.id);
    await caretAtEnd(page, archivedEditor);
    expect(await issueArchived(api, owner.token, archived.id)).toMatchObject({ status: 200 });
    const targetRefused = page.waitForResponse(
      (r) => r.request().method() === "POST" && new URL(r.url()).pathname === duplicatePath,
      { timeout: 15_000 }
    );
    await pasteInto(archivedEditor, clipboard);
    expect((await targetRefused).status()).toBeGreaterThanOrEqual(400);
    await expect(archivedEditor.getByRole("alert")).toBeVisible();
    expect(saves(archived.id).some((html) => imageIDs(html).length > 0)).toBe(false);
    expect(files.calls("POST", /\/duplicate$/)).toHaveLength(4);
    await files.finish((failure) => / POST \/api\/v0\/assets\/[^/]+\/duplicate$/.test(failure));
  }
);
