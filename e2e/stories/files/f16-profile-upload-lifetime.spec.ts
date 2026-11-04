import { changeRole, createProject, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { anotherTabSignsIn, bearer } from "../../fixtures/auth";
import { deferred } from "../../fixtures/deferred";
import { modalDialog } from "../../fixtures/dialogs";
import { holdImageBoundary } from "../../fixtures/profile-image-lifetime";
import { expectDecoded, profileImagePage, redPixel } from "../../fixtures/profile-images-page";
import { answerTo, holdAnswer, moveWithinApp } from "../../fixtures/settings-pages";
import { filePixel, storedFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";
import { test } from "../../fixtures/profile-image-wire";

for (const boundary of ["prepare", "put", "complete"] as const) {
  test(`F16 (page): cancelling at ${boundary} prevents later binding and permits a fresh selection`, async ({
    db,
    imageWorld,
    page,
  }, info) => {
    const image = await profileImagePage(imageWorld, info);
    await page.goto("/settings/profile/general");
    await page.getByRole("button", { name: "Change avatar", exact: true }).click();
    await image.select();
    const held = await holdImageBoundary(page, boundary);
    try {
      await page.getByRole("button", { name: "Upload and save", exact: true }).click();
      await held.wait();
      await page.getByRole("button", { name: "Cancel upload", exact: true }).click();
      await expect(page.getByLabel("Choose or replace image", { exact: true })).toBeFocused();
      await held.release();
      await image.select(redPixel);
      const saved = await answerTo(page, "PATCH", "/api/v0/me", () =>
        page.getByRole("button", { name: "Upload and save", exact: true }).click()
      );
      expect(saved.status()).toBe(200);
      await expect(modalDialog(page)).toHaveCount(0);
      const id = (await saved.json()).avatar_asset_id;
      await image.persisted(id, redPixel);
      expect(image.calls("POST", "/api/v0/assets")).toHaveLength(2);
      expect(image.bodies("PATCH", "/api/v0/me")).toEqual([{ avatar_asset_id: id }]);
      const assets = await db.query<{ id: string; bound: boolean }>(
        "SELECT id, bound_at IS NOT NULL AS bound FROM file_assets WHERE user_id=$1 ORDER BY created_at",
        [image.owner.id]
      );
      expect(assets).toHaveLength(2);
      expect(assets.filter((asset) => asset.bound)).toEqual([{ id, bound: true }]);
      const first = assets[0];
      if (!first) throw new Error("Missing cancelled preparation");
      expect(image.calls("POST", `/api/v0/assets/${first.id}/complete`)).toHaveLength(boundary === "complete" ? 1 : 0);
      expect(image.watch.oldApiRequests).toEqual([]);
      expect(image.watch.pageErrors).toEqual([]);
    } finally {
      await held.release();
    }
  });
}

test("F16 (page): Back and Forward across workspace and project retire a pending cover PUT", async ({
  api,
  db,
  imageWorld,
  page,
}, info) => {
  const image = await profileImagePage(imageWorld, info);
  const beta = await createWorkspace(api, image.owner.token, { name: "Other workspace", slug: slugFor(info, "beta") });
  const betaProject = await createProject(api, image.owner.token, beta.slug, {
    name: "Other workspace project",
    identifier: "BETA",
  });
  const neighbor = await createProject(api, image.owner.token, image.workspace.slug, {
    name: "Neighbor project",
    identifier: "NEXT",
  });
  await page.goto(`/${image.workspace.slug}/projects`);
  const current = `/${image.workspace.slug}/settings/projects/${image.project.id}`;
  await [
    { path: `/${beta.slug}/settings/projects/${betaProject.id}`, name: betaProject.name },
    { path: `/${image.workspace.slug}/settings/projects/${neighbor.id}`, name: neighbor.name },
  ].reduce(async (previous, entry) => {
    await previous;
    await moveWithinApp(page, entry.path);
    await expect(page.locator("#name")).toHaveValue(entry.name);
    await moveWithinApp(page, current);
    await expect(page.locator("#name")).toHaveValue(image.project.name);
    await page.getByRole("button", { name: "Change cover", exact: true }).click();
    await page.getByRole("button", { name: "Upload image", exact: true }).click();
    await image.select();
    const held = await holdImageBoundary(page, "put");
    try {
      await page.getByRole("button", { name: "Upload and save", exact: true }).click();
      await held.wait();
      await page.goBack();
      await expect(page.locator("#name")).toHaveValue(entry.name);
      await expect(modalDialog(page)).toHaveCount(0);
      await page.goForward();
      await expect(page.locator("#name")).toHaveValue(image.project.name);
      await expect(modalDialog(page)).toHaveCount(0);
      await held.release();
      await page.getByRole("button", { name: "Change cover", exact: true }).click();
      await expect(page.getByRole("button", { name: "Cover 1", exact: true })).toBeVisible();
      await expect(page.getByAltText("Selected image preview")).toHaveCount(0);
      await page.keyboard.press("Escape");
      await expect(modalDialog(page)).toHaveCount(0);
    } finally {
      await held.release();
    }
  }, Promise.resolve());
  expect(image.calls("PATCH", `/api/v0/projects/${image.project.id}`)).toHaveLength(0);
  expect(await db.query("SELECT bound_at FROM file_assets WHERE project_id=$1", [image.project.id])).toEqual([
    { bound_at: null },
    { bound_at: null },
  ]);
  expect(image.watch.apiRequests.filter((request) => request.endsWith("/complete"))).toEqual([]);
  expect(image.watch.oldApiRequests).toEqual([]);
  expect(image.watch.pageErrors).toEqual([]);
});

test("F16 (page): an account switch during confirmation cannot bind the former account's image", async ({
  api,
  db,
  imageWorld,
  page,
  context,
}, info) => {
  const image = await profileImagePage(imageWorld, info);
  expect(
    (await api.PATCH("/api/v0/me/profile", { headers: bearer(image.other.token), body: { is_onboarded: true } }))
      .response.status
  ).toBe(200);
  await page.goto("/settings/profile/general");
  await page.getByRole("button", { name: "Change avatar", exact: true }).click();
  await image.select();
  const held = await holdImageBoundary(page, "complete");
  let otherTab;
  try {
    await page.getByRole("button", { name: "Upload and save", exact: true }).click();
    await held.wait();
    otherTab = await anotherTabSignsIn(context, api, image.other.email);
    await expect(page.locator("#display_name")).toHaveValue(image.other.email.split("@")[0] ?? "");
    await expect(modalDialog(page)).toHaveCount(0);
    await held.release();
    await page.getByRole("button", { name: "Change avatar", exact: true }).click();
    await expect(page.getByRole("button", { name: "Upload and save", exact: true })).toBeDisabled();
    await expect(page.getByAltText("Selected image preview")).toHaveCount(0);
    expect(image.calls("PATCH", "/api/v0/me")).toHaveLength(0);
    expect(
      await db.query("SELECT avatar_asset_id FROM users WHERE id=ANY($1::uuid[])", [[image.owner.id, image.other.id]])
    ).toEqual([{ avatar_asset_id: null }, { avatar_asset_id: null }]);
    expect(await db.query("SELECT bound_at FROM file_assets WHERE user_id=$1", [image.owner.id])).toEqual([
      { bound_at: null },
    ]);
    expect(image.watch.pageErrors).toEqual([]);
  } finally {
    await held.release();
    await otherTab?.close();
  }
});

for (const outcome of ["success", "refusal"] as const) {
  test(`F16 (page): workspace permission ABA rejects an old owner ${outcome} after a new icon is authorized`, async ({
    api,
    imageWorld,
    page,
  }, info) => {
    const image = await profileImagePage(imageWorld, info);
    const { workspace, owner, other } = image;
    await inviteAndAccept(api, owner.token, workspace.slug, other, 20);
    const seed = async (bytes = filePixel) => {
      const asset = await storedFile(
        api,
        owner.token,
        "WORKSPACE_LOGO",
        { kind: "workspace", id: workspace.id },
        bytes,
        "image/png",
        "logo.png"
      );
      expect(
        (
          await api.PATCH("/api/v0/workspaces/{slug}", {
            headers: bearer(owner.token),
            params: { path: { slug: workspace.slug } },
            body: { logo_asset_id: asset.id },
          })
        ).response.status
      ).toBe(200);
      return asset.id;
    };
    await seed();
    await page.goto(`/${workspace.slug}/settings`);
    const logo = page
      .getByRole("button", { name: "Open workspace switcher", exact: true, includeHidden: true })
      .locator("img");
    await expectDecoded(logo);
    const release = deferred();
    const finished = deferred();
    let received = false;
    const endpoint = `/api/v0/workspaces/${workspace.slug}`;
    await page.route(
      (url) => url.pathname === endpoint,
      async (route) => {
        if (route.request().method() !== "PATCH" || received) return route.fallback();
        if (outcome === "refusal") await changeRole(api, other.token, workspace.slug, owner.id, 15);
        const response = await route.fetch();
        expect(response.status()).toBe(outcome === "success" ? 200 : 403);
        received = true;
        try {
          await release.promise;
          await route.fulfill({ response });
        } finally {
          finished.resolve();
        }
      }
    );
    try {
      await page.getByRole("button", { name: "Change workspace icon", exact: true }).first().click();
      await image.select();
      await page.getByRole("button", { name: "Upload and save", exact: true }).click();
      await expect.poll(() => received).toBe(true);
      if (outcome === "success") await changeRole(api, other.token, workspace.slug, owner.id, 15);
      // A real image error causes the product's own authorized directory refresh; no store is injected.
      await logo.dispatchEvent("error");
      await expect(page.locator('input[name="name"]')).toBeDisabled();
      await expect(modalDialog(page)).toHaveCount(0);
      await changeRole(api, other.token, workspace.slug, owner.id, 20);
      const next = await seed(redPixel);
      await expectDecoded(logo);
      await logo.dispatchEvent("error");
      await expect(page.locator('input[name="name"]')).toBeEnabled();
      await expectDecoded(logo);
      const currentURL = await logo.getAttribute("src");
      const delivered = page.waitForResponse(
        (response) => response.request().method() === "PATCH" && new URL(response.url()).pathname === endpoint
      );
      release.resolve();
      await finished.promise;
      await (await delivered).finished();
      await expect(page.locator('input[name="name"]')).toBeEnabled();
      await expect(logo).toHaveAttribute("src", currentURL ?? "missing");
      await expect(modalDialog(page)).toHaveCount(0);
      await expect(page.getByRole("alert")).toHaveCount(0);
      await page.getByRole("button", { name: "Change workspace icon", exact: true }).first().click();
      await expect(modalDialog(page).locator("img")).toHaveAttribute("src", currentURL ?? "missing");
      await page.keyboard.press("Escape");
      await expect(modalDialog(page)).toHaveCount(0);
      expect(
        (
          await api.GET("/api/v0/workspaces/{slug}", {
            headers: bearer(owner.token),
            params: { path: { slug: workspace.slug } },
          })
        ).data?.logo_asset_id
      ).toBe(next);
      await image.persisted(next, redPixel);
      expect(image.watch.pageErrors).toEqual([]);
      await info.attach("workspace-owner-lifetime", {
        body: JSON.stringify({ outcome, next, observedRoleLossAndRecovery: true }),
        contentType: "application/json",
      });
    } finally {
      release.resolve();
      if (received) await finished.promise;
    }
  });
}

test("F16 (page): an old authorized user read cannot overwrite a newly bound avatar", async ({
  api,
  imageWorld,
  page,
}, info) => {
  const image = await profileImagePage(imageWorld, info);
  const old = await storedFile(
    api,
    image.owner.token,
    "USER_AVATAR",
    { kind: "user" },
    filePixel,
    "image/png",
    "old.png"
  );
  expect(
    (await api.PATCH("/api/v0/me", { headers: bearer(image.owner.token), body: { avatar_asset_id: old.id } })).response
      .status
  ).toBe(200);
  await page.goto("/settings/profile/general");
  const opener = page.getByRole("button", { name: "Change avatar", exact: true });
  const avatar = opener.locator("img");
  await expectDecoded(avatar);
  let received = false;
  const release = await holdAnswer(page, "GET", "/api/v0/me", 1, () => {
    received = true;
  });
  try {
    await avatar.dispatchEvent("error");
    await expect.poll(() => received).toBe(true);
    await opener.click();
    await image.select(redPixel);
    const saved = await answerTo(page, "PATCH", "/api/v0/me", () =>
      page.getByRole("button", { name: "Upload and save", exact: true }).click()
    );
    expect(saved.status()).toBe(200);
    await expect(modalDialog(page)).toHaveCount(0);
    const updated = await saved.json();
    const id = updated.avatar_asset_id;
    await (await answerTo(page, "GET", "/api/v0/me", release)).finished();
    await expectDecoded(avatar);
    const latest = await avatar.getAttribute("src");
    expect(new URL(latest ?? "missing", imageWorld.baseURL).pathname).toBe(
      new URL(updated.avatar_url, imageWorld.baseURL).pathname
    );
    // Opening a new dialog commits another consumer after the held read has settled.
    await opener.click();
    await expect(modalDialog(page).locator("img")).toHaveAttribute("src", latest ?? "missing");
    await page.keyboard.press("Escape");
    await expect(avatar).toHaveAttribute("src", latest ?? "missing");
    await image.persisted(id, redPixel);
    expect(image.calls("PATCH", "/api/v0/me")).toHaveLength(1);
    await image.finish();
  } finally {
    await release();
  }
});

test("F16 (page): an expired capability is removed, refreshed once, then explicitly retried without rewriting the image", async ({
  api,
  imageWorld,
  page,
}, info) => {
  const image = await profileImagePage(imageWorld, info);
  const avatar = await storedFile(
    api,
    image.owner.token,
    "USER_AVATAR",
    { kind: "user" },
    filePixel,
    "image/png",
    "expiry.png"
  );
  expect(
    (await api.PATCH("/api/v0/me", { headers: bearer(image.owner.token), body: { avatar_asset_id: avatar.id } }))
      .response.status
  ).toBe(200);
  const now = Date.now();
  await page.clock.install({ time: new Date(now) });
  let inject = true;
  let reads = 0;
  await page.route(
    (url) => url.pathname === "/api/v0/me",
    async (route) => {
      if (route.request().method() !== "GET") return route.fallback();
      const response = await route.fetch();
      const body = await response.json();
      reads++;
      // UI expiry injection only: the backend clock and the signed capability TTL are unchanged.
      if (inject) body.avatar_url_expires_at = new Date(now + 60_000).toISOString();
      await route.fulfill({ response, json: body });
    }
  );
  await page.goto("/settings/profile/general");
  const avatarImage = page.getByRole("button", { name: "Change avatar", exact: true }).locator("img");
  await expectDecoded(avatarImage);
  const initialReads = reads;
  await page.clock.setSystemTime(new Date(now + 65_000));
  await page.evaluate(() => window.dispatchEvent(new Event("focus")));
  await expect(avatarImage).toHaveCount(0);
  const retry = page.getByRole("button", { name: "Retry image", exact: true });
  await expect(retry).toBeEnabled();
  expect(reads).toBe(initialReads + 1);
  await page.evaluate(() => {
    window.dispatchEvent(new Event("focus"));
    window.dispatchEvent(new Event("online"));
    document.dispatchEvent(new Event("visibilitychange"));
  });
  inject = false;
  await retry.click();
  await expectDecoded(avatarImage);
  expect(reads).toBe(initialReads + 2);
  expect(image.calls("POST", "/api/v0/assets")).toHaveLength(0);
  expect(image.calls("PATCH", "/api/v0/me")).toHaveLength(0);
  await info.attach("ui-expiry-injection", {
    body: JSON.stringify({ browserTimeOnly: true, initialReads, finalReads: reads, automaticReads: 1, manualReads: 1 }),
    contentType: "application/json",
  });
  await image.finish();
});
