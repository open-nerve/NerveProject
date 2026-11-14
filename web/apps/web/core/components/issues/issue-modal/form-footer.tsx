/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDescriptionLoadState } from "./types";
import type { MutableRefObject } from "react";
// i18n
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
// hooks
import { Switch } from "@makeplane/propel/components/switch";

type IssueFormFooterProps = {
  /** The edited work item's or draft's id; a new one has none. */
  id: string | undefined;
  isDraft: boolean;
  canSaveDraft: boolean;
  descriptionLoadState: IssueDescriptionLoadState;
  /**
   * Whether the lists of the project the form opened with are out: until they answer, what it holds of the cycle and
   * the modules a caller gave is not yet what it may send (P7a spec 2.6; re-73 M1), and it saves nothing.
   */
  planningLoading: boolean;
  isCreateMoreToggleEnabled: boolean;
  onCreateMoreToggleChange: (value: boolean) => void;
  isSubmitting: boolean;
  isPending: boolean;
  isMoving: boolean;
  moveToIssue: boolean;
  primaryButtonText: {
    default: string;
    loading: string;
  };
  submitBtnRef: MutableRefObject<HTMLButtonElement | null>;
  onClose: () => void;
  onSaveDraft: () => Promise<void>;
  onMoveToProjects: () => Promise<void>;
};

/** The form's buttons: create more, discard, save, save to drafts and publish. */
export function IssueFormFooter(props: IssueFormFooterProps) {
  const { t } = useTranslation();
  const {
    id,
    isDraft,
    canSaveDraft,
    descriptionLoadState,
    planningLoading,
    isCreateMoreToggleEnabled,
    onCreateMoreToggleChange,
    isSubmitting,
    isPending,
    isMoving,
    moveToIssue,
    primaryButtonText,
    submitBtnRef,
    onClose,
    onSaveDraft,
    onMoveToProjects,
  } = props;

  return (
    <div className="flex items-center justify-end gap-4 border-t-[0.5px] border-subtle pt-6 pb-3">
      {!id && (
        <div className="inline-flex items-center gap-1.5">
          <Switch
            size="sm"
            checked={isCreateMoreToggleEnabled}
            onCheckedChange={onCreateMoreToggleChange}
            disabled={isSubmitting || isPending}
            aria-label={t("create_more")}
          />
          <span className="text-caption-sm-regular">{t("create_more")}</span>
        </div>
      )}
      <div className="flex items-center gap-2">
        <div>
          <Button variant="secondary" size="lg" disabled={isSubmitting || isPending || isMoving} onClick={onClose}>
            {t("discard")}
          </Button>
        </div>
        <div>
          <Button
            variant={moveToIssue ? "secondary" : "primary"}
            size="lg"
            type="submit"
            ref={submitBtnRef}
            loading={isSubmitting}
            disabled={isSubmitting || isPending || isMoving || descriptionLoadState !== "ready" || planningLoading}
          >
            {isSubmitting ? primaryButtonText.loading : primaryButtonText.default}
          </Button>
        </div>

        {!id && !isDraft && canSaveDraft && (
          <Button
            variant="secondary"
            type="button"
            size="lg"
            disabled={isSubmitting || isPending || descriptionLoadState !== "ready"}
            onClick={onSaveDraft}
          >
            {t("save_to_drafts")}
          </Button>
        )}

        {moveToIssue && (
          <Button
            variant="primary"
            type="button"
            loading={isMoving}
            onClick={onMoveToProjects}
            disabled={isMoving || isSubmitting || isPending || descriptionLoadState !== "ready"}
            size="lg"
          >
            {t("add_to_project")}
          </Button>
        )}
      </div>
    </div>
  );
}
