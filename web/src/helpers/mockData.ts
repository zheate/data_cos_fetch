import type { CosGroupResponse, CosRow, DataFetchResponse, DataFetchRow, Step1Params } from './types';
import { useDataFetchStore } from '../stores/data-fetch-store';
import { useCosFilterStore } from '../stores/cos-filter-store';

export function getMockDataFetchResponse(): DataFetchResponse {
  const records: DataFetchRow[] = [];
  const entries = ['SH-MOD-2026-001', 'SH-MOD-2026-002', 'SH-MOD-2026-003', 'CP-CHIP-B08'];
  const categories = ['LVI', 'Rth', 'lambd'];
  const currents = [10, 12, 14, 16, 18, 20];

  let idCounter = 1;
  for (const entry of entries) {
    for (const cat of categories) {
      for (const curr of currents) {
        const basePower = 8.5 + (curr - 10) * 1.4 + (idCounter % 3) * 0.2;
        const voltage = 1.72 + (curr - 10) * 0.04;
        const efficiency = (basePower / (curr * voltage)) * 100;
        const lambda = 904.8 + (curr - 10) * 0.12 + (idCounter % 4) * 0.05;

        records.push({
          entry_id: entry,
          test_category: cat,
          current_a: curr,
          power_w: Number(basePower.toFixed(3)),
          voltage_v: Number(voltage.toFixed(3)),
          efficiency_pct: Number(efficiency.toFixed(1)),
          lambda_nm: Number(lambda.toFixed(3)),
          shift_nm: Number(((curr - 10) * 0.12).toFixed(3)),
          wavelength_2a_nm: Number((lambda - 0.32).toFixed(3)),
          wavelength_cold_nm: Number((lambda - 5.8).toFixed(3)),
        });
        idCounter++;
      }
    }
  }

  return {
    total: records.length,
    records,
    errors: [
      '[Entry: SH-MOD-ERR-09] Excel 解析异常：检测到文档透明加密锁，无法读取字节流。',
      '[Entry: SH-MOD-ERR-12] 路径未找到：Z:/Ldtd/archive/202602/raw_lvi.xlsx (os error 2)',
    ],
    infos: [
      '[Entry: SH-MOD-2026-001] 匹配到 3 个测试站别目录：FCP_Station_01, FCP_Station_02',
      '[Entry: CP-CHIP-B08] 自动关联晶圆批次号 WAFER-2026-B8',
    ],
  };
}

export function getMockCosFilterData() {
  const warehouses = ['良品仓', '良品仓', '良品仓', '待确认仓', '产线暂存仓'];
  const owners = ['研发一部', '制造工程部', '质量中心'];
  const itemNums = ['ITM-905-2026A', 'ITM-905-2026B'];
  const boxNums = ['BOX-01', 'BOX-02', 'BOX-03', 'BOX-04', 'BOX-05', 'BOX-06'];

  const allRows: CosRow[] = [];
  const totalChips = 110;

  for (let i = 1; i <= totalChips; i++) {
    const center = 904.0 + (i % 12) * 0.28 + Math.sin(i) * 0.4;
    allRows.push({
      device_id: `COS-2026-09A-${String(i).padStart(4, '0')}`,
      warehouse: warehouses[i % warehouses.length],
      isolation: i % 18 === 0 ? '是' : '否',
      item_num: itemNums[i % itemNums.length],
      box_num: boxNums[i % boxNums.length],
      owner: owners[i % owners.length],
      center_wavelength_nm: Number(center.toFixed(3)),
      two_a_wavelength_nm: Number((center - 0.35).toFixed(3)),
      cold_wavelength_nm: Number((center - 5.82).toFixed(3)),
      peak_wavelength_nm: Number((center + 0.12).toFixed(3)),
    });
  }

  // 10 groups of 8 chips = 80 chips grouped
  const groups: CosRow[][] = [];
  const groupCount = 10;
  const groupSize = 8;

  for (let g = 0; g < groupCount; g++) {
    const groupRows: CosRow[] = [];
    const baseWavelength = 904.2 + g * 0.25;
    for (let c = 0; c < groupSize; c++) {
      const chipIndex = g * groupSize + c;
      const row = allRows[chipIndex] || allRows[0];
      const tunedCenter = baseWavelength + (c - 4) * 0.025;
      groupRows.push({
        ...row,
        center_wavelength_nm: Number(tunedCenter.toFixed(3)),
        two_a_wavelength_nm: Number((tunedCenter - 0.35).toFixed(3)),
        cold_wavelength_nm: Number((tunedCenter - 5.82).toFixed(3)),
      });
    }
    groups.push(groupRows);
  }

  const remaining = allRows.slice(80, 98); // 18 remaining

  const groupResult: CosGroupResponse = {
    group_count: groups.length,
    remaining_count: remaining.length,
    groups,
    remaining,
  };

  const step1Params: Step1Params = {
    wavelength_field: 'center',
    wavelength_min_nm: 903.5,
    wavelength_max_nm: 907.5,
    required_count: 8,
  };

  return {
    loadedCosCount: totalChips,
    step1Rows: allRows,
    step2Rows: allRows.filter((r) => r.isolation !== '是'),
    step1Params,
    groupResult,
    groupingDedupRemoved: 2,
  };
}

