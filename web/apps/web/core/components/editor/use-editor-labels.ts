import type { EditorLabels } from "@nerve/editor";
import { useTranslation } from "@nerve/i18n";

/** Both editor hosts supply the current language to their shared React controls and NodeViews. */
export function useEditorLabels(): EditorLabels {
  const { t } = useTranslation();
  return {
    image: {
      select: t("content_files.image.select"),
      cancel: t("content_files.image.cancel"),
      retry: t("content_files.image.retry"),
      unavailable: t("content_files.image.unavailable"),
      loading: t("content_files.image.loading"),
      copyUnknown: t("content_files.image.copyUnknown"),
      cancelled: t("content_files.image.cancelled"),
      invalid: t("content_files.image.invalid"),
      size: t("content_files.image.size"),
      name: t("content_files.image.name"),
      tooMany: t("content_files.image.tooMany"),
      queued: t("content_files.image.queued"),
      preparing: t("content_files.image.preparing"),
      uploading: t("content_files.image.uploading"),
      confirming: t("content_files.image.confirming"),
      download: t("content_files.image.download"),
      downloading: t("content_files.image.downloading"),
      downloadFailed: t("content_files.image.downloadFailed"),
      fullScreen: t("content_files.image.fullScreen"),
      close: t("content_files.image.close"),
      zoomIn: t("content_files.image.zoomIn"),
      zoomOut: t("content_files.image.zoomOut"),
      align: t("content_files.image.align"),
      left: t("content_files.image.left"),
      center: t("content_files.image.center"),
      right: t("content_files.image.right"),
    },
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
