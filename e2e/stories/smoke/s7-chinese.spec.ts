import type { Page, TestInfo } from "@playwright/test";

import {
  changeWorkspacePreferences,
  createLabel,
  createProject,
  createWorkspace,
  invitationTo,
  invite,
  inviteAndAccept,
  slugFor,
  type Api,
} from "../../fixtures/api";
import { bearer, emailFor, shownNameOf, type AuthTokens } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { closedWithin, modalDialog } from "../../fixtures/dialogs";
import { labelRow } from "../../fixtures/project-pages";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { invitationLinkOf, memberRow } from "../../fixtures/workspace-pages";

// S7, the workspace's pages in Chinese (M3 closeout: the text Plane wrote in English on M3's pages is in the user's
// language): an account whose language is zh-CN sees its pages, menus and dialogs in Chinese.

/** Makes the language of the account of tokens Chinese, the home's tour taken already. */
async function speakChinese(api: Api, tokens: AuthTokens): Promise<void> {
  const chinese = await api.PATCH("/api/v0/me/profile", {
    body: { language: "zh-CN", is_tour_completed: true },
    headers: bearer(tokens.access_token),
  });
  expect(chinese.response.status).toBe(200);
}

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
  await speakChinese(api, admin);
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

test("S7 (page): an admin whose language is Chinese sees the workspace's home, menu, projects, general and members pages and their dialogs in Chinese", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { admin, slug, bobEmail } = await chineseAdmin(api, testInfo);
  const carolEmail = emailFor(testInfo, "carol");
  await invite(api, admin.access_token, slug, [{ email: carolEmail, role: 5 }]);
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
  // The menu names his role in Acme in Chinese too.
  await page.getByRole("button", { name: "打开工作空间切换器" }).click();
  await expect(page.getByRole("menu").getByText("管理员", { exact: true })).toBeVisible();
  await expect(page.getByRole("menuitem", { name: "离开工作区" })).toBeVisible();
  await page.keyboard.press("Escape");

  // The sidebar, by its name: its buttons by theirs; the project navigation's dialog, each mode's description.
  const sidebar = page.getByRole("complementary", { name: "主侧边栏" });
  await expect(sidebar.getByRole("button", { name: "切换侧边栏" })).toBeVisible();
  // the workspace menu's button, one: the chevron beside it, which does the same for the pointer, is hidden
  await expect(sidebar.getByRole("button", { name: /^(收起|展开)工作区菜单$/ })).toHaveCount(1);
  await sidebar.getByRole("button", { name: "项目导航" }).click();
  await expect(modalDialog(page).getByText("功能标签显示为项目下的嵌套项，可以展开和收起。")).toBeVisible();
  await expect(modalDialog(page).getByText("功能标签显示为项目内的横向标签。")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);

  // The projects page: its search, by its name.
  await page.goto(`/${slug}/projects`);
  await expect(page.getByRole("button", { name: "搜索", exact: true })).toBeVisible();

  // The general page: the settings' sidebar and its way back, the size, and the deletion's dialog, the workspace's
  // name in it.
  await page.goto(`/${slug}/settings`);
  await expect(page.getByText("工作区设置", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "返回", exact: true })).toBeVisible();
  await expect(page.getByText("只有我自己", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "删除", exact: true }).click();
  await expect(modalDialog(page).getByText(/^您将要删除工作区 Acme。/)).toBeVisible();
  await expect(modalDialog(page).getByText("输入这个工作区的名称以继续。")).toBeVisible();
  await expect(modalDialog(page).getByText("最后确认，请在下方输入 delete my workspace。")).toBeVisible();
  await cancel(page);

  // The members page: its title, bob's role, carol's invitation's, each row's menu by its name, and the removal's
  // dialog, the member's name in it.
  await page.goto(`/${slug}/settings/members`);
  await expect(page).toHaveTitle("Acme - 成员");
  await expect(memberRow(page, bobEmail).getByRole("button", { name: "成员", exact: true })).toBeVisible();
  const carols = page.locator("div.group", { has: page.getByRole("heading", { name: carolEmail, exact: true }) });
  await expect(carols.getByRole("button", { name: "访客", exact: true })).toBeVisible();
  await expect(carols.getByRole("button", { name: `${carolEmail} 的选项`, exact: true })).toBeVisible();
  await memberRow(page, bobEmail)
    .getByRole("button", { name: `${shownNameOf(bobEmail)} 的选项`, exact: true })
    .click();
  await page.getByRole("menuitem", { name: "移除", exact: true }).click();
  await expect(modalDialog(page).getByRole("heading")).toHaveText(`移出 ${shownNameOf(bobEmail)}？`);
  await expect(modalDialog(page).getByText(/^确定要将 .+ 移出工作区吗？/)).toBeVisible();
  await cancel(page);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: Array(4).fill(EMOJI_CHECK_WARNING) });
});

