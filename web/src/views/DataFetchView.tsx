import type { ReactNode } from 'react';
import { useDeferredValue, useMemo, useState } from 'react';
import {
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  Download,
  FileSpreadsheet,
  FolderOpen,
  Info,
  ListChecks,
  ListFilter,
  Loader2,
  Play,
  Rows3,
  ShieldCheck,
  X,
} from 'lucide-react';
import { toast } from 'sonner';
import { useAppStore } from '../stores/app-store';
import { useDataFetchStore } from '../stores/data-fetch-store';
import { request } from '../helpers/api';
import { parseLines, parseCurrentPoints, DEFAULT_TEST_CATEGORIES, MEASUREMENTS } from '../helpers/utils';
import { downloadDataFetchAsXlsx } from '../helpers/csv';
import { DataFetchTable } from '../components/DataFetchTable';
import type {
  DataFetchExtractPayload,
  DataFetchResponse,
  ExtractionDiagnosticsResponse,
  ExtractionDiagnosticStatus,
  ExtractionMode,
} from '../helpers/types';
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

const ERROR_CLASSIFICATIONS = [
  {
    label: '访问权限',
    matches: ['permission denied', 'access denied', '拒绝访问', '权限不足', 'os error 5'],
    guidance: '后端进程没有读取权限。请检查加密软件白名单及共享目录权限。',
  },
  {
    label: '文件解密或格式',
    matches: ['failed to open workbook', 'excel parser', 'content probe', 'zip', 'ole', 'calamine'],
    guidance: '文件可被定位，但无法按 Excel 内容解析。透明加密未向后端提供明文时常出现此类错误。',
  },
  {
    label: '路径或盘符',
    matches: ['not found', 'cannot find', '找不到', 'failed to read directory', 'os error 2', 'os error 3'],
    guidance: '后端看不到该路径。请检查映射盘、UNC路径以及是否以管理员身份运行。',
  },
] as const;

function classifyExtractionError(message: string) {
  const normalized = message.toLowerCase();
  return ERROR_CLASSIFICATIONS.find((item) => item.matches.some((token) => normalized.includes(token))) ?? {
    label: '数据读取',
    guidance: '请保留完整错误并结合运行日志进一步定位。',
  };
}

