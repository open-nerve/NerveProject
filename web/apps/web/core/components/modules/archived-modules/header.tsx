/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { ArchiveTabsList } from "@/components/archives";
// local imports
import { ModuleViewHeader } from "../module-view-header";

type Props = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * The header of a project's archive of modules: the archive's tabs, and the search, the order and the filters of its
 * modules (the archive's own search and filters; the project's one order, which its list shares).
 */
export function ArchivedModulesHeader({ workspaceSlug, projectId }: Props) {
  return (
    <div className="group relative flex border-b border-subtle">
      <div className="horizontal-scrollbar scrollbar-sm flex w-full items-center gap-2 overflow-x-auto px-4">
        <ArchiveTabsList />
      </div>
      <div className="flex h-full items-center gap-3 self-end px-8">
        <ModuleViewHeader workspaceSlug={workspaceSlug} projectId={projectId} list="archived" />
      </div>
    </div>
  );
}
