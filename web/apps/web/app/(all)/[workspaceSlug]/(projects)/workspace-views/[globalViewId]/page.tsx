/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
// components
import { PageHead } from "@/components/core/page-title";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";

function GlobalViewIssuesPage() {
  // router
  // store hooks
  const { currentWorkspace } = useWorkspace();
  // states

  // derived values
  const pageTitle = currentWorkspace?.name ? `${currentWorkspace?.name} - All Views` : undefined;

  // handlers
  return (
    <>
      <PageHead title={pageTitle} />
    </>
  );
}

export default observer(GlobalViewIssuesPage);
