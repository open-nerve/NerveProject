/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Label } from "@nerve/api-client";
import { heldChange, lateSettlingsAnswering, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { labelOf, projectOf } from "@/store/project/fake-projects";
import { useLabelChanges } from "./use-label-changes";

// How the label settings change a project's labels (M3 design 2 P7, 3.16, 7.1, 7.6): the hook runs as a plain
// function, with stand-ins for the label store's changes, which nerve answers when the test says, and for the form's
// setError. Its session is fake-tab.ts's.

const store = vi.hoisted(() => ({
  createLabel: vi.fn(),
  updateLabel: vi.fn(),
  updateLabelPosition: vi.fn(),
  deleteLabel: vi.fn(),
}));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => store }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme");
const bug = labelOf(web, "Bug", 10000);
const ui = labelOf(web, "UI", 20000);
const setError = vi.fn();
const done = vi.fn();
const underField = (field: string, message: string) => setError(field, { message });

/** Each change of the page, sent as the page sends it, with the store's change it makes and what nerve answers. */
const changes = {
  creation: {
    change: "a creation",
    send: () => useLabelChanges().create(web.id, { name: "Bug", color: "#F59E0B" }, underField, done),
    sent: () => store.createLabel,
    call: [web.id, { name: "Bug", color: "#F59E0B" }],
    answer: bug,
  },
  change: {
    change: "a change",
    send: () => useLabelChanges().update(bug.id, { name: "Defect", color: "#F59E0B" }, underField, done),
    sent: () => store.updateLabel,
    call: [bug.id, { name: "Defect", color: "#F59E0B" }],
    answer: bug,
  },
  move: {
    change: "a move",
    send: () => useLabelChanges().move(ui.id, bug.id, undefined, false),
    sent: () => store.updateLabelPosition,
    call: [ui.id, bug.id, undefined, false],
    answer: ui,
  },
  removal: {
    change: "a removal from its group",
    send: () => useLabelChanges().removeFromGroup(ui.id),
    sent: () => store.updateLabel,
    call: [ui.id, { parent_id: null }],
    answer: ui,
  },
  deletion: {
    change: "a deletion",
    send: () => useLabelChanges().remove(bug.id, done),
    sent: () => store.deleteLabel,
    call: [bug.id],
    answer: undefined,
  },
};
const { creation, change, move, removal, deletion } = changes;

beforeEach(() => {
  signedIn();
  for (const sent of Object.values(store)) {
    sent.mockReset();
    sent.mockResolvedValue(bug);
  }
  setError.mockReset();
  done.mockReset();
  toasts.length = 0;
});

describe("useLabelChanges", () => {
  it.each(Object.values(changes))("sends $change to the store as the page asks", async ({ send, sent, call }) => {
    await send();
    expect(sent().mock.calls).toEqual([call]);
  });

  it.each([creation, change, deletion])("follows $change nerve makes with done, and says nothing", async ({ send }) => {
    await send();
    expect([done.mock.calls.length, underFields(setError), toasts]).toEqual([1, [], []]);
  });

  it.each([
    { by: creation, refused: "a name another label has, in another case", code: "project.label_name_taken" },
    { by: change, refused: "a name another label has, in another case", code: "project.label_name_taken" },
  ])("shows $refused under the name, of $by.change, and does not follow it", async ({ by, code }) => {
    by.sent().mockRejectedValueOnce(refusal(409, code));
    await by.send();
    expect([underFields(setError), toasts, done.mock.calls]).toEqual([
      [["name", "errors.project_label_name_taken"]],
      [],
      [],
    ]);
  });

  it("shows a name too long under the name", async () => {
    store.createLabel.mockRejectedValueOnce(refusal(422, "validation_failed", [{ field: "name", code: "too_long" }]));
    await creation.send();
    expect([underFields(setError), toasts]).toEqual([[["name", "errors.field.too_long"]], []]);
  });

  it("shows nerve's reason for a colour it does not allow, which the form shows nothing under, in a toast", async () => {
    store.updateLabel.mockRejectedValueOnce(refusal(422, "validation_failed", [{ field: "color", code: "too_long" }]));
    await change.send();
    expect([underFields(setError), toasts]).toEqual([
      [],
      [{ type: "error", title: "toast.error", message: "errors.validation_failed" }],
    ]);
  });

  it("says a place nerve does not allow for a move: labels have two levels", async () => {
    move.sent().mockRejectedValueOnce(refusal(422, "validation_failed", [{ field: "parent_id", code: "not_allowed" }]));
    await move.send();
    expect(toasts).toEqual([
      { type: "error", title: "toast.error", message: "project_settings.labels.parent_not_allowed" },
    ]);
  });

  it.each([
    { by: move, status: 404, code: "project.label_not_found" },
    { by: removal, status: 403, code: "forbidden" },
    { by: deletion, status: 404, code: "project.label_not_found" },
  ])("shows nerve's reason when it refuses $by.change otherwise, in a toast", async ({ by, status, code }) => {
    by.sent().mockRejectedValueOnce(refusal(status, code));
    await by.send();
    const message = `errors.${code.replace(".", "_")}`;
    expect([toasts, done.mock.calls]).toEqual([[{ type: "error", title: "toast.error", message }], []]);
  });

  describe.each(Object.values(changes))("$change", ({ send, sent, answer }) => {
    it.each(lateSettlingsAnswering(answer))(
      "does nothing on the page when it $settles after another tab moved this one",
      async ({ settle }) => {
        const held = heldChange<Label | undefined>();
        sent().mockReturnValueOnce(held.sent);
        const followed = send();
        switchAccount();
        settle(held);
        await followed;
        await pageSettled();
        expect([done.mock.calls, underFields(setError), toasts]).toEqual([[], [], []]);
      }
    );
  });
});
