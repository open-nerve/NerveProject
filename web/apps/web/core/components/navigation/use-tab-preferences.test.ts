/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectNavigation } from "@nerve/api-client";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import type { NavigationChange } from "@/store/project/preferences.store";
import { useTabPreferences } from "./use-tab-preferences";

// The tab bar the project's header and the sidebar show (M3 design 3.18), and its changes (7.1): nerve's
// ProjectNavigation, the caller's once the project wrapper has fetched it. The hook runs as a plain function, outside
// React, with fake-store-hooks.ts for the tab bar store, whose changes record their arguments. Its session is
// fake-tab.ts's.

const preferences = vi.hoisted(() => ({
  updateNavigation: vi.fn((_projectId: string, _change: NavigationChange): Promise<unknown> => Promise.resolve()),
}));
vi.mock("@/hooks/store/use-project-preferences", async () => {
  const fake = await import("@/hooks/store/fake-store-hooks");
  return {
    useProjectPreferences: () => ({ ...fake.useProjectPreferences(), updateNavigation: preferences.updateNavigation }),
  };
});
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** The caller's tab bar in Web as nerve last answered it. */
const held: ProjectNavigation = { default_tab: "cycles", hide_in_more_menu: ["views"] };

/**
 * Each change the store was asked for: its project, and what it makes of the tab bar nerve answered the change before
 * it with (the first, of the tab bar held), as the store makes it in its turn, nerve answering each with the tab bar
 * sent. A change decided as it was asked for, of the tab bar shown then, would undo the one before it.
 */
const made = () => {
  let answered = held;
  return preferences.updateNavigation.mock.calls.map(([projectId, change]) => {
    answered = change(answered);
    return [projectId, answered];
  });
};

beforeEach(() => {
  signedIn();
  emptyStores();
  stores.navigations = { "p-web": held };
  preferences.updateNavigation.mockClear();
  toasts.length = 0;
});

describe("useTabPreferences", () => {
  it("gives nerve's default tab bar and no change until the caller's is fetched, then his and its changes", () => {
    stores.navigations = {};
    expect(useTabPreferences("p-web")).toEqual({
      navigation: { default_tab: "work_items", hide_in_more_menu: [] },
      changes: undefined,
    });
    stores.navigations = { "p-web": held };
    const { navigation, changes } = useTabPreferences("p-web");
    expect([navigation, changes && Object.keys(changes)]).toEqual([held, ["toggleDefault", "hide", "show"]]);
  });

  it("makes each change to the tab bar nerve last answered, in the change's turn", () => {
    const { changes } = useTabPreferences("p-web");
    changes?.toggleDefault("modules");
    changes?.hide("modules");
    changes?.show("views");
    expect(made()).toEqual([
      ["p-web", { default_tab: "modules", hide_in_more_menu: ["views"] }],
      ["p-web", { default_tab: "modules", hide_in_more_menu: ["views", "modules"] }],
      ["p-web", { default_tab: "modules", hide_in_more_menu: ["modules"] }],
    ]);
  });

  it("says a new default once nerve has it, and nerve's reason for refusing a change", async () => {
    const { changes } = useTabPreferences("p-web");
    changes?.toggleDefault("modules");
    changes?.hide("modules");
    preferences.updateNavigation.mockImplementationOnce(() => Promise.reject(refusal(422, "validation_failed")));
    changes?.show("views");
    await pageSettled();
    expect(toasts).toEqual([
      { type: "success", title: "Success!", message: "Default tab updated successfully." },
      { type: "error", title: "toast.error", message: "errors.validation_failed" },
    ]);
  });

  it.each(lateSettlings)("says nothing when a change $settles after another tab moved this one", async ({ settle }) => {
    const change = heldChange<undefined>();
    preferences.updateNavigation.mockReturnValueOnce(change.sent);
    useTabPreferences("p-web").changes?.toggleDefault("modules");
    switchAccount();
    settle(change);
    await pageSettled();
    expect(toasts).toEqual([]);
  });
});
