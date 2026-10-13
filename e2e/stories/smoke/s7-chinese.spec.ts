import type { Page, TestInfo } from "@playwright/test";

import { createProject, createWorkspace, inviteAndAccept, slugFor, type Api } from "../../fixtures/api";
import { bearer, emailFor, shownNameOf, type AuthTokens } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { modalDialog } from "../../fixtures/dialogs";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { memberRow } from "../../fixtures/workspace-pages";

// S7, the workspace's pages in Chinese (M3 closeout: the text Plane wrote in English on M3's pages is in the user's
// language): an account whose language is zh-CN sees its pages, menus and dialogs in Chinese.

/**
 * An admin of this run of the test whose language is Chinese, the home's tour taken already, and Acme, his workspace,
 * of the size "Just myself", with bob, a member.
 */
async function chineseAdmin(
  api: Api,
  testInfo: TestInfo
): Promise<{ admin: AuthTokens; slug: string; bobEmail: string }> {
  const admin = await registerOnboarded(api, emailFor(testInfo, "admin"));
  const bobEmail = emailFor(testInfo, "bob");
  const bob = await registerOnboarded(api, bobEmail);
  const chinese = await api.PATCH("/api/v0/me/profile", {
    body: { language: "zh-CN", is_tour_completed: true },
    headers: bearer(admin.access_token),
  });
  expect(chinese.response.status).toBe(200);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug, organization_size: "Just myself" });
  await inviteAndAccept(api, admin.access_token, slug, { email: bobEmail, token: bob.access_token }, 15);
  return { admin, slug, bobEmail };
}

/** Closes the one modal dialog page shows by its Cancel (取消). */
async function cancel(page: Page): Promise<void> {
  await modalDialog(page).getByRole("button", { name: "取消" }).click();
  await expect(modalDialog(page)).toHaveCount(0);
}

test("S7 (page): an admin whose language is Chinese sees the workspace's home, menu, general and members pages and their dialogs in Chinese", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { admin, slug, bobEmail } = await chineseAdmin(api, testInfo);
  const page = await signedInPage(admin);
  const watch = await watchPage(page);

  // The home's greeting, his name in it, the search's placeholder, the keyboard shortcuts' dialog (its title, which
  // names it, and its search), the workspace menu's leaving.
  await page.goto(`/${slug}`);
  const greeting = new RegExp(`^(早上好|下午好|晚上好)，${shownNameOf(emailFor(testInfo, "admin"))}$`);
  await expect(page.getByRole("heading", { level: 2 })).toHaveText(greeting);
  await expect(page.getByPlaceholder("搜索命令...")).toBeVisible();
  await page.keyboard.press("ControlOrMeta+/");
  await expect(modalDialog(page)).toHaveAccessibleName("键盘快捷键");
  await expect(modalDialog(page).getByRole("textbox", { name: "搜索快捷键" })).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);
  await page.getByRole("button", { name: "打开工作空间切换器" }).click();
  await expect(page.getByRole("menuitem", { name: "离开工作区" })).toBeVisible();
  await page.keyboard.press("Escape");

  // The sidebar, by its name; the project navigation's dialog, each mode's description.
  await page.getByRole("complementary", { name: "主侧边栏" }).getByRole("button", { name: "项目导航" }).click();
  await expect(modalDialog(page).getByText("功能标签显示为项目下的嵌套项，可以展开和收起。")).toBeVisible();
  await expect(modalDialog(page).getByText("功能标签显示为项目内的横向标签。")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);

  // The general page: the settings' sidebar, the size, and the deletion's dialog, the workspace's name in it.
  await page.goto(`/${slug}/settings`);
  await expect(page.getByText("工作区设置", { exact: true })).toBeVisible();
  await expect(page.getByText("只有我自己", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "删除", exact: true }).click();
  await expect(modalDialog(page).getByText(/^您将要删除工作区 Acme。/)).toBeVisible();
  await expect(modalDialog(page).getByText("输入这个工作区的名称以继续。")).toBeVisible();
  await expect(modalDialog(page).getByText("最后确认，请在下方输入 delete my workspace。")).toBeVisible();
  await cancel(page);

  // The members page: its title, and the removal's dialog, the member's name in it.
  await page.goto(`/${slug}/settings/members`);
  await expect(page).toHaveTitle("Acme - 成员");
  await memberRow(page, bobEmail).locator("button").first().click();
  await page.getByRole("button", { name: "移除", exact: true }).click();
  await expect(modalDialog(page).getByRole("heading")).toHaveText(`移出 ${shownNameOf(bobEmail)}？`);
  await expect(modalDialog(page).getByText(/^确定要将 .+ 移出工作区吗？/)).toBeVisible();
  await cancel(page);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: Array(3).fill(EMOJI_CHECK_WARNING) });
});

test("S7 (page): an admin whose language is Chinese sees a project's general and members pages and their dialogs in Chinese", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { admin, slug } = await chineseAdmin(api, testInfo);
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const page = await signedInPage(admin);
  const watch = await watchPage(page);

  // The general page: the settings' sidebar, the deletion's description, the archive's and the deletion's dialogs,
  // the project's name in them.
  await page.goto(`/${slug}/settings/projects/${web.id}`);
  await expect(page.getByText("项目设置", { exact: true })).toBeVisible();
  await expect(page.getByText("删除项目时，项目中的全部数据和资源将被永久删除，且无法恢复。")).toBeVisible();
  await page.getByRole("button", { name: "归档", exact: true }).click();
  await expect(modalDialog(page).getByRole("heading")).toHaveText("归档 Web");
  await cancel(page);
  await page.getByRole("button", { name: "删除", exact: true }).click();
  await expect(modalDialog(page).getByRole("heading")).toHaveText("删除项目");
  await expect(modalDialog(page).getByText(/^确定要删除项目 Web 吗？/)).toBeVisible();
  await cancel(page);

  // The members page: the member defaults.
  await page.goto(`/${slug}/settings/projects/${web.id}/members`);
  await expect(page.getByText("项目负责人", { exact: true })).toBeVisible();
  await expect(page.getByText("默认负责人", { exact: true })).toBeVisible();
  await expect(page.getByRole("switch", { name: "访客访问" })).toBeVisible();
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: Array(2).fill(EMOJI_CHECK_WARNING) });
});
