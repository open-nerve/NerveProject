/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown, submitModalForm } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { DeleteWorkspaceModal } from "./delete-workspace-modal";

// What the deletion of a workspace sends and does on the page (M3 design 7.1, 9.5): the modal renders on the server,
// with stand-ins for its inputs and buttons (fake-controls.ts), which keep the props they were given; the test types
// as a person would, through the inputs' onChange, and submits the form. Its session is fake-tab.ts's.

const store = vi.hoisted(() => ({ deleteWorkspace: vi.fn() }));
const navigate = vi.hoisted(() => vi.fn());
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => ({ deleteWorkspace: store.deleteWorkspace }) }));
vi.mock("react-router", () => ({ useNavigate: () => navigate }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@makeplane/propel/components/input", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const acme = workspaceOf("acme", { name: "Acme" });

/** Renders the modal of Acme, types name and words into its two inputs, and submits it; gives onClose. */
async function confirm(name: string, words: string) {
  const onClose = vi.fn();
  renderToStaticMarkup(<DeleteWorkspaceModal isOpen workspace={acme} onClose={onClose} />);
  const [nameInput, wordsInput] = shown.inputs;
  nameInput?.onChange({ target: { value: name } });
  wordsInput?.onChange({ target: { value: words } });
  await submitModalForm();
  return onClose;
}

beforeEach(() => {
  signedIn();
  store.deleteWorkspace.mockReset();
  store.deleteWorkspace.mockResolvedValue(undefined);
  navigate.mockClear();
  toasts.length = 0;
  emptyShown();
});

describe("DeleteWorkspaceModal", () => {
  it("deletes the workspace once its name and the words are typed, then lands at the root and says so", async () => {
    const onClose = await confirm("Acme", "delete my workspace");
    expect(store.deleteWorkspace.mock.calls).toEqual([[acme]]);
    expect([onClose.mock.calls.length, navigate.mock.calls]).toEqual([1, [["/"]]]);
    expect(toasts).toEqual([
      {
        type: "success",
        title: "workspace_settings.settings.general.delete_modal.success_title",
        message: "workspace_settings.settings.general.delete_modal.success_message",
      },
    ]);
  });

  it.each([
    { typed: "another name", name: "Acme Corp", words: "delete my workspace" },
    { typed: "other words", name: "Acme", words: "delete it" },
  ])("deletes nothing when the form has $typed", async ({ name, words }) => {
    const onClose = await confirm(name, words);
    expect([store.deleteWorkspace.mock.calls, onClose.mock.calls, navigate.mock.calls, toasts]).toEqual([
      [],
      [],
      [],
      [],
    ]);
  });

  it("shows nerve's reason when it refuses, and stays", async () => {
    store.deleteWorkspace.mockRejectedValueOnce(refusal(403, "forbidden"));
    const onClose = await confirm("Acme", "delete my workspace");
    expect([onClose.mock.calls, navigate.mock.calls]).toEqual([[], []]);
    expect(toasts).toEqual([
      {
        type: "error",
        title: "workspace_settings.settings.general.delete_modal.error_title",
        message: "errors.forbidden",
      },
    ]);
  });

  it.each(lateSettlings)(
    "neither moves nor speaks when the deletion $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const deletion = heldChange<undefined>();
      store.deleteWorkspace.mockReturnValueOnce(deletion.sent);
      const submitted = confirm("Acme", "delete my workspace");
      await vi.waitFor(() => expect(store.deleteWorkspace).toHaveBeenCalledTimes(1));
      switchAccount();
      settle(deletion);
      const onClose = await submitted;
      expect([onClose.mock.calls, navigate.mock.calls, toasts]).toEqual([[], [], []]);
    }
  );
});
