/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, pageSettled, settledYet, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { useOpenWorkspace } from "./use-open-workspace";

// How a page opens a workspace the caller has just joined (M3 design 3.14, 7.1): the hook runs as a plain function,
// with stand-ins for the router and the profile's store, whose change nerve answers when the test says. Its session is
// fake-tab.ts's.

const page = vi.hoisted(() => ({ updateUserProfile: vi.fn(), navigate: vi.fn() }));
vi.mock("react-router", () => ({ useNavigate: () => page.navigate }));
vi.mock("@/hooks/store/user", () => ({ useUserProfile: () => ({ updateUserProfile: page.updateUserProfile }) }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));

const acme = { id: "id-acme", slug: "acme" };

beforeEach(() => {
  signedIn();
  page.updateUserProfile.mockReset();
  page.updateUserProfile.mockResolvedValue({});
  page.navigate.mockClear();
});

describe("useOpenWorkspace", () => {
  it("writes the workspace as the one opened last, then goes there", async () => {
    await useOpenWorkspace()(acme);
    expect(page.updateUserProfile.mock.calls).toEqual([[{ last_workspace_id: "id-acme" }]]);
    expect(page.navigate.mock.calls).toEqual([["/acme"]]);
  });

  it("settles once the workspace's page has opened", async () => {
    const opening = heldChange<undefined>();
    page.navigate.mockReturnValueOnce(opening.sent);
    const settled = settledYet(useOpenWorkspace()(acme));
    await pageSettled();
    expect([page.navigate.mock.calls, settled()]).toEqual([[["/acme"]], false]);
    opening.answer(undefined);
    await pageSettled();
    expect(settled()).toBe(true);
  });

  it("goes there when nerve does not save it", async () => {
    page.updateUserProfile.mockRejectedValueOnce(refusal(500, "internal_error"));
    await useOpenWorkspace()(acme);
    expect(page.navigate.mock.calls).toEqual([["/acme"]]);
  });

  it.each(lateSettlings)(
    "stays when the write $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const written = heldChange<undefined>();
      page.updateUserProfile.mockReturnValueOnce(written.sent);
      const opened = useOpenWorkspace()(acme);
      switchAccount();
      settle(written);
      await opened;
      expect(page.navigate.mock.calls).toEqual([]);
    }
  );
});
