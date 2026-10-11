/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useFormContext } from "react-hook-form";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { ProjectCreationForm } from "@/components/projects/create/use-create-project";

type Props = {
  handleClose: () => void;
};

function ProjectCreateButtons(props: Props) {
  const { t } = useTranslation();
  const { handleClose } = props;
  const {
    formState: { isSubmitting },
  } = useFormContext<ProjectCreationForm>();

  return (
    <div className="flex justify-end gap-2 border-t border-subtle py-4">
      <Button variant="secondary" size="lg" onClick={handleClose} disabled={isSubmitting}>
        {t("common.cancel")}
      </Button>
      <Button variant="primary" size="lg" type="submit" loading={isSubmitting}>
        {isSubmitting ? t("creating") : t("create_project")}
      </Button>
    </div>
  );
}

export default ProjectCreateButtons;
