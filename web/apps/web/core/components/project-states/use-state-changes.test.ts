/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { State } from "@nerve/api-client";
import { heldChange, lateSettlingsAnswering, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf, stateOf } from "@/store/project/fake-projects";
import { useStateChanges } from "./use-state-changes";

// How the state settings change a project's states (M3 design 2 P6, 7.1, 7.6): the hook runs as a plain function,
// with stand-ins for the state store's changes, which nerve answers when the test says, and for the form's setError.
// Its session is fake-tab.ts's.

const store = vi.hoisted(() => ({
  createState: vi.fn(),
  updateState: vi.fn(),
  moveState: vi.fn(),
  deleteState: vi.fn(),
  markStateAsDefault: vi.fn(),
}));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => store }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme");
const review = stateOf(web, "Review", "started", 70000);
const setError = vi.fn();
const done = vi.fn();
const underField = (field: string, message: string) => setError(field, { message });

/** Each change of the page, sent as the page sends it, with the store's change it makes and what nerve answers. */
const changes = {
  creation: {
    change: "a creation",
    send: () =>
      useStateChanges().create(web.id, { name: "Review", color: "#8B5CF6", group: "started" }, underField, done),
    sent: () => store.createState,
    call: [web.id, { name: "Review", color: "#8B5CF6", group: "started" }],
    answer: review,
  },
  change: {
    change: "a change",
    send: () => useStateChanges().update(review.id, { name: "Checked" }, underField, done),
    sent: () => store.updateState,
    call: [review.id, { name: "Checked" }],
    answer: review,
  },
  move: {
    change: "a move",
    send: () => useStateChanges().move(review.id, "started", "s-WEB-Todo", true),
    sent: () => store.moveState,
    call: [review.id, "started", "s-WEB-Todo", true],
    answer: review,
  },
  deletion: {
    change: "a deletion",
    send: () => useStateChanges().remove(review.id, done),
    sent: () => store.deleteState,
    call: [review.id],
    answer: undefined,
  },
  newDefault: {
    change: "a new default",
    send: () => useStateChanges().markDefault(review.id),
    sent: () => store.markStateAsDefault,
    call: [review.id],
    answer: undefined,
  },
};
const { creation, change, move, deletion, newDefault } = changes;

beforeEach(() => {
  signedIn();
  for (const sent of Object.values(store)) {
    sent.mockReset();
    sent.mockResolvedValue(review);
  }
  setError.mockReset();
  done.mockReset();
  toasts.length = 0;
});

describe("useStateChanges", () => {
  it.each(Object.values(changes))("sends $change to the store as the page asks", async ({ send, sent, call }) => {
    await send();
    expect(sent().mock.calls).toEqual([call]);
  });

  it.each([creation, change, deletion])("follows $change nerve makes with done, and says nothing", async ({ send }) => {
    await send();
    expect([done.mock.calls.length, underFields(setError), toasts]).toEqual([1, [], []]);
  });

  /** Each row for a creation and for a change alike, the form's two changes. */
  const ofBoth = <R extends object>(rows: R[]) =>
    rows.flatMap((row) => [creation, change].map((by) => Object.assign({ by, change: by.change }, row)));

  it.each(
    ofBoth([
      {
        refused: "a name taken",
        error: refusal(409, "project.state_name_taken"),
        fields: [["name", "errors.project_state_name_taken"]],
      },
      {
        refused: "a name too long",
        error: refusal(422, "validation_failed", [{ field: "name", code: "too_long" }]),
        fields: [["name", "errors.field.too_long"]],
      },
      {
        refused: "a description nerve does not allow",
        error: refusal(422, "validation_failed", [{ field: "description", code: "not_allowed" }]),
        fields: [["description", "errors.field.not_allowed"]],
      },
    ])
  )("shows $refused under its field, of $change, and does not follow it", async ({ by, error, fields }) => {
    by.sent().mockRejectedValueOnce(error);
    await by.send();
    expect([underFields(setError), toasts, done.mock.calls]).toEqual([fields, [], []]);
  });

  it.each(
    ofBoth([
      {
        refused: "a colour nerve does not allow, which the form shows nothing under",
        error: refusal(422, "validation_failed", [{ field: "color", code: "not_allowed" }]),
        message: "errors.validation_failed",
      },
      { refused: "a change by one no longer its admin", error: refusal(403, "forbidden"), message: "errors.forbidden" },
    ])
  )("shows nerve's reason for $refused, of $change, in a toast", async ({ by, error, message }) => {
    by.sent().mockRejectedValueOnce(error);
    await by.send();
    expect([underFields(setError), toasts]).toEqual([[], [{ type: "error", title: "toast.error", message }]]);
  });

  it.each([
    { refused: "the deletion of the default", by: deletion, status: 409, code: "project.state_default" },
    { refused: "the deletion of a group's only state", by: deletion, status: 409, code: "project.state_last_in_group" },
    { refused: "the move of a group's only state", by: move, status: 409, code: "project.state_last_in_group" },
    {
      refused: "a new default of a state deleted meanwhile",
      by: newDefault,
      status: 404,
      code: "project.state_not_found",
    },
  ])("shows nerve's reason when it refuses $refused, in a toast, and does not follow it", async (row) => {
    const { by, status, code } = row;
    by.sent().mockRejectedValueOnce(refusal(status, code));
    await by.send();
    const message = `errors.${code.replace(".", "_")}`;
    expect([toasts, done.mock.calls]).toEqual([[{ type: "error", title: "toast.error", message }], []]);
  });

  describe.each(Object.values(changes))("$change", ({ send, sent, answer }) => {
    it.each(lateSettlingsAnswering(answer))(
      "does nothing on the page when it $settles after another tab moved this one",
      async ({ settle }) => {
        const held = heldChange<State | undefined>();
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
