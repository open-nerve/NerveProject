// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, expect, it, vi } from "vitest";
import { rootStore } from "@/lib/store-context";
import { CycleStore } from "@/store/cycle.store";
import { CyclePeekOverview } from "./cycle-peek-overview";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const container = document.createElement("div");
document.body.append(container);
const root = createRoot(container);
afterEach(async () => {
  await act(async () => root.render(null));
  vi.restoreAllMocks();
});

it.each([false, true])(
  "loads only the matching detail endpoint for a cold preview, archived=%s",
  async (isArchived) => {
    const store = rootStore.cycle as CycleStore;
    const pending = new Promise<never>(() => {});
    const ordinary = vi.spyOn(store.cycleService, "getCycleDetails").mockReturnValue(pending);
    const archived = vi.spyOn(store.cycleArchiveService, "getArchivedCycleDetails").mockReturnValue(pending);
    const config = { provider: () => new Map(), shouldRetryOnError: false };
    await act(async () =>
      root.render(
        <MemoryRouter initialEntries={["/?peekCycle=cold"]}>
          <SWRConfig value={config}>
            <CyclePeekOverview workspaceSlug="ws" projectId="project" isArchived={isArchived} />
          </SWRConfig>
        </MemoryRouter>
      )
    );
    expect(ordinary).toHaveBeenCalledTimes(isArchived ? 0 : 1);
    expect(archived).toHaveBeenCalledTimes(isArchived ? 1 : 0);
    expect(isArchived ? archived : ordinary).toHaveBeenCalledWith("ws", "project", "cold");
  }
);
