import { createHash } from "node:crypto";
import { readdir, readFile } from "node:fs/promises";
import type { Page } from "@playwright/test";
import { expect, test } from "../../fixtures/test";
import { issueListProject, openIssueList } from "../../fixtures/issue-list-page";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { createIssue } from "../../fixtures/issues";
import { password } from "../../fixtures/auth";
import { signInPath, submitSignIn } from "../../fixtures/auth-pages";
import { answerTo, expectPopupBesideButton, transitionsEnded } from "../../fixtures/settings-pages";
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

async function holdEditorImplementation(page: Page) {
  const chunks = (await editorChunks()).filter(({ markers }) =>
    markers.includes("Adding different instances of a keyed plugin")
  );
  let release!: () => void;
  const loaded = new Promise<void>((resolve) => {
    release = resolve;
  });
  const held: string[] = [];
  await page.route("**/assets/*.js", async (route) => {
    const file = new URL(route.request().url()).pathname.split("/").at(-1);
    if (chunks.some((chunk) => chunk.file === file)) {
      held.push(file!);
      await loaded;
    }
    await route.continue();
  });
  return { held, release };
}

test("C10 (page): an open peek display panel follows its button when the editor finishes loading", async ({
  api,
  page,
}, info) => {
  const { held, release } = await holdEditorImplementation(page);
  try {
    const seed = await issueListProject(api, info);
    const parent = await createIssue(api, seed.member.token, seed.project.id, { name: "Loading parent" });
    await createIssue(api, seed.member.token, seed.project.id, { name: "Loading child", parent_id: parent.id });
    const { watch } = await openIssueList(page, seed);
    await page.locator(`#issue-${parent.id}`).click();
    const peek = page.locator("#full-screen-portal");
    await expect(peek.getByText("Loading child", { exact: true })).toBeVisible();
    const trigger = peek.getByRole("button", { name: "Display", exact: true });
    await trigger.focus();
    await page.keyboard.press("Enter");
    await expect(trigger).toHaveAttribute("aria-expanded", "true");
    const panel = page.locator(`[id="${await trigger.getAttribute("aria-controls")}"]`);
    await transitionsEnded(panel);
    await expectPopupBesideButton(panel, trigger);
    await expect.poll(() => held.length).toBeGreaterThan(0);
    const editor = peek.locator(`[id="editor-container-${parent.id}"] [contenteditable]`);
    await expect(editor).toHaveCount(0);
    const before = (await trigger.boundingBox())!;
    release();
    await expect(editor).toBeVisible();
    expect((await trigger.boundingBox())!.y).toBeGreaterThan(before.y);
    // The open panel catches up on a rendered frame; no resize or scroll is sent to make it move.
    await expect(async () => expectPopupBesideButton(panel, trigger)).toPass({ timeout: 3000 });
    await page.keyboard.press("Escape");
    await expect(panel).toHaveCount(0);
    await expect(trigger).toBeFocused();
    expect(watch.apiFailures).toEqual([]);
    expect(watch.oldApiRequests).toEqual([]);
    expect(watch.pageErrors).toEqual([]);
    expect(watch.cspViolations).toEqual([]);
    expect(requestsElsewhere(page, watch)).toEqual([]);
    await expectQuietConsole(page, watch);
  } finally {
    release();
  }
});

