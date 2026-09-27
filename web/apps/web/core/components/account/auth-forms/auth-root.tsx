/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useSearchParams } from "react-router";
// helpers
import type { EAuthModes } from "@/helpers/authentication.helper";
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
  // query params: a workspace invitation's, for M3's "join the workspace" title (M2 design 7.3)
  const invitation_id = searchParams.get("invitation_id");
  const workspaceSlug = searchParams.get("slug");

  return (
    <div className="mt-10 flex w-full flex-grow flex-col items-center justify-center py-6">
      <div className="relative flex w-full max-w-[22.5rem] flex-col gap-6">
        <AuthHeader
          workspaceSlug={workspaceSlug || undefined}
          invitationId={invitation_id || undefined}
          authMode={authMode}
        />
        <AuthPasswordForm mode={authMode} />
      </div>
    </div>
  );
});
