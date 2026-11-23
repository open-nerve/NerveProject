/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDetail } from "@nerve/api-client";

import { observer } from "mobx-react";
// nerve imports
import { Field, FieldError } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { useTranslation } from "@nerve/i18n";

type TInboxIssueTitle = {
  data: Partial<IssueDetail>;
  handleData: (
    issueKey: keyof Partial<IssueDetail>,
    issueValue: Partial<IssueDetail>[keyof Partial<IssueDetail>]
  ) => void;
  /** The name's hint (P5b spec 2.1, R5): the field is invalid, and described by it, exactly while it shows. */
  error: string | undefined;
};

export const InboxIssueTitle = observer(function InboxIssueTitle(props: TInboxIssueTitle) {
  const { data, handleData, error } = props;
  const { t } = useTranslation();
  return (
    <div className="space-y-1">
      <Field name="name" invalid={error !== undefined}>
        <InputGroup size="2xl">
          <Input
            size="2xl"
            id="name"
            name="name"
            type="text"
            value={data?.name}
            onChange={(e) => handleData("name", e.target.value)}
            placeholder={t("title")}
            aria-label={t("title")}
          />
        </InputGroup>
        {error !== undefined && (
          <FieldError size="2xl" match>
            {error}
          </FieldError>
        )}
      </Field>
    </div>
  );
});
