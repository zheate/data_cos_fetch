import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type {
  CosBatchFile,
  CosGroupResultTab,
  CosGroupResponse,
  CosRow,
  FlatTopStrategy,
  GroupingMode,
  Step1Params,
  WavelengthLabel,
  CosBatchListResponse,
  CosStepResponse,
  DataFetchResponse,
  CosStep4ExtractPayload,
} from '../helpers/types';
import { request } from '../helpers/api';
import { useAppStore } from './app-store';
import {
  dedupeByDeviceId,
  parseCurrentPoints,
  parseLines,
  parseOptionalNumber,
  WAVELENGTH_LABEL_TO_FIELD,
} from '../helpers/utils';

export type CosFilterPreset = {
  name: string;
  wavelengthLabel: WavelengthLabel;
  wavelengthMin: number;
  wavelengthMax: number;
  requiredCount: number;
  selectedItemNums: string[];
  boxCountInput: number;
  manualBoxOverride: boolean;
  manualSelectedBoxes: string[];
  groupingMode: GroupingMode;
  groupSize: number;
  maxDiff: number;
  avgMinInput: string;
  avgMaxInput: string;
  flatTopStrategy: FlatTopStrategy;
  huangLowMin: number;
  huangLowMax: number;
  huangHighMin: number;
  huangHighMax: number;
};

type CosFilterState = {
  // File selection
  batchDirectory: string;
  batchFiles: CosBatchFile[];
  cosFilePath: string;

  // Data pipeline
  loadedCosCount: number;
  step1Rows: CosRow[];
  step1Params: Step1Params | null;
  step2Rows: CosRow[];
  groupResult: CosGroupResponse | null;
  groupingDedupRemoved: number;

  // Step 1 form
  wavelengthLabel: WavelengthLabel;
  wavelengthMin: number;
  wavelengthMax: number;
  requiredCount: number;

  // Step 2 form
  selectedItemNums: string[];
  boxCountInput: number;
  manualBoxOverride: boolean;
  manualSelectedBoxes: string[];

  // Step 3 form
  groupingMode: GroupingMode;
  groupSize: number;
  maxDiff: number;
  avgMinInput: string;
  avgMaxInput: string;
  flatTopStrategy: FlatTopStrategy;
  huangLowMin: number;
  huangLowMax: number;
  huangHighMin: number;
  huangHighMax: number;

  // Presets
  presets: CosFilterPreset[];

  // Result tab
  groupResultTab: CosGroupResultTab;
  selectedGroupIndex: number;

  // Actions
  setBatchDirectory: (v: string) => void;
  setBatchFiles: (v: CosBatchFile[]) => void;
  setCosFilePath: (v: string) => void;
  setLoadedCosCount: (v: number) => void;
  setStep1Rows: (v: CosRow[]) => void;
  setStep1Params: (v: Step1Params | null) => void;
  setStep2Rows: (v: CosRow[]) => void;
  setGroupResult: (v: CosGroupResponse | null) => void;
  setGroupingDedupRemoved: (v: number) => void;
  setWavelengthLabel: (v: WavelengthLabel) => void;
  setWavelengthMin: (v: number) => void;
  setWavelengthMax: (v: number) => void;
  setRequiredCount: (v: number) => void;
  setSelectedItemNums: (v: string[]) => void;
  setBoxCountInput: (v: number) => void;
  setManualBoxOverride: (v: boolean) => void;
  setManualSelectedBoxes: (v: string[]) => void;
  setGroupingMode: (v: GroupingMode) => void;
  setGroupSize: (v: number) => void;
  setMaxDiff: (v: number) => void;
  setAvgMinInput: (v: string) => void;
  setAvgMaxInput: (v: string) => void;
  setFlatTopStrategy: (v: FlatTopStrategy) => void;
  setHuangLowMin: (v: number) => void;
  setHuangLowMax: (v: number) => void;
  setHuangHighMin: (v: number) => void;
  setHuangHighMax: (v: number) => void;
  setGroupResultTab: (v: CosGroupResultTab) => void;
  setSelectedGroupIndex: (v: number) => void;

  // Step 4 (Power & Electrical Data Extraction)
  step4Result: DataFetchResponse | null;
  step4Target: 'grouped' | 'selected_group' | 'step2' | 'step1';
  step4Measurements: string[];
  step4CurrentInput: string;
  step4ChipRootsInput: string;
  setStep4Result: (v: DataFetchResponse | null) => void;
  setStep4Target: (v: 'grouped' | 'selected_group' | 'step2' | 'step1') => void;
  setStep4Measurements: (v: string[]) => void;
  setStep4CurrentInput: (v: string) => void;
  setStep4ChipRootsInput: (v: string) => void;

  savePreset: (name: string) => void;
  loadPreset: (name: string) => void;
  deletePreset: (name: string) => void;

  /** Reset entire COS flow data (keeps form defaults). */
  resetCosFlow: () => void;

  // UX State
  activeStep: number;
  setActiveStep: (v: number) => void;

  // Async Actions
  loadCosBatchFiles: (overrideDirectory?: string) => Promise<number>;
  loadCosFile: () => Promise<void>;
  runStep1: () => Promise<void>;
  runStep2: (step2SourceRows: CosRow[]) => Promise<void>;
  runGrouping: (step2Rows: CosRow[]) => Promise<void>;
  runStep4Extract: () => Promise<void>;
};

