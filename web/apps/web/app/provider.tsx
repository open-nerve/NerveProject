/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { lazy, Suspense } from "react";
import { useTheme } from "next-themes";
import { SWRConfig } from "swr";
// Nerve Imports
import { WEB_SWR_CONFIG } from "@nerve/constants";
import { TranslationProvider, useTranslation } from "@nerve/i18n";
import { Toast } from "@nerve/propel/toast";
// helpers
import { resolveGeneralTheme } from "@nerve/utils";
// mobx store provider
import { StoreProvider } from "@/lib/store-context";

// lazy imports
const AppProgressBar = lazy(function AppProgressBar() {
  return import("@/lib/b-progress/AppProgressBar");
});

const StoreWrapper = lazy(function StoreWrapper() {
  return import("@/lib/wrappers/store-wrapper");
});

const InstanceWrapper = lazy(function InstanceWrapper() {
  return import("@/lib/wrappers/instance-wrapper");
});

/** The toasts, their region named in the user's language: the translation's provider holds it. */
function AppToast({ theme }: { theme: string | undefined }) {
  const { t } = useTranslation();
  return <Toast theme={resolveGeneralTheme(theme)} label={t("aria_labels.toasts")} />;
}

export interface IAppProvider {
  children: React.ReactNode;
}

export function AppProvider(props: IAppProvider) {
  const { children } = props;
  // themes
  const { resolvedTheme } = useTheme();

  return (
    <StoreProvider>
      <>
        <AppProgressBar />
        <TranslationProvider>
          <AppToast theme={resolvedTheme} />
          <StoreWrapper>
            <InstanceWrapper>
              <Suspense>
                <SWRConfig value={WEB_SWR_CONFIG}>{children}</SWRConfig>
              </Suspense>
            </InstanceWrapper>
          </StoreWrapper>
        </TranslationProvider>
      </>
    </StoreProvider>
  );
}
