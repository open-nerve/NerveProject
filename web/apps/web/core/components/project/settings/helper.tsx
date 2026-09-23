/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Switch } from "@makeplane/propel/components/switch";

type Props = {
  featureItem: { key: string; property: string };
  value: boolean;
  handleSubmit: (featureKey: string, featureProperty: string) => void;
  disabled?: boolean;
};

export function ProjectFeatureToggle(props: Props) {
  const { featureItem, value, handleSubmit, disabled } = props;
  return (
    <Switch
      size="sm"
      checked={value}
      onCheckedChange={() => handleSubmit(featureItem.key, featureItem.property)}
      disabled={disabled}
      aria-label="Toggle project feature"
    />
  );
}
