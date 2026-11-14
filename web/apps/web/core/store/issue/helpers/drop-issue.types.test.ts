/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A drop's change gives a work item's modules as additions and removals, made into the whole set in the write's turn
// (P7b spec 2.7, R3): a drag function that gives module_ids, as an update or a work item's fields carry them, cannot be
// handed to the drop of a page's list or of the page's actions.
import { expect, it } from "vitest";
import type { Issue, IssueUpdate } from "@nerve/api-client";
import type { IssueActions } from "@/hooks/use-issues-actions";
import type { BaseIssuesStore } from "./base-issues.store";

/** Drag functions: one giving the whole set, one giving an update, and one giving a change. */
const whole = (held: Issue) => ({ module_ids: [...held.module_ids, "beta"] });
const update = (held: Issue): IssueUpdate => ({ priority: held.priority });
const change = (held: Issue) => ({ modules: { remove: held.module_ids.slice(0, 1), add: ["beta"] } });

/** Never called: the drops of a page's list and of its actions, given each drag function. */
function drops(store: BaseIssuesStore, actions: Required<IssueActions>) {
  return [
    // @ts-expect-error a drop's change cannot give the modules as a whole set
    store.dropIssue("ws", "A", "x", whole, { group: "beta" }),
    // @ts-expect-error nor as an update, which may carry them
    store.dropIssue("ws", "A", "x", update, { group: "beta" }),
    // @ts-expect-error nor through the page's actions
    actions.dropIssue("A", "x", whole, { group: "beta" }),
    store.dropIssue("ws", "A", "x", change, { group: "beta" }),
  ];
}

it("takes a drop's modules as additions and removals only", () => {
  expect(drops).toBeTypeOf("function");
});
