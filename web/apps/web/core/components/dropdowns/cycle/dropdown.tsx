/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { DirectoryReadError } from "@/components/issues/issue-detail-widgets/directory-read";
import { useCycles } from "@/hooks/store/use-cycles";
import { useCycleOptions } from "@/hooks/use-planning-options";
import type { TCycleDropdownBaseProps } from "./base";
import { CycleDropdownBase } from "./base";

type TCycleDropdownProps = Omit<
  TCycleDropdownBaseProps,
  "getCycleById" | "onDropdownOpen" | "options" | "readStatus"
> & {
  /** The work item's project, whose cycles are its options. */
  projectId: string | undefined;
};

/**
 * A work item's cycle dropdown (P7a spec 2.4): its options read as it opens (useCycleOptions), its value named by the
 * planning store; loading while that read is out, or while its caller says so (a form waiting for its project's lists,
 * P7a spec 2.6). Keyed by the project: a list open as the project changes closes, and opens next on the new project's
 * options (M2 closeout 4).
 */
export const CycleDropdown = observer(function CycleDropdown(props: TCycleDropdownProps) {
  const { projectId, isInitializing, ...dropdown } = props;
  const { workspaceSlug } = useParams();
  const { getCycleById } = useCycles();
  const { options, read } = useCycleOptions(workspaceSlug, projectId);
  return (
    <CycleDropdownBase
      key={projectId}
      {...dropdown}
      getCycleById={getCycleById}
      isInitializing={read.loading || isInitializing}
      options={options}
      readStatus={read.error ? <DirectoryReadError error={read.error} retry={read.load} /> : undefined}
      onDropdownOpen={read.load}
    />
  );
});
