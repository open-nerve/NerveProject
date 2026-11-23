/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it } from "vitest";
import type { IssueUpdate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { errorMessageKey } from "@/lib/error-messages";
import { intakeUpdateBody, updateBody } from "./issue-write-fields";

// The content change of a work item in its project's intake (P5a spec 2.2, the contract's trap C9): nerve takes seven
// fields of one (IntakeIssueUpdate, additionalProperties false), so any other field a caller gives fails the change
// before anything is sent, refused as nerve refuses a change of a work item in triage (issue.in_triage), in the same
// sentence (P7b spec 2.4, R4); the fields come from an update's request (updateBody), whose skip_activity rule holds.

/** What intakeUpdateBody throws for body, or nothing. */
function refusalOf(body: IssueUpdate): unknown {
  try {
    intakeUpdateBody(updateBody(body));
  } catch (error) {
    return error;
  }
  return undefined;
}

it("sends the seven fields of an intake's change as given", () => {
  const body: IssueUpdate = {
    name: "Renamed",
    description_html: "<p>new</p>",
    priority: "high",
    assignee_ids: ["u-ann"],
    label_ids: [],
    target_date: null,
  };
  expect(intakeUpdateBody(updateBody(body))).toEqual(body);
  expect(JSON.parse(JSON.stringify(intakeUpdateBody(updateBody({ name: "Only" }))))).toEqual({ name: "Only" });
  expect(intakeUpdateBody(updateBody({ description_html: "<p>same</p>", skip_activity: true }))).toEqual({
    description_html: "<p>same</p>",
    skip_activity: true,
  });
});

it.each<[keyof IssueUpdate, IssueUpdate]>([
  ["state_id", { name: "x", state_id: "s-todo" }],
  ["sort_order", { sort_order: 500 }],
  ["start_date", { start_date: "2026-11-01" }],
  ["parent_id", { parent_id: null }],
  ["cycle_id", { cycle_id: "c-1" }],
  ["module_ids", { module_ids: [] }],
])(
  "refuses an intake's change that gives %s, a field nerve does not take of one, as nerve's 409 says it",
  (_, body) => {
    const triage = new ApiError(409, { status: 409, code: "issue.in_triage", title: "Conflict" });
    expect(errorMessageKey(refusalOf(body))).toBe(errorMessageKey(triage));
  }
);

it("keeps updateBody's rule: skip_activity goes with description_html alone", () => {
  expect(() => intakeUpdateBody(updateBody({ name: "x", description_html: "<p>x</p>", skip_activity: true }))).toThrow(
    "skip_activity requires description_html alone"
  );
});
