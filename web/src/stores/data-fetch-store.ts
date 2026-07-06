import { create } from 'zustand';
import type { DataFetchResponse, ExtractionMode } from '../helpers/types';
import { DEFAULT_TEST_CATEGORIES, MEASUREMENTS } from '../helpers/utils';

type DataFetchState = {
  mode: ExtractionMode;
  entriesInput: string;
  selectedTests: string[];
  selectedMeasurements: string[];
  currentInput: string;
  moduleDefaultRoot: string;
  chipDefaultRootsInput: string;
  result: DataFetchResponse | null;

  customTests: string[];

  setMode: (v: ExtractionMode) => void;
  setEntriesInput: (v: string) => void;
  setSelectedTests: (v: string[]) => void;
  setSelectedMeasurements: (v: string[]) => void;
  setCurrentInput: (v: string) => void;
  setModuleDefaultRoot: (v: string) => void;
  setChipDefaultRootsInput: (v: string) => void;
  setResult: (v: DataFetchResponse | null) => void;

  toggleTest: (category: string, checked: boolean) => void;
  toggleMeasurement: (measurement: string, checked: boolean) => void;
  addCustomTest: (category: string) => void;
  removeCustomTest: (category: string) => void;
};

export const useDataFetchStore = create<DataFetchState>((set) => ({
  mode: 'module',
  entriesInput: '',
  selectedTests: [...DEFAULT_TEST_CATEGORIES],
  selectedMeasurements: [...MEASUREMENTS],
  currentInput: '',
  moduleDefaultRoot: 'Z:/Ldtd/fcp/',
  chipDefaultRootsInput: 'Z:/Ldtd/',
  result: null,
  customTests: [],

  setMode: (v) => set({ mode: v }),
  setEntriesInput: (v) => set({ entriesInput: v }),
  setSelectedTests: (v) => set({ selectedTests: v }),
  setSelectedMeasurements: (v) => set({ selectedMeasurements: v }),
  setCurrentInput: (v) => set({ currentInput: v }),
  setModuleDefaultRoot: (v) => set({ moduleDefaultRoot: v }),
  setChipDefaultRootsInput: (v) => set({ chipDefaultRootsInput: v }),
  setResult: (v) => set({ result: v }),

  toggleTest: (category, checked) =>
    set((s) => ({
      selectedTests: checked
        ? Array.from(new Set([...s.selectedTests, category]))
        : s.selectedTests.filter((item) => item !== category),
    })),

  toggleMeasurement: (measurement, checked) =>
    set((s) => ({
      selectedMeasurements: checked
        ? Array.from(new Set([...s.selectedMeasurements, measurement]))
        : s.selectedMeasurements.filter((item) => item !== measurement),
    })),

  addCustomTest: (category) => 
    set((s) => ({
      customTests: Array.from(new Set([...s.customTests, category])),
      selectedTests: Array.from(new Set([...s.selectedTests, category])),
    })),
    
  removeCustomTest: (category) =>
    set((s) => ({
      customTests: s.customTests.filter((item) => item !== category),
      selectedTests: s.selectedTests.filter((item) => item !== category),
    })),
}));
