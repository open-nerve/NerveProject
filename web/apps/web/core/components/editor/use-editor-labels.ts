import type { EditorLabels } from "@nerve/editor";
import { useTranslation } from "@nerve/i18n";

/** Both editor hosts supply the current language to their shared React controls and NodeViews. */
export function useEditorLabels(): EditorLabels {
  const { t } = useTranslation();
  return {
    continueWriting: t("aria_labels.editor.continue_writing"),
    resizeImage: t("aria_labels.editor.resize_image"),
    tableRowOptions: t("aria_labels.editor.table_row_options"),
    tableColumnOptions: t("aria_labels.editor.table_column_options"),
    tableColors: {
      gray: t("aria_labels.editor.table_colors.gray"),
      peach: t("aria_labels.editor.table_colors.peach"),
      pink: t("aria_labels.editor.table_colors.pink"),
      orange: t("aria_labels.editor.table_colors.orange"),
      green: t("aria_labels.editor.table_colors.green"),
      "light-blue": t("aria_labels.editor.table_colors.light-blue"),
      "dark-blue": t("aria_labels.editor.table_colors.dark-blue"),
      purple: t("aria_labels.editor.table_colors.purple"),
      none: t("aria_labels.editor.table_colors.none"),
    },
  };
}
