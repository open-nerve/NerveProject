import { bearer } from "../../fixtures/auth";
import { modalDialog } from "../../fixtures/dialogs";
import { pageFileStory } from "../../fixtures/file-story";
import { profileImagePage, redPixel } from "../../fixtures/profile-images-page";
import { answerTo } from "../../fixtures/settings-pages";
import { expect } from "../../fixtures/test";

pageFileStory(
  "F7 (page): a prepared cover binds in one project POST, then settings replace, select a stable preset and clear it",
  async (world, info) => {
    info.setTimeout(90_000);
    const { api, page, db, baseURL } = world;
    const image = await profileImagePage(world, info);
    const slug = image.workspace.slug;
    const createPath = `/api/v0/workspaces/${slug}/projects`;
    await page.goto(`${baseURL}/${slug}/projects`);
    await image.probe();
    await page.getByRole("button", { name: "Add Project", exact: true }).click();
    await page.locator("#name").fill("Image project");
    await page.locator("#identifier").fill("IMAGE");
    await page.getByRole("button", { name: "Change cover", exact: true }).click();
    await expect(page.getByRole("button", { name: /^Cover \d+$/ })).toHaveCount(29);
    await expect
      .poll(() =>
        page
          .getByRole("button", { name: /^Cover \d+$/ })
          .locator("img")
          .evaluateAll((images) =>
            images.every((element) => element instanceof HTMLImageElement && element.naturalWidth === 1600)
          )
      )
      .toBe(true);
    await page.getByRole("button", { name: "Upload image", exact: true }).click();
    await image.select();
    await page.getByRole("button", { name: "Upload and save", exact: true }).click();
    await expect(page.getByLabel("Choose or replace image", { exact: true })).toHaveCount(0);
    const id = await image.transfer(0, "PROJECT_COVER", { kind: "new_project", id: image.workspace.id });
    expect(image.calls("POST", createPath)).toHaveLength(0);
    const created = await answerTo(page, "POST", createPath, () =>
      page.getByRole("button", { name: "Create project", exact: true }).dblclick()
    );
    expect(created.status()).toBe(201);
    const project = await created.json();
    expect(project).toMatchObject({ name: "Image project", cover_image_asset_id: id, cover_builtin_id: null });
    expect(image.calls("POST", createPath)).toHaveLength(1);
    expect(image.bodies("POST", createPath)[0]).toMatchObject({ cover_image_asset_id: id });
    const updatePath = `/api/v0/projects/${project.id}`;
    await image.observeOwner("PATCH", updatePath);
    expect(image.calls("PATCH", updatePath)).toHaveLength(0);
    await image.persisted(id);
    expect(
      await db.query("SELECT count(*)::int AS n FROM projects WHERE workspace_id=$1 AND identifier='IMAGE'", [
        image.workspace.id,
      ])
    ).toEqual([{ n: 1 }]);
    await page.goto(`${baseURL}/${slug}/settings/projects/${project.id}`);
    await page.locator("#name").fill("Edited image project");
    await page.getByRole("button", { name: "Change cover", exact: true }).click();
    await page.getByRole("button", { name: "Upload image", exact: true }).click();
    await image.select(redPixel);
    await page.getByRole("button", { name: "Upload and save", exact: true }).click();
    await expect(modalDialog(page)).toHaveCount(0);
    const replacement = await image.transfer(1, "PROJECT_COVER", { kind: "project", id: project.id }, redPixel);
    expect(image.calls("PATCH", updatePath)).toHaveLength(0);
    await expect(page.locator("#name")).toHaveValue("Edited image project");
    expect(
      (
        await answerTo(page, "PATCH", updatePath, () =>
          page.getByRole("button", { name: "Update project", exact: true }).dblclick()
        )
      ).status()
    ).toBe(200);
    expect(image.calls("PATCH", updatePath)).toHaveLength(1);
    expect(image.bodies("PATCH", updatePath)[0]).toMatchObject({
      name: "Edited image project",
      cover_image_asset_id: replacement,
    });
    await image.persisted(replacement, redPixel);
    await page.getByRole("button", { name: "Change cover", exact: true }).click();
    await page.getByRole("button", { name: "Cover 29", exact: true }).click();
    const preset = await answerTo(page, "PATCH", updatePath, () =>
      page.getByRole("button", { name: "Update project", exact: true }).click()
    );
    expect(await preset.json()).toMatchObject({
      cover_builtin_id: "cover-29",
      cover_image_asset_id: null,
      cover_image_url: null,
    });
    await page.goto(`${baseURL}/${slug}/projects`);
    const card = page
      .locator("div")
      .filter({ has: page.getByRole("link", { name: "Edited image project", exact: true }) })
      .filter({ has: page.locator('img[src*="cover-29-"]') })
      .last();
    await expect(card.locator('img[src*="cover-29-"]')).toBeVisible();
    await page.goto(`${baseURL}/${slug}/settings/projects/${project.id}`);
    await page.getByRole("button", { name: "Change cover", exact: true }).click();
    await page.getByRole("button", { name: "Random cover", exact: true }).click();
    const random = await answerTo(page, "PATCH", updatePath, () =>
      page.getByRole("button", { name: "Update project", exact: true }).click()
    );
    expect((await random.json()).cover_builtin_id).toMatch(/^cover-(0[1-9]|1\d|2\d)$/);
    await page.getByRole("button", { name: "Change cover", exact: true }).click();
    await page.getByRole("button", { name: "Remove cover", exact: true }).click();
    expect(
      (
        await answerTo(page, "PATCH", updatePath, () =>
          page.getByRole("button", { name: "Update project", exact: true }).click()
        )
      ).status()
    ).toBe(200);
    expect(image.bodies("PATCH", updatePath).at(-1)).toMatchObject({
      cover_image_asset_id: null,
      cover_builtin_id: null,
    });
    expect(image.calls("POST", "/api/v0/assets")).toHaveLength(2);
    const final = await api.GET("/api/v0/projects/{project_id}", {
      params: { path: { project_id: project.id } },
      headers: bearer(image.owner.token),
    });
    expect(final.data).toMatchObject({ cover_image_asset_id: null, cover_builtin_id: null, cover_image_url: null });
    expect(
      await db.query("SELECT cover_image_asset_id, cover_builtin_id FROM projects WHERE id=$1", [project.id])
    ).toEqual([{ cover_image_asset_id: null, cover_builtin_id: null }]);
    await image.finish();
  }
);
