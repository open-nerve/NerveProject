// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, expect, it, vi } from "vitest";
import type { IFavorite } from "@nerve/types";
import { FavoriteFolder } from "./favorite-folder";
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/hooks/store/use-favorite", () => ({ useFavorite: () => ({ fetchGroupedFavorites: vi.fn() }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const favorite: IFavorite = {
  id: "folder",
  name: "Saved",
  entity_type: "folder",
  entity_data: { name: "Saved" },
  is_folder: true,
  sort_order: 1,
  parent: null,
  children: [],
  project_id: null,
  sequence: 1,
  workspace_id: "ws",
};
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
afterEach(async () => {
  await act(async () => root?.unmount());
  host?.remove();
});
it("keeps the pointer's drag grip out of the folder's native disclosure button", async () => {
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await act(async () =>
    root.render(
      <FavoriteFolder
        favorite={favorite}
        isLastChild
        handleRemoveFromFavorites={vi.fn()}
        handleRemoveFromFavoritesFolder={vi.fn()}
        handleDrop={vi.fn()}
      />
    )
  );
  expect(host.querySelector("button button")).toBeNull();
  const trigger = [...host.querySelectorAll("button")].find((button) => button.textContent === "Saved")!;
  expect(trigger).toBeDefined();
  expect(trigger.tabIndex).toBe(0);
  await act(async () => {
    trigger.focus();
    trigger.dispatchEvent(new KeyboardEvent("keydown", { key: " ", bubbles: true, cancelable: true }));
  });
  expect(trigger.getAttribute("aria-expanded")).toBe("true");
});
