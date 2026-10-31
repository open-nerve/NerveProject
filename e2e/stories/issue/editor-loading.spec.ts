import { createHash } from "node:crypto";
import { readdir, readFile } from "node:fs/promises";
import { expect, test } from "../../fixtures/test";
import { issueListProject } from "../../fixtures/issue-list-page";
import { createIssue } from "../../fixtures/issues";
import { password } from "../../fixtures/auth";
import { signInPath, submitSignIn } from "../../fixtures/auth-pages";
import { answerTo } from "../../fixtures/settings-pages";
import { expectQuietConsole, requestsElsewhere, watchPage } from "../../fixtures/browser";

// Identify the actual build by production code in it, not an assumed bundler filename.
// These are the editor's own suggestion plugin and ProseMirror's state implementation.
async function editorChunks() {
  const assets = new URL("../../../web/apps/web/build/client/assets/", import.meta.url);
  const files = (await readdir(assets)).filter((file) => file.endsWith(".js"));
  const chunks = await Promise.all(
    files.map(async (file) => {
      const code = await readFile(new URL(file, assets), "utf8");
      const markers = ["emojiSuggestion", "Adding different instances of a keyed plugin"].filter((marker) =>
        code.includes(marker)
      );
      return { file, markers, sha256: createHash("sha256").update(code).digest("hex") };
    })
  );
  const editor = chunks.filter(({ markers }) => markers.length > 0);
  expect(editor.flatMap(({ markers }) => markers)).toEqual(
    expect.arrayContaining(["emojiSuggestion", "Adding different instances of a keyed plugin"])
  );
  return editor;
}

for (const entry of ["number", "id"] as const) {
  test(`C10 (page): anonymous ${entry} entry loads no editor implementation; signing in loads it and saves real text without warnings`, async ({
    api,
    db,
    page,
  }, info) => {
    const chunks = await editorChunks();
    await info.attach("editor-build.json", { body: JSON.stringify(chunks, null, 2), contentType: "application/json" });
    const seed = await issueListProject(api, info);
    const issue = await createIssue(api, seed.member.token, seed.project.id, {
      name: "Lazy editor",
      description_html: "<p>Ready</p>",
    });
    const numbered = `/${seed.slug}/browse/WEB-${issue.sequence_id}`;
    const target = entry === "number" ? numbered : `/${seed.slug}/projects/${seed.project.id}/issues/${issue.id}`;
    const watch = await watchPage(page);
    await page.goto(target);
    await expect(page.getByRole("button", { name: "Go to workspace", exact: true })).toBeVisible();
    const loaded = () =>
      chunks.filter(({ file }) => watch.requestUrls.some((url) => new URL(url).pathname === `/assets/${file}`));
    expect(loaded(), "signed-out routing must not evaluate the editor").toEqual([]);
    expect(await submitSignIn(page, seed.member.email, password)).toBe(200);
    await expect(page).toHaveURL(numbered);
    const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
    await expect(editor).toHaveText("Ready");
    expect(loaded()).toEqual(chunks);
    await expect
      .poll(
        async () =>
          (
            await db.query<{ description_html: string }>("SELECT description_html FROM issues WHERE id=$1", [issue.id])
          )[0]?.description_html
      )
      .toContain("data-id=");
    expect(
      (
        await answerTo(page, "PATCH", `/api/v0/issues/${issue.id}`, async () => {
          await editor.click();
          await editor.press("End");
          await page.keyboard.insertText(" after loading");
        })
      ).status()
    ).toBe(200);
    await expect(editor).toHaveText("Ready after loading");
    expect(await db.query("SELECT description_stripped FROM issues WHERE id=$1", [issue.id])).toEqual([
      { description_stripped: "Ready after loading" },
    ]);
    expect(watch.apiFailures).toEqual([]);
    expect(watch.oldApiRequests).toEqual([]);
    expect(watch.pageErrors).toEqual([]);
    expect(watch.cspViolations).toEqual([]);
    expect(requestsElsewhere(page, watch)).toEqual([]);
    await expectQuietConsole(page, watch);
  });
}

test("C10 (page): personal settings keeps the command palette without loading a work-item editor", async ({
  api,
  page,
}, info) => {
  const chunks = await editorChunks();
  const seed = await issueListProject(api, info);
  const watch = await watchPage(page);
  await page.goto(signInPath("/settings/profile/security"));
  expect(await submitSignIn(page, seed.member.email, password)).toBe(200);
  await expect(page).toHaveURL("/settings/profile/security");
  await expect(page.getByRole("textbox", { name: "Old password", exact: true })).toBeVisible();
  await page.keyboard.press("ControlOrMeta+k");
  await expect(page.getByRole("dialog")).toHaveCount(1);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  expect(
    chunks.filter(({ file }) => watch.requestUrls.some((url) => new URL(url).pathname === `/assets/${file}`))
  ).toEqual([]);
  expect(watch.apiFailures).toEqual([]);
  expect(watch.oldApiRequests).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  expect(watch.cspViolations).toEqual([]);
  await expectQuietConsole(page, watch);
});

test("C10 (page): a native callout saves and reopens without external images or workers", async ({
  api,
  db,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Native callout" });
  const workers: string[] = [];
  page.on("worker", (worker) => workers.push(worker.url()));
  const watch = await watchPage(page);
  await page.goto(signInPath(`/${seed.slug}/browse/WEB-${issue.sequence_id}`));
  expect(await submitSignIn(page, seed.member.email, password)).toBe(200);
  const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
  await expect(editor).toBeVisible();
  await editor.pressSequentially("/callout");
  await page.getByRole("button", { name: "Callout", exact: true }).click();
  await expect(editor.getByText("💡", { exact: true })).toBeVisible();
  await page.keyboard.insertText("Native callout content");
  const savedHTML = async () =>
    (await db.query<{ description_html: string }>("SELECT description_html FROM issues WHERE id=$1", [issue.id]))[0]
      ?.description_html;
  await expect.poll(savedHTML).toContain("Native callout content");
  expect(await savedHTML()).toContain('data-emoji-unicode="128161"');
  expect(await savedHTML()).not.toMatch(/data-emoji-url|<img/);
  await page.reload();
  await expect(editor).toContainText("Native callout content");
  await expect(editor.getByText("💡", { exact: true })).toBeVisible();
  expect(workers).toEqual([]);
  expect(requestsElsewhere(page, watch)).toEqual([]);
  expect(watch.cspViolations).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  expect(watch.apiFailures).toEqual([]);
  expect(watch.oldApiRequests).toEqual([]);
  await expectQuietConsole(page, watch);
});
