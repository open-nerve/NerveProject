/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Switch } from "@makeplane/propel/components/switch";

type Props = {
  featureItem: { property: string };
  value: boolean;
  handleSubmit: (featureProperty: string) => void;
};

export function ProjectFeatureToggle(props: Props) {
  const { featureItem, value, handleSubmit } = props;
  return (
    <Switch
      size="sm"
      checked={value}
      onCheckedChange={() => handleSubmit(featureItem.property)}
      aria-label="Toggle project feature"
    />
  );
}
