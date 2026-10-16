/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { pageSettled, signedIn } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf, stateOf } from "@/store/project/fake-projects";
import { StateDelete } from "./delete";

// Which states the state settings offer to delete, and what confirming it sends (M3 design 2 P6, 3.17, 7.6): the
// button renders on the server, with stand-ins for its dialog, which keeps the props it was given, and its tooltip
// (fake-controls.ts); the test presses the dialog's button, as a person would. A default and a group's only state
// cannot be deleted, as nerve refuses (409), and their button is disabled, its tooltip (its label rendered beside it)
// saying why; nerve's refusal all the same (another tab changed them meanwhile) is said in a toast. Its session is
// fake-tab.ts's; its texts are the keys of the page's language (fake-i18n.ts).

const store = vi.hoisted(() => ({ deleteState: vi.fn() }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => store }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@makeplane/propel/components/tooltip", () => import("@/lib/fake-controls"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme");
const review = stateOf(web, "Review", "started", 70000);

/** Renders the deletion of state among totalStates of its group: whether its button is disabled, its dialog, its text. */
function deletionOf(state = review, totalStates = 2) {
  emptyShown();
  const markup = renderToStaticMarkup(<StateDelete state={state} totalStates={totalStates} />);
  const [alert] = shown.alerts;
  if (!alert) throw new Error("no dialog");
  return { disabled: /<button[^>]* disabled=""/.test(markup), alert, markup };
}

beforeEach(() => {
  signedIn();
  store.deleteState.mockReset();
  store.deleteState.mockResolvedValue(undefined);
  toasts.length = 0;
});

describe("StateDelete", () => {
  it.each([
    {
      state: "the default",
      deleted: { ...review, default: true },
      totalStates: 2,
      why: "project_settings.states.delete.default",
    },
    {
      state: "its group's only state",
      deleted: review,
      totalStates: 1,
      why: "project_settings.states.delete.only_one",
    },
  ])("offers no deletion of $state, says why, and sends none", async ({ deleted, totalStates, why }) => {
    const { disabled, alert, markup } = deletionOf(deleted, totalStates);
    await alert.handleSubmit();
    expect([disabled, markup.includes(why), store.deleteState.mock.calls]).toEqual([true, true, []]);
  });

  it("asks in the page's language, its buttons too", () => {
    const { alert } = deletionOf();
    const { title, content, primaryButtonText, secondaryButtonText } = alert;
    expect([title, renderToStaticMarkup(content), primaryButtonText, secondaryButtonText]).toEqual([
      "project_settings.states.delete.title",
      "project_settings.states.delete.body",
      { default: "delete", loading: "deleting" },
      "cancel",
    ]);
  });

  it("deletes a state its group has another of, once its dialog is confirmed, the dialog closed until then", async () => {
    const { disabled, alert } = deletionOf();
    expect([disabled, alert.isOpen, store.deleteState.mock.calls]).toEqual([false, false, []]);
    await alert.handleSubmit();
    expect([store.deleteState.mock.calls, toasts]).toEqual([[[review.id]], []]);
  });

  it("says nerve's reason when it refuses the deletion", async () => {
    store.deleteState.mockRejectedValueOnce(refusal(409, "project.state_default"));
    await deletionOf().alert.handleSubmit();
    await pageSettled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.project_state_default" }]);
  });
});
