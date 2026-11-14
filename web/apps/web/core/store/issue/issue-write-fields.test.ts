/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it } from "vitest";
import type { IssueCreate, IssueDetail, IssueUpdate } from "@nerve/api-client";
import type { IssueChange, ModuleSetChange } from "@nerve/types";
import { changeBody, createBody, updateBody } from "./issue-write-fields";

/** What the client puts on the wire: it sends the body as JSON, so a field left undefined is not sent at all. */
const sent = (body: IssueCreate | IssueUpdate): unknown => JSON.parse(JSON.stringify(body));
/** A held issue's own fields spread into a request by a caller; none of them is a field of the request. */
const withHeld = <T extends object>(input: T): T => ({
  id: "x",
  project_id: "A",
  workspace_id: "W",
  sequence_id: 7,
  updated_at: "2026-11-14T00:00:00Z",
  ...input,
});
const refused = "skip_activity requires description_html alone";

it("a creation sends the fields it names and leaves out whatever else the caller passed", () => {
  const input: IssueCreate = {
    name: "Write the notes",
    description_html: "<p>notes</p>",
    state_id: "S",
    priority: "high",
    assignee_ids: ["U"],
    label_ids: ["L"],
    start_date: "2026-11-01",
    target_date: "2026-11-30",
    parent_id: "P",
    cycle_id: "C",
  };
  expect(sent(createBody(withHeld(input)))).toStrictEqual(input);
});
it("a creation sends the cycle it is given and leaves out one it is not", () => {
  expect(sent(createBody({ name: "Write the notes", cycle_id: "C" }))).toStrictEqual({
    name: "Write the notes",
    cycle_id: "C",
  });
  expect(sent(createBody({ name: "Write the notes", cycle_id: undefined }))).toStrictEqual({ name: "Write the notes" });
});
it("an update sends the fields it names and leaves out whatever else the caller passed", () => {
  const input: IssueUpdate = {
    name: "Write the notes",
    description_html: "<p>notes</p>",
    state_id: "S",
    priority: "low",
    assignee_ids: [],
    label_ids: ["L"],
    sort_order: 0,
    start_date: "2026-11-01",
    target_date: "2026-11-30",
    parent_id: "P",
    cycle_id: "C",
  };
  expect(sent(updateBody(withHeld(input)))).toStrictEqual(input);
});
it("an update sends a cleared cycle as null and leaves out a field it was not given", () => {
  expect(sent(updateBody({ cycle_id: null }))).toStrictEqual({ cycle_id: null });
  expect(sent(updateBody({ parent_id: null, start_date: null, target_date: null }))).toStrictEqual({
    parent_id: null,
    start_date: null,
    target_date: null,
  });
  expect(sent(updateBody({ name: "Write the notes", cycle_id: undefined, skip_activity: undefined }))).toStrictEqual({
    name: "Write the notes",
  });
});
it("skip_activity goes with description_html alone", () => {
  for (const skip_activity of [true, false]) {
    const input = { description_html: "<p>same</p>", skip_activity };
    expect(sent(updateBody(input))).toStrictEqual(input);
    expect(sent(updateBody(withHeld(input))), "the held fields are left out before the check").toStrictEqual(input);
  }
});
it("skip_activity without description_html, beside another field, or not a boolean is refused before sending", () => {
  expect(() => updateBody({ skip_activity: true })).toThrow(refused);
  const beside: IssueUpdate[] = [
    { name: "Write the notes" },
    { state_id: "S" },
    { priority: "low" },
    { assignee_ids: [] },
    { label_ids: [] },
    { sort_order: 0 },
    { start_date: null },
    { target_date: null },
    { parent_id: null },
    { cycle_id: null },
    { cycle_id: "C" },
  ];
  for (const skip_activity of [true, false])
    for (const field of beside)
      expect(
        () => updateBody({ description_html: "<p>same</p>", skip_activity, ...field }),
        JSON.stringify({ skip_activity, ...field })
      ).toThrow(refused);
  expect(() => updateBody({ description_html: "<p>same</p>", skip_activity: "yes" } as unknown as IssueUpdate)).toThrow(
    refused
  );
});
// M6 P6b spec 2.5: the modules are passed through as the caller gives them; nerve alone decides whether they may be.
it("a creation sends the modules it is given and leaves out none it is not", () => {
  expect(sent(createBody(withHeld({ name: "Write the notes", module_ids: ["M1", "M2"] })))).toStrictEqual({
    name: "Write the notes",
    module_ids: ["M1", "M2"],
  });
  expect(sent(createBody({ name: "Write the notes", module_ids: [] }))).toStrictEqual({
    name: "Write the notes",
    module_ids: [],
  });
  expect(sent(createBody({ name: "Write the notes", module_ids: undefined }))).toStrictEqual({
    name: "Write the notes",
  });
});
it("an update sends the whole set of modules it is given, empty for none, and leaves them out when not given", () => {
  expect(sent(updateBody(withHeld({ module_ids: ["M2", "M3"] })))).toStrictEqual({ module_ids: ["M2", "M3"] });
  expect(sent(updateBody({ module_ids: [] }))).toStrictEqual({ module_ids: [] });
  expect(sent(updateBody({ name: "Write the notes", module_ids: undefined }))).toStrictEqual({
    name: "Write the notes",
  });
  expect(() => updateBody({ description_html: "<p>same</p>", skip_activity: true, module_ids: [] })).toThrow(refused);
});
// M6 P7a spec 2.2, R3: a change of a work item's modules is additions and removals, made into the whole set nerve
// replaces from the set the work item holds when its turn comes; ids the page does not know stay as they are held.
it("a change of the modules removes and adds to the set held, keeping its order and the ids it does not name", () => {
  const held = ["A", "G"];
  const modulesAfter = (from: readonly string[], modules: ModuleSetChange) => changeBody({ modules }, from).module_ids;
  expect(modulesAfter(held, { add: ["B"] })).toEqual(["A", "G", "B"]);
  expect(modulesAfter(held, { remove: ["A"] })).toEqual(["G"]);
  expect(modulesAfter(held, { add: ["A"] }), "adding one it is in").toEqual(["A", "G"]);
  expect(modulesAfter(held, { remove: ["B"] }), "removing one it is not in").toEqual(["A", "G"]);
  expect(modulesAfter(["G", "A", "C"], { add: ["B", "D"], remove: ["A"] })).toEqual(["G", "C", "B", "D"]);
  expect(modulesAfter([], { add: ["B"] })).toEqual(["B"]);
  expect(held, "the set held is not changed").toEqual(["A", "G"]);
});
it("a change's request names each field it sends: its fields as given, the modules as the whole set they make", () => {
  const held = ["A", "G"];
  const fields: IssueChange = {
    name: "Write the notes",
    state_id: "S",
    priority: "low",
    assignee_ids: [],
    label_ids: ["L"],
    sort_order: 0,
    start_date: "2026-11-01",
    target_date: null,
    parent_id: "P",
    cycle_id: null,
  };
  expect(sent(changeBody(withHeld(fields), held))).toStrictEqual(fields);
  expect(sent(changeBody({ name: "Write the notes", modules: { add: ["B"] } }, held))).toStrictEqual({
    name: "Write the notes",
    module_ids: ["A", "G", "B"],
  });
  expect(sent(changeBody({ modules: { remove: ["A", "G"] } }, held))).toStrictEqual({ module_ids: [] });
  expect(sent(changeBody({ description_html: "<p>same</p>", skip_activity: true }, held))).toStrictEqual({
    description_html: "<p>same</p>",
    skip_activity: true,
  });
});
it("a change's request leaves out whatever else a caller slipped into it, a whole set of modules too", () => {
  const slipped = { ...withHeld({ name: "Write the notes" }), module_ids: ["Z"] } as unknown as IssueChange;
  expect(sent(changeBody(slipped, ["A", "G"]))).toStrictEqual({ name: "Write the notes" });
  const both = { module_ids: ["Z"], modules: { add: ["B"] } } as unknown as IssueChange;
  expect(sent(changeBody(both, ["A", "G"]))).toStrictEqual({ module_ids: ["A", "G", "B"] });
});
it("a change cannot carry a whole set of modules, from a literal or from a work item's fields", () => {
  const detail: Partial<IssueDetail> = { name: "Write the notes", module_ids: ["A"] };
  // @ts-expect-error the change declares module_ids as never
  const declared: Required<IssueChange>["module_ids"] = ["A"];
  // @ts-expect-error a literal whole set is not a change
  const literal: IssueChange = { module_ids: ["A"] };
  // @ts-expect-error a work item's fields, as the rows and the spreadsheet held them, carry the whole set
  const fromDetail: IssueChange = detail;
  const change: IssueChange = { name: "Write the notes", cycle_id: null, modules: { add: ["B"], remove: ["A"] } };
  expect([declared, literal, fromDetail, change]).toHaveLength(4);
});
