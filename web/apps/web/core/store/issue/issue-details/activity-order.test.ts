/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it } from "vitest";
import { compareActivityOrder } from "./activity-order";

it("orders nanoseconds before ids across equivalent timezone spellings", () => {
  const rows = [
    { id: "a", created_at: "2026-10-30T08:00:00.000000009+08:00" },
    { id: "z", created_at: "2026-10-30T00:00:00.000000001Z" },
    { id: "b", created_at: "2026-10-30T00:00:00.000000009Z" },
  ];
  expect(rows.toSorted(compareActivityOrder).map((row) => row.id)).toEqual(["z", "a", "b"]);
});
it("uses kind only for entries with the same timestamp and id", () => {
  const time = "2026-10-30T00:00:00Z";
  const rows = [
    { id: "same", kind: "comment" as const, created_at: time },
    { id: "same", kind: "activity" as const, created_at: time },
  ];
  expect(rows.toSorted(compareActivityOrder).map((row) => row.kind)).toEqual(["activity", "comment"]);
});
