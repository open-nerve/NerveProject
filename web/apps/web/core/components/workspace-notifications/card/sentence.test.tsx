// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { MemberUser, NotificationChange } from "@nerve/api-client";
import { notificationOf } from "@/store/notification/fake-notifications";
import { NotificationSentence } from "./sentence";

// A card's sentence (P4 spec 2.6, R11, R22): who made the change (useActorName) and what, as describeChange says it,
// in the page's language; a day as the calendar day nerve gives ("2026-10-17", re-93 m15), never as a moment, which a
// zone west of UTC shows as the day before. @nerve/i18n's text is its key, then the values it was given.

const members = vi.hoisted(() => new Map<string, Pick<MemberUser, "id" | "display_name">>());
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: (id: string) => members.get(id) }) }));

let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  members.clear();
  members.set("u-ann", { id: "u-ann", display_name: "ann" });
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

const base = notificationOf("one");
/** The sentence of a notification of the change, made by the account (none: nerve). */
async function sentence(change: Partial<NotificationChange>, actor: string | null = "u-ann") {
  const notification = {
    ...base,
    triggered_by_id: actor,
    data: { ...base.data, change: { ...base.data.change, ...change } },
  };
  await act(async () => root.render(<NotificationSentence notification={notification} />));
  return host.textContent;
}

describe("NotificationSentence", () => {
  it("says who made the change and what, as describeChange says it", async () => {
    expect(await sentence({ field: "name", verb: "updated", old_value: "Fix", new_value: "Mend" })).toBe(
      "ann notification.card.renamed Mend"
    );
  });

  it("names nerve for a change nerve made, and a deactivated user for an account the members no longer list", async () => {
    expect(await sentence({ field: "archived_at", verb: "updated", new_value: "archive" }, null)).toBe(
      "Nerve notification.card.archived"
    );
    expect(await sentence({ field: "attachment", verb: "created" }, "u-gone")).toBe(
      "common.deactivated_user notification.card.attachment_added"
    );
  });

  it("shows a priority by its text", async () => {
    expect(await sentence({ field: "priority", verb: "updated", old_value: "high", new_value: "urgent" })).toBe(
      "ann notification.card.priority urgent"
    );
  });

  describe("in a zone west of UTC", () => {
    // A "YYYY-MM-DD" read as a moment is UTC's midnight, the evening before here. The zone is the test's alone
    // (vi.stubEnv); the CI's zone is UTC, where it does not show.
    beforeAll(() => {
      vi.stubEnv("TZ", "America/Los_Angeles");
    });
    afterAll(() => {
      vi.unstubAllEnvs();
    });

    it("runs west of UTC", () => {
      expect(new Date(2026, 9, 17).getTimezoneOffset()).toBe(420);
    });

    it("shows a day nerve gives as that calendar day (re-93 m15)", async () => {
      expect(await sentence({ field: "start_date", verb: "updated", old_value: "", new_value: "2026-10-17" })).toBe(
        "ann notification.card.start_date_set Oct 17, 2026"
      );
      expect(await sentence({ field: "target_date", verb: "updated", old_value: "", new_value: "2026-11-01" })).toBe(
        "ann notification.card.due_date_set Nov 01, 2026"
      );
    });
  });
});
