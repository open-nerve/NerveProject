// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { LogoProps } from "@nerve/api-client";
import type { IFavorite, TLogoProps } from "@nerve/types";
import { its } from "@/store/user/fake-permissions";
import { ME, loadViews, viewOf, viewsTab } from "@/store/views/fake-views";
import { useFavoriteItemDetails } from "./use-favorite-item-details";

// A view's favorite in the sidebar (P6a spec 2.9, R24): it shows the name and the icon of the view as the views' store
// holds it (RootStore.views, getViewById), and its own name and icon while the store holds no such view (not read
// yet, deleted here, or of a project the store no longer gives). The favorites' writes are P7's. The store is a views
// store over a fake nerve that answers each request when the test says (fake-views.ts); the store hooks run as plain
// functions (fake-store-hooks.ts).

type Tab = Awaited<ReturnType<typeof viewsTab>>;
const held = vi.hoisted(() => ({ tab: undefined as unknown as Tab }));
vi.mock("@/hooks/store/use-views", () => ({ useViews: () => held.tab.views }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => held.tab.projects }));

beforeEach(async () => {
  vi.useFakeTimers();
  held.tab = await viewsTab(its.member);
});
afterEach(() => {
  vi.useRealTimers();
});

/** The view's icon as nerve gives it now, and the one the favorite was given when it was made. */
const rocket: LogoProps = { in_use: "emoji", emoji: { value: "128640" } };
const star: TLogoProps = { in_use: "emoji", emoji: { value: "11088" } };

/** The caller's favorite of web's view of that id, named and with the icon nerve gave it when it was made. */
function favoriteOf(viewId: string): IFavorite {
  const { acme, web } = held.tab;
  return {
    id: `f-${viewId}`,
    name: "",
    entity_type: "view",
    entity_data: { id: viewId, name: "Old board", logo_props: star },
    is_folder: false,
    sort_order: 1,
    parent: null,
    entity_identifier: viewId,
    children: [],
    project_id: web.id,
    sequence: 1,
    workspace_id: acme.id,
  };
}

it("a view's favorite shows the name and the icon of the view the views' store holds", async () => {
  const { acme, nerve, views, web } = held.tab;
  const board = viewOf(web, "Sprint board", ME, { logo_props: rocket });
  await loadViews(nerve, views, web, [board]);

  const { itemTitle, itemIcon, itemLink } = useFavoriteItemDetails(acme.slug, favoriteOf(board.id));

  expect(itemTitle).toBe("Sprint board");
  expect(itemIcon?.props).toEqual({ type: "view", logo: rocket });
  expect(itemLink).toBe(`/${acme.slug}/projects/${web.id}/views/${board.id}`);
});

it("a view's favorite shows its own name and icon while the views' store holds no such view", async () => {
  const { acme, nerve, views, web } = held.tab;
  const other = viewOf(web, "Bugs", ME, { logo_props: rocket });
  const gone = favoriteOf("v-web-gone");

  // Not read yet, then read without it.
  expect(useFavoriteItemDetails(acme.slug, gone)).toMatchObject({ itemTitle: "Old board" });
  await loadViews(nerve, views, web, [other]);
  const { itemTitle, itemIcon } = useFavoriteItemDetails(acme.slug, gone);

  expect(itemTitle).toBe("Old board");
  expect(itemIcon?.props).toEqual({ type: "view", logo: star });
});
