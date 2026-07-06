import { useMemo } from 'react';
import {
  FolderOpen,
  Layers,
  Loader2,
  Play,
  RefreshCw,
  RotateCcw,
  SlidersHorizontal,
} from 'lucide-react';
import { formatEpoch, formatSize } from '../helpers/utils';
import type { WavelengthLabel, GroupingMode, FlatTopStrategy } from '../helpers/types';
import { useCosFilterStore } from '../stores/cos-filter-store';
import { useShallow } from 'zustand/react/shallow';
import { useAppStore } from '../stores/app-store';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { ScrollArea } from '@/components/ui/scroll-area';

const getFirstSelectionKey = (selection: string) => selection;
const isWavelengthLabel = (value: string): value is WavelengthLabel => ['2A波长', '冷波长', '中心波长', '峰值波长'].includes(value);
const isGroupingMode = (value: string): value is GroupingMode => ['greedy', 'optimal', 'flat_top', 'huang_meng'].includes(value);
const isFlatTopStrategy = (value: string): value is FlatTopStrategy => ['max_group_rate', 'max_uniformity'].includes(value);

function ChoiceList({
  legend,
  items,
  selectedKeys,
  onToggle,
  emptyText,
}: {
  legend: string;
  items: Array<{ key: string; label: string; meta?: string }>;
  selectedKeys: string[];
  onToggle: (key: string, checked: boolean) => void;
  emptyText: string;
}) {
  return (
    <FieldSet>
      <FieldLegend variant="label">{legend}</FieldLegend>
      <ScrollArea className="h-44 rounded-lg border bg-background">
        <div className="flex flex-col gap-2 p-3">
          {items.length === 0 ? (
            <p className="text-sm text-muted-foreground">{emptyText}</p>
          ) : (
            items.map((item, index) => {
              const id = `${legend.replace(/\s+/g, '-').toLowerCase()}-${index}`;
              return (
                <Field key={item.key} orientation="horizontal" className="rounded-lg border bg-card px-3 py-2">
                  <Checkbox
                    id={id}
                    checked={selectedKeys.includes(item.key)}
                    onCheckedChange={(value) => onToggle(item.key, value === true)}
                  />
                  <FieldContent>
                    <FieldLabel htmlFor={id}>{item.label}</FieldLabel>
                    {item.meta ? <FieldDescription>{item.meta}</FieldDescription> : null}
                  </FieldContent>
                </Field>
              );
            })
          )}
        </div>
      </ScrollArea>
    </FieldSet>
  );
}

