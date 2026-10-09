/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Workspace } from "@nerve/api-client";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { fetchHanded, handed, response } from "@/lib/fake-session-swr";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { OnboardingRoot } from "./root";

// What the onboarding writes as its steps end (M3 design 7.4, 7.1): the root renders on the server, its steps a
// stand-in that keeps what it was given, and the test ends a step as the step would; the profile's store is a
// stand-in too, which nerve answers when the test says, and so is the page of an unreachable nerve, which keeps its
// props. The workspaces are fake-store-hooks.ts's, the fetch fake-session-swr.ts's, the session fake-tab.ts's.

/** How the steps tell the root they are done (OnboardingStepRoot's props). */
type Ends = {
  onNamed: () => void;
  onCreated: (workspace: Workspace, alone: boolean) => Promise<void>;
  onDone: () => void;
};
/** What the page of an unreachable nerve is given (SessionUnavailable's props). */
type Unavailable = { onRetry: () => void; autoRetry: boolean };

const page = vi.hoisted(
  (): {
    ends: Ends | undefined;
    unavailable: Unavailable[];
    updateUserProfile: ReturnType<typeof vi.fn>;
    finishUserOnboarding: ReturnType<typeof vi.fn>;
  } => ({
    ends: undefined,
    unavailable: [],
    updateUserProfile: vi.fn(),
    finishUserOnboarding: vi.fn(),
  })
);
vi.mock("./steps", () => ({
  OnboardingStepRoot: (props: Ends) => {
    page.ends = props;
    return null;
  },
}));
vi.mock("./header", () => ({ OnboardingHeader: () => null }));
vi.mock("@/components/account/session-unavailable", () => ({
  SessionUnavailable: (props: Unavailable) => {
    page.unavailable.push(props);
    return null;
  },
}));
vi.mock("@/hooks/store/user", () => ({
  useUserProfile: () => ({
    data: undefined,
    updateUserProfile: page.updateUserProfile,
    finishUserOnboarding: page.finishUserOnboarding,
  }),
}));
vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const alpha = workspaceOf("alpha");
const zeta = workspaceOf("zeta", { role: 20 });
/**
 * Each change the onboarding follows, by the profile's store change: a step's, the end's, and the end after a
 * workspace created for its creator alone, which follows a refusal its own way.
 */
const changes = [
  { change: "a step's change", store: page.updateUserProfile, make: () => opened([]).onNamed() },
  { change: "the end of the onboarding", store: page.finishUserOnboarding, make: () => opened([alpha]).onDone() },
  {
    change: "the end of the onboarding of a workspace created for its creator alone",
    store: page.finishUserOnboarding,
    make: () => void opened([alpha, zeta]).onCreated(zeta, true),
  },
];

/** Opens the onboarding of one whose workspaces nerve listed as workspaces; gives how its steps end. */
function opened(workspaces: Workspace[]): Ends {
  stores.workspaces = workspaces;
  renderToStaticMarkup(<OnboardingRoot />);
  if (!page.ends) throw new Error("the onboarding showed no step");
  return page.ends;
}

beforeEach(() => {
  signedIn();
  emptyStores();
  handed.length = 0;
  response.current = {};
  page.ends = undefined;
  page.unavailable.length = 0;
  page.updateUserProfile.mockReset();
  page.updateUserProfile.mockResolvedValue({});
  page.finishUserOnboarding.mockReset();
  page.finishUserOnboarding.mockResolvedValue(undefined);
  toasts.length = 0;
});

describe("OnboardingRoot", () => {
  it("lists the caller's workspaces, and shows no step before nerve has", async () => {
    renderToStaticMarkup(<OnboardingRoot />);
    expect(page.ends).toBeUndefined();
    await fetchHanded();
    expect([handed.map(([fetch]) => fetch), stores.fetched]).toEqual([[["WORKSPACES"]], ["the workspaces"]]);
  });

  it("says when nerve cannot list the caller's workspaces, shows no step, and asks again on a retry", () => {
    const mutate = vi.fn(() => Promise.resolve(undefined));
    response.current = { error: new Error("nerve cannot be reached"), mutate };
    renderToStaticMarkup(<OnboardingRoot />);
    expect([page.ends, page.unavailable.map(({ autoRetry }) => autoRetry)]).toEqual([undefined, [false]]);
    page.unavailable[0]?.onRetry();
    expect(mutate).toHaveBeenCalledOnce();
  });

  it("ends the onboarding of one who has a workspace with the profile step", () => {
    opened([alpha]).onNamed();
    expect([page.finishUserOnboarding.mock.calls, page.updateUserProfile.mock.calls]).toEqual([[[]], []]);
  });

  it("writes the profile step done for one who has none, whom the creation step follows", () => {
    opened([]).onNamed();
    expect([page.finishUserOnboarding.mock.calls, page.updateUserProfile.mock.calls]).toEqual([
      [],
      [[{ onboarding_step: { profile_complete: true } }]],
    ]);
  });

  it.each([
    { workspace: "for others too", alone: false, finished: [] },
    { workspace: "for its creator alone", alone: true, finished: [[]] },
  ])("writes a workspace created $workspace as the one opened last, with its step", async ({ alone, finished }) => {
    await opened([alpha, zeta]).onCreated(zeta, alone);
    expect([page.updateUserProfile.mock.calls, page.finishUserOnboarding.mock.calls]).toEqual([
      [[{ onboarding_step: { workspace_create: true }, last_workspace_id: zeta.id }]],
      finished,
    ]);
  });

  it("says why nerve refused a step's change", async () => {
    page.updateUserProfile.mockRejectedValueOnce(refusal(422, "validation_failed"));
    opened([]).onNamed();
    await pageSettled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.validation_failed" }]);
  });

  describe.each(changes)("$change", ({ store, make }) => {
    it.each(lateSettlings)(
      "says nothing when it $settles after another tab moved this one to another account",
      async ({ settle }) => {
        const change = heldChange<undefined>();
        store.mockReturnValueOnce(change.sent);
        make();
        switchAccount();
        settle(change);
        await pageSettled();
        expect(toasts).toEqual([]);
      }
    );
  });
});
