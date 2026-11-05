/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Locator } from "@playwright/test";

import { addProjectMembers, inviteAndAccept } from "../../fixtures/api";
import { named } from "../../fixtures/auth";
import { assetRows, imageHTML } from "../../fixtures/assert/storage";
import { contentFilesPage, dropFiles, pngFile } from "../../fixtures/content-files-page";
import { deferred } from "../../fixtures/deferred";
import { pageFileStory } from "../../fixtures/file-story";
import { descriptionPage, issueRead, updateIssue } from "../../fixtures/issues";
import { expectDecoded, redPixel } from "../../fixtures/profile-images-page";
import { filePixel, storedFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";

const imageIDs = (html: string) => [...html.matchAll(/src="([0-9a-f-]{36})"/g)].map((match) => match[1]!);
type Saved = { description_html?: string; skip_activity?: boolean };

pageFileStory(
  "F12 (page): a stored image gains only its block identity once; replacing and reordering save ordinary versions by identity alone; the history shows a removed image read-only",
  async (world, info) => {
    info.setTimeout(180_000);
    const { api, page, baseURL, db, uploadStatus } = world;
    const files = await contentFilesPage(world, info);
    const { owner, other, workspace, project, issue } = files;
    // Another member wrote the stored body, so its version stays apart from the owner's merged edits.
    await named(api, other.token, { display_name: "Ada" });
    await inviteAndAccept(api, owner.token, workspace.slug, other, 15);
    await addProjectMembers(api, owner.token, project.id, [{ member_id: other.id, role: 15 }]);
    const first = await storedFile(
      api,
      other.token,
      "ISSUE_DESCRIPTION",
      { kind: "issue", id: issue.id },
      filePixel,
      "image/png",
      "first.png",
      uploadStatus
    );
    // Stored before block identities existed: an image without its id.
    await updateIssue(api, other.token, issue.id, {
      description_html: `<p data-id="base">Original body</p>${imageHTML(first.id)}`,
    });
    expect(imageHTML(first.id)).not.toMatch(/\sid=/);
    const versions = async () => (await descriptionPage(api, owner.token, issue.id, undefined, 100)).data.length;
    const versionsBefore = await versions();
    const path = `/api/v0/issues/${issue.id}`;
    const sent: Saved[] = [];
    page.on("request", (request) => {
      if (request.method() === "PATCH" && new URL(request.url()).pathname === path)
        sent.push(request.postDataJSON() as Saved);
    });
    const savedWhere = (accept: (body: Saved) => boolean) =>
      page.waitForResponse(
        (r) =>
          r.request().method() === "PATCH" &&
          new URL(r.url()).pathname === path &&
          accept(r.request().postDataJSON() as Saved),
        { timeout: 20_000 }
      );

    // Opening it writes the block identity once, as a migration that records no activity.
    const migrated = savedWhere((body) => body.skip_activity === true);
    await page.goto(`${baseURL}/${workspace.slug}/browse/FILE-${issue.sequence_id}`);
    const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
    await expectDecoded(editor.locator("img.image-component"));
    expect((await migrated).status()).toBe(200);
    const migration = (await migrated).request().postDataJSON() as Required<Saved>;
    expect(imageIDs(migration.description_html)).toEqual([first.id]);
    expect(migration.description_html).toMatch(/<image-component[^>]*\bid="[0-9a-f-]{36}"/);
    await page.waitForTimeout(3000);
    expect(sent).toHaveLength(1);
    expect(await versions()).toBe(versionsBefore);

    // Replacing it: the old image leaves the body, the new one is saved by identity, and the old asset is kept.
    await editor.locator("img.image-component").click();
    await page.keyboard.press("Delete");
    await expect(editor.locator("img.image-component")).toHaveCount(0);
    const replacement = pngFile("replacement.png", redPixel);
    let saved = savedWhere((body) => imageIDs(body.description_html ?? "").length === 1 && !body.skip_activity);
    expect(await dropFiles(editor.locator("p").last(), [replacement])).toBe(true);
    const replacementID = await files.transfer(0, "ISSUE_DESCRIPTION", { kind: "issue", id: issue.id }, replacement);
    await expectDecoded(editor.locator("img.image-component"));
    let body = (await saved).request().postDataJSON() as Saved;
    expect(imageIDs(body.description_html!)).toEqual([replacementID]);
    expect(body).not.toHaveProperty("skip_activity");
    await files.persisted(replacementID, replacement);
    expect(await assetRows(db, [first.id])).toMatchObject([{ is_deleted: false, issue_id: issue.id }]);
    expect(files.calls("DELETE", /^\/api\/v0\/assets\//)).toHaveLength(0);

    // A second image, then the editor's own move puts it above the first: the same identities in the new order, no copy.
    const second = pngFile("second.png");
    saved = savedWhere((patch) => imageIDs(patch.description_html ?? "").length === 2);
    expect(await dropFiles(editor.locator("p").last(), [second])).toBe(true);
    const secondID = await files.transfer(1, "ISSUE_DESCRIPTION", { kind: "issue", id: issue.id }, second);
    body = (await saved).request().postDataJSON() as Saved;
    const order = imageIDs(body.description_html!);
    expect(order.toSorted()).toEqual([replacementID, secondID].toSorted());
    await expect(editor.locator("img.image-component")).toHaveCount(2);
    const reversed = order.toReversed();
    saved = savedWhere((patch) => imageIDs(patch.description_html ?? "").join() === reversed.join());
    await moveLowerImageAbove(editor);
    body = (await saved).request().postDataJSON() as Saved;
    expect(body).not.toHaveProperty("skip_activity");
    expect(files.calls("POST", /\/duplicate$/)).toHaveLength(0);
    const read = await issueRead(api, owner.token, issue.id);
    expect(imageIDs("issue" in read ? (read.issue.description_html ?? "") : "")).toEqual(reversed);
    expect(await versions()).toBe(versionsBefore + 1);

    // The other member's version shows the image the body no longer holds, read-only: no picker, no write.
    const writes = sent.length;
    const reads = files.calls("GET", `/api/v0/assets/${first.id}/content`).length;
    await page.getByRole("button", { name: /^Last edited by/ }).click();
    await page.getByRole("menuitem", { name: /\bAda\b/ }).press("Enter");
    const history = page.getByRole("dialog", { name: "Description history", exact: true });
    await expectDecoded(history.locator("img.image-component").first());
    expect(files.calls("GET", `/api/v0/assets/${first.id}/content`).length).toBeGreaterThan(reads);
    await expect(history.locator('input[type="file"]')).toHaveCount(0);
    await page.keyboard.press("Escape");
    await expect(history).toHaveCount(0);
    expect(sent).toHaveLength(writes);
    await files.finish();
  }
);

pageFileStory(
  "F12 (page): another image chosen for a placeholder whose first image is still uploading replaces it; only the chosen one completes and is saved, and the first one's end shows nowhere",
  async (world, info) => {
    info.setTimeout(120_000);
    const { page, baseURL, db } = world;
    const files = await contentFilesPage(world, info);
    const { workspace, issue } = files;
    const path = `/api/v0/issues/${issue.id}`;
    // Every raw upload waits here until the story lets it go on.
    const held: (() => void)[] = [];
    await page.route(
      () => true,
      async (route) => {
        if (route.request().method() !== "PUT") return route.fallback();
        const gate = deferred();
        held.push(gate.resolve);
        await gate.promise;
        // A request the page has dropped sends nothing on.
        await route.continue().catch(() => undefined);
      }
    );
    await page.goto(`${baseURL}/${workspace.slug}/browse/FILE-${issue.sequence_id}`);
    const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
    await expect(editor).toBeVisible();
    const first = pngFile("first-choice.png");
    const second = pngFile("second-choice.png", redPixel);
    expect(await dropFiles(editor.locator("p").last(), [first])).toBe(true);
    const placeholder = editor.locator(".image-upload-component");
    await expect.poll(() => held.length).toBe(1);
    await expect(placeholder.getByText("Uploading image", { exact: true })).toBeVisible();

    // The person chooses another image for the same placeholder while the first is still on its way.
    const chooser = page.waitForEvent("filechooser");
    // The visible button, not its hidden file input of the same name, opens the native chooser.
    await placeholder.locator("button", { hasText: "Choose images" }).click();
    await (await chooser).setFiles({ name: second.name, mimeType: second.type, buffer: second.bytes });
    await expect.poll(() => held.length).toBe(2);
    await expect(placeholder.getByText("Uploading image", { exact: true })).toBeVisible();
    // The first transfer ended with its own generation: its end reaches nothing the second one shows.
    await page.waitForTimeout(1000);
    await expect(placeholder.getByRole("alert")).toHaveCount(0);
    await expect(placeholder.getByRole("button", { name: "Retry", exact: true })).toHaveCount(0);
    const prepares = files.calls("POST", "/api/v0/assets");
    expect(prepares.map((request) => (request.postDataJSON() as { name: string }).name)).toEqual([
      first.name,
      second.name,
    ]);
    const [firstID, secondID] = await Promise.all(
      prepares.map(
        async (request) => ((await (await request.response())!.json()) as { asset: { id: string } }).asset.id
      )
    );

    // Only the second completes; the body saves its identity alone.
    const saved = page.waitForResponse(
      (r) =>
        r.request().method() === "PATCH" &&
        new URL(r.url()).pathname === path &&
        imageIDs((r.request().postDataJSON() as Saved).description_html ?? "").length === 1,
      { timeout: 20_000 }
    );
    held[1]!();
    await expectDecoded(editor.locator("img.image-component"));
    expect(imageIDs(((await saved).request().postDataJSON() as Saved).description_html!)).toEqual([secondID]);
    await files.persisted(secondID!, second);
    held[0]!();
    await page.waitForTimeout(1000);
    expect(files.calls("POST", `/api/v0/assets/${secondID}/complete`)).toHaveLength(1);
    expect(files.calls("POST", `/api/v0/assets/${firstID}/complete`)).toHaveLength(0);
    expect(await assetRows(db, [firstID!])).toMatchObject([{ is_uploaded: false }]);
    await expect(editor.locator("img.image-component")).toHaveCount(1);
    await files.finish();
  }
);

/**
 * The editor's own move of the lower image above the upper one, in one transaction, as its drop handler performs a
 * dragged node's move. Playwright's drag emulation does not start ProseMirror's node drag on this page, for a
 * paragraph either, so the gesture itself is not driven here; what the page saves, reads and stores is.
 */
async function moveLowerImageAbove(editor: Locator) {
  await editor.evaluate((dom) => {
    type Node = { type: { name: string }; nodeSize: number };
    type View = {
      state: {
        doc: { descendants(visit: (node: Node, pos: number) => void): void };
        tr: { delete(from: number, to: number): { insert(pos: number, node: Node): unknown } };
      };
      dispatch(tr: unknown): void;
    };
    const view = (dom as unknown as { editor: { view: View } }).editor.view;
    const images: { node: Node; pos: number }[] = [];
    view.state.doc.descendants((node, pos) => {
      if (node.type.name === "imageComponent") images.push({ node, pos });
    });
    const [upper, lower] = images;
    if (!upper || !lower) throw new Error("Two images are needed");
    view.dispatch(view.state.tr.delete(lower.pos, lower.pos + lower.node.nodeSize).insert(upper.pos, lower.node));
  });
}