function diagnosticStatusMeta(status: ExtractionDiagnosticStatus) {
  if (status === 'pass') {
    return { label: '通过', className: 'text-emerald-600 dark:text-emerald-500', icon: CheckCircle2 };
  }
  if (status === 'warning') {
    return { label: '警告', className: 'text-amber-600 dark:text-amber-400', icon: AlertTriangle };
  }
  return { label: '失败', className: 'text-destructive', icon: AlertCircle };
}

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
  const { apiBase, token, busy, withTask, logPath } = useAppStore();
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
  const [showErrors, setShowErrors] = useState(false);
  const [showDiagnostics, setShowDiagnostics] = useState(false);
  const [diagnostics, setDiagnostics] = useState<ExtractionDiagnosticsResponse | null>(null);

  const resultStats = useMemo(() => {
    const result = store.result;
    if (!result) {
      return {
        total: 0,
        entries: 0,
        success: 0,
        failures: 0,
        errors: [],
        infos: [],
      };
    }

    return {
      total: result.total,
      entries: new Set(result.records.map((row) => row.entry_id)).size,
      success: result.records.length,
      failures: result.errors?.length || 0,
      errors: result.errors || [],
      infos: result.infos || [],
    };
  }, [store.result]);

  const runDataFetch = async () => {
    const toastId = toast.loading('正在提取...');
    const result = await withTask(async () => {
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

      return request<DataFetchResponse>(apiBase, token, '/api/v1/data-fetch/extract', payload);
    });

    if (!result) {
      toast.dismiss(toastId);
      return;
    }

    store.setResult(result);
    const hasErrors = result.errors.length > 0;
    const hasInfos = result.infos.length > 0;
    if (hasErrors || (result.total === 0 && hasInfos)) {
      setShowErrors(true);
    }

    if (result.total === 0 && hasErrors) {
      toast.error(`提取失败：0 条记录，${result.errors.length} 条错误`, { id: toastId });
    } else if (hasErrors) {
      toast.warning(`部分完成：${result.total} 条记录，${result.errors.length} 条错误`, { id: toastId });
    } else if (result.total === 0) {
      toast.warning('未提取到记录，请查看诊断详情或运行环境检测。', { id: toastId });
    } else if (hasInfos) {
      toast.warning(`已提取 ${result.total} 条记录，另有 ${result.infos.length} 条查找提示`, { id: toastId });
    } else {
      toast.success(`已提取 ${result.total} 条记录`, { id: toastId });
    }
  };

  const runEnvironmentDiagnostics = async () => {
    const roots = store.mode === 'module'
      ? [store.moduleDefaultRoot.trim()].filter(Boolean)
      : parseLines(store.chipDefaultRootsInput);
    const result = await withTask(
      () => request<ExtractionDiagnosticsResponse>(apiBase, token, '/api/v1/data-fetch/diagnostics', { roots }),
    );
    if (!result) return;

    setDiagnostics(result);
    setShowDiagnostics(true);
    if (result.failed > 0) {
      toast.error(`环境检测发现 ${result.failed} 项失败`);
    } else if (result.warnings > 0) {
      toast.warning(`环境检测完成，有 ${result.warnings} 项警告`);
    } else {
      toast.success('环境检测全部通过');
    }
  };

  const openLogsFolder = async () => {
    const error = await window.desktopRuntime?.openLogsFolder?.();
    if (error) toast.error(`无法打开日志目录：${error}`);
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

              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={busy}
                onClick={runEnvironmentDiagnostics}
                className="w-full"
              >
                <ShieldCheck data-icon="inline-start" />
                检测读取环境
              </Button>
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
            {store.result && (
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-3 text-sm font-medium">
                  <span className="flex items-center gap-1.5 text-emerald-600 dark:text-emerald-500">
                    <CheckCircle2 className="size-4" />
                    成功: {resultStats.success}
                  </span>
                  {resultStats.failures > 0 ? (
                    <button
                      type="button"
                      onClick={() => setShowErrors(true)}
                      className="flex items-center gap-1.5 rounded-sm text-destructive hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    >
                      <AlertCircle className="size-4" />
                      失败: {resultStats.failures}
                    </button>
                  ) : (
                    <span className="flex items-center gap-1.5 text-muted-foreground">
                      <AlertCircle className="size-4" />
                      失败: 0
                    </span>
                  )}
                  {resultStats.failures === 0 && resultStats.infos.length > 0 && (
                    <button
                      type="button"
                      onClick={() => setShowErrors(true)}
                      className="flex items-center gap-1.5 rounded-sm text-amber-600 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-ring dark:text-amber-400"
                    >
                      <Info className="size-4" />
                      提示: {resultStats.infos.length}
                    </button>
                  )}
                </div>
                {store.result.records.length > 0 && (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => downloadDataFetchAsXlsx(store.result!.records, 'Data_Fetch_Output.xlsx')}
                    className="rounded-lg"
                  >
                    <Download className="mr-1.5 h-4 w-4" />
                    导出
                  </Button>
                )}
              </div>
            )}
          </CardHeader>

          <CardContent className="flex flex-col gap-4 pt-4">
            {store.result && <DataFetchTable rows={store.result.records} />}
          </CardContent>
        </Card>
      </div>

      {showErrors && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 animate-in fade-in duration-200">
          <div className="flex w-full max-w-2xl flex-col rounded-xl border bg-background shadow-lg">
            <div className="flex items-center justify-between border-b p-4">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-foreground">
                <AlertCircle className="size-5" />
                提取诊断详情
              </h2>
              <button
                type="button"
                onClick={() => setShowErrors(false)}
                className="rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <X className="size-5" />
              </button>
            </div>
            <div className="p-4 text-sm">
              <div className="max-h-[55vh] space-y-3 overflow-y-auto pr-1">
                {resultStats.errors.map((error, index) => {
                  const classification = classifyExtractionError(error);
                  return (
                    <div key={`error-${index}`} className="rounded-lg border border-destructive/25 bg-destructive/5 p-3">
                      <div className="flex items-center gap-2 text-xs font-semibold text-destructive">
                        <AlertCircle className="size-3.5" />
                        {classification.label}
                      </div>
                      <p className="mt-2 whitespace-pre-wrap break-words font-mono text-xs text-foreground">{error}</p>
                      <p className="mt-2 text-xs text-muted-foreground">{classification.guidance}</p>
                    </div>
                  );
                })}
                {resultStats.infos.map((info, index) => (
                  <div key={`info-${index}`} className="rounded-lg border border-amber-500/25 bg-amber-500/5 p-3">
                    <div className="flex items-center gap-2 text-xs font-semibold text-amber-600 dark:text-amber-400">
                      <Info className="size-3.5" />
                      查找提示
                    </div>
                    <p className="mt-2 whitespace-pre-wrap break-words font-mono text-xs text-foreground">{info}</p>
                  </div>
                ))}
                {resultStats.errors.length === 0 && resultStats.infos.length === 0 && (
                  <p className="text-muted-foreground">没有找到具体的诊断信息。</p>
                )}
              </div>
            </div>
            <div className="flex justify-end border-t p-4">
              <Button onClick={() => setShowErrors(false)}>关闭</Button>
            </div>
          </div>
        </div>
      )}

      {showDiagnostics && diagnostics && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 animate-in fade-in duration-200">
          <div role="dialog" aria-modal="true" aria-labelledby="environment-diagnostics-title" className="flex w-full max-w-3xl flex-col rounded-xl border bg-background shadow-lg">
            <div className="flex items-start justify-between gap-3 border-b p-4">
              <div>
                <h2 id="environment-diagnostics-title" className="flex items-center gap-2 text-lg font-semibold">
                  <ShieldCheck className="size-5 text-primary" />
                  读取环境检测
                </h2>
                <p className="mt-1 text-xs text-muted-foreground">
                  后端 PID {diagnostics.process_id} · {diagnostics.failed} 项失败 · {diagnostics.warnings} 项警告
                </p>
              </div>
              <button
                type="button"
                onClick={() => setShowDiagnostics(false)}
                className="rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                aria-label="关闭环境检测"
              >
                <X className="size-5" />
              </button>
            </div>
            <div className="max-h-[60vh] space-y-3 overflow-y-auto p-4">
              {diagnostics.checks.map((check, index) => {
                const meta = diagnosticStatusMeta(check.status);
                const StatusIcon = meta.icon;
                return (
                  <div key={`${check.code}-${check.path ?? index}`} className="rounded-lg border bg-muted/15 p-3">
                    <div className="flex items-center justify-between gap-3">
                      <div className={`flex items-center gap-2 text-sm font-semibold ${meta.className}`}>
                        <StatusIcon className="size-4" />
                        {check.label}
                      </div>
                      <Badge variant="outline" className={meta.className}>{meta.label}</Badge>
                    </div>
                    {check.path && <p className="mt-2 break-all font-mono text-xs text-muted-foreground">{check.path}</p>}
                    <p className="mt-2 text-xs text-foreground">{check.detail}</p>
                    {check.os_error_code !== null && (
                      <p className="mt-1 font-mono text-[11px] text-muted-foreground">系统错误码：{check.os_error_code}</p>
                    )}
                  </div>
                );
              })}

              <div className="rounded-lg border bg-muted/10 p-3">
                <p className="text-xs font-semibold text-foreground">供管理员识别的后端程序</p>
                <p className="mt-2 break-all font-mono text-xs text-muted-foreground">{diagnostics.executable_path}</p>
                <p className="mt-3 text-xs font-semibold text-foreground">缓存目录</p>
                <p className="mt-2 break-all font-mono text-xs text-muted-foreground">{diagnostics.cache_directory}</p>
                {logPath && (
                  <>
                    <p className="mt-3 text-xs font-semibold text-foreground">运行日志</p>
                    <p className="mt-2 break-all font-mono text-xs text-muted-foreground">{logPath}</p>
                  </>
                )}
              </div>
            </div>
            <div className="flex justify-end gap-2 border-t p-4">
              {window.desktopRuntime?.openLogsFolder && (
                <Button type="button" variant="outline" onClick={openLogsFolder}>
                  <FolderOpen data-icon="inline-start" />
                  打开日志目录
                </Button>
              )}
              <Button type="button" onClick={() => setShowDiagnostics(false)}>关闭</Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
