/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// hooks
import { useProject } from "@/hooks/store/use-project";
// local imports
import { HomeLoader, NoProjectsEmptyState } from "./widgets";

/**
 * The home body (M1 design 3.14) is the no-projects empty state once the projects are loaded. The recent visits come
 * back with M7 (M3 design 3.1).
 */
export const HomeBody = observer(function HomeBody() {
  // store hooks
  const { loader } = useProject();

  if (loader !== "loaded") return <HomeLoader />;

  return <NoProjectsEmptyState />;
});
