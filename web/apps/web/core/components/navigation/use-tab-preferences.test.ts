/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { useTabPreferences } from "./use-tab-preferences";

// The tab bar the project's header and the sidebar show (M3 design 3.18): nerve's ProjectNavigation, the caller's once
// the project wrapper has fetched it. The hook runs as a plain function, outside React, with fake-store-hooks.ts for
// the tab bar store.

vi.mock("@/hooks/store/use-project-preferences", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));

beforeEach(() => {
  emptyStores();
});

describe("useTabPreferences", () => {
  it("gives nerve's default tab bar until the caller's is fetched, then his", () => {
    expect(useTabPreferences("p-web").navigation).toEqual({ default_tab: "work_items", hide_in_more_menu: [] });
    stores.navigations = { "p-web": { default_tab: "cycles", hide_in_more_menu: ["views"] } };
    expect(useTabPreferences("p-web").navigation).toEqual({ default_tab: "cycles", hide_in_more_menu: ["views"] });
  });
});
