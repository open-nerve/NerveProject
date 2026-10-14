/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { cn } from "@nerve/utils";
// assets
import ProjectNotAuthorizedImg from "@/app/assets/auth/project-not-authorized.svg?url";
import Unauthorized from "@/app/assets/auth/unauthorized.svg?url";
import WorkspaceNotAuthorizedImg from "@/app/assets/auth/workspace-not-authorized.svg?url";
// components
import { PageHead } from "@/components/core/page-title";
// layouts
import DefaultLayout from "@/layouts/default-layout";

type Props = {
  section?: "settings" | "general";
  isProjectView?: boolean;
  className?: string;
};

export const NotAuthorizedView = observer(function NotAuthorizedView(props: Props) {
  const { section = "general", isProjectView = false, className } = props;
  const { t } = useTranslation();

  // assets
  const settingAsset = isProjectView ? ProjectNotAuthorizedImg : WorkspaceNotAuthorizedImg;
  const asset = section === "settings" ? settingAsset : Unauthorized;

  return (
    <DefaultLayout className={cn("bg-surface-1", className)}>
      <PageHead title={t("not_authorized_to_view")} />
      <div className="flex h-full w-full flex-col items-center justify-center gap-y-5 text-center">
        <div className="h-44 w-72">
          <img src={asset} className="h-[176px] w-[288px] object-contain" alt="" />
        </div>
        <h1 className="text-18 font-medium text-primary">{t("not_authorized_to_view")}</h1>
      </div>
    </DefaultLayout>
  );
});