for (const dirty of [false, true]) {
  for (const action of ["Escape", "Discard"] as const) {
    test(`C10 (page): ${action} can leave a ${dirty ? "dirty" : "clean"} creation while the real editor chunk is still pending`, async ({
      api,
      db,
      page,
    }, info) => {
      const { held, release } = await holdEditorImplementation(page);
      try {
        const seed = await issueListProject(api, info);
        const { watch } = await openIssueList(page, seed);
        const trigger = page.getByRole("button", { name: "Add work item", exact: true });
        await trigger.click();
        const form = page.getByRole("dialog", { name: "Create new work item", exact: true });
        const title = form.getByRole("textbox", { name: "Title", exact: true });
        await expect(title).toBeVisible();
        await expect.poll(() => held.length).toBeGreaterThan(0);
        await expect(form.locator('[contenteditable="true"]')).toHaveCount(0);
        if (dirty) {
          await title.fill("Retained before the editor loads");
        }
        if (action === "Escape") {
          await title.focus();
          await page.keyboard.press("Escape");
        } else await form.getByRole("button", { name: "Discard", exact: true }).click();
        if (dirty) {
          const confirm = page.getByRole("dialog", { name: "Save this draft?", exact: true });
          await expect(confirm.getByRole("heading", { name: "Save this draft?", exact: true })).toBeVisible();
          await confirm.getByRole("button", { name: "Save to Drafts", exact: true }).click();
          await expect(page.getByText("Editor is not ready to discard changes", { exact: true })).toBeVisible();
          await confirm.getByRole("button", { name: "Cancel", exact: true }).click();
          await expect(title).toHaveValue("Retained before the editor loads");
          await form.getByRole("button", { name: "Discard", exact: true }).click();
          await confirm.getByRole("button", { name: "Discard", exact: true }).click();
        }
        await expect(form).toHaveCount(0);
        await expect(trigger).toBeFocused();
        expect(watch.apiRequests.filter((request) => /^POST .*\/(issues|drafts)$/.test(request))).toEqual([]);
        expect(await db.query("SELECT id FROM issues WHERE project_id=$1", [seed.project.id])).toEqual([]);
        release();
        await expectQuietConsole(page, watch);
        expect(watch.apiFailures).toEqual([]);
        expect(watch.oldApiRequests).toEqual([]);
        expect(watch.pageErrors).toEqual([]);
        expect(watch.cspViolations).toEqual([]);
        expect(requestsElsewhere(page, watch)).toEqual([]);
      } finally {
        release();
      }
    });
  }
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

for (const owner of ["detail", "creation"] as const) {
  test(`C12 (page): ${owner} has no image, attachment or subscription action; image paste and drop make no upload and text still saves`, async ({
    api,
    db,
    page,
  }, info) => {
    const seed = await issueListProject(api, info);
    const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Image boundary" });
    const watch =
      owner === "detail" ? await openIssueDetail(page, seed, issue) : (await openIssueList(page, seed)).watch;
    if (owner === "creation") {
      await page.getByRole("button", { name: "Add work item", exact: true }).click();
      await page.getByRole("textbox", { name: "Title", exact: true }).fill("Created without images");
    }
    const editor = page.locator(
      owner === "detail"
        ? `[id="editor-container-${issue.id}"] .tiptap[contenteditable="true"]`
        : '[id="editor-container-issue-modal-editor"] .tiptap[contenteditable="true"]'
    );
    await expect(editor).toBeVisible();
    await editor.pressSequentially("/");
    await expect(page.getByRole("button", { name: "Callout", exact: true })).toBeVisible();
    await editor.pressSequentially("image");
    await expect(page.locator("#slash-command")).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Image", exact: true })).toHaveCount(0);
    await editor.press("ControlOrMeta+a");
    await editor.press("Backspace");
    await editor.evaluate((element) => {
      const png = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+ip1sAAAAASUVORK5CYII=";
      const file = new File([Uint8Array.from(atob(png), (char) => char.charCodeAt(0))], "boundary.png", {
        type: "image/png",
      });
      const clipboard = new DataTransfer();
      clipboard.items.add(file);
      // Firefox's synthetic ClipboardEvent ignores the constructor's clipboardData. Give every
      // engine the same actual DataTransfer; editor handlers and persistence remain real.
      const paste = new ClipboardEvent("paste", { bubbles: true, cancelable: true });
      Object.defineProperty(paste, "clipboardData", { value: clipboard });
      element.dispatchEvent(paste);
      const dropped = new DataTransfer();
      dropped.items.add(file);
      const rect = element.getBoundingClientRect();
      element.dispatchEvent(
        new DragEvent("drop", {
          dataTransfer: dropped,
          clientX: rect.left + 5,
          clientY: rect.top + 5,
          bubbles: true,
          cancelable: true,
        })
      );
    });
    // ProseMirror may briefly own a hidden paste target for a file without insertable text.
    // Wait for its real focus restoration before sending the next, separate clipboard input.
    await expect(editor).toBeFocused();
    await editor.evaluate((element) => {
      // A file clipboard is handled as a file. Separately paste document HTML, whose text remains
      // available even though its image node is absent from the M4 editor schema.
      const documentClipboard = new DataTransfer();
      const png = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+ip1sAAAAASUVORK5CYII=";
      documentClipboard.setData("text/html", `<p>Kept paste text</p><img src="data:image/png;base64,${png}">`);
      documentClipboard.setData("text/plain", "Kept paste text");
      const paste = new ClipboardEvent("paste", { bubbles: true, cancelable: true });
      Object.defineProperty(paste, "clipboardData", { value: documentClipboard });
      element.dispatchEvent(paste);
    });
    await expect(editor).toHaveText("Kept paste text");
    await page.keyboard.insertText(" plus typed");
    await expect(editor).toHaveText("Kept paste text plus typed");
    await expect(editor.locator("img,input[type=file],image-component")).toHaveCount(0);
    await expect(page.getByText("Attachments", { exact: true })).toHaveCount(0);
    await expect(page.getByRole("button", { name: /^(un)?subscribe$/i })).toHaveCount(0);
    if (owner === "creation") {
      expect(
        (
          await answerTo(page, "POST", `/api/v0/projects/${seed.project.id}/issues`, () =>
            page
              .getByRole("dialog", { name: "Create new work item", exact: true })
              .getByRole("button", { name: "Save", exact: true })
              .click()
          )
        ).status()
      ).toBe(201);
    }
    await expect
      .poll(
        async () =>
          (
            await db.query<{ description_stripped: string }>(
              "SELECT description_stripped FROM issues WHERE project_id=$1 AND name=$2",
              [seed.project.id, owner === "detail" ? "Image boundary" : "Created without images"]
            )
          )[0]?.description_stripped
      )
      .toBe("Kept paste text plus typed");
    expect(watch.apiRequests.filter((request) => /asset|upload|attachment|subscribe/i.test(request))).toEqual([]);
    expect(watch.oldApiRequests).toEqual([]);
    expect(watch.apiFailures).toEqual([]);
    expect(watch.pageErrors).toEqual([]);
    expect(watch.cspViolations).toEqual([]);
    expect(requestsElsewhere(page, watch)).toEqual([]);
    await expectQuietConsole(page, watch);
  });
}
