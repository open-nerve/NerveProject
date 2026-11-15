/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { isValidElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { EViewAccess, type IProjectView } from "@nerve/types";
import { heldChange, lateSettlings, lateSettlingsAnswering, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { toasts } from "@/lib/fake-toast";
import { DeleteProjectViewModal } from "./delete-view-modal";
import { CreateUpdateProjectViewModal } from "./modal";

// What follows a view's creation, change and deletion on the page (M3 handoff §1, v0 7.7, P7c spec §2.9): the dialog
// closes, goes on and says so only while the tab is in the session the change was sent in; after another tab moved it
// to another account, the page is that account's, and nothing follows. The dialogs render on the server with stand-ins
// for the UI kit's controls (fake-controls.ts) and the view's form (its submit is the dialog's handler); the views'
// store is a stand-in whose changes the test settles; the session is fake-tab.ts's, the address's params are params'.

const store = vi.hoisted(() => ({ createView: vi.fn(), updateView: vi.fn(), deleteView: vi.fn() }));
const navigate = vi.hoisted(() => vi.fn());
const params = vi.hoisted(() => ({ workspaceSlug: "acme", projectId: "p-web" }));
vi.mock("@/hooks/store/use-project-view", () => ({ useProjectView: () => store }));
vi.mock("react-router", () => ({ useNavigate: () => navigate, useParams: () => params }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("./form", () => ({ ProjectViewForm: () => null }));

const release: IProjectView = {
  id: "v-release",
  name: "Release view",
  access: EViewAccess.PUBLIC,
  created_at: new Date(),
  updated_at: new Date(),
  is_favorite: false,
  created_by: "u",
  updated_by: "u",
  description: "",
  rich_filters: {},
  display_filters: {},
  display_properties: {},
  query: {},
  query_data: {},
  project: "p-web",
  workspace: "id-acme",
  logo_props: undefined,
  is_locked: false,
  owned_by: "u",
};

/** Renders the view's dialog, of web's view data (none: a new view), and submits its form; gives onClose. */
async function submitted(data?: IProjectView) {
  const onClose = vi.fn();
  renderToStaticMarkup(
    <CreateUpdateProjectViewModal isOpen data={data} onClose={onClose} workspaceSlug="acme" projectId="p-web" />
  );
  const form = shown.modals.at(-1)?.children;
  if (!isValidElement<{ handleFormSubmit: (values: IProjectView) => Promise<void> }>(form)) {
    throw new Error("the dialog showed no form");
  }
  await form.props.handleFormSubmit(release);
  return onClose;
}

/** Renders the deletion's dialog of web's view, and confirms it; gives onClose. */
async function confirmed() {
  const onClose = vi.fn();
  renderToStaticMarkup(<DeleteProjectViewModal isOpen data={release} onClose={onClose} />);
  await shown.alerts.at(-1)?.handleSubmit();
  return onClose;
}

beforeEach(() => {
  signedIn();
  store.createView.mockReset().mockResolvedValue(release);
  store.updateView.mockReset().mockResolvedValue(release);
  store.deleteView.mockReset().mockResolvedValue(undefined);
  navigate.mockClear();
  toasts.length = 0;
  emptyShown();
});

describe("CreateUpdateProjectViewModal", () => {
  it("creates the view, closes, says so and goes to it", async () => {
    const onClose = await submitted();
    expect(store.createView.mock.calls).toEqual([["acme", "p-web", release]]);
    expect([onClose.mock.calls.length, navigate.mock.calls]).toEqual([1, [["/acme/projects/p-web/views/v-release"]]]);
    expect(toasts).toEqual([{ type: "success", title: "Success!", message: "View created successfully." }]);
  });

  it("changes the view and closes", async () => {
    const onClose = await submitted(release);
    expect(store.updateView.mock.calls).toEqual([["acme", "p-web", "v-release", release]]);
    expect([onClose.mock.calls.length, navigate.mock.calls, toasts]).toEqual([1, [], []]);
  });

  it.each([
    { change: "creation", data: undefined, message: "Failed to create view. Please try again." },
    { change: "change", data: release, message: "Failed to update view. Please try again." },
  ])("says the $change failed, and stays", async ({ data, message }) => {
    store.createView.mockRejectedValueOnce(new Error("refused"));
    store.updateView.mockRejectedValueOnce(new Error("refused"));
    const onClose = await submitted(data);
    expect([onClose.mock.calls, navigate.mock.calls]).toEqual([[], []]);
    expect(toasts).toEqual([{ type: "error", title: "Error!", message }]);
  });

  describe.each([
    { change: "creation", data: undefined, sent: store.createView },
    { change: "change", data: release, sent: store.updateView },
  ])("the $change", ({ data, sent }) => {
    it.each(lateSettlingsAnswering(release))(
      "neither closes, moves nor speaks when it $settles after another tab moved this one to another account",
      async ({ settle }) => {
        const change = heldChange<IProjectView>();
        sent.mockReturnValueOnce(change.sent);
        const settled = submitted(data);
        await vi.waitFor(() => expect(sent).toHaveBeenCalledTimes(1));
        switchAccount();
        settle(change);
        const onClose = await settled;
        expect([onClose.mock.calls, navigate.mock.calls, toasts]).toEqual([[], [], []]);
      }
    );
  });
});

describe("DeleteProjectViewModal", () => {
  it("deletes the view, closes, says so and goes to the project's views", async () => {
    const onClose = await confirmed();
    expect(store.deleteView.mock.calls).toEqual([["acme", "p-web", "v-release"]]);
    expect([onClose.mock.calls.length, navigate.mock.calls]).toEqual([1, [["/acme/projects/p-web/views"]]]);
    expect(toasts).toEqual([{ type: "success", title: "Success!", message: "View deleted successfully." }]);
  });

  it("says the deletion failed, and stays", async () => {
    store.deleteView.mockRejectedValueOnce(new Error("refused"));
    const onClose = await confirmed();
    expect([onClose.mock.calls, navigate.mock.calls]).toEqual([[], []]);
    expect(toasts).toEqual([
      { type: "error", title: "Error!", message: "View could not be deleted. Please try again." },
    ]);
  });

  it.each(lateSettlings)(
    "neither closes, moves nor speaks when the deletion $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const deletion = heldChange<undefined>();
      store.deleteView.mockReturnValueOnce(deletion.sent);
      const settled = confirmed();
      await vi.waitFor(() => expect(store.deleteView).toHaveBeenCalledTimes(1));
      switchAccount();
      settle(deletion);
      const onClose = await settled;
      expect([onClose.mock.calls, navigate.mock.calls, toasts]).toEqual([[], [], []]);
    }
  );
});
