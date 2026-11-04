import { bearer, login } from "../../fixtures/auth";
import { modalDialog } from "../../fixtures/dialogs";
import { pageFileStory } from "../../fixtures/file-story";
import { expectDecoded, profileImagePage, redPixel } from "../../fixtures/profile-images-page";
import { answerTo } from "../../fixtures/settings-pages";
import { filePixel } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";
import { test } from "../../fixtures/profile-image-wire";
import { anotherBrowser } from "../../fixtures/workspace-pages";

pageFileStory(
  "F5 (page): real avatar and cover uploads bind separately, replace same-name bytes, survive reload and a new login, and clear",
  async (world, info) => {
    info.setTimeout(90_000);
    const { page, baseURL, db, api } = world;
    const image = await profileImagePage(world, info);
    await page.goto(`${baseURL}/settings/profile/general`);
    await image.probe();
    await page.locator("#first_name").fill("Unsaved name");
    const upload = async (kind: "avatar" | "cover", ordinal: number, bytes = filePixel) => {
      await page
        .getByRole("button", { name: kind === "avatar" ? "Change avatar" : "Change cover", exact: true })
        .click();
      await expect(modalDialog(page)).toHaveAccessibleName(kind === "avatar" ? "Change avatar" : "Change cover");
      await image.select(bytes);
      const saved = await answerTo(page, "PATCH", "/api/v0/me", () =>
        page.getByRole("button", { name: "Upload and save", exact: true }).dblclick()
      );
      expect(saved.status()).toBe(200);
      await expect(modalDialog(page)).toHaveCount(0);
      const id = await image.transfer(
        ordinal,
        kind === "avatar" ? "USER_AVATAR" : "USER_COVER",
        { kind: "user" },
        bytes
      );
      expect(image.bodies("PATCH", "/api/v0/me").at(-1)).toEqual({
        [kind === "avatar" ? "avatar_asset_id" : "cover_image_asset_id"]: id,
      });
      await image.persisted(id, bytes);
      await expect(page.locator("#first_name")).toHaveValue("Unsaved name");
      return id;
    };
    const first = await upload("avatar", 0);
    const cover = await upload("cover", 1, redPixel);
    const avatar = await upload("avatar", 2, redPixel);
    expect(avatar).not.toBe(first);
    expect(
      await db.query("SELECT avatar_asset_id, cover_image_asset_id, first_name FROM users WHERE id=$1", [
        image.owner.id,
      ])
    ).toEqual([{ avatar_asset_id: avatar, cover_image_asset_id: cover, first_name: "" }]);
    expect(await db.query("SELECT is_deleted FROM file_assets WHERE id=$1", [first])).toEqual([{ is_deleted: true }]);
    await page.reload();
    await expectDecoded(page.getByRole("button", { name: "Change avatar", exact: true }).locator("img"));
    const browser = page.context().browser();
    if (!browser) throw new Error("No browser for fresh login");
    const fresh = await anotherBrowser(browser, baseURL, await login(api, image.owner.email));
    try {
      await fresh.page.goto("/settings/profile/general");
      await expectDecoded(fresh.page.getByRole("button", { name: "Change avatar", exact: true }).locator("img"));
    } finally {
      await fresh.close();
    }
    const clear = async (kind: "avatar" | "cover") => {
      await page
        .getByRole("button", { name: kind === "avatar" ? "Change avatar" : "Change cover", exact: true })
        .click();
      const cleared = await answerTo(page, "PATCH", "/api/v0/me", () =>
        page.getByRole("button", { name: "Remove image", exact: true }).click()
      );
      expect(cleared.status()).toBe(200);
      await expect(modalDialog(page)).toHaveCount(0);
      expect(image.bodies("PATCH", "/api/v0/me").at(-1)).toEqual({
        [kind === "avatar" ? "avatar_asset_id" : "cover_image_asset_id"]: null,
      });
    };
    await clear("avatar");
    await clear("cover");
    expect(image.calls("POST", "/api/v0/assets")).toHaveLength(3);
    expect(image.calls("PATCH", "/api/v0/me")).toHaveLength(5);
    const me = await api.GET("/api/v0/me", { headers: bearer(image.owner.token) });
    expect(me.data).toMatchObject({
      avatar_asset_id: null,
      cover_image_asset_id: null,
      avatar_url: null,
      cover_image_url: null,
    });
    await image.finish();
  }
);

test("F5 (page): onboarding avatar binds without completing or submitting the profile step", async ({
  api,
  imageWorld,
  page,
}, info) => {
  const image = await profileImagePage(imageWorld, info, false);
  await page.goto("/onboarding");
  await page.getByRole("button", { name: "Change avatar", exact: true }).first().click();
  await image.select();
  expect(
    (
      await answerTo(page, "PATCH", "/api/v0/me", () => page.getByRole("button", { name: "Upload and save" }).click())
    ).status()
  ).toBe(200);
  await expect(modalDialog(page)).toHaveCount(0);
  const id = await image.transfer(0, "USER_AVATAR", { kind: "user" });
  await image.persisted(id);
  expect(image.calls("PATCH", "/api/v0/me/profile")).toHaveLength(0);
  const profile = await api.GET("/api/v0/me/profile", { headers: bearer(image.owner.token) });
  expect(profile.data?.is_onboarded).toBe(false);
  await image.finish();
});
