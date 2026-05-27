import { useMemo } from 'react';
import type { CosGroupResponse, Step1Params } from '../helpers/types';
import { buildHistogram, getWavelengthValue } from '../helpers/utils';
import { WavelengthHistogram } from './WavelengthHistogram';

type CosTrendPanelProps = {
  groupResult: CosGroupResponse | null;
  step1Params: Step1Params | null;
};

export function CosTrendPanel({ groupResult, step1Params }: CosTrendPanelProps) {
  const groupedWavelengths = useMemo(() => {
    if (!groupResult || !step1Params) {
      return [];
    }

    const values: number[] = [];
    for (const group of groupResult.groups ?? []) {
      for (const row of group ?? []) {
        const wavelength = getWavelengthValue(row, step1Params.wavelength_field);
        if (wavelength !== null) {
          values.push(wavelength);
        }
      }
    }
    return values;
  }, [groupResult, step1Params]);

  const groupDiffValues = useMemo(() => {
    if (!groupResult || !step1Params) {
      return [];
    }

    const values: number[] = [];
    for (const group of groupResult.groups ?? []) {
      let min = Number.POSITIVE_INFINITY;
      let max = Number.NEGATIVE_INFINITY;
      let count = 0;

      for (const row of group ?? []) {
        const wavelength = getWavelengthValue(row, step1Params.wavelength_field);
        if (wavelength === null) {
          continue;
        }
        min = Math.min(min, wavelength);
        max = Math.max(max, wavelength);
        count += 1;
      }

      if (count > 0) {
        const diff = max - min;
        if (diff > 0) {
          values.push(diff);
        }
      }
    }

    return values;
  }, [groupResult, step1Params]);

  const groupedWavelengthHistogram = useMemo(
    () => buildHistogram(groupedWavelengths, 24),
    [groupedWavelengths],
  );
  const groupDiffHistogram = useMemo(() => buildHistogram(groupDiffValues, 20), [groupDiffValues]);

  return (
    <div className="grid gap-4 xl:grid-cols-2">
      <WavelengthHistogram
        title="入组波长直方分布"
        unit="nm"
        bins={groupedWavelengthHistogram}
        valueCount={groupedWavelengths.length}
      />
      <WavelengthHistogram
        title="组内极差频数图"
        unit="nm"
        bins={groupDiffHistogram}
        valueCount={groupDiffValues.length}
      />
    </div>
  );
}
