import { modalDialog } from "../../fixtures/dialogs";
import { dropImage, holdImageBoundary } from "../../fixtures/profile-image-lifetime";
import { expectDecoded, profileImagePage } from "../../fixtures/profile-images-page";
import { answerTo } from "../../fixtures/settings-pages";
import { expect } from "../../fixtures/test";
import { test } from "../../fixtures/profile-image-wire";

test("F17 (page): native file and drop selection, alert focus, busy cancellation, keyboard close and raw upload", async ({
  imageWorld,
  page,
}, info) => {
  const image = await profileImagePage(imageWorld, info);
  await page.goto("/settings/profile/general");
  await image.probe();
  const opener = page.getByRole("button", { name: "Change avatar", exact: true });
  await opener.focus();
  await page.keyboard.press("Enter");
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Change avatar");
  const input = dialog.getByLabel("Choose or replace image", { exact: true });
  await expect(input).toHaveAttribute("type", "file");
  await expect(input).toHaveAttribute("aria-describedby", /.+-help/);
  await input.setInputFiles({ name: "wrong.txt", mimeType: "text/plain", buffer: Buffer.from("not an image") });
  await expect(dialog.getByRole("alert")).toBeFocused();
  await expect(input).toHaveAttribute("aria-invalid", "true");
  await dropImage(page, undefined, 2);
  await expect(dialog.getByRole("alert")).toHaveText("Choose one image at a time.");
  await expect(dialog.getByRole("alert")).toBeFocused();
  await dropImage(page);
  await expectDecoded(dialog.getByAltText("Selected image preview"));
  await expect(dialog.getByRole("button", { name: "Upload and save", exact: true })).toBeFocused();
  const held = await holdImageBoundary(page, "prepare");
  try {
    await page.keyboard.press("Enter");
    await held.wait();
    await expect(dialog.locator('[aria-live="polite"]')).toHaveText("Preparing upload…");
    await expect(input).toBeDisabled();
    await page.keyboard.press("Escape");
    await expect(dialog).toHaveCount(1);
    await dialog.getByRole("button", { name: "Cancel upload", exact: true }).click();
    await expect(input).toBeFocused();
    await expect(input).toBeEnabled();
    await held.release();
    await page.keyboard.press("Escape");
    await expect(dialog).toHaveCount(0);
    await expect(opener).toBeFocused();
    await page.keyboard.press("Enter");
    await dropImage(page);
    const saved = await answerTo(page, "PATCH", "/api/v0/me", () =>
      dialog.getByRole("button", { name: "Upload and save", exact: true }).click()
    );
    expect(saved.status()).toBe(200);
    await expect(dialog).toHaveCount(0);
    await expect(opener).toBeFocused();
    const id = (await saved.json()).avatar_asset_id;
    await image.persisted(id);
    expect(image.calls("POST", "/api/v0/assets")).toHaveLength(2);
    expect(image.calls("PATCH", "/api/v0/me")).toHaveLength(1);
    expect(image.watch.oldApiRequests).toEqual([]);
    expect(image.watch.pageErrors).toEqual([]);
  } finally {
    await held.release();
  }
});
