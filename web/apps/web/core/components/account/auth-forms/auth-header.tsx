/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import type { InvitationPreview } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
import { WorkspaceLogo } from "@/components/workspace/logo";
// helpers
import { EAuthModes } from "@/helpers/authentication.helper";
// hooks
import { useInvitationPreview } from "@/hooks/use-invitation-preview";

type TAuthHeader = {
  invitationId: string | null;
  token: string | null;
  authMode: EAuthModes;
};

/** The keys of the heading and the line under it, for each mode. */
const TITLE_KEYS = {
  [EAuthModes.SIGN_IN]: {
    header: "auth.common.header",
    subHeader: "auth.sign_in.sub_header",
  },
  [EAuthModes.SIGN_UP]: {
    header: "auth.common.header",
    subHeader: "auth.sign_up.sub_header",
  },
};

export const AuthHeader = observer(function AuthHeader(props: TAuthHeader) {
  const { invitationId, token, authMode } = props;
  // nerve imports
  const { t } = useTranslation();

  const { data: invitation, isLoading } = useInvitationPreview(invitationId, token);

  const getHeaderSubHeader = (mode: EAuthModes, current: InvitationPreview | undefined) => {
    if (current) {
      return {
        header: (
          <div className="relative inline-flex items-center gap-2">
            {t("common.join")}{" "}
            <WorkspaceLogo
              logo={current.workspace_logo_url}
              name={current.workspace_name}
              classNames="size-9 flex-shrink-0"
            />{" "}
            {current.workspace_name}
          </div>
        ),
        subHeader: t(
          mode === EAuthModes.SIGN_UP ? "auth.sign_up.invitation_sub_header" : "auth.sign_in.invitation_sub_header"
        ),
      };
    }

    return { header: t(TITLE_KEYS[mode].header), subHeader: t(TITLE_KEYS[mode].subHeader) };
  };

  const { header, subHeader } = getHeaderSubHeader(authMode, invitation);

  if (isLoading)
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LogoSpinner />
      </div>
    );

  return <AuthHeaderBase subHeader={subHeader} header={header} />;
});

type TAuthHeaderBase = {
  header: React.ReactNode;
  subHeader: string;
};

function AuthHeaderBase(props: TAuthHeaderBase) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-h4-semibold text-primary">{props.header}</span>
      <span className="text-h4-semibold text-placeholder">{props.subHeader}</span>
    </div>
  );
}
