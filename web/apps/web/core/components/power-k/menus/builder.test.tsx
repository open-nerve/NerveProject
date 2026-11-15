/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeAll, expect, it, vi } from "vitest";
import { initPromise, setLanguage } from "@nerve/i18n";
import { PowerKMenuBuilder } from "./builder";
import { PowerKCyclesMenu } from "./cycles";
import { PowerKLabelsMenu } from "./labels";
import { PowerKModulesMenu } from "./modules";
import { PowerKProjectsMenu } from "./projects";
import { PowerKSettingsMenu } from "./settings";
import { PowerKViewsMenu } from "./views";
import { PowerKWorkspacesMenu } from "./workspaces";

// A menu of the command palette with no items (P7c spec 2.3, R4) says so by the text of the key its builder is given,
// in the page's language, and a builder given none by that of power_k.search_menu.no_results. The texts are the
// locales' own (the real @nerve/i18n, in both languages): a menu that shows a key, or English in Chinese, fails here.

// A workspace's logo, which a menu with no items does not show, loads the session's stores as it is imported.
vi.mock("@/components/workspace/logo", () => ({ WorkspaceLogo: () => null }));

beforeAll(async () => {
  await initPromise;
});

const none = () => {};
const own = (item: string) => item;

/** The text a menu shows in a language. */
async function shown(language: "en" | "zh-CN", menu: ReactElement) {
  await setLanguage(language);
  return renderToStaticMarkup(menu).replace(/<[^>]*>/g, "");
}

it.each([
  ["en", "No results found", "No cycles found"],
  ["zh-CN", "未找到结果", "未找到周期"],
] as const)(
  "with no items the builder shows, in %s, the text of the key it is given, or of no_results given none",
  async (language, fallback, cycles) => {
    const withKey = (
      <PowerKMenuBuilder<string>
        items={[]}
        getKey={own}
        getValue={own}
        getLabel={own}
        onSelect={none}
        emptyTextKey="power_k.menus.no_cycles"
      />
    );
    const withNone = (
      <PowerKMenuBuilder<string> items={[]} getKey={own} getValue={own} getLabel={own} onSelect={none} />
    );
    expect(await shown(language, withKey)).toBe(cycles);
    expect(await shown(language, withNone)).toBe(fallback);
  }
);

it.each([
  ["cycles", () => <PowerKCyclesMenu cycles={[]} onSelect={none} />, "No cycles found", "未找到周期"],
  ["modules", () => <PowerKModulesMenu modules={[]} onSelect={none} />, "No modules found", "未找到模块"],
  ["labels", () => <PowerKLabelsMenu labels={[]} onSelect={none} />, "No labels found", "未找到标签"],
  ["views", () => <PowerKViewsMenu views={[]} onSelect={none} />, "No views found", "未找到视图"],
  ["projects", () => <PowerKProjectsMenu projects={[]} onSelect={none} />, "No projects found", "未找到项目"],
  ["settings", () => <PowerKSettingsMenu settings={[]} onSelect={none} />, "No settings found", "未找到设置"],
  ["workspaces", () => <PowerKWorkspacesMenu workspaces={[]} onSelect={none} />, "No workspaces found", "未找到工作区"],
] as const)("the %s menu with no items says so in the page's language", async (_, menu, en, zh) => {
  expect(await shown("en", menu())).toBe(en);
  expect(await shown("zh-CN", menu())).toBe(zh);
});

/** Never rendered: builders given a text, and a key of another namespace's texts, where they take a palette's key. */
function given() {
  return [
    <PowerKMenuBuilder<string>
      key="text"
      items={[]}
      getKey={own}
      getValue={own}
      getLabel={own}
      onSelect={none}
      // @ts-expect-error the builder takes a key of the palette's texts (power-k.json), not a text
      emptyTextKey="No cycles found"
    />,
    <PowerKMenuBuilder<string>
      key="other"
      items={[]}
      getKey={own}
      getValue={own}
      getLabel={own}
      onSelect={none}
      // @ts-expect-error nor a key of another namespace's texts
      emptyTextKey="common.cycles"
    />,
  ];
}

it("takes a key of the palette's texts for its empty text, not a text", () => {
  expect(given).toBeTypeOf("function");
});
