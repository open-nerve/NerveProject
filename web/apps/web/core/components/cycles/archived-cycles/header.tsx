/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { ArchiveTabsList } from "@/components/archives";
// local imports
import { CyclesViewHeader } from "../cycles-view-header";

type Props = {
  projectId: string;
};

/** The header of a project's archive of cycles: the archive's tabs, and the search and the filters of its cycles. */
export function ArchivedCyclesHeader({ projectId }: Props) {
  return (
    <div className="group relative flex border-b border-subtle">
      <div className="horizontal-scrollbar scrollbar-sm flex w-full items-center gap-2 overflow-x-auto px-4">
        <ArchiveTabsList />
      </div>
      <div className="flex h-full items-center gap-3 self-end px-8">
        <CyclesViewHeader projectId={projectId} list="archived" />
      </div>
    </div>
  );
}
