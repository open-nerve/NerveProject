/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ArrowNarrowRightOutline } from "@makeplane/propel/icons";
import { Tooltip } from "@makeplane/propel/components/tooltip";
// assets

// components
import { DetailResourceError } from "../issue-detail-widgets/resource-error";
// hooks
import { useTranslation } from "@nerve/i18n";
import { usePlatformOS } from "@/hooks/use-platform-os";

type TIssuePeekOverviewError = {
  removeRoutePeekId: () => void;
  error: unknown;
  retry?: () => Promise<unknown>;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
};

export function IssuePeekOverviewError(props: TIssuePeekOverviewError) {
  const { removeRoutePeekId } = props;
  // hooks
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();

  return (
    <div className="relative flex h-full w-full flex-col overflow-hidden">
      <div className="flex flex-shrink-0 justify-start">
        <Tooltip label={t("common.close_peek_view")} disabled={isMobile}>
          <button
            type="button"
            aria-label={t("common.close_peek_view")}
            onClick={removeRoutePeekId}
            className="m-5 h-5 w-5"
          >
            <ArrowNarrowRightOutline className="h-4 w-4 text-tertiary hover:text-secondary" />
          </button>
        </Tooltip>
      </div>

      <div className="h-full w-full">{props.retry && <DetailResourceError {...props} retry={props.retry} />}</div>
    </div>
  );
}
