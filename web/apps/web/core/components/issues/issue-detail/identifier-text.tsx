/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@nerve/i18n";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { TIdentifierTextProps, TIdentifierTextVariant, TIssueIdentifierSize } from "@nerve/types";
import { cn } from "@nerve/utils";
import { useCopyText } from "@/hooks/use-copy-link";

const SIZE_MAP: Record<TIssueIdentifierSize, string> = {
  xs: "text-caption-sm-regular",
  sm: "text-caption-sm-medium",
  md: "text-caption-md-medium",
  lg: "text-caption-lg-medium",
};

const VARIANT_MAP: Record<TIdentifierTextVariant, string> = {
  default: "text-tertiary",
  secondary: "text-secondary",
  tertiary: "text-tertiary",
  primary: "text-primary",
  "primary-subtle": "text-primary/80",
  success: "text-success-primary",
};

export function IdentifierText(props: TIdentifierTextProps) {
  const { identifier, enableClickToCopyIdentifier = false, size = "lg", variant = "default" } = props;
  const { t } = useTranslation();
  const copyText = useCopyText();

  const textSizeClassName = SIZE_MAP[size];
  const variantClassName = VARIANT_MAP[variant];
  const className = cn("text-12 font-medium whitespace-nowrap text-tertiary", textSizeClassName, variantClassName);
  // A display-only identifier is also rendered inside links, menu triggers and listbox options.
  if (!enableClickToCopyIdentifier) return <span className={className}>{identifier}</span>;

  return (
    <Tooltip label={t("common.actions.copy_id")}>
      <button
        type="button"
        className={cn(className, "cursor-pointer")}
        aria-label={`${t("common.actions.copy_id")}: ${identifier}`}
        onClick={() => copyText(identifier, t("common.work_item_id_copied"))}
      >
        {identifier}
      </button>
    </Tooltip>
  );
}