export function DataSourcePanel() {
  const { loadCosBatchFiles, resetCosFlow, batchDirectory, setBatchDirectory, cosFilePath, setCosFilePath, batchFiles, loadCosFile, loadedCosCount } = useCosFilterStore(useShallow((state) => ({
    loadCosBatchFiles: state.loadCosBatchFiles,
    resetCosFlow: state.resetCosFlow,
    batchDirectory: state.batchDirectory,
    setBatchDirectory: state.setBatchDirectory,
    cosFilePath: state.cosFilePath,
    setCosFilePath: state.setCosFilePath,
    batchFiles: state.batchFiles,
    loadCosFile: state.loadCosFile,
    loadedCosCount: state.loadedCosCount,
  })));
  const busy = useAppStore((state) => state.busy);
  const withTask = useAppStore((state) => state.withTask);
  const setMessage = useAppStore((state) => state.setMessage);

  const refreshBatchFiles = async () => {
    await withTask(async () => {
      const total = await loadCosBatchFiles();
      setMessage(`批次文件已刷新：${total} 个`);
    });
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-medium">选择批次文件</p>
        </div>
        <div className="flex items-center gap-2">
          <Button type="button" variant="outline" size="icon-sm" disabled={busy} onClick={() => void refreshBatchFiles()}>
            <RefreshCw />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            disabled={busy}
            onClick={() => {
              resetCosFlow();
              setMessage('已重置');
            }}
          >
            <RotateCcw />
          </Button>
        </div>
      </div>

      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="batch-directory">批次目录（可选）</FieldLabel>
          <Input
            id="batch-directory"
            placeholder="/abs/path/to/data"
            value={batchDirectory}
            onChange={(event) => setBatchDirectory(event.target.value)}
          />
        </Field>

        <Field>
          <FieldLabel htmlFor="batch-file-select">批次文件</FieldLabel>
          <Select value={cosFilePath || undefined} onValueChange={setCosFilePath}>
            <SelectTrigger id="batch-file-select" className="w-full">
              <SelectValue placeholder="请选择批次文件" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {batchFiles.map((file) => (
                  <SelectItem key={file.file_path} value={file.file_path}>
                    <div className="flex flex-col gap-0.5">
                      <span>{file.file_name}</span>
                      <span className="text-xs text-muted-foreground">
                        {formatEpoch(file.modified_epoch_s)} · {formatSize(file.size_bytes)}
                      </span>
                    </div>
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </Field>

        <Field>
          <FieldLabel htmlFor="cos-file-path">精确文件路径</FieldLabel>
          <Input
            id="cos-file-path"
            value={cosFilePath}
            onChange={(event) => setCosFilePath(event.target.value)}
            placeholder="/abs/path/to/file.xlsx"
          />
        </Field>
      </FieldGroup>

      <Alert className="border-primary/20 bg-primary/5 rounded-xl">
        <FolderOpen className="h-4 w-4 text-primary" />
        <AlertTitle className="text-primary font-bold">当前状态</AlertTitle>
        <AlertDescription className="text-muted-foreground">
          {loadedCosCount > 0 ? `已加载 ${loadedCosCount} 条 COS 记录。` : `当前找到 ${batchFiles.length} 个候选批次文件。`}
        </AlertDescription>
      </Alert>

      <Button
        type="button"
        className="w-full h-10 rounded-xl font-bold bg-primary text-primary-foreground hover:bg-primary/95 shadow-premium active:scale-[0.99] transition-all duration-300"
        disabled={busy || cosFilePath.trim().length === 0}
        onClick={() => void loadCosFile()}
      >
        {busy ? <Loader2 className="animate-spin" data-icon="inline-start" /> : <Play data-icon="inline-start" className="h-4 w-4 mr-1" />}
        加载数据
      </Button>
    </div>
  );
}

export function Step1Config() {
  const { wavelengthLabel, setWavelengthLabel, requiredCount, setRequiredCount, wavelengthMin, setWavelengthMin, wavelengthMax, setWavelengthMax, loadedCosCount, runStep1 } = useCosFilterStore(useShallow((state) => ({
    wavelengthLabel: state.wavelengthLabel,
    setWavelengthLabel: state.setWavelengthLabel,
    requiredCount: state.requiredCount,
    setRequiredCount: state.setRequiredCount,
    wavelengthMin: state.wavelengthMin,
    setWavelengthMin: state.setWavelengthMin,
    wavelengthMax: state.wavelengthMax,
    setWavelengthMax: state.setWavelengthMax,
    loadedCosCount: state.loadedCosCount,
    runStep1: state.runStep1,
  })));
  const busy = useAppStore((state) => state.busy);

  return (
    <div className="flex flex-col gap-4">
      <Alert className="border-primary/20 bg-primary/5 rounded-xl">
        <SlidersHorizontal className="h-4 w-4 text-primary" />
        <AlertTitle className="text-primary font-bold">基础波长区间</AlertTitle>
        <AlertDescription className="text-muted-foreground">
          当前设定范围 {Math.min(wavelengthMin, wavelengthMax)} ~ {Math.max(wavelengthMin, wavelengthMax)} nm。
        </AlertDescription>
      </Alert>

      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="wavelength-label">波长类型</FieldLabel>
          <Select
            value={wavelengthLabel}
            onValueChange={(value) => {
              const selected = getFirstSelectionKey(value);
              if (isWavelengthLabel(selected)) {
                setWavelengthLabel(selected);
              }
            }}
          >
            <SelectTrigger id="wavelength-label" className="w-full">
              <SelectValue placeholder="选择波长类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="2A波长">2A波长</SelectItem>
                <SelectItem value="冷波长">冷波长</SelectItem>
                <SelectItem value="中心波长">中心波长</SelectItem>
                <SelectItem value="峰值波长">峰值波长</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </Field>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field>
            <FieldLabel htmlFor="required-count">需求数量</FieldLabel>
            <Input
              id="required-count"
              type="number"
              value={String(requiredCount)}
              onChange={(event) => setRequiredCount(parseInt(event.target.value, 10) || 1)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="wavelength-min">波长最小值</FieldLabel>
            <Input
              id="wavelength-min"
              type="number"
              value={String(wavelengthMin)}
              onChange={(event) => setWavelengthMin(parseFloat(event.target.value) || 0)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="wavelength-max">波长最大值</FieldLabel>
            <Input
              id="wavelength-max"
              type="number"
              value={String(wavelengthMax)}
              onChange={(event) => setWavelengthMax(parseFloat(event.target.value) || 0)}
            />
          </Field>
        </div>
      </FieldGroup>

      <Button
        type="button"
        className="w-full h-10 rounded-xl font-bold bg-primary text-primary-foreground hover:bg-primary/95 shadow-premium active:scale-[0.99] transition-all duration-300"
        disabled={busy || loadedCosCount === 0}
        onClick={() => void runStep1()}
      >
        {busy ? <Loader2 className="animate-spin" data-icon="inline-start" /> : <Play data-icon="inline-start" className="h-4 w-4 mr-1" />}
        执行第一步
      </Button>
    </div>
  );
}

export function Step2Config() {
  const {
    step1Rows,
    selectedItemNums,
    setSelectedItemNums,
    boxCountInput,
    setBoxCountInput,
    manualBoxOverride,
    setManualBoxOverride,
    manualSelectedBoxes,
    setManualSelectedBoxes,
    runStep2,
  } = useCosFilterStore(useShallow((state) => ({
    step1Rows: state.step1Rows,
    selectedItemNums: state.selectedItemNums,
    setSelectedItemNums: state.setSelectedItemNums,
    boxCountInput: state.boxCountInput,
    setBoxCountInput: state.setBoxCountInput,
    manualBoxOverride: state.manualBoxOverride,
    setManualBoxOverride: state.setManualBoxOverride,
    manualSelectedBoxes: state.manualSelectedBoxes,
    setManualSelectedBoxes: state.setManualSelectedBoxes,
    runStep2: state.runStep2,
  })));
  const busy = useAppStore((state) => state.busy);

  const step2Candidates = useMemo(() => {
    if (selectedItemNums.length === 0) return step1Rows;
    const selected = new Set(selectedItemNums);
    return step1Rows.filter((row) => row.item_num && selected.has(row.item_num));
  }, [selectedItemNums, step1Rows]);

  const boxCountEntries = useMemo(() => {
    const countMap = new Map<string, number>();
    for (const row of step2Candidates) {
      const box = (row.box_num ?? '').trim();
      if (box.length === 0) continue;
      countMap.set(box, (countMap.get(box) ?? 0) + 1);
    }
    return Array.from(countMap.entries()).sort((left, right) => (left[1] !== right[1] ? right[1] - left[1] : left[0].localeCompare(right[0])));
  }, [step2Candidates]);

  const sortedBoxes = useMemo(() => boxCountEntries.map((entry) => entry[0]), [boxCountEntries]);
  const autoSelectedBoxes = useMemo(
    () => sortedBoxes.slice(0, Math.max(0, Math.min(boxCountInput, sortedBoxes.length))),
    [boxCountInput, sortedBoxes],
  );
  const selectedBoxes = manualBoxOverride ? manualSelectedBoxes : autoSelectedBoxes;

  const step2SourceRows = useMemo(() => {
    if (selectedBoxes.length === 0) return step2Candidates;
    const selected = new Set(selectedBoxes);
    return step2Candidates.filter((row) => row.box_num && selected.has(row.box_num));
  }, [selectedBoxes, step2Candidates]);

  const availableItemNums = useMemo(
    () => Array.from(new Set(step1Rows.map((row) => (row.item_num ?? '').trim()).filter((value) => value.length > 0))).sort(),
    [step1Rows],
  );

  return (
    <div className="flex flex-col gap-4">
      <Alert className="border-primary/20 bg-primary/5 rounded-xl">
        <Layers className="h-4 w-4 text-primary" />
        <AlertTitle className="text-primary font-bold">候选池概况</AlertTitle>
        <AlertDescription className="text-muted-foreground">
          当前候选 {step2Candidates.length} 条，
          {manualBoxOverride ? ` 手动选盒 ${manualSelectedBoxes.length} 个。` : ` 自动入选盒 ${autoSelectedBoxes.length} 个。`}
        </AlertDescription>
      </Alert>

      <FieldGroup>
        <ChoiceList
          legend="ItemNum 多选"
          items={availableItemNums.map((value) => ({ key: value, label: value }))}
          selectedKeys={selectedItemNums}
          onToggle={(key, checked) => {
            const next = checked
              ? Array.from(new Set([...selectedItemNums, key]))
              : selectedItemNums.filter((item) => item !== key);
            setSelectedItemNums(next);
            setManualBoxOverride(false);
            setManualSelectedBoxes([]);
          }}
          emptyText="暂无可选择的 ItemNum。"
        />

        <Field>
          <FieldLabel htmlFor="box-count-input">自动选盒数量</FieldLabel>
          <Input
            id="box-count-input"
            type="number"
            value={String(boxCountInput)}
            onChange={(event) => {
              setBoxCountInput(parseInt(event.target.value, 10) || 0);
              setManualBoxOverride(false);
            }}
          />
        </Field>

        <ChoiceList
          legend="手动选盒覆盖"
          items={boxCountEntries.map(([box, count]) => ({ key: box, label: box, meta: `${count} 条` }))}
          selectedKeys={manualSelectedBoxes}
          onToggle={(key, checked) => {
            const next = checked
              ? Array.from(new Set([...manualSelectedBoxes, key]))
              : manualSelectedBoxes.filter((item) => item !== key);
            setManualSelectedBoxes(next);
            setManualBoxOverride(next.length > 0);
          }}
          emptyText="当前没有可供覆盖的盒号。"
        />
      </FieldGroup>

      {manualBoxOverride && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => {
            setManualBoxOverride(false);
            setManualSelectedBoxes([]);
          }}
        >
          <RotateCcw data-icon="inline-start" />
          恢复自动选盒
        </Button>
      )}

      <Button
        type="button"
        className="w-full h-10 rounded-xl font-bold bg-primary text-primary-foreground hover:bg-primary/95 shadow-premium active:scale-[0.99] transition-all duration-300"
        disabled={busy || step2SourceRows.length === 0}
        onClick={() => void runStep2(step2SourceRows)}
      >
        {busy ? <Loader2 className="animate-spin" data-icon="inline-start" /> : <Play data-icon="inline-start" className="h-4 w-4 mr-1" />}
        执行第二步
      </Button>
    </div>
  );
}

export function Step3Config() {
  const {
    groupingMode,
    setGroupingMode,
    groupSize,
    setGroupSize,
    maxDiff,
    setMaxDiff,
    avgMinInput,
    setAvgMinInput,
    avgMaxInput,
    setAvgMaxInput,
    flatTopStrategy,
    setFlatTopStrategy,
    huangLowMax,
    setHuangLowMax,
    huangLowMin,
    setHuangLowMin,
    huangHighMax,
    setHuangHighMax,
    huangHighMin,
    setHuangHighMin,
    step2Rows,
    runGrouping,
  } = useCosFilterStore(useShallow((state) => ({
    groupingMode: state.groupingMode,
    setGroupingMode: state.setGroupingMode,
    groupSize: state.groupSize,
    setGroupSize: state.setGroupSize,
    maxDiff: state.maxDiff,
    setMaxDiff: state.setMaxDiff,
    avgMinInput: state.avgMinInput,
    setAvgMinInput: state.setAvgMinInput,
    avgMaxInput: state.avgMaxInput,
    setAvgMaxInput: state.setAvgMaxInput,
    flatTopStrategy: state.flatTopStrategy,
    setFlatTopStrategy: state.setFlatTopStrategy,
    huangLowMax: state.huangLowMax,
    setHuangLowMax: state.setHuangLowMax,
    huangLowMin: state.huangLowMin,
    setHuangLowMin: state.setHuangLowMin,
    huangHighMax: state.huangHighMax,
    setHuangHighMax: state.setHuangHighMax,
    huangHighMin: state.huangHighMin,
    setHuangHighMin: state.setHuangHighMin,
    step2Rows: state.step2Rows,
    runGrouping: state.runGrouping,
  })));
  const busy = useAppStore((state) => state.busy);
  const isHuangMengError =
    groupingMode === 'huang_meng' &&
    (huangLowMax <= huangLowMin || huangHighMax <= huangHighMin || huangLowMax >= huangHighMin);

  return (
    <div className="flex flex-col gap-4">
      <Alert className="border-primary/20 bg-primary/5 rounded-xl">
        <Layers className="h-4 w-4 text-primary" />
        <AlertTitle className="text-primary font-bold">成组规则摘要</AlertTitle>
        <AlertDescription className="text-muted-foreground">
          每组 {groupSize} 个，极差上限 {maxDiff} nm，待成组池 {step2Rows.length} 条。
        </AlertDescription>
      </Alert>

      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="grouping-mode">算法模式</FieldLabel>
          <Select
            value={groupingMode}
            onValueChange={(value) => {
              const selected = getFirstSelectionKey(value);
              if (isGroupingMode(selected)) {
                setGroupingMode(selected);
              }
            }}
          >
            <SelectTrigger id="grouping-mode" className="w-full">
              <SelectValue placeholder="选择算法模式" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="greedy">快速贪婪</SelectItem>
                <SelectItem value="optimal">中心优化</SelectItem>
                <SelectItem value="flat_top">平顶均匀</SelectItem>
                <SelectItem value="huang_meng">黄盟算法</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </Field>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field>
            <FieldLabel htmlFor="group-size">每组组合数</FieldLabel>
            <Input
              id="group-size"
              type="number"
              value={String(groupSize)}
              onChange={(event) => setGroupSize(parseInt(event.target.value, 10) || 1)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="max-diff">极大极差 (nm)</FieldLabel>
            <Input
              id="max-diff"
              type="number"
              step={0.1}
              value={String(maxDiff)}
              onChange={(event) => setMaxDiff(parseFloat(event.target.value) || 0.01)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="avg-min">平均下限</FieldLabel>
            <Input
              id="avg-min"
              placeholder="964.10"
              value={avgMinInput}
              onChange={(event) => setAvgMinInput(event.target.value)}
            />
          </Field>
          {groupingMode !== 'huang_meng' && (
            <Field>
              <FieldLabel htmlFor="avg-max">平均上限</FieldLabel>
              <Input
                id="avg-max"
                placeholder="964.40"
                value={avgMaxInput}
                onChange={(event) => setAvgMaxInput(event.target.value)}
              />
            </Field>
          )}
          {groupingMode === 'flat_top' && (
            <Field>
              <FieldLabel htmlFor="flat-top-strategy">平顶策略</FieldLabel>
              <Select
                value={flatTopStrategy}
                onValueChange={(value) => {
                  const selected = getFirstSelectionKey(value);
                  if (isFlatTopStrategy(selected)) {
                    setFlatTopStrategy(selected);
                  }
                }}
              >
                <SelectTrigger id="flat-top-strategy" className="w-full">
                  <SelectValue placeholder="选择平顶策略" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="max_group_rate">最大分组率</SelectItem>
                    <SelectItem value="max_uniformity">最大均匀性</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
          )}
        </div>

        {groupingMode === 'huang_meng' && (
          <FieldSet>
            <FieldLegend variant="label">黄盟算法阈值</FieldLegend>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="avg-max-huang">带外平均上限</FieldLabel>
                <Input
                  id="avg-max-huang"
                  placeholder="964.40"
                  value={avgMaxInput}
                  onChange={(event) => setAvgMaxInput(event.target.value)}
                />
              </Field>
              <Field data-invalid={huangLowMin >= huangLowMax}>
                <FieldLabel htmlFor="huang-low-min">低区下限</FieldLabel>
                <Input
                  id="huang-low-min"
                  type="number"
                  aria-invalid={huangLowMin >= huangLowMax}
                  value={String(huangLowMin)}
                  onChange={(event) => setHuangLowMin(parseFloat(event.target.value) || huangLowMin)}
                />
              </Field>
              <Field data-invalid={huangLowMax <= huangLowMin || huangLowMax >= huangHighMin}>
                <FieldLabel htmlFor="huang-low-max">低区上限</FieldLabel>
                <Input
                  id="huang-low-max"
                  type="number"
                  aria-invalid={huangLowMax <= huangLowMin || huangLowMax >= huangHighMin}
                  value={String(huangLowMax)}
                  onChange={(event) => setHuangLowMax(parseFloat(event.target.value) || huangLowMax)}
                />
              </Field>
              <Field data-invalid={huangHighMin <= huangLowMax || huangHighMin >= huangHighMax}>
                <FieldLabel htmlFor="huang-high-min">高区下限</FieldLabel>
                <Input
                  id="huang-high-min"
                  type="number"
                  aria-invalid={huangHighMin <= huangLowMax || huangHighMin >= huangHighMax}
                  value={String(huangHighMin)}
                  onChange={(event) => setHuangHighMin(parseFloat(event.target.value) || huangHighMin)}
                />
              </Field>
              <Field data-invalid={huangHighMax <= huangHighMin}>
                <FieldLabel htmlFor="huang-high-max">高区上限</FieldLabel>
                <Input
                  id="huang-high-max"
                  type="number"
                  aria-invalid={huangHighMax <= huangHighMin}
                  value={String(huangHighMax)}
                  onChange={(event) => setHuangHighMax(parseFloat(event.target.value) || huangHighMax)}
                />
              </Field>
            </div>
          </FieldSet>
        )}
      </FieldGroup>

      {isHuangMengError && (
        <Alert variant="destructive">
          <RotateCcw />
          <AlertTitle>阈值配置有冲突</AlertTitle>
          <AlertDescription>高低区范围重叠或顺序错误，请调整后再执行。</AlertDescription>
        </Alert>
      )}

      <Button
        type="button"
        className="w-full h-10 rounded-xl font-bold bg-primary text-primary-foreground hover:bg-primary/95 shadow-premium active:scale-[0.99] transition-all duration-300"
        disabled={busy || isHuangMengError || step2Rows.length === 0}
        onClick={() => void runGrouping(step2Rows)}
      >
        {busy ? <Loader2 className="animate-spin" data-icon="inline-start" /> : <Layers data-icon="inline-start" className="h-4 w-4 mr-1" />}
        开始进行分组
      </Button>
    </div>
  );
}
