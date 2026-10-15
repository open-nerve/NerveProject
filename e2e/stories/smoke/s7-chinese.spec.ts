import { invitationTo, invite } from "../../fixtures/api";
import { emailFor, register, shownNameOf } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { cancel, chineseAdmin, speakChinese } from "../../fixtures/chinese";
import { modalDialog } from "../../fixtures/dialogs";
import { registerOnboarded, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { askToEndMembership, invitationLinkOf, invitationRow, memberRow } from "../../fixtures/workspace-pages";

// S7, the workspace's pages in Chinese (M3 closeout: the text Plane wrote in English on M3's pages is in the user's
// language): an account whose language is zh-CN sees its pages, menus and dialogs in Chinese. This file has the
// workspace's pages, the onboarding and the invitation; a project's pages and the filters are
// s7-chinese-project.spec.ts.

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
  // The menu, by its button's name (工作区, as everywhere), names his role in Acme in Chinese too.
  await page.getByRole("button", { name: "打开工作区切换器", exact: true }).click();
  await expect(page.getByRole("menu").getByText("管理员", { exact: true })).toBeVisible();
  await expect(page.getByRole("menuitem", { name: "离开工作区" })).toBeVisible();
  await page.keyboard.press("Escape");

  // The sidebar, by its name: its buttons by theirs; the project navigation's dialog, each mode's description.
  const sidebar = page.getByRole("complementary", { name: "主侧边栏" });
  await expect(sidebar.getByRole("button", { name: "切换侧边栏" })).toBeVisible();
  // the workspace menu's heading, one button named by its words: the chevron beside it, which does the same for the
  // pointer, is hidden
  const workspaceMenu = sidebar.getByRole("button", { name: "工作区", exact: true });
  await expect(workspaceMenu).toHaveCount(1);
  await expect(workspaceMenu).toHaveAttribute("aria-expanded", "true");
  await sidebar.getByRole("button", { name: "项目导航" }).click();
  await expect(modalDialog(page).getByText("功能标签显示为项目下的嵌套项，可以展开和收起。")).toBeVisible();
  await expect(modalDialog(page).getByText("功能标签显示为项目内的横向标签。")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);

  // The projects page: its search, by its name. Its box, closed, is no Tab stop; Enter opens it with the focus in it,
  // marked, and Escape closes it, the focus back on the button; so does its clear button, by its name.
  await page.goto(`/${slug}/projects`);
  const searchButton = page.getByRole("button", { name: "搜索", exact: true });
  const searchBox = page.getByRole("textbox", { name: "搜索", exact: true });
  await expect(searchButton).toBeVisible();
  await searchButton.focus();
  await page.keyboard.press("Tab");
  await expect(searchBox).not.toBeFocused();
  await searchButton.focus();
  await page.keyboard.press("Enter");
  await expect(searchBox).toBeFocused();
  await expect(searchBox.locator("..")).not.toHaveCSS("box-shadow", "none");
  await page.keyboard.press("Escape");
  await expect(searchButton).toBeFocused();
  await page.keyboard.press("Enter");
  await page.keyboard.type("Web");
  const clear = page.getByRole("button", { name: "清除搜索", exact: true });
  await expect(clear).toBeVisible();
  await clear.click();
  await expect(searchButton).toBeFocused();

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
  const carols = invitationRow(page, carolEmail);
  await expect(carols.getByRole("button", { name: "访客", exact: true })).toBeVisible();
  await expect(carols.getByRole("button", { name: `${carolEmail} 的选项`, exact: true })).toBeVisible();
  await expect(
    memberRow(page, bobEmail).getByRole("button", { name: `${shownNameOf(bobEmail)} 的选项`, exact: true })
  ).toBeVisible();
  await askToEndMembership(page, bobEmail, "移除");
  await expect(modalDialog(page).getByRole("heading")).toHaveText(`移除 ${shownNameOf(bobEmail)}？`);
  await expect(modalDialog(page).getByText(/^确定要从工作区中移除 .+ 吗？/)).toBeVisible();
  await cancel(page);
  // The invitation's dialog, by its name: its button, while nerve has not answered, says so as the app's busy buttons
  // do.
  await page.getByRole("button", { name: "添加成员", exact: true }).click();
  await expect(modalDialog(page)).toHaveAccessibleName("邀请人员协作");
  await modalDialog(page).locator('[id="invitations.0.email"]').fill(emailFor(testInfo, "dave"));
  const sending = await sentHeld(page, "POST", `/api/v0/workspaces/${slug}/invitations`, () =>
    modalDialog(page).getByRole("button", { name: "发送邀请", exact: true }).click()
  );
  await expect(modalDialog(page).getByRole("button", { name: "发送中", exact: true })).toBeVisible();
  expect((await sending.release()).status()).toBe(201);
  await expect(modalDialog(page)).toHaveCount(0);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: Array(4).fill(EMOJI_CHECK_WARNING) });
});

test("S7 (page): one whose language is Chinese, not onboarded yet, reads the onboarding's title in Chinese", async ({
  api,
  signedInPage,
}, testInfo) => {
  const dave = await register(api, emailFor(testInfo, "dave"));
  await speakChinese(api, dave);
  const page = await signedInPage(dave);
  const watch = await watchPage(page);

  await page.goto("/onboarding");
  await expect(page).toHaveTitle("新手引导");
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
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
