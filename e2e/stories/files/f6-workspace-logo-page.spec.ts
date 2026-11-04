import { inviteAndAccept } from "../../fixtures/api";
import { emailFor, bearer } from "../../fixtures/auth";
import { watchPage } from "../../fixtures/browser";
import { modalDialog } from "../../fixtures/dialogs";
import { pageFileStory } from "../../fixtures/file-story";
import { expectDecoded, profileImagePage } from "../../fixtures/profile-images-page";
import { answerTo, registerOnboarded } from "../../fixtures/settings-pages";
import { filePixel, storedFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";

pageFileStory(
  "F6 (page): admin binds a workspace icon without losing draft fields; members and guests only display it",
  async (world, info) => {
    info.setTimeout(90_000);
    const { page, api, db, baseURL } = world;
    const image = await profileImagePage(world, info);
    const { workspace } = image;
    const avatar = await storedFile(
      api,
      image.owner.token,
      "USER_AVATAR",
      { kind: "user" },
      filePixel,
      "image/png",
      "member.png",
      world.uploadStatus
    );
    expect(
      (
        await api.PATCH("/api/v0/me", {
          headers: bearer(image.owner.token),
          body: { avatar_asset_id: avatar.id },
        })
      ).response.status
    ).toBe(200);
    await page.goto(`${baseURL}/${workspace.slug}/settings`);
    await image.probe();
    await page.locator('input[name="name"]').fill("Unsaved workspace");
    await page.getByRole("button", { name: "Upload workspace icon", exact: true }).first().click();
    await image.select();
    const endpoint = `/api/v0/workspaces/${workspace.slug}`;
    expect(
      (
        await answerTo(page, "PATCH", endpoint, () => page.getByRole("button", { name: "Upload and save" }).dblclick())
      ).status()
    ).toBe(200);
    await expect(modalDialog(page)).toHaveCount(0);
    const id = await image.transfer(0, "WORKSPACE_LOGO", { kind: "workspace", id: workspace.id });
    expect(image.bodies("PATCH", endpoint)).toEqual([{ logo_asset_id: id }]);
    await image.persisted(id);
    await expect(page.locator('input[name="name"]')).toHaveValue("Unsaved workspace");
    expect(await db.query("SELECT logo_asset_id, name FROM workspaces WHERE id=$1", [workspace.id])).toEqual([
      { logo_asset_id: id, name: workspace.name },
    ]);
    const browser = page.context().browser();
    if (!browser) throw new Error("Missing browser");
    const view = async (role: 5 | 15) => {
      const email = emailFor(info, `logo-${role}`);
      const tokens = await registerOnboarded(api, email);
      await inviteAndAccept(api, image.owner.token, workspace.slug, { email, token: tokens.access_token }, role);
      const reader = await anotherBrowser(browser, baseURL, tokens);
      try {
        const watch = await watchPage(reader.page);
        await reader.page.goto(`/${workspace.slug}/projects`);
        await expectDecoded(reader.page.getByRole("button", { name: "Open workspace switcher" }).locator("img"));
        await reader.page.goto(`/${workspace.slug}/settings`);
        if (role === 5)
          await expect(
            reader.page.getByRole("heading", { name: "Oops! You are not authorized to view this page" })
          ).toBeVisible();
        else await expect(reader.page.locator('input[name="name"]')).toBeDisabled();
        await expectDecoded(reader.page.getByRole("button", { name: "Open workspace switcher" }).locator("img"));
        await expect(reader.page.getByRole("button", { name: /workspace icon/ })).toHaveCount(0);
        if (role === 5) {
          // The existing workspace layout reads a redacted member directory once per document.
          // Displaying its icon adds no member-image directory request for a guest.
          expect(
            watch.apiRequests.filter((request) => request === `GET /api/v0/workspaces/${workspace.slug}/members`)
          ).toHaveLength(2);
        } else {
          await reader.page.goto(`/${workspace.slug}/settings/members`);
          const row = reader.page.getByRole("row").filter({ hasText: image.owner.email });
          await expectDecoded(row.locator("img"));
        }
        expect(watch.oldApiRequests).toEqual([]);
        expect(watch.apiFailures).toEqual([]);
      } finally {
        await reader.close();
      }
    };
    await view(15);
    await view(5);
    await page.getByRole("button", { name: "Change workspace icon", exact: true }).first().click();
    expect(
      (
        await answerTo(page, "PATCH", endpoint, () => page.getByRole("button", { name: "Remove image" }).click())
      ).status()
    ).toBe(200);
    await expect(modalDialog(page)).toHaveCount(0);
    expect(image.bodies("PATCH", endpoint)).toEqual([{ logo_asset_id: id }, { logo_asset_id: null }]);
    const current = await api.GET("/api/v0/workspaces/{slug}", {
      params: { path: { slug: workspace.slug } },
      headers: bearer(image.owner.token),
    });
    expect(current.data).toMatchObject({ logo_asset_id: null, logo_url: null });
    await image.finish();
  }
);