test("S7 (page): an admin whose language is Chinese sees a project's general, states, labels and members pages and their dialogs in Chinese", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { admin, slug } = await chineseAdmin(api, testInfo);
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  await createProject(api, admin.access_token, slug, {
    name: "Ops",
    identifier: "OPS",
    logo_props: { in_use: "emoji", emoji: { value: "128640" } },
  });
  await createProject(api, admin.access_token, slug, {
    name: "Lab",
    identifier: "LAB",
    logo_props: { in_use: "icon", icon: { name: "rocket" } },
  });
  await changeWorkspacePreferences(api, admin.access_token, slug, {
    navigation_control_preference: "TABBED",
    navigation_project_limit: 2,
  });
  const bug = await createLabel(api, admin.access_token, web.id, { name: "Bug" });
  await createLabel(api, admin.access_token, web.id, { name: "UI", parent_id: bug.id });
  const page = await signedInPage(admin);
  const watch = await watchPage(page);

  // The projects page: the sidebar shows two projects, as its limit says, Lab and Ops, made last, which it puts first,
  // and offers the rest by a button its words name. Each link, its tabs beside it, is named by its project's name
  // alone, its logo no part of it: Ops's emoji, and Lab's icon, a word the icon font draws once it is in.
  await page.goto(`/${slug}/projects`);
  const sidebar = page.getByRole("complementary", { name: "主侧边栏" });
  await expect(sidebar.getByRole("link", { name: "Ops", exact: true })).toBeVisible();
  const lab = sidebar.getByRole("link").filter({ has: page.locator(".material-symbols-rounded") });
  await expect(lab).toHaveAccessibleName("Lab");
  await expect(sidebar.getByRole("button", { name: "更多", exact: true })).toHaveText("更多");
  // the extended sidebar, which the button opens, by its heading
  await sidebar.getByRole("button", { name: "更多", exact: true }).click();
  await expect(page.locator("#extended-project-sidebar-toggle").getByText("项目", { exact: true })).toBeVisible();

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

  // The states page: a group by its name, the states not the default offered to be it, each named by its state.
  await page.goto(`/${slug}/settings/projects/${web.id}/states`);
  await expect(page.getByRole("button", { name: "已开始", exact: true })).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByRole("button", { name: /^设为默认：/ })).toHaveCount(4);
  await expect(page.getByRole("button", { name: "设为默认：Todo", exact: true })).toHaveText("设为默认");

  // The labels page: a group's menu, named by the group, the menu of a label in it, and the deletion's dialog, the
  // group's name in it.
  await page.goto(`/${slug}/settings/projects/${web.id}/labels`);
  await labelRow(page, "UI").getByRole("button", { name: "选项" }).click();
  await expect(page.getByRole("menuitem")).toHaveText(["移出组", "编辑标签"]);
  await page.keyboard.press("Escape");
  await labelRow(page, "Bug").getByRole("button", { name: "Bug 的选项", exact: true }).click();
  await expect(page.getByRole("menuitem")).toHaveText(["编辑标签", "删除标签"]);
  await page.getByRole("menuitem", { name: "删除标签" }).click();
  await expect(modalDialog(page).getByRole("heading")).toHaveText("删除标签");
  await expect(modalDialog(page).getByText(/^确定要删除 Bug 吗？/)).toBeVisible();
  await cancel(page);

  // The members page: the member defaults.
  await page.goto(`/${slug}/settings/projects/${web.id}/members`);
  await expect(page.getByText("项目负责人", { exact: true })).toBeVisible();
  await expect(page.getByText("默认负责人", { exact: true })).toBeVisible();
  // The lead's select finds its "none" by the word it shows.
  await page.getByRole("heading", { name: "项目负责人" }).locator("xpath=following::button[1]").click();
  await page.keyboard.type("无");
  await expect(page.getByRole("option")).toHaveText(["无"]);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("switch", { name: "访客访问" })).toBeVisible();
  // His row's menu, named by him, offers his leaving.
  await page.getByRole("button", { name: `${shownNameOf(emailFor(testInfo, "admin"))} 的选项`, exact: true }).click();
  await expect(page.getByRole("menuitem", { name: "离开" })).toBeVisible();
  await page.keyboard.press("Escape");

  // The adding of members: a row with no member picked says so.
  await page.getByRole("button", { name: "添加成员", exact: true }).click();
  await modalDialog(page).getByRole("button", { name: "添加成员", exact: true }).click();
  await expect(modalDialog(page).getByText("请选择成员", { exact: true })).toBeVisible();
  await cancel(page);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: Array(5).fill(EMOJI_CHECK_WARNING) });
});

