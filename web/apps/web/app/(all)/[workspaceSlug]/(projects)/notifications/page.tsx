/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
// components
import { PageHead } from "@/components/core/page-title";
import { NotificationsRoot } from "@/components/workspace-notifications";
// hooks
import { useRouteWorkspace } from "@/hooks/use-route-workspace";

/**
 * The notifications page (M7 P4 spec 2.6): titled the inbox of the route's workspace (useRouteWorkspace, never the
 * router store's workspace, which follows the address a commit late), and its preview; the layout holds its sidebar
 * and its state.
 */
function NotificationsPage() {
  const { t } = useTranslation();
  const workspace = useRouteWorkspace();
  const pageTitle = workspace ? t("notification.page_label", { workspace: workspace.name }) : undefined;

  return (
    <>
      <PageHead title={pageTitle} />
      <NotificationsRoot />
    </>
  );
}

export default observer(NotificationsPage);
