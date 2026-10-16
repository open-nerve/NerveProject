import {
  changeProject,
  changeProjectPreferences,
  changeWorkspacePreferences,
  createLabel,
  createProject,
} from "../../fixtures/api";
import { emailFor, shownNameOf } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { cancel, chineseAdmin } from "../../fixtures/chinese";
import { closedWithin, modalDialog } from "../../fixtures/dialogs";
import { workItemFilters } from "../../fixtures/mounts";
import { labelRow } from "../../fixtures/project-pages";
import { expect, test } from "../../fixtures/test";

// S7, the workspace's pages in Chinese (M3 closeout), its project half: a project's pages and their dialogs, the
// filters of the projects and the members, and a project's settings, as an admin whose language is zh-CN sees them.
// The workspace's pages, the onboarding and the invitation are s7-chinese.spec.ts.

test("S7 (page): an admin whose language is Chinese sees a project's header, general, states, labels and members pages and their dialogs in Chinese", async ({
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
  // Web shows its modules and its views, both under its header's more menu, the views the tab it opens on.
  const shown = await changeProject(api, admin.access_token, web.id, { module_view: true, issue_views_view: true });
  expect(shown.status).toBe(200);
  const placed = await changeProjectPreferences(api, admin.access_token, web.id, {
    navigation: { default_tab: "views", hide_in_more_menu: ["modules", "views"] },
  });
  expect(placed.status).toBe(200);
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
  // the extended sidebar, which the button opens, by its heading; its search, focused, is marked
  await sidebar.getByRole("button", { name: "更多", exact: true }).click();
  const extended = page.locator("#extended-project-sidebar-toggle");
  await expect(extended.getByText("项目", { exact: true })).toBeVisible();
  await extended.getByPlaceholder("搜索", { exact: true }).focus();
  await expect(extended.getByPlaceholder("搜索", { exact: true }).locator("..")).not.toHaveCSS("box-shadow", "none");

  // The project's header, the tabbed navigation's: its project switcher's search and its words for no match. Its page,
  // the work items', asks Plane's address of the filters, M4's (P8b spec §5), as P8 sees.
  await page.goto(`/${slug}/projects/${web.id}/issues`);
  // the switcher beside the header's quick actions: the innermost element that holds both (the breadcrumbs of the
  // work items' page, M4's, have a switcher named by the project too)
  const header = page
    .locator("div", { has: page.getByRole("button", { name: "切换快速操作菜单", exact: true }) })
    .filter({ has: page.getByRole("button", { name: "Web", exact: true }) })
    .last();
  await header.getByRole("button", { name: "Web", exact: true }).click();
  await expect(page.getByRole("combobox", { name: "搜索", exact: true })).toBeFocused();
  await page.keyboard.type("zz");
  await expect(page.getByText("未找到匹配项", { exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  // The header's more menu, by its button's name: each hidden tab's actions are named by it in Chinese, the views' (the
  // tab Web opens on) offering to clear the default; the arrow keys reach the modules' actions after the modules.
  const more = page.getByRole("button", { name: "更多标签页", exact: true });
  await expect(more).toBeVisible();
  await more.press("Enter");
  await expect(page.getByRole("menuitem", { name: "显示视图", exact: true })).toHaveCount(1);
  await expect(page.getByRole("menuitem", { name: "取消视图的默认", exact: true })).toHaveCount(1);
  const modules = page.getByRole("menuitem", { name: "模块", exact: true });
  await expect
    .poll(
      async () => {
        await page.keyboard.press("ArrowDown");
        return modules.evaluate((item) => item === document.activeElement);
      },
      { timeout: 5_000 }
    )
    .toBe(true);
  await page.keyboard.press("ArrowDown");
  await expect(page.getByRole("menuitem", { name: "显示模块", exact: true })).toBeFocused();
  await page.keyboard.press("ArrowDown");
  await expect(page.getByRole("menuitem", { name: "将模块设为默认", exact: true })).toBeFocused();
  await page.keyboard.press("Escape");

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
  // The lead's select finds its "none" by the word it shows; its search and its words for no match are Chinese.
  await page.getByRole("heading", { name: "项目负责人" }).locator("xpath=following::button[1]").click();
  await expect(page.getByRole("combobox", { name: "搜索", exact: true })).toBeFocused();
  await page.keyboard.type("无");
  await expect(page.getByRole("option")).toHaveText(["无"]);
  await page.keyboard.type("zz");
  await expect(page.getByText("未找到匹配项", { exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("switch", { name: "访客访问" })).toBeVisible();
  // His row's menu, named by him, offers his leaving.
  await page.getByRole("button", { name: `${shownNameOf(emailFor(testInfo, "admin"))} 的选项`, exact: true }).click();
  await expect(page.getByRole("menuitem", { name: "离开" })).toBeVisible();
  await page.keyboard.press("Escape");

  // The adding of members: a row with no member picked says so, its member select focused and described by it (M3
  // closeout, C11-1); its member select's search and words for no match are Chinese.
  await page.getByRole("button", { name: "添加成员", exact: true }).click();
  await modalDialog(page).getByRole("button", { name: "添加成员", exact: true }).click();
  await expect(modalDialog(page).getByText("请选择成员", { exact: true })).toBeVisible();
  const memberSelect = modalDialog(page).getByRole("button", { name: "选择成员" });
  await expect(memberSelect).toBeFocused();
  await expect(memberSelect).toHaveAttribute("aria-invalid", "true");
  await expect(memberSelect).toHaveAccessibleDescription("请选择成员");
  await memberSelect.click();
  await expect(page.getByRole("combobox", { name: "搜索", exact: true })).toBeFocused();
  await page.keyboard.type("zz");
  await expect(page.getByText("未找到匹配项", { exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  await cancel(page);
  const filters = workItemFilters(slug, web.id);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [filters.failure],
    [filters.request],
    [],
  ]);
  await expectQuietConsole(page, watch, { warnings: Array(6).fill(EMOJI_CHECK_WARNING), errors: [filters.logged] });
});

test("S7 (page): an admin whose language is Chinese filters the projects and the members, and reads a project's settings, its custom range and a toast in Chinese", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { admin, slug } = await chineseAdmin(api, testInfo);
  const page = await signedInPage(admin);
  const watch = await watchPage(page);

  // The project settings of a workspace with no project yet, and the create-workspace page's title.
  await page.goto(`/${slug}/settings/projects`);
  await expect(page.getByText("没有项目", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "开始您的第一个项目" })).toBeVisible();
  await page.goto("/create-workspace");
  await expect(page).toHaveTitle("创建您的工作区");

  // The projects page: the sidebar's heading, the card's date and its members.
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
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
  // their search, which has the focus as they open, is marked
  const filtersSearch = page.getByPlaceholder("搜索", { exact: true }).last();
  await expect(filtersSearch).toBeFocused();
  await expect(filtersSearch.locator("..")).not.toHaveCSS("box-shadow", "none");
  const filters = page.getByRole("button", { name: "我的项目" }).locator("xpath=../..");
  await expect(filters.getByText(/^(网络|负责人|成员|创建日期)$/)).toHaveText(["网络", "负责人", "成员", "创建日期"]);
  await expect(filters.getByRole("button", { name: /^(今天|昨天|最近 7 天|最近 30 天|自定义)$/ })).toHaveCount(5);
  // the custom date's dialog, which the filters open, named by the filter, its select and its options, the range it
  // shows and its buttons in Chinese, stays and leaves them open: closed by Escape, it gives the focus back to the
  // custom date's button in them
  await filters.getByRole("button", { name: "自定义", exact: true }).click();
  const custom = modalDialog(page);
  await expect(custom).toHaveAccessibleName("创建日期");
  await expect(custom.getByRole("button", { name: "创建日期范围", exact: true })).toBeVisible();
  await expect(custom.getByText("晚于：", { exact: true })).toBeVisible();
  await expect(custom.getByText("早于：", { exact: true })).toBeVisible();
  await expect(custom.getByRole("button", { name: "取消", exact: true })).toBeVisible();
  await expect(custom.getByRole("button", { name: "应用", exact: true })).toBeVisible();
  expect(await closedWithin(page)).toBe(false);
  // the select's options, in its list (the calendars' month and year selects hold options too); Escape closes the
  // list alone
  await custom.getByRole("button", { name: "创建日期范围", exact: true }).click();
  await expect(page.getByRole("listbox").getByRole("option")).toHaveText([
    "创建日期早于",
    "创建日期晚于",
    "创建日期范围",
  ]);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(custom).toHaveCount(1);
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);
  await expect(filters.getByRole("button", { name: "自定义", exact: true })).toBeFocused();
  // the caller, in the lead's filter and the members', is 您
  await expect(filters.getByRole("button", { name: / 您$/ })).toHaveCount(2);
  await filtersSearch.fill("公开");
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
  await expect(page.getByRole("button", { name: "移除 公开", exact: true })).toBeVisible();
  await page.getByText("1/1", { exact: true }).hover();
  await expect(page.getByText("1 个项目中有 1 个符合筛选条件。", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "移除 网络", exact: true }).click();
  await expect(page.getByRole("button", { name: "移除 公开", exact: true })).toHaveCount(0);

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

  // A project's settings: each page's title, the members' search; a feature's switch, whose toast says it, in the
  // region of the toasts, named.
  const settings = `/${slug}/settings/projects/${web.id}`;
  await page.goto(settings);
  await expect(page).toHaveTitle("Web - 常规设置");
  await page.goto(`${settings}/members`);
  await expect(page).toHaveTitle("Web - 成员");
  await expect(page.getByPlaceholder("搜索", { exact: true })).toBeVisible();
  await page.goto(`${settings}/features/cycles`);
  await page.getByRole("switch", { name: "启用周期" }).click();
  const toasts = page.getByRole("region", { name: "消息提示" });
  await expect(toasts.getByText("项目功能已更新。", { exact: true })).toBeVisible();

  // The auto-archiving: its months, and its custom range, which says what it takes.
  await page.goto(`${settings}/automations`);
  await page.getByRole("switch", { name: "自动归档已关闭的工作项" }).click();
  await page.getByRole("button", { name: "1 个月", exact: true }).click();
  await page.getByRole("option", { name: "自定义时间范围" }).click();
  const months = modalDialog(page).getByPlaceholder("输入月数");
  await expect(modalDialog(page).getByText("个月", { exact: true })).toBeVisible();
  await months.fill("");
  await modalDialog(page).getByRole("button", { name: "更新", exact: true }).click();
  await expect(modalDialog(page).getByText("请选择 1 到 12 之间的月数。", { exact: true })).toBeVisible();
  await cancel(page);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: Array(8).fill(EMOJI_CHECK_WARNING) });
});