test("S7 (page): an admin whose language is Chinese filters the projects and the members in Chinese", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { admin, slug } = await chineseAdmin(api, testInfo);
  const page = await signedInPage(admin);
  const watch = await watchPage(page);

  // The projects page: the sidebar's heading, the card's date and its members.
  await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  await page.goto(`/${slug}/projects`);
  const navigation = page.getByRole("complementary", { name: "主侧边栏" }).getByRole("button", { name: "项目导航" });
  await expect(navigation.locator("xpath=../preceding-sibling::span")).toHaveText("项目");
  await expect(page.getByText(/^创建于 /)).toBeVisible();
  // The filters, by the keyboard: Enter on their button opens them, which it says. Each filter by its name, the
  // accesses searched as the page names them.
  const filtersButton = page.getByRole("button", { name: "筛选", exact: true });
  await filtersButton.focus();
  await page.keyboard.press("Enter");
  await expect(filtersButton).toHaveAttribute("aria-expanded", "true");
  const filters = page.getByRole("button", { name: "我的项目" }).locator("xpath=../..");
  await expect(filters.getByText(/^(网络|负责人|成员|创建日期)$/)).toHaveText(["网络", "负责人", "成员", "创建日期"]);
  await expect(filters.getByRole("button", { name: /^(今天|昨天|最近 7 天|最近 30 天|自定义)$/ })).toHaveCount(5);
  // the custom date's dialog, which the filters open, named by the filter, stays and leaves them open: closed by
  // Escape, it gives the focus back to the custom date's button in them
  await filters.getByRole("button", { name: "自定义", exact: true }).click();
  await expect(modalDialog(page).getByRole("button", { name: /^创建日期/ })).toBeVisible();
  expect(await closedWithin(page)).toBe(false);
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);
  await expect(filters.getByRole("button", { name: "自定义", exact: true })).toBeFocused();
  // the caller, in the lead's filter and the members', is 您
  await expect(filters.getByRole("button", { name: / 您$/ })).toHaveCount(2);
  await page.getByPlaceholder("搜索", { exact: true }).last().fill("公开");
  await expect(page.getByRole("button", { name: "公开", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "私有", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "公开", exact: true }).click();
  // Escape closes them, the focus back on their button, which says so
  await page.keyboard.press("Escape");
  await expect(filtersButton).toBeFocused();
  await expect(filtersButton).toHaveAttribute("aria-expanded", "false");
  // opened again, their search emptied, Tab past their last filter closes them, the focus moving on to the button
  // after theirs
  await page.keyboard.press("Enter");
  await page.getByPlaceholder("搜索", { exact: true }).last().fill("");
  await filters.getByRole("button", { name: "自定义", exact: true }).focus();
  await page.keyboard.press("Tab");
  await expect(page.getByRole("button", { name: "添加项目", exact: true })).toBeFocused();
  await expect(filtersButton).toHaveAttribute("aria-expanded", "false");
  // The bar of the filters applied: the filter by its name, each button named by what it removes, and the count.
  await expect(page.getByRole("button", { name: "移除公开", exact: true })).toBeVisible();
  await page.getByText("1/1", { exact: true }).hover();
  await expect(page.getByText("1 个项目中有 1 个符合筛选条件。", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "移除网络", exact: true }).click();
  await expect(page.getByRole("button", { name: "移除公开", exact: true })).toHaveCount(0);

  // The archived projects: the header's breadcrumb says so.
  await page.goto(`/${slug}/projects/archives`);
  await expect(page.getByText("已归档", { exact: true })).toHaveCount(1);

  // The members page: one button names its filter, a popover whose roles' heading and roles have their names.
  await page.goto(`/${slug}/settings/members`);
  await expect(page.getByRole("button", { name: "筛选" })).toHaveCount(1);
  await page.getByRole("button", { name: "筛选" }).click();
  // the filter, its button and its popover
  const roleFilter = page.getByRole("button", { name: "筛选" }).locator("xpath=..");
  await expect(roleFilter.getByRole("button", { name: "角色", exact: true })).toHaveAttribute("aria-expanded", "true");
  const roles = roleFilter.getByRole("button", { name: /^(管理员|成员|访客|已停用)$/ });
  await expect(roles).toHaveText(["管理员", "成员", "访客", "已停用"]);
  await page.keyboard.press("Escape");
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: Array(3).fill(EMOJI_CHECK_WARNING) });
});

test("S7 (page): one whose language is Chinese reads her invitation to Acme, as a guest, in Chinese", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { admin, slug } = await chineseAdmin(api, testInfo);
  const carolEmail = emailFor(testInfo, "carol");
  const carol = await registerOnboarded(api, carolEmail);
  await speakChinese(api, carol);
  await invite(api, admin.access_token, slug, [{ email: carolEmail, role: 5 }]);
  const page = await signedInPage(carol);
  const watch = await watchPage(page);

  await page.goto(invitationLinkOf(await invitationTo(api, admin.access_token, slug, carolEmail)));
  await expect(page.getByText("您受邀以 访客 的身份加入 Acme。", { exact: true })).toBeVisible();
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
