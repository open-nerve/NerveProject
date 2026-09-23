/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useParams } from "next/navigation";
// hooks
import { useProject } from "@/hooks/store/use-project";
// local imports
import { HomeLoader, NoProjectsEmptyState, RecentActivityWidget } from "./widgets";

/** The home body is a fixed list (M1 design 3.14): the no-projects empty state and the recent visits. */
export const HomeBody = observer(function HomeBody() {
  // router
  const { workspaceSlug } = useParams();
  // store hooks
  const { loader } = useProject();

  if (!workspaceSlug) return null;
  if (loader !== "loaded") return <HomeLoader />;

  return (
    <div className="relative flex h-full w-full flex-col gap-7">
      <NoProjectsEmptyState />
      <div className="py-4">
        <RecentActivityWidget workspaceSlug={workspaceSlug.toString()} />
      </div>
    </div>
  );
});
