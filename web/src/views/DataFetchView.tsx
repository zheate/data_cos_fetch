import type { ReactNode } from 'react';
import { useDeferredValue, useMemo } from 'react';
import {
  AlertCircle,
  CheckCircle2,
  Download,
  FileSpreadsheet,
  FolderOpen,
  ListChecks,
  ListFilter,
  Loader2,
  Play,
  Rows3,
  X,
} from 'lucide-react';
import { useAppStore } from '../stores/app-store';
import { useDataFetchStore } from '../stores/data-fetch-store';
import { request } from '../helpers/api';
import { parseLines, parseCurrentPoints, DEFAULT_TEST_CATEGORIES, MEASUREMENTS } from '../helpers/utils';
import { downloadDataFetchAsCsv } from '../helpers/csv';
import { DataFetchTable } from '../components/DataFetchTable';
import type { DataFetchExtractPayload, DataFetchResponse, ExtractionMode } from '../helpers/types';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Field,
  FieldContent,
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
import { Textarea } from '@/components/ui/textarea';
import { cn } from '@/lib/utils';

const MEASUREMENT_META: Record<string, { label: string; detail: string }> = {
  LVI: { label: 'LVI', detail: '功率 / 电压 / 效率' },
  Rth: { label: 'Rth', detail: '热阻' },
  lambd: { label: 'lambd', detail: '波长' },
};

function ConfigSection({
  index,
  title,
  meta,
  icon,
  children,
}: {
  index: string;
  title: string;
  meta?: ReactNode;
  icon: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="overflow-hidden rounded-lg border bg-background/70">
      <div className="flex items-center justify-between gap-3 border-b bg-muted/25 px-3.5 py-2.5">
        <div className="flex min-w-0 items-center gap-2.5">
          <span className="flex size-7 shrink-0 items-center justify-center rounded-md border bg-card text-muted-foreground [&_svg]:size-3.5">
            {icon}
          </span>
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-mono text-[10px] font-semibold text-muted-foreground">{index}</span>
              <h2 className="truncate text-sm font-semibold text-foreground">{title}</h2>
            </div>
          </div>
        </div>
        {meta && <div className="shrink-0 text-xs text-muted-foreground">{meta}</div>}
      </div>
      <div className="p-3.5">{children}</div>
    </section>
  );
}

function SelectionField({
  id,
  label,
  checked,
  onCheckedChange,
}: {
  id: string;
  label: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <Field
      orientation="horizontal"
      className={cn(
        'min-h-10 rounded-lg border bg-background px-3 py-2.5 transition-colors',
        checked ? 'border-primary/35 bg-primary/5' : 'hover:bg-muted/35',
      )}
    >
      <Checkbox id={id} checked={checked} onCheckedChange={(value) => onCheckedChange(value === true)} />
      <FieldContent>
        <FieldLabel htmlFor={id} className="text-xs font-medium">
          {label}
        </FieldLabel>
      </FieldContent>
    </Field>
  );
}

function MeasurementToggle({
  measurement,
  checked,
  onToggle,
}: {
  measurement: string;
  checked: boolean;
  onToggle: () => void;
}) {
  const meta = MEASUREMENT_META[measurement] ?? { label: measurement, detail: '测试文件' };

  return (
    <button
      type="button"
      aria-pressed={checked}
      onClick={onToggle}
      className={cn(
        'flex min-h-[70px] flex-col items-start justify-between rounded-lg border px-3 py-2.5 text-left transition-colors focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/40',
        checked
          ? 'border-primary/45 bg-primary/5 text-foreground'
          : 'border-border bg-background text-muted-foreground hover:bg-muted/35 hover:text-foreground',
      )}
    >
      <span className="flex w-full items-center justify-between gap-2">
        <span className="text-sm font-semibold">{meta.label}</span>
        {checked ? (
          <CheckCircle2 className="size-4 text-primary" />
        ) : (
          <span className="size-4 rounded-full border border-muted-foreground/30" />
        )}
      </span>
      <span className="text-[11px] leading-snug text-muted-foreground">{meta.detail}</span>
    </button>
  );
}

