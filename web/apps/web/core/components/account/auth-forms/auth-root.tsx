/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useState } from "react";
import { observer } from "mobx-react";
import { useSearchParams } from "react-router";
// nerve imports
import { Banner } from "@makeplane/propel/components/banner";
// helpers
import type { EAuthModes, EAuthenticationErrorCodes, TAuthErrorInfo } from "@/helpers/authentication.helper";
import { authErrorHandler } from "@/helpers/authentication.helper";
// local imports
import { AuthHeader } from "./auth-header";
import { AuthPasswordForm } from "./password";

type TAuthRoot = {
  authMode: EAuthModes;
};

export const AuthRoot = observer(function AuthRoot(props: TAuthRoot) {
  const { authMode } = props;
  //router
  const [searchParams] = useSearchParams();
  // query params
  const emailParam = searchParams.get("email");
  const invitation_id = searchParams.get("invitation_id");
  const workspaceSlug = searchParams.get("slug");
  const error_code = searchParams.get("error_code");
  const nextPath = searchParams.get("next_path");
  // states
  const [errorInfo, setErrorInfo] = useState<TAuthErrorInfo | undefined>(undefined);

  useEffect(() => {
    if (error_code) setErrorInfo(authErrorHandler(error_code as EAuthenticationErrorCodes, emailParam ?? undefined));
  }, [error_code, emailParam]);

  return (
    <div className="mt-10 flex w-full flex-grow flex-col items-center justify-center py-6">
      <div className="relative flex w-full max-w-[22.5rem] flex-col gap-6">
        {errorInfo && (
          <Banner
            placement="inline"
            variant="accent"
            role="alert"
            description={errorInfo.message}
            onDismiss={() => setErrorInfo(undefined)}
          />
        )}
        <AuthHeader
          workspaceSlug={workspaceSlug || undefined}
          invitationId={invitation_id || undefined}
          invitationEmail={emailParam || undefined}
          authMode={authMode}
        />
        <AuthPasswordForm mode={authMode} email={emailParam ?? ""} nextPath={nextPath || undefined} />
      </div>
    </div>
  );
});
