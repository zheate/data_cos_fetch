export type WavelengthField = 'cold' | 'center' | 'two_a' | 'peak';
export type ExtractionMode = 'module' | 'chip';
export type GroupingMode = 'greedy' | 'optimal' | 'flat_top' | 'huang_meng';
export type FlatTopStrategy = 'max_group_rate' | 'max_uniformity';
export type WavelengthLabel = '2A波长' | '冷波长' | '中心波长' | '峰值波长';
export type CosGroupResultTab = 'groups' | 'remaining' | 'trend' | 'export';

export type DataFetchRow = {
  entry_id: string;
  test_category: string;
  current_a: number | null;
  power_w: number | null;
  voltage_v: number | null;
  efficiency_pct: number | null;
  lambda_nm: number | null;
  shift_nm: number | null;
  wavelength_2a_nm: number | null;
  wavelength_cold_nm: number | null;
};

export type CosRow = {
  device_id: string;
  warehouse?: string | null;
  isolation?: string | null;
  item_num?: string | null;
  box_num?: string | null;
  owner?: string | null;
  cold_wavelength_nm?: number | null;
  center_wavelength_nm?: number | null;
  two_a_wavelength_nm?: number | null;
  peak_wavelength_nm?: number | null;
};

export type DataFetchResponse = {
  total: number;
  records: DataFetchRow[];
  errors: string[];
  infos: string[];
};

export type CosStepResponse = {
  total: number;
  records: CosRow[];
};

export type CosBatchFile = {
  file_path: string;
  file_name: string;
  modified_epoch_s: number;
  size_bytes: number;
};

export type CosBatchListResponse = {
  files: CosBatchFile[];
  total: number;
};

export type CosGroupResponse = {
  group_count: number;
  remaining_count: number;
  groups: CosRow[][];
  remaining: CosRow[];
};

export type Step1Params = {
  wavelength_field: WavelengthField;
  wavelength_min_nm: number;
  wavelength_max_nm: number;
  required_count: number;
};

export type GroupSummaryRow = {
  groupId: string;
  count: number;
  avg: number | null;
  min: number | null;
  max: number | null;
  diff: number | null;
  cv: number | null;
};

export type HistogramBin = {
  start: number;
  end: number;
  count: number;
};
