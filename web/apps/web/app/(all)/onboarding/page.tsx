/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
import { OnboardingRoot } from "@/components/onboarding";
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useUser } from "@/hooks/store/user";
// wrappers
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";

// The onboarding lists the caller's workspaces, which decide its steps (OnboardingRoot, M3 design 7.4).
function OnboardingPage() {
  // store hooks
  const { data: user } = useUser();

  return (
    <AuthenticationWrapper pageType={EPageTypes.ONBOARDING}>
      <div className="relative flex size-full overflow-hidden rounded-lg bg-canvas transition-all duration-300 ease-in-out">
        <div className="size-full flex-grow overflow-hidden p-2 transition-all duration-300 ease-in-out">
          <div className="shadow-md relative flex h-full w-full flex-col overflow-hidden rounded-lg border border-subtle bg-surface-1">
            {user ? (
              <OnboardingRoot />
            ) : (
              <div className="grid h-full w-full place-items-center">
                <LogoSpinner />
              </div>
            )}
          </div>
        </div>
      </div>
    </AuthenticationWrapper>
  );
}

export default observer(OnboardingPage);
