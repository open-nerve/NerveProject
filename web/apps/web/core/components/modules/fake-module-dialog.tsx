/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// Stand-ins for what a module's dialog (modal.tsx, form.tsx) renders beside its own fields, for the tests of the dialog
// and of its page's host (actions-host.tsx): a test file mocks each of these modules with this one, for instance
// vi.mock("@/components/dropdowns/project/dropdown", () => import("@/components/modules/fake-module-dialog")):
// - the project's picker (ProjectDropdown) and the dates' (DateRangeDropdown), each a button that keeps its props, so
//   that the test picks as their lists would (their onChange, onSelect);
// - under the real members' dropdown (dropdown/member/dropdown.tsx), which lists a project's members as the member
//   store gives them: that store (useMember), its read (useDirectoryRead), and its list (MemberDropdownBase), a button
//   named by its placeholder ("lead", "members") that keeps its props, the members it lists among them.
// The member store holds the members of projects A and B, as their stores give them: A's ann and bob, and gus, its
// guest; B's cat, and dan, its guest; the workspace's, all five.

import type { ComponentProps } from "react";
import type { DateRangeDropdown as RealDateRangeDropdown } from "@/components/dropdowns/date-range";

/** The project's picker's props, as the form gives them. */
type ProjectPicker = {
  value: string | null;
  onChange: (projectId: string) => void;
  renderCondition?: (id: string) => boolean;
};
/** A members' list's props, as the members' dropdown gives them. */
type MemberList = {
  placeholder?: string;
  memberIds?: string[];
  value: string | string[] | null;
  onChange: (value: never) => void;
  invalid?: boolean;
  describedBy?: string;
};

/** The props each picker was last given: the project's, the dates', and each members' list by its placeholder. */
export const pickers: {
  project: ProjectPicker | undefined;
  dates: ComponentProps<typeof RealDateRangeDropdown> | undefined;
  members: Record<string, MemberList>;
} = { project: undefined, dates: undefined, members: {} };

/** The pickers as a test starts: none rendered. */
export function emptyPickers() {
  Object.assign(pickers, { project: undefined, dates: undefined, members: {} });
}

/** Each project's members, as its store gives them: by account id, whether a guest. */
const PROJECT_MEMBERS: Record<string, [id: string, guest: boolean][]> = {
  A: [
    ["u-ann", false],
    ["u-bob", false],
    ["u-gus", true],
  ],
  B: [
    ["u-cat", false],
    ["u-dan", true],
  ],
};

/** What a picker's button says as one of the form's fields, as the real ones pass it on to theirs. */
const fieldOf = ({ invalid, describedBy }: { invalid?: boolean; describedBy?: string }) => ({
  "aria-invalid": invalid,
  "aria-describedby": describedBy,
});

export function ProjectDropdown(props: ProjectPicker) {
  pickers.project = props;
  return <button type="button">{props.value}</button>;
}

export function DateRangeDropdown(props: ComponentProps<typeof RealDateRangeDropdown>) {
  pickers.dates = props;
  return (
    <button type="button" {...fieldOf(props)}>
      dates
    </button>
  );
}

export function MemberDropdownBase(props: MemberList) {
  pickers.members[props.placeholder ?? ""] = props;
  return (
    <button type="button" {...fieldOf(props)}>
      {props.placeholder}
    </button>
  );
}

export function useMember() {
  return {
    getUserDetails: () => undefined,
    project: {
      getProjectMemberIds: (projectId: string, includeGuestUsers: boolean) =>
        (PROJECT_MEMBERS[projectId] ?? []).filter(([, guest]) => includeGuestUsers || !guest).map(([id]) => id),
      fetchProjectMembers: () => Promise.resolve(undefined),
    },
    workspace: {
      workspaceMemberIds: Object.values(PROJECT_MEMBERS).flatMap((members) => members.map(([id]) => id)),
      fetchWorkspaceMembers: () => Promise.resolve(undefined),
    },
  };
}

export function useDirectoryRead() {
  return { load: () => Promise.resolve(), loading: false, error: undefined };
}

export function DirectoryReadError() {
  return null;
}
