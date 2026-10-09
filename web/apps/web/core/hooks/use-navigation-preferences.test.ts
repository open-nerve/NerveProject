/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspacePreferences } from "@nerve/api-client";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { useProjectNavigationPreferences } from "./use-navigation-preferences";

// What the sidebar shows of the caller's settings in the address's workspace, and what its project navigation dialog
// sends (M3 design 3.18, 7.5): the hook runs as a plain function, with stand-ins for the store's settings and their
// change, which nerve answers when the test says. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({ settings: new Map<string, WorkspacePreferences>(), updatePreferences: vi.fn() }));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }) }));
vi.mock("./store/use-workspace", () => ({
  useWorkspace: () => ({
    preferences: {
      getPreferences: (slug: string) => page.settings.get(slug),
      updatePreferences: page.updatePreferences,
    },
  }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** The caller's settings in acme: a limit of 3, in tabs. */
const tabbed: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 3 };

beforeEach(() => {
  signedIn();
  page.settings.clear();
  page.updatePreferences.mockReset();
  page.updatePreferences.mockResolvedValue(tabbed);
  toasts.length = 0;
});

describe("useProjectNavigationPreferences", () => {
  it("shows the caller's settings in the address's workspace, and nerve's defaults until they arrive", () => {
    expect(useProjectNavigationPreferences().preferences).toEqual({
      navigationMode: "ACCORDION",
      limitedProjectsCount: 10,
      showLimitedProjects: true,
    });
    page.settings.set("acme", tabbed);
    expect(useProjectNavigationPreferences().preferences).toEqual({
      navigationMode: "TABBED",
      limitedProjectsCount: 3,
      showLimitedProjects: true,
    });
  });

  it("sends a change to the address's workspace, made in its turn to the settings nerve last answered", async () => {
    page.settings.set("acme", tabbed);
    await useProjectNavigationPreferences().changeNavigation({ limitToggled: true });
    expect(page.updatePreferences.mock.calls).toEqual([["acme", expect.any(Function)]]);
    // nerve answered every project to the turn before it: the sidebar showed a limit of 3 as this one was asked for,
    // and in this one's turn the store has that answer, as its settings show
    const answered: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 0 };
    page.settings.set("acme", answered);
    expect(page.updatePreferences.mock.calls[0]?.[1](answered)).toEqual({ navigation_project_limit: 10 });
    expect(toasts).toEqual([]);
  });

  it("shows nerve's reason when it refuses a change", async () => {
    page.updatePreferences.mockRejectedValueOnce(refusal(422, "validation_failed"));
    await useProjectNavigationPreferences().changeNavigation({ limitedProjectsCount: 3_000_000_000 });
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.validation_failed" }]);
  });

  it.each(lateSettlings)(
    "says nothing of a change that $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const change = heldChange<undefined>();
      page.updatePreferences.mockReturnValueOnce(change.sent);
      const made = useProjectNavigationPreferences().changeNavigation({ navigationMode: "TABBED" });
      switchAccount();
      settle(change);
      await made;
      expect(toasts).toEqual([]);
    }
  );
});
