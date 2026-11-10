/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { AssigneeDistribution, LabelDistribution } from "@nerve/api-client";
import { userOf } from "@/store/member/fake-users";
import { labelOf, projectOf } from "@/store/project/fake-projects";
import { assigneeRows, labelRows } from "./distribution-rows";

// The rows of a cycle's work items by assignee and by label, named for the page (P5a spec 2.4): nerve's rows carry
// ids alone (P4 R3), the project's members and labels give the names; one no longer the project's (in a frozen
// progress) is named by a fallback text. A text is its key here (t gives the key back).

const t = (key: string) => key;
const web = projectOf("WEB", "w-acme");

const ann = userOf("Ann");
const bob = userOf("bob");
const zoe = userOf("Zoe");
/** The project's members' profiles by account id, as the project's members store gives them. */
const memberOf = (accountId: string) => [zoe, ann, bob].find((member) => member.id === accountId);

/** nerve's row of the work items of an assignee (null: without one), of which completed are completed. */
const assigned = (assigneeId: string | null, total = 2, completed = 1): AssigneeDistribution => ({
  assignee_id: assigneeId,
  total_issues: total,
  completed_issues: completed,
  pending_issues: total - completed,
});

const bug = labelOf(web, "bug", 1, { color: "#EF4444" });
const docs = labelOf(web, "Docs", 2, { color: "#3B82F6" });
const api = labelOf(web, "api", 3, { color: "" });
const labels = [docs, bug, api];

/** nerve's row of the work items of a label (null: without one), of which completed are completed. */
const labelled = (labelId: string | null, total = 2, completed = 1): LabelDistribution => ({
  label_id: labelId,
  total_issues: total,
  completed_issues: completed,
  pending_issues: total - completed,
});

describe("assigneeRows", () => {
  it("names an assignee by the project's member, with his avatar, and keeps nerve's counts", () => {
    expect(assigneeRows([assigned(ann.id, 3, 1)], memberOf, t)).toEqual([
      { assignee_id: ann.id, total_issues: 3, completed_issues: 1, pending_issues: 2, name: "Ann", avatarOf: ann.id },
    ]);
  });

  it("names one no longer the project's member by the fallback text, without an avatar", () => {
    expect(assigneeRows([assigned("u-gone", 4, 4)], memberOf, t)).toEqual([
      {
        assignee_id: "u-gone",
        total_issues: 4,
        completed_issues: 4,
        pending_issues: 0,
        name: "distribution.former_member",
        avatarOf: null,
      },
    ]);
  });

  it("names the work items without an assignee", () => {
    expect(assigneeRows([assigned(null, 5, 2)], memberOf, t)).toEqual([
      {
        assignee_id: null,
        total_issues: 5,
        completed_issues: 2,
        pending_issues: 3,
        name: "no_assignee",
        avatarOf: null,
      },
    ]);
  });

  it("orders by name, then those no longer members, the work items without an assignee last", () => {
    const rows = assigneeRows(
      [
        assigned("u-gone-1"),
        assigned(zoe.id),
        assigned(null),
        assigned(bob.id),
        assigned("u-gone-2"),
        assigned(ann.id),
      ],
      memberOf,
      t
    );
    expect(rows.map(({ assignee_id, name }) => [assignee_id, name])).toEqual([
      [ann.id, "Ann"],
      [bob.id, "bob"],
      [zoe.id, "Zoe"],
      ["u-gone-1", "distribution.former_member"],
      ["u-gone-2", "distribution.former_member"],
      [null, "no_assignee"],
    ]);
  });
});

describe("labelRows", () => {
  it("names a label by the project's label, with its color, and keeps nerve's counts", () => {
    expect(labelRows([labelled(bug.id, 3, 2)], labels, t)).toEqual([
      { label_id: bug.id, total_issues: 3, completed_issues: 2, pending_issues: 1, name: "bug", color: "#EF4444" },
    ]);
  });

  it("gives a label without a color none", () => {
    expect(labelRows([labelled(api.id)], labels, t)).toEqual([
      { label_id: api.id, total_issues: 2, completed_issues: 1, pending_issues: 1, name: "api", color: null },
    ]);
  });

  it("names a label no longer the project's by the fallback text, without a color", () => {
    expect(labelRows([labelled("l-gone", 1, 0)], labels, t)).toEqual([
      {
        label_id: "l-gone",
        total_issues: 1,
        completed_issues: 0,
        pending_issues: 1,
        name: "distribution.deleted_label",
        color: null,
      },
    ]);
  });

  it("names the work items without a label", () => {
    expect(labelRows([labelled(null, 6, 3)], labels, t)).toEqual([
      {
        label_id: null,
        total_issues: 6,
        completed_issues: 3,
        pending_issues: 3,
        name: "distribution.no_labels",
        color: null,
      },
    ]);
  });

  it("orders by name, then the labels no longer the project's, the work items without a label last", () => {
    const rows = labelRows(
      [
        labelled("l-gone-1"),
        labelled(docs.id),
        labelled(null),
        labelled(bug.id),
        labelled("l-gone-2"),
        labelled(api.id),
      ],
      labels,
      t
    );
    expect(rows.map(({ label_id, name }) => [label_id, name])).toEqual([
      [api.id, "api"],
      [bug.id, "bug"],
      [docs.id, "Docs"],
      ["l-gone-1", "distribution.deleted_label"],
      ["l-gone-2", "distribution.deleted_label"],
      [null, "distribution.no_labels"],
    ]);
  });
});
