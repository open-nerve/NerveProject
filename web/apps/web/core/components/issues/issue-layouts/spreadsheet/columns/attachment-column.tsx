/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import type { Issue } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Row } from "@nerve/ui";

/** The server's count of the work item's live attachments, never the length of a loaded page. */
export const SpreadsheetAttachmentColumn = observer(function SpreadsheetAttachmentColumn({ issue }: { issue: Issue }) {
  const { t } = useTranslation();
  const count = issue.attachment_count;
  return (
    <Row className="flex h-11 w-full items-center border-b-[0.5px] border-subtle px-page-x py-1 text-11 hover:bg-layer-1">
      {count} {t("content_files.count_unit", { count })}
    </Row>
  );
});
