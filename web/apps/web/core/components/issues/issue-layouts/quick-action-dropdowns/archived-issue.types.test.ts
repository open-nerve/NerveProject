/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The archive's row menu (P7c spec 2.6, A18): its items (restore, a new tab, the link, deletion) open no dialog, so it
// is given none: neither the work item's edit (the setters of a dialog it holds, a dialog to name) nor whether it is
// allowed, which only the edit and the copy read.
import { expect, it } from "vitest";
import type { Issue } from "@nerve/api-client";
import type { useArchivedIssueMenuItems } from "./helper";

type ArchivedMenuInput = Parameters<typeof useArchivedIssueMenuItems>[0];

const setIssueToEdit = (_issue: Issue | undefined) => {};
const setCreateUpdateIssueModal = (_open: boolean) => {};
const openDialog = (_kind: "edit" | "copy") => {};

/** Never called: what the archive's row gives its menu, then with each part of the edit's. */
function inputs(issue: Issue): ArchivedMenuInput[] {
  const row = { issue, workspaceSlug: "ws", isDeletingAllowed: true, setDeleteIssueModal: () => {} };
  return [
    row,
    // @ts-expect-error not the setters of a dialog it holds
    { ...row, setIssueToEdit, setCreateUpdateIssueModal },
    // @ts-expect-error nor a dialog to name
    { ...row, openDialog },
    // @ts-expect-error nor whether it may edit
    { ...row, isEditingAllowed: false },
    // @ts-expect-error nor the empty setters, its edit turned off, it was once given
    { ...row, isEditingAllowed: false, setIssueToEdit: () => {}, setCreateUpdateIssueModal: () => {} },
  ];
}

it("gives the archive's row menu no dialog to open", () => {
  expect(inputs).toBeTypeOf("function");
});
