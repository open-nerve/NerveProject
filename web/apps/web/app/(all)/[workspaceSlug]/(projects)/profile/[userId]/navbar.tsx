/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { NavLink, useParams } from "react-router";
// plane imports
import { PROFILE_TABS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Header, EHeaderVariant } from "@nerve/ui";
import { cn } from "@nerve/utils";

type Props = {
  isAuthorized: boolean;
};

export function ProfileNavbar(props: Props) {
  const { isAuthorized } = props;
  const { t } = useTranslation();
  const { workspaceSlug, userId } = useParams();

  const tabsList = isAuthorized ? PROFILE_TABS : [];

  return (
    <Header variant={EHeaderVariant.SECONDARY} showOnMobile={false}>
      <div className="flex items-center overflow-x-scroll">
        {tabsList.map((tab) => (
          <NavLink key={tab.route} to={`/${workspaceSlug}/profile/${userId}/${tab.route}`}>
            {({ isActive }) => (
              <span
                className={cn(
                  `flex border-b-2 p-4 text-13 font-medium whitespace-nowrap text-tertiary outline-none hover:text-primary ${
                    isActive
                      ? "border-accent-strong text-accent-primary hover:text-accent-primary"
                      : "border-transparent"
                  }`
                )}
              >
                {t(tab.i18n_label)}
              </span>
            )}
          </NavLink>
        ))}
      </div>
    </Header>
  );
}
