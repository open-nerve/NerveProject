/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Workspace } from "@nerve/api-client";
import { heldChange, lateSettlings, lateSettlingsAnswering, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import {
  creationRefusal,
  slugFrom,
  useCreateWorkspace,
  type CreationForm,
  type CreationRefusal,
} from "./use-create-workspace";

// How a creation form creates a workspace (M3 design 2 W1, 7.1): the hook runs as a plain function, with stand-ins
// for the workspace store's check of the slug and its creation, which nerve answers when the test says, and for the
// form's setError. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({ checkWorkspaceSlug: vi.fn(), createWorkspace: vi.fn(), setError: vi.fn() }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({ checkWorkspaceSlug: page.checkWorkspaceSlug, createWorkspace: page.createWorkspace }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const form: CreationForm = { name: "Acme", slug: "acme", organization_size: "2-10" };
const acme = workspaceOf("acme", { name: "Acme", role: 20 });
const create = (values = form) => useCreateWorkspace()(values, page.setError);

beforeEach(() => {
  signedIn();
  page.checkWorkspaceSlug.mockReset();
  page.checkWorkspaceSlug.mockResolvedValue({ available: true });
  page.createWorkspace.mockReset();
  page.createWorkspace.mockResolvedValue(acme);
  page.setError.mockReset();
  toasts.length = 0;
});

describe("useCreateWorkspace", () => {
  it("asks whether the slug is free, creates the workspace from the form, says so and gives it", async () => {
    expect(await create()).toBe(acme);
    expect([page.checkWorkspaceSlug.mock.calls, page.createWorkspace.mock.calls]).toEqual([[["acme"]], [[form]]]);
    expect(page.setError.mock.calls).toEqual([]);
    expect(toasts).toEqual([
      {
        type: "success",
        title: "workspace_creation.toast.success.title",
        message: "workspace_creation.toast.success.message",
      },
    ]);
  });

  it("sends no size when the form has none chosen", async () => {
    await create({ ...form, organization_size: null });
    expect(page.createWorkspace.mock.calls).toEqual([[{ name: "Acme", slug: "acme" }]]);
  });

  it.each([
    { reason: "taken", message: "workspace_creation.errors.validation.url_already_taken" },
    { reason: "reserved", message: "workspace_creation.errors.validation.url_reserved" },
    { reason: "invalid", message: "workspace_creation.errors.validation.url_alphanumeric" },
  ])("sends nothing for a slug nerve says is $reason, and says so under it", async ({ reason, message }) => {
    page.checkWorkspaceSlug.mockResolvedValueOnce({ available: false, reason });
    expect(await create()).toBeUndefined();
    expect([page.createWorkspace.mock.calls, page.setError.mock.calls, toasts]).toEqual([
      [],
      [["slug", { type: "server", message }]],
      [],
    ]);
  });

  it("shows nerve's refusal under the fields it names", async () => {
    page.createWorkspace.mockRejectedValueOnce(
      refusal(422, "validation_failed", [
        { field: "name", code: "contains_url" },
        { field: "slug", code: "not_allowed" },
      ])
    );
    expect(await create()).toBeUndefined();
    expect([page.setError.mock.calls, toasts]).toEqual([
      [
        ["name", { type: "server", message: "errors.field.contains_url" }],
        ["slug", { type: "server", message: "workspace_creation.errors.validation.url_reserved" }],
      ],
      [],
    ]);
  });

  it("shows nerve's reason when it names no field of the form", async () => {
    page.createWorkspace.mockRejectedValueOnce(refusal(403, "workspace.creation_disabled"));
    expect(await create()).toBeUndefined();
    expect(page.setError.mock.calls).toEqual([]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.workspace_creation_disabled" }]);
  });

  it("sends nothing when nerve refuses the slug's check, and shows its reason", async () => {
    page.checkWorkspaceSlug.mockRejectedValueOnce(refusal(429, "rate_limited"));
    expect(await create()).toBeUndefined();
    expect([page.createWorkspace.mock.calls, page.setError.mock.calls, toasts]).toEqual([
      [],
      [],
      [{ type: "error", title: "toast.error", message: "errors.rate_limited" }],
    ]);
  });

  it.each(lateSettlingsAnswering(acme))(
    "neither speaks nor gives the workspace when the creation $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const created = heldChange<Workspace>();
      page.createWorkspace.mockReturnValueOnce(created.sent);
      const creating = create();
      await vi.waitFor(() => expect(page.createWorkspace).toHaveBeenCalledTimes(1));
      switchAccount();
      settle(created);
      expect(await creating).toBeUndefined();
      expect([page.setError.mock.calls, toasts]).toEqual([[], []]);
    }
  );

  it.each(lateSettlings)(
    "neither says why nor sends anything when the slug's check $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const checked = heldChange<undefined>();
      page.checkWorkspaceSlug.mockReturnValueOnce(checked.sent.then(() => ({ available: false, reason: "taken" })));
      const creating = create();
      await vi.waitFor(() => expect(page.checkWorkspaceSlug).toHaveBeenCalledTimes(1));
      switchAccount();
      settle(checked);
      expect(await creating).toBeUndefined();
      expect([page.createWorkspace.mock.calls, page.setError.mock.calls, toasts]).toEqual([[], [], []]);
    }
  );
});

describe("creationRefusal", () => {
  it.each<{ refused: string; error: unknown; shown: CreationRefusal }>([
    {
      refused: "a taken slug",
      error: refusal(409, "workspace.slug_taken"),
      shown: { kind: "fields", fields: { slug: "workspace_creation.errors.validation.url_already_taken" } },
    },
    {
      refused: "a reserved slug and a name with a web address",
      error: refusal(422, "validation_failed", [
        { field: "slug", code: "not_allowed" },
        { field: "name", code: "contains_url" },
      ]),
      shown: {
        kind: "fields",
        fields: { slug: "workspace_creation.errors.validation.url_reserved", name: "errors.field.contains_url" },
      },
    },
    {
      refused: "a field the form does not have",
      error: refusal(422, "validation_failed", [
        { field: "name", code: "too_long" },
        { field: "timezone", code: "invalid_format" },
      ]),
      shown: { kind: "toast" },
    },
    {
      refused: "creation switched off",
      error: refusal(403, "workspace.creation_disabled"),
      shown: { kind: "toast" },
    },
    {
      refused: "a failure without an answer",
      error: new TypeError("offline"),
      shown: { kind: "toast" },
    },
  ])("shows $refused", ({ error, shown }) => {
    expect(creationRefusal(error)).toEqual(shown);
  });
});

describe("slugFrom", () => {
  it.each([
    { typed: "Acme Two", slug: "acme-two" },
    { typed: "acme ", slug: "acme-" },
  ])("makes '$typed' into '$slug'", ({ typed, slug }) => {
    expect(slugFrom(typed)).toBe(slug);
  });
});