export function DataFetchView() {
  const { apiBase, token, busy, withTask } = useAppStore();
  const store = useDataFetchStore();
  const deferredEntriesInput = useDeferredValue(store.entriesInput);

  const entryCount = useMemo(() => parseLines(deferredEntriesInput).length, [deferredEntriesInput]);
  const isEntryCountPending = deferredEntriesInput !== store.entriesInput;
  const selectedTestCount = store.mode === 'module' ? store.selectedTests.length : 0;
  const selectedMeasurementCount = store.selectedMeasurements.length;
  const allCategories = useMemo(
    () => [...DEFAULT_TEST_CATEGORIES, ...store.customTests],
    [store.customTests],
  );
  const currentModeRootLabel = store.mode === 'module' ? '模块根目录' : '芯片根目录';
  const currentSummary = store.currentInput.trim() || '最大电流';
  const measurementSummary = selectedMeasurementCount > 0 ? store.selectedMeasurements.join(' / ') : '未选择';
  const canRun = !busy && entryCount > 0 && selectedMeasurementCount > 0;

  const resultStats = useMemo(() => {
    const result = store.result;
    if (!result) {
      return {
        total: 0,
        entries: 0,
      };
    }

    return {
      total: result.total,
      entries: new Set(result.records.map((row) => row.entry_id)).size,
    };
  }, [store.result]);

  const runDataFetch = async () => {
    await withTask(async () => {
      const entries = parseLines(store.entriesInput);
      if (entries.length === 0) {
        throw new Error('请输入条目。');
      }
      if (store.selectedMeasurements.length === 0) {
        throw new Error('请选择测试文件。');
      }

      const moduleDefaultRoot = store.moduleDefaultRoot.trim();
      const chipDefaultRoots = parseLines(store.chipDefaultRootsInput);
      const payload: DataFetchExtractPayload = {
        mode: store.mode,
        entries,
        test_categories: store.mode === 'module' ? store.selectedTests : undefined,
        measurements: store.selectedMeasurements,
        current_points: parseCurrentPoints(store.currentInput),
        module_default_root: moduleDefaultRoot || undefined,
        chip_default_root: chipDefaultRoots[0] || undefined,
        chip_default_roots: chipDefaultRoots.length > 0 ? chipDefaultRoots : undefined,
      };

      const result = await request<DataFetchResponse>(apiBase, token, '/api/v1/data-fetch/extract', payload);
      store.setResult(result);
      return result;
    }, {
      loading: '正在提取...',
      success: (result) => `已提取 ${result.total} 条记录`
    });
  };

  return (
    <div className="data-fetch-workbench grid gap-5 lg:grid-cols-[minmax(360px,420px)_minmax(0,1fr)]">
      <Card className="h-fit border bg-card/95 shadow-sm">
        <CardContent className="flex flex-col gap-4 pt-4">
          <ConfigSection
            index="01"
            title="来源"
            icon={<FolderOpen />}
            meta={<span className="font-mono">{store.mode}</span>}
          >
            <FieldGroup className="gap-4">
              <Field>
                <FieldLabel htmlFor="data-fetch-mode">模式</FieldLabel>
                <Select
                  value={store.mode}
                  onValueChange={(value) => {
                    if (value === 'module' || value === 'chip') {
                      store.setMode(value as ExtractionMode);
                    }
                  }}
                >
                  <SelectTrigger id="data-fetch-mode" className="w-full">
                    <SelectValue placeholder="选择模式" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="module">模块</SelectItem>
                      <SelectItem value="chip">芯片</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>

              <Field>
                <FieldLabel htmlFor="default-root-input">{currentModeRootLabel}</FieldLabel>
                {store.mode === 'module' ? (
                  <Input
                    id="default-root-input"
                    value={store.moduleDefaultRoot}
                    onChange={(event) => store.setModuleDefaultRoot(event.target.value)}
                  />
                ) : (
                  <Textarea
                    id="default-root-input"
                    rows={3}
                    className="min-h-[84px] max-h-[160px] resize-y overflow-y-auto font-mono text-xs [field-sizing:fixed]"
                    placeholder={'Z:/Ldtd/\nD:/Ldtd/'}
                    value={store.chipDefaultRootsInput}
                    onChange={(event) => store.setChipDefaultRootsInput(event.target.value)}
                  />
                )}
              </Field>
            </FieldGroup>
          </ConfigSection>

          <ConfigSection
            index="02"
            title="条目"
            icon={<Rows3 />}
            meta={<Badge variant="secondary">{isEntryCountPending ? '计算中' : `${entryCount} 条`}</Badge>}
          >
            <Textarea
              id="entries-input"
              rows={8}
              aria-label="条目列表"
              className="min-h-[188px] max-h-[260px] resize-y overflow-y-auto font-mono text-xs leading-relaxed [field-sizing:fixed]"
              placeholder="/abs/path/to/shellA&#10;/abs/path/to/shellB"
              value={store.entriesInput}
              onChange={(event) => store.setEntriesInput(event.target.value)}
            />
          </ConfigSection>

          <ConfigSection
            index="03"
            title="测试文件"
            icon={<FileSpreadsheet />}
            meta={`${selectedMeasurementCount} / ${MEASUREMENTS.length}`}
          >
            <div className="grid grid-cols-3 gap-2">
              {MEASUREMENTS.map((measurement) => (
                <MeasurementToggle
                  key={measurement}
                  measurement={measurement}
                  checked={store.selectedMeasurements.includes(measurement)}
                  onToggle={() => store.toggleMeasurement(measurement, !store.selectedMeasurements.includes(measurement))}
                />
              ))}
            </div>
            {selectedMeasurementCount === 0 && (
              <p className="mt-2 flex items-center gap-1.5 text-xs text-destructive">
                <AlertCircle className="size-3.5" />
                至少选择一个测试文件
              </p>
            )}
          </ConfigSection>

          <ConfigSection
            index="04"
            title="测试条件"
            icon={<ListFilter />}
            meta={store.mode === 'module' ? `${selectedTestCount} 个站别` : '芯片'}
          >
            <FieldGroup className="gap-4">
              {store.mode === 'module' && (
                <FieldSet className="gap-3">
                  <div className="flex items-center justify-between gap-3">
                    <FieldLegend variant="label" className="mb-0">站别</FieldLegend>
                    <form
                      onSubmit={(event) => {
                        event.preventDefault();
                        const input = new FormData(event.currentTarget).get('newCategory') as string;
                        if (input && input.trim()) {
                          store.addCustomTest(input.trim());
                          event.currentTarget.reset();
                        }
                      }}
                      className="flex items-center gap-2"
                    >
                      <Input
                        name="newCategory"
                        aria-label="新增站别"
                        placeholder="新增站别"
                        className="h-7 w-32 text-xs"
                      />
                      <Button type="submit" size="sm" variant="secondary" className="h-7 px-2 text-xs">添加</Button>
                    </form>
                  </div>
                  <div className="grid gap-2 sm:grid-cols-2">
                    {allCategories.map((category) => {
                      const isCustom = store.customTests.includes(category);
                      return (
                        <div key={category} className="group relative">
                          <SelectionField
                            id={`test-category-${category}`}
                            label={category}
                            checked={store.selectedTests.includes(category)}
                            onCheckedChange={(checked) => store.toggleTest(category, checked)}
                          />
                          {isCustom && (
                            <button
                              type="button"
                              className="absolute right-2 top-1/2 flex size-5 -translate-y-1/2 items-center justify-center rounded text-muted-foreground opacity-0 transition-opacity hover:bg-muted hover:text-destructive group-hover:opacity-100 focus:opacity-100"
                              onClick={() => store.removeCustomTest(category)}
                              title="移除站别"
                              aria-label={`移除 ${category}`}
                            >
                              <X className="size-3" />
                            </button>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </FieldSet>
              )}

              <Field>
                <FieldLabel htmlFor="current-input">电流点</FieldLabel>
                <Input
                  id="current-input"
                  placeholder='最大电流 / a / 12~19'
                  value={store.currentInput}
                  onChange={(event) => store.setCurrentInput(event.target.value)}
                />
              </Field>
            </FieldGroup>
          </ConfigSection>

          <section className="rounded-lg border border-primary/20 bg-primary/5 p-3.5">
            <div className="grid grid-cols-2 gap-2 text-xs">
              <div>
                <p className="text-muted-foreground">条目</p>
                <p className="mt-1 font-semibold text-foreground">{entryCount}</p>
              </div>
              <div>
                <p className="text-muted-foreground">文件</p>
                <p className="mt-1 truncate font-semibold text-foreground">{measurementSummary}</p>
              </div>
              <div>
                <p className="text-muted-foreground">站别</p>
                <p className="mt-1 font-semibold text-foreground">{store.mode === 'module' ? selectedTestCount : '-'}</p>
              </div>
              <div>
                <p className="text-muted-foreground">电流</p>
                <p className="mt-1 truncate font-semibold text-foreground">{currentSummary}</p>
              </div>
            </div>

            <Button
              type="button"
              size="lg"
              className="mt-3 h-10 w-full"
              disabled={!canRun}
              onClick={runDataFetch}
            >
              {busy ? <Loader2 className="animate-spin" data-icon="inline-start" /> : <Play data-icon="inline-start" />}
              提取数据
            </Button>
          </section>
        </CardContent>
      </Card>

      <div className="flex min-w-0 flex-col gap-4">
        <Card className="min-h-[680px] border bg-card/95 shadow-sm">
          <CardHeader className="flex flex-row items-center justify-between gap-3 border-b bg-card/80 px-4 py-4">
            <div className="min-w-0">
              <CardTitle className="flex items-center gap-2 text-base font-semibold">
                <ListChecks className="size-4 text-primary" />
                提取结果
              </CardTitle>
              {store.result && (
                <CardDescription className="mt-1 text-xs">
                  {resultStats.total} 条记录，{resultStats.entries} 个条目
                </CardDescription>
              )}
            </div>
            {store.result && store.result.records.length > 0 && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => downloadDataFetchAsCsv(store.result!.records, 'Data_Fetch_Output.csv')}
                className="rounded-lg"
              >
                <Download className="mr-1.5 h-4 w-4" />
                导出
              </Button>
            )}
          </CardHeader>

          <CardContent className="flex flex-col gap-4 pt-4">
            {store.result && <DataFetchTable rows={store.result.records} />}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
