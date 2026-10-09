/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
import { Link, useLocation } from "react-router";
import { SITE_NAME } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { NerveLockup } from "@/components/common/nerve-logo";
import { PageHead } from "@/components/core/page-title";
import { EAuthModes } from "@/helpers/authentication.helper";
import { useInstance } from "@/hooks/store/use-instance";

const authContentMap = {
  [EAuthModes.SIGN_IN]: {
    pageTitle: "auth.sign_in.title",
    text: "auth.common.new_to_nerve",
    linkText: "auth.sign_up.title",
    linkHref: "/sign-up",
  },
  [EAuthModes.SIGN_UP]: {
    pageTitle: "auth.sign_up.title",
    text: "auth.common.already_have_an_account",
    linkText: "auth.sign_in.title",
    linkHref: "/",
  },
};

type AuthHeaderProps = {
  type: EAuthModes;
};

export const AuthHeader = observer(function AuthHeader({ type }: AuthHeaderProps) {
  const { t } = useTranslation();
  // router: the other page gets this one's query, an invitation's link and the way back to it (M3 design 7.4)
  const { search } = useLocation();
  // store
  const { config } = useInstance();
  // derived values: the link to the sign-up page shows only while sign-up is open; the link to the sign-in page always
  // does, as an invitation's link opens the sign-up page of a closed nerve too (M3 design 7.4)
  const showsLink = type === EAuthModes.SIGN_UP || (config?.signup_enabled ?? false);

  return (
    <AuthHeaderBase
      pageTitle={t(authContentMap[type].pageTitle)}
      additionalAction={
        showsLink && (
          <div className="flex flex-col items-end text-center text-13 font-medium text-tertiary sm:flex-row sm:items-center sm:gap-2">
            <span className="text-body-sm-regular text-tertiary">{t(authContentMap[type].text)}</span>
            <Link
              to={{ pathname: authContentMap[type].linkHref, search }}
              className="text-body-sm-semibold text-accent-primary hover:underline"
            >
              {t(authContentMap[type].linkText)}
            </Link>
          </div>
        )
      }
    />
  );
});

type TAuthHeaderBase = {
  pageTitle: string;
  additionalAction?: React.ReactNode;
};

function AuthHeaderBase(props: TAuthHeaderBase) {
  const { pageTitle, additionalAction } = props;
  return (
    <>
      <PageHead title={`${pageTitle} - ${SITE_NAME}`} />
      <div className="sticky top-0 flex w-full flex-shrink-0 items-center justify-between gap-6">
        <Link to="/">
          <NerveLockup className="h-5 w-auto" />
        </Link>
        {additionalAction}
      </div>
    </>
  );
}
