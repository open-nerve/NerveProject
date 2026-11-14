/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { DirectoryReadError } from "@/components/issues/issue-detail-widgets/directory-read";
import { useModules } from "@/hooks/store/use-modules";
import { useModuleOptions } from "@/hooks/use-planning-options";
import type { TModuleDropdownBaseProps } from "./base";
import { ModuleDropdownBase } from "./base";

type TModuleDropdownProps = Omit<
  TModuleDropdownBaseProps,
  "getModuleById" | "isInitializing" | "onDropdownOpen" | "options" | "readStatus"
> & {
  /** The work item's project, whose modules are its options. */
  projectId: string | undefined;
};

/**
 * A work item's module dropdown (P7a spec 2.4): its options read as it opens (useModuleOptions), its values named by
 * the planning store. Keyed by the project: a list open as the project changes closes, and opens next on the new
 * project's options (M2 closeout 4).
 */
export const ModuleDropdown = observer(function ModuleDropdown(props: TModuleDropdownProps) {
  const { projectId, ...dropdown } = props;
  const { workspaceSlug } = useParams();
  const { getModuleById } = useModules();
  const { options, read } = useModuleOptions(workspaceSlug, projectId);
  return (
    <ModuleDropdownBase
      key={projectId}
      {...dropdown}
      getModuleById={getModuleById}
      isInitializing={read.loading}
      options={options}
      readStatus={read.error ? <DirectoryReadError error={read.error} retry={read.load} /> : undefined}
      onDropdownOpen={read.load}
    />
  );
});