export function getMockCosStep4Data(groups: CosRow[][]): DataFetchResponse {
  const records: DataFetchRow[] = [];
  const flatChips = groups.flat();
  for (const chip of flatChips) {
    const chipNum = parseInt(chip.device_id.replace(/\D/g, '').slice(-3)) || 1;
    const power = 9.5 + (chipNum % 7) * 0.15 + Math.sin(chipNum) * 0.2;
    const voltage = 1.82 + (chipNum % 5) * 0.02;
    const curr = 10;
    const eff = (power / (curr * voltage)) * 100;

    records.push({
      entry_id: chip.device_id,
      test_category: 'LVI',
      current_a: curr,
      power_w: Number(power.toFixed(3)),
      voltage_v: Number(voltage.toFixed(3)),
      efficiency_pct: Number(eff.toFixed(1)),
      lambda_nm: chip.center_wavelength_nm ?? null,
      shift_nm: 0.15,
      wavelength_2a_nm: chip.two_a_wavelength_nm ?? null,
      wavelength_cold_nm: chip.cold_wavelength_nm ?? null,
    });
  }
  return {
    total: records.length,
    records,
    errors: [],
    infos: [`匹配成功: 共从测试数据源提取到 ${records.length} 颗 COS 器件的实测 LVI 数据`],
  };
}

export function injectAllMockData() {
  // 1. Data Fetch Store
  const dataFetch = getMockDataFetchResponse();
  useDataFetchStore.getState().setEntriesInput('SH-MOD-2026-001\nSH-MOD-2026-002\nSH-MOD-2026-003\nCP-CHIP-B08');
  useDataFetchStore.getState().setResult(dataFetch);

  // 2. COS Filter Store
  const cosData = getMockCosFilterData();
  useCosFilterStore.getState().setLoadedCosCount(cosData.loadedCosCount);
  useCosFilterStore.getState().setStep1Rows(cosData.step1Rows);
  useCosFilterStore.getState().setStep2Rows(cosData.step2Rows);
  useCosFilterStore.getState().setStep1Params(cosData.step1Params);
  useCosFilterStore.getState().setGroupResult(cosData.groupResult);
  useCosFilterStore.getState().setGroupingDedupRemoved(cosData.groupingDedupRemoved);
  useCosFilterStore.getState().setSelectedGroupIndex(0);

  // 3. Step 4 Electrical Mock Data
  const step4Data = getMockCosStep4Data(cosData.groupResult.groups);
  useCosFilterStore.getState().setStep4Result(step4Data);
}

export function clearAllMockData() {
  useDataFetchStore.getState().setResult(null);
  useCosFilterStore.getState().setLoadedCosCount(0);
  useCosFilterStore.getState().setStep1Rows([]);
  useCosFilterStore.getState().setStep2Rows([]);
  useCosFilterStore.getState().setStep1Params(null);
  useCosFilterStore.getState().setGroupResult(null);
  useCosFilterStore.getState().setGroupingDedupRemoved(0);
  useCosFilterStore.getState().setStep4Result(null);
}