export const useCosFilterStore = create<CosFilterState>()(
  persist(
    (set, get) => ({
      batchDirectory: '',
      batchFiles: [],
      cosFilePath: '',
      loadedCosCount: 0,
      step1Rows: [],
      step1Params: null,
      step2Rows: [],
      groupResult: null,
      groupingDedupRemoved: 0,
      wavelengthLabel: '2A波长',
      wavelengthMin: 900,
      wavelengthMax: 1000,
      requiredCount: 100000,
      selectedItemNums: [],
      boxCountInput: 0,
      manualBoxOverride: false,
      manualSelectedBoxes: [],
      groupingMode: 'greedy',
      groupSize: 20,
      maxDiff: 0.2,
      avgMinInput: '',
      avgMaxInput: '',
      flatTopStrategy: 'max_group_rate',
      huangLowMin: 963.2,
      huangLowMax: 963.5,
      huangHighMin: 965.0,
      huangHighMax: 965.2,
      presets: [],
      groupResultTab: 'groups',
      selectedGroupIndex: 0,
      activeStep: 0,

      // Step 4 state
      step4Result: null,
      step4Target: 'grouped',
      step4Measurements: ['LVI'],
      step4CurrentInput: '',
      step4ChipRootsInput: 'Z:/Ldtd/\nZ:/Ldtd/Ldtd/',

      setBatchDirectory: (v) => set({ batchDirectory: v }),
      setBatchFiles: (v) => set({ batchFiles: v }),
      setCosFilePath: (v) => set({ cosFilePath: v }),
      setLoadedCosCount: (v) => set({ loadedCosCount: v }),
      setStep1Rows: (v) => set({ step1Rows: v }),
      setStep1Params: (v) => set({ step1Params: v }),
      setStep2Rows: (v) => set({ step2Rows: v }),
      setGroupResult: (v) => set({ groupResult: v }),
      setGroupingDedupRemoved: (v) => set({ groupingDedupRemoved: v }),
      setWavelengthLabel: (v) => set({ wavelengthLabel: v }),
      setWavelengthMin: (v) => set({ wavelengthMin: v }),
      setWavelengthMax: (v) => set({ wavelengthMax: v }),
      setRequiredCount: (v) => set({ requiredCount: v }),
      setSelectedItemNums: (v) => set({ selectedItemNums: v }),
      setBoxCountInput: (v) => set({ boxCountInput: v }),
      setManualBoxOverride: (v) => set({ manualBoxOverride: v }),
      setManualSelectedBoxes: (v) => set({ manualSelectedBoxes: v }),
      setGroupingMode: (v) => set({ groupingMode: v }),
      setGroupSize: (v) => set({ groupSize: v }),
      setMaxDiff: (v) => set({ maxDiff: v }),
      setAvgMinInput: (v) => set({ avgMinInput: v }),
      setAvgMaxInput: (v) => set({ avgMaxInput: v }),
      setFlatTopStrategy: (v) => set({ flatTopStrategy: v }),
      setHuangLowMin: (v) => set({ huangLowMin: v }),
      setHuangLowMax: (v) => set({ huangLowMax: v }),
      setHuangHighMin: (v) => set({ huangHighMin: v }),
      setHuangHighMax: (v) => set({ huangHighMax: v }),
      setGroupResultTab: (v) => set({ groupResultTab: v }),
      setSelectedGroupIndex: (v) => set({ selectedGroupIndex: v }),

      setStep4Result: (v) => set({ step4Result: v }),
      setStep4Target: (v) => set({ step4Target: v }),
      setStep4Measurements: (v) => set({ step4Measurements: v }),
      setStep4CurrentInput: (v) => set({ step4CurrentInput: v }),
      setStep4ChipRootsInput: (v) => set({ step4ChipRootsInput: v }),

      savePreset: (name) => {
        const state = get();
        const newPreset: CosFilterPreset = {
          name,
          wavelengthLabel: state.wavelengthLabel,
          wavelengthMin: state.wavelengthMin,
          wavelengthMax: state.wavelengthMax,
          requiredCount: state.requiredCount,
          selectedItemNums: state.selectedItemNums,
          boxCountInput: state.boxCountInput,
          manualBoxOverride: state.manualBoxOverride,
          manualSelectedBoxes: state.manualSelectedBoxes,
          groupingMode: state.groupingMode,
          groupSize: state.groupSize,
          maxDiff: state.maxDiff,
          avgMinInput: state.avgMinInput,
          avgMaxInput: state.avgMaxInput,
          flatTopStrategy: state.flatTopStrategy,
          huangLowMin: state.huangLowMin,
          huangLowMax: state.huangLowMax,
          huangHighMin: state.huangHighMin,
          huangHighMax: state.huangHighMax,
        };
        set({ presets: [...state.presets.filter((p) => p.name !== name), newPreset] });
      },
      loadPreset: (name) => {
        const preset = get().presets.find((p) => p.name === name);
        if (preset) {
          const { name: presetName, ...rest } = preset;
          void presetName;
          set(rest as Partial<CosFilterState>);
        }
      },
      deletePreset: (name) => set((s) => ({ presets: s.presets.filter((p) => p.name !== name) })),

      resetCosFlow: () =>
        set({
          loadedCosCount: 0,
          step1Rows: [],
          step1Params: null,
          step2Rows: [],
          groupResult: null,
          groupingDedupRemoved: 0,
          selectedItemNums: [],
          manualBoxOverride: false,
          manualSelectedBoxes: [],
          groupResultTab: 'groups',
          selectedGroupIndex: 0,
          activeStep: 0,
          step4Result: null,
        }),
      setActiveStep: (v) => set({ activeStep: v }),
      
      loadCosBatchFiles: async (overrideDirectory) => {
        const state = get();
        const { apiBase, token } = useAppStore.getState();
        const directory = (overrideDirectory ?? state.batchDirectory).trim();
        const payload = directory.length > 0 ? { directory } : {};
        const result = await request<CosBatchListResponse>(apiBase, token, '/api/v1/cos-filter/files', payload);
        const files = result?.files ?? [];
        set({ batchFiles: files });

        if (files.length > 0) {
          const first = files[0].file_path;
          if (!state.cosFilePath || !files.some((f) => f.file_path === state.cosFilePath)) {
            set({ cosFilePath: first });
          }
        }
        return result?.total ?? 0;
      },

      loadCosFile: async () => {
        const state = get();
        const { apiBase, token, withTask, setMessage } = useAppStore.getState();
        await withTask(async () => {
          if (state.cosFilePath.trim().length === 0) throw new Error('请先选择或输入 COS 文件路径。');
          const result = await request<{ total: number }>(apiBase, token, '/api/v1/cos-filter/load', {
            file_path: state.cosFilePath.trim(),
          });
          set({
            loadedCosCount: result.total ?? 0,
            step1Rows: [],
            step1Params: null,
            step2Rows: [],
            groupResult: null,
            groupingDedupRemoved: 0,
            groupResultTab: 'groups',
            selectedGroupIndex: 0,
            selectedItemNums: [],
            manualBoxOverride: false,
            manualSelectedBoxes: [],
            activeStep: 1, // Auto advance to next step
          });
          setMessage(`COS 数据已加载：${result.total} 条`);
        });
      },

      runStep1: async () => {
        const state = get();
        const { apiBase, token, withTask, setMessage } = useAppStore.getState();
        await withTask(async () => {
          if (state.loadedCosCount === 0) throw new Error('请先加载 COS 数据。');
          let minNm = state.wavelengthMin;
          let maxNm = state.wavelengthMax;
          if (minNm > maxNm) [minNm, maxNm] = [maxNm, minNm];

          const field = WAVELENGTH_LABEL_TO_FIELD[state.wavelengthLabel];
          const result = await request<CosStepResponse>(apiBase, token, '/api/v1/cos-filter/step1', {
            params: { wavelength_field: field, wavelength_min_nm: minNm, wavelength_max_nm: maxNm },
          });

          const params: Step1Params = {
            wavelength_field: field,
            wavelength_min_nm: minNm,
            wavelength_max_nm: maxNm,
            required_count: state.requiredCount,
          };

          set({
            step1Rows: result.records,
            step1Params: params,
            step2Rows: [],
            groupResult: null,
            groupingDedupRemoved: 0,
            groupResultTab: 'groups',
            selectedGroupIndex: 0,
            selectedItemNums: [],
            manualBoxOverride: false,
            manualSelectedBoxes: [],
            avgMinInput: minNm.toFixed(2),
            avgMaxInput: maxNm.toFixed(2),
            activeStep: 2, // Auto advance to step 2
          });
          setMessage(`第一步完成：${result.total} 条`);
        });
      },

      runStep2: async () => {
        const state = get();
        const { apiBase, token, withTask, setMessage } = useAppStore.getState();
        await withTask(async () => {
          if (!state.step1Params || state.step1Rows.length === 0) throw new Error('请先完成第一步筛选。');
          const result = await request<CosStepResponse>(apiBase, token, '/api/v1/cos-filter/step2', {
            // Server uses cached step1 result; no need to send full records
            params: {
              wavelength_field: state.step1Params.wavelength_field,
              wavelength_min_nm: state.step1Params.wavelength_min_nm,
              wavelength_max_nm: state.step1Params.wavelength_max_nm,
              required_count: state.step1Params.required_count,
              item_num_filter: state.selectedItemNums.length > 0 ? state.selectedItemNums : undefined,
            },
          });
          set({
            step2Rows: result.records,
            groupResult: null,
            groupingDedupRemoved: 0,
            groupResultTab: 'groups',
            selectedGroupIndex: 0,
            activeStep: 3, // Auto advance to step 3
          });
          setMessage(`第二步完成：${result.total} 条`);
        });
      },

      runGrouping: async (step2Rows) => {
        const state = get();
        const { apiBase, token, withTask } = useAppStore.getState();
        await withTask(async () => {
          if (!state.step1Params || state.step2Rows.length === 0) throw new Error('请先完成第二步筛选。');
          const deduped = dedupeByDeviceId(step2Rows);
          if (deduped.rows.length === 0) throw new Error('去重后无可分组数据。');

          const endpoint: Record<string, string> = {
            greedy: '/api/v1/cos-filter/group/greedy',
            optimal: '/api/v1/cos-filter/group/optimal',
            flat_top: '/api/v1/cos-filter/group/flat-top',
            huang_meng: '/api/v1/cos-filter/group/huang-meng',
          };

          const avgMinNm = parseOptionalNumber(state.avgMinInput);
          const avgMaxNm = parseOptionalNumber(state.avgMaxInput);

          const payload = {
            // Send only the deduped records to group (post-step2 filtering);
            // server will use these instead of the raw step1 cache.
            records: deduped.rows,
            params: {
              wavelength_field: state.step1Params.wavelength_field,
              group_size: state.groupSize,
              max_diff_nm: state.maxDiff,
              avg_min_nm: avgMinNm,
              avg_max_nm: avgMaxNm,
              strict_mode: state.groupingMode === 'flat_top' ? state.flatTopStrategy === 'max_uniformity' : undefined,
              low_min_nm: state.groupingMode === 'huang_meng' ? state.huangLowMin : undefined,
              low_max_nm: state.groupingMode === 'huang_meng' ? state.huangLowMax : undefined,
              high_min_nm: state.groupingMode === 'huang_meng' ? state.huangHighMin : undefined,
              high_max_nm: state.groupingMode === 'huang_meng' ? state.huangHighMax : undefined,
            },
          };

          const result = await request<CosGroupResponse>(apiBase, token, endpoint[state.groupingMode], payload);
          set({
            groupResult: result,
            groupingDedupRemoved: deduped.removed,
            groupResultTab: 'groups',
            selectedGroupIndex: 0,
            activeStep: 4, // Advance to step 4
          });
        });
      },

      runStep4Extract: async () => {
        const state = get();
        const { apiBase, token, withTask, setMessage } = useAppStore.getState();
        await withTask(async () => {
          let candidates: CosRow[] = [];
          if (state.step4Target === 'selected_group') {
            const groups = state.groupResult?.groups ?? [];
            const idx = state.selectedGroupIndex;
            candidates = groups[idx] ?? [];
            if (candidates.length === 0) {
              throw new Error('当前未选中有效分组，请先在第三步完成成组。');
            }
          } else if (state.step4Target === 'grouped') {
            candidates = (state.groupResult?.groups ?? []).flat();
            if (candidates.length === 0) {
              // If not grouped yet, fallback to step2 or step1
              candidates = state.step2Rows.length > 0 ? state.step2Rows : state.step1Rows;
            }
            if (candidates.length === 0) {
              throw new Error('当前无可提取的芯片，请先完成波长筛选。');
            }
          } else if (state.step4Target === 'step2') {
            candidates = state.step2Rows;
            if (candidates.length === 0) {
              throw new Error('第二步筛选未产生芯片，请先执行第二步。');
            }
          } else {
            candidates = state.step1Rows;
            if (candidates.length === 0) {
              throw new Error('第一步波长筛选未产生芯片，请先执行第一步。');
            }
          }

          const measurements = state.step4Measurements.length > 0 ? state.step4Measurements : ['LVI'];
          const currentPoints = parseCurrentPoints(state.step4CurrentInput);
          const rawRoots = state.step4ChipRootsInput?.trim() ? state.step4ChipRootsInput : 'Z:/Ldtd/\nZ:/Ldtd/Ldtd/';
          const chipRoots = parseLines(rawRoots);

          const payload: CosStep4ExtractPayload = {
            records: candidates,
            measurements,
            current_points: currentPoints,
            chip_default_root: chipRoots[0] || 'Z:/Ldtd/',
            chip_default_roots: chipRoots.length > 0 ? chipRoots : ['Z:/Ldtd/', 'Z:/Ldtd/Ldtd/'],
          };

          const result = await request<DataFetchResponse>(
            apiBase,
            token,
            '/api/v1/cos-filter/step4/extract',
            payload,
          );

          set({
            step4Result: result,
            groupResultTab: 'power',
          });
          setMessage(`电性能提取完成：共提取 ${result.records.length} 条实测记录（共 ${result.total} 条源数据）`);
        });
      },
    }),
    {
      name: 'data-cos-suite-filter-storage',
      version: 2,
      migrate: (persistedState: unknown, version: number) => {
        const state = (persistedState || {}) as Record<string, unknown>;
        if (version < 2) {
          if (!state.step4ChipRootsInput || state.step4ChipRootsInput === 'Z:/Ldtd/') {
            state.step4ChipRootsInput = 'Z:/Ldtd/\nZ:/Ldtd/Ldtd/';
          }
        }
        return state as any;
      },
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        batchDirectory: state.batchDirectory,
        wavelengthLabel: state.wavelengthLabel,
        wavelengthMin: state.wavelengthMin,
        wavelengthMax: state.wavelengthMax,
        requiredCount: state.requiredCount,
        selectedItemNums: state.selectedItemNums,
        boxCountInput: state.boxCountInput,
        manualBoxOverride: state.manualBoxOverride,
        manualSelectedBoxes: state.manualSelectedBoxes,
        groupingMode: state.groupingMode,
        groupSize: state.groupSize,
        maxDiff: state.maxDiff,
        avgMinInput: state.avgMinInput,
        avgMaxInput: state.avgMaxInput,
        flatTopStrategy: state.flatTopStrategy,
        huangLowMin: state.huangLowMin,
        huangLowMax: state.huangLowMax,
        huangHighMin: state.huangHighMin,
        huangHighMax: state.huangHighMax,
        presets: state.presets,
        step4Target: state.step4Target,
        step4Measurements: state.step4Measurements,
        step4CurrentInput: state.step4CurrentInput,
        step4ChipRootsInput: state.step4ChipRootsInput,
      }),
    }
  )
);
