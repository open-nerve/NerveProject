/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { WorkspaceHomeView } from "./root";

// What the workspace's home writes as its tour ends (M3 design 7.1, 7.5): the home renders on the server, the tour a
// stand-in that keeps how it ends, and the test ends it as the tour would; the profile's store is a stand-in too,
// which nerve answers when the test says. The session is fake-tab.ts's.

const page = vi.hoisted((): { complete: (() => void) | undefined; updateTourCompleted: ReturnType<typeof vi.fn> } => ({
  complete: undefined,
  updateTourCompleted: vi.fn(),
}));
vi.mock("@/components/onboarding/tour/root", () => ({
  TourRoot: (props: { onComplete: () => void }) => {
    page.complete = props.onComplete;
    return null;
  },
}));
vi.mock("./home-body", () => ({ HomeBody: () => null }));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => ({ data: undefined }),
  useUserProfile: () => ({ data: { is_tour_completed: false }, updateTourCompleted: page.updateTourCompleted }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Opens the home of one who has not ended the tour, and gives how the tour ends. */
function opened(): () => void {
  renderToStaticMarkup(<WorkspaceHomeView />);
  if (!page.complete) throw new Error("the home showed no tour");
  return page.complete;
}

beforeEach(() => {
  signedIn();
  page.complete = undefined;
  page.updateTourCompleted.mockReset();
  page.updateTourCompleted.mockResolvedValue({});
  toasts.length = 0;
});

describe("WorkspaceHomeView", () => {
  it("writes the tour's end in the caller's profile", async () => {
    opened()();
    await pageSettled();
    expect(page.updateTourCompleted).toHaveBeenCalledTimes(1);
    expect(toasts).toEqual([]);
  });

  it("shows nerve's reason when it refuses the tour's end", async () => {
    page.updateTourCompleted.mockRejectedValueOnce(refusal(503, "server_busy"));
    opened()();
    await pageSettled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.server_busy" }]);
  });

  it.each(lateSettlings)(
    "says nothing of the tour's end when it $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const change = heldChange<undefined>();
      page.updateTourCompleted.mockReturnValueOnce(change.sent);
      opened()();
      switchAccount();
      settle(change);
      await pageSettled();
      expect(toasts).toEqual([]);
    }
  );
});
