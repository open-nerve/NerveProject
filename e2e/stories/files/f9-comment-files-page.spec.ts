/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { addProjectMembers, inviteAndAccept } from "../../fixtures/api";
import { assetRows } from "../../fixtures/assert/storage";
import { bearer } from "../../fixtures/auth";
import { chooseAttachment as choose, contentFilesPage, textFile as text } from "../../fixtures/content-files-page";
import { modalDialog } from "../../fixtures/dialogs";
import { pageFileStory } from "../../fixtures/file-story";
import { bodiesSentTo } from "../../fixtures/settings-pages";
import { readFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";

pageFileStory(
  "F9 (page): two comments keep private preparations apart, a refused send keeps text and files for an explicit resend, and an unknown answer is checked, never resent",
  async (world, info) => {
    info.setTimeout(180_000);
    const { api, page, baseURL, db } = world;
    const files = await contentFilesPage(world, info);
    const { owner, other, workspace, project, issue } = files;
    await inviteAndAccept(api, owner.token, workspace.slug, other, 15);
    await addProjectMembers(api, owner.token, project.id, [{ member_id: other.id, role: 15 }]);
    const visibleTo = async (id: string) =>
      (
        await api.GET("/api/v0/assets/{id}/content", {
          params: { path: { id } },
          headers: bearer(other.token),
        })
      ).response.status;
    const scope = { kind: "new_comment", id: issue.id } as const;
    const path = `/api/v0/issues/${issue.id}/comments`;
    const sent = await bodiesSentTo(page, "POST", path);

    await page.goto(`${baseURL}/${workspace.slug}/browse/FILE-${issue.sequence_id}`);
    const editor = page.locator(`[id^="editor-container-add_comment_${issue.id}_"] [contenteditable="true"]`);
    const submit = page.getByRole("button", { name: "Comment", exact: true });
    const composer = page.getByRole("region", { name: "Files for this comment", exact: true });
    await expect(editor).toBeVisible();
    const write = async (value: string) => {
      await editor.click();
      await editor.press("ControlOrMeta+A");
      await page.keyboard.insertText(value);
      await expect(editor).toHaveText(value);
    };
    const bound = async (id: string) =>
      (await db.query<{ comment_id: string | null }>("SELECT comment_id FROM file_assets WHERE id=$1", [id]))[0]
        ?.comment_id;

    // A: the preparation stays private and unbound until the one create binds exactly it.
    const first = text("first-comment.txt", "bytes of the first comment's file");
    await write("First comment with a file");
    await choose(page, composer, first);
    const firstID = await files.transfer(0, "COMMENT_ATTACHMENT", scope, first);
    await expect(composer.getByText(first.name, { exact: true })).toBeVisible();
    await expect(composer.getByRole("button", { name: `Download ${first.name}`, exact: true })).toHaveCount(0);
    await files.persisted(firstID, first, false);
    expect(await visibleTo(firstID)).toBe(404);
    const createdA = page.waitForResponse((r) => r.request().method() === "POST" && new URL(r.url()).pathname === path);
    await submit.click();
    expect((await createdA).status()).toBe(201);
    const commentA = await (await createdA).json();
    expect(sent.at(-1)).toEqual({
      comment_html: expect.stringMatching(/First comment with a file/),
      attachment_ids: [firstID],
    });
    await expect(editor).toBeEmpty();
    await expect(composer.getByText(first.name, { exact: true })).toHaveCount(0);
    expect(await bound(firstID)).toBe(commentA.id);
    expect(await visibleTo(firstID)).toBe(200);
    expect(await readFile(api, other.token, firstID)).toEqual(first.bytes);

    // B: a refusal keeps the text and the preparation; nothing is sent again until the person sends it.
    const second = text("second-comment.txt", "bytes of the second comment's file");
    await write("Second comment after a refusal");
    await choose(page, composer, second);
    const secondID = await files.transfer(1, "COMMENT_ATTACHMENT", scope, second);
    let refusals = 0;
    await page.route(
      (url) => url.pathname === path,
      async (route) => {
        if (route.request().method() !== "POST" || refusals > 0) return route.fallback();
        refusals++;
        // This route answers before the recorder below it sees the request.
        sent.push(route.request().postDataJSON());
        await route.fulfill({
          status: 503,
          contentType: "application/problem+json",
          body: JSON.stringify({ type: "about:blank", title: "Busy", status: 503, code: "server_busy" }),
        });
      }
    );
    await submit.click();
    await expect(page.getByRole("alert").filter({ hasText: /busy|try again/i })).toBeVisible();
    await expect(editor).toHaveText("Second comment after a refusal");
    await expect(composer.getByText(second.name, { exact: true })).toBeVisible();
    expect(sent).toHaveLength(2);
    expect(await bound(secondID)).toBeNull();
    const createdB = page.waitForResponse((r) => r.request().method() === "POST" && new URL(r.url()).pathname === path);
    await submit.click();
    expect((await createdB).status()).toBe(201);
    const commentB = await (await createdB).json();
    expect(sent).toHaveLength(3);
    expect(sent[1]).toEqual(sent[2]);
    expect(sent[2]).toEqual({
      comment_html: expect.stringMatching(/Second comment after a refusal/),
      attachment_ids: [secondID],
    });
    expect(await bound(secondID)).toBe(commentB.id);
    expect(await bound(firstID)).toBe(commentA.id);

    // C: the answer is lost after the server created it; the page reads, and never sends it by itself.
    const third = text("third-comment.txt", "bytes of the third comment's file");
    await write("Third comment whose answer is lost");
    await choose(page, composer, third);
    const thirdID = await files.transfer(2, "COMMENT_ATTACHMENT", scope, third);
    await page.route(
      (url) => url.pathname === path,
      async (route) => {
        if (route.request().method() !== "POST") return route.fallback();
        sent.push(route.request().postDataJSON());
        await route.fetch();
        await route.abort("failed");
      },
      { times: 1 }
    );
    await submit.click();
    const unknown = page.getByRole("status").filter({ hasText: "The comment may have been saved." });
    await expect(unknown).toBeVisible();
    await expect(submit).toBeDisabled();
    await expect(editor).toHaveText("Third comment whose answer is lost");
    const reads = files.calls("GET", path).length;
    await unknown.getByRole("button", { name: "Read comments to check", exact: true }).click();
    const allow = unknown.getByRole("button", { name: "I checked; allow another submission", exact: true });
    await expect(allow).toBeVisible();
    expect(files.calls("GET", path).length).toBeGreaterThan(reads);
    // The read shows the comment the lost answer created, and the file that comment took stays unresolved here.
    await expect(page.getByText("Third comment whose answer is lost", { exact: true })).toHaveCount(2);
    await expect(composer.getByText(/^Unverified selected file 1\b/)).toBeVisible();
    await expect(composer.getByRole("button", { name: "Remove selection", exact: true })).toBeVisible();
    await allow.click();
    await expect(submit).toBeDisabled();
    expect(sent).toHaveLength(4);
    const [commentC] = await db.query<{ id: string }>(
      "SELECT id FROM issue_comments WHERE issue_id=$1 AND comment_html LIKE '%answer is lost%'",
      [issue.id]
    );
    expect(await bound(thirdID)).toBe(commentC!.id);

    // Each created comment shows only its own file, read and downloaded through its own list.
    for (const [comment, file] of [
      [commentA, first],
      [commentB, second],
    ] as const) {
      const row = page.locator(`[id="comment-${comment.id}"]`);
      // eslint-disable-next-line no-await-in-loop -- Each comment's section is opened and read in turn.
      await row.getByRole("button", { name: "Attachments", exact: true }).click();
      // eslint-disable-next-line no-await-in-loop
      await expect(row.getByRole("button", { name: /^Download / })).toHaveCount(1);
      // eslint-disable-next-line no-await-in-loop
      await files.downloaded(row.getByRole("button", { name: `Download ${file.name}`, exact: true }), file);
    }
    // An existing comment's own section binds a file on completion, with no comment write, and keeps the focus.
    const section = page
      .locator(`[id="comment-${commentA.id}"]`)
      .getByRole("button", { name: "Attachments", exact: true })
      .locator("xpath=..");
    const later = text("added-later.txt", "bytes added to an existing comment");
    await choose(page, section, later);
    const laterID = await files.transfer(3, "COMMENT_ATTACHMENT", { kind: "comment", id: commentA.id }, later);
    await expect(section.getByRole("button", { name: `Download ${later.name}`, exact: true })).toBeVisible();
    await expect(section.getByLabel("Choose or replace file", { exact: true })).toBeFocused();
    expect(await bound(laterID)).toBe(commentA.id);
    expect(files.calls("PATCH", `/api/v0/issue-comments/${commentA.id}`)).toHaveLength(0);
    // Its author deletes it after one explicit confirmation: exactly one DELETE, and only the comment's list is read.
    await section.getByRole("button", { name: `Delete file ${later.name}`, exact: true }).click();
    await modalDialog(page).getByRole("button", { name: "Delete file", exact: true }).click();
    await expect(section.getByRole("button", { name: `Download ${later.name}`, exact: true })).toHaveCount(0);
    await expect(page.getByRole("status").filter({ hasText: "File deleted." })).toBeVisible();
    await page.waitForTimeout(1000);
    expect(files.calls("DELETE", `/api/v0/assets/${laterID}`)).toHaveLength(1);
    expect(await assetRows(db, [laterID])).toMatchObject([{ is_deleted: true }]);
    expect(files.calls("PATCH", `/api/v0/issue-comments/${commentA.id}`)).toHaveLength(0);
    expect(sent).toHaveLength(4);
    // The lost answer and the refusal are the only failures; the bound preparation's read answered 200.
    await files.finish((failure) => failure.endsWith(`POST ${path}`) && !failure.startsWith("4"));
  }
);
