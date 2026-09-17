import type { ReactNode } from 'react';
import { useDeferredValue, useMemo, useState } from 'react';
import {
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  Download,
  FolderOpen,
  Hash,
  Info,
  Layers,
  ListChecks,
  Loader2,
  Play,
  ShieldCheck,
  SlidersHorizontal,
  Trash2,
  X,
} from 'lucide-react';
import { toast } from 'sonner';
import { useAppStore } from '../stores/app-store';
import { useDataFetchStore } from '../stores/data-fetch-store';
import { request } from '../helpers/api';
import { parseLines, parseCurrentPoints, DEFAULT_TEST_CATEGORIES, MEASUREMENTS } from '../helpers/utils';
import { downloadDataFetchAsXlsx } from '../helpers/csv';
import { DataFetchTable } from '../components/DataFetchTable';
import { MetricCard } from '../components/MetricCard';
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
  title,
  meta,
  children,
}: {
  title: string;
  meta?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-2.5">
      <div className="flex items-center justify-between">
        <h3 className="text-xs font-semibold text-foreground">{title}</h3>
        {meta && <div className="shrink-0 text-xs text-muted-foreground">{meta}</div>}
      </div>
      {children}
    </div>
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
  const canRun = !busy && entryCount > 0 && selectedMeasurementCount > 0;
  const [configOpen, setConfigOpen] = useState(false);
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
      const chipDefaultRoots = parseLines(store.chipDefaultRootsInput?.trim() ? store.chipDefaultRootsInput : 'Z:/Ldtd/\nZ:/Ldtd/Ldtd/');
      const payload: DataFetchExtractPayload = {
        mode: store.mode,
        entries,
        test_categories: store.mode === 'module' ? store.selectedTests : undefined,
        measurements: store.selectedMeasurements,
        current_points: parseCurrentPoints(store.currentInput),
        module_default_root: moduleDefaultRoot || undefined,
        chip_default_root: chipDefaultRoots[0] || 'Z:/Ldtd/',
        chip_default_roots: chipDefaultRoots.length > 0 ? chipDefaultRoots : ['Z:/Ldtd/', 'Z:/Ldtd/Ldtd/'],
      };

      return request<DataFetchResponse>(apiBase, token, '/api/v1/data-fetch/extract', payload);
    });

    if (!result) {
      toast.dismiss(toastId);
      return;
    }

    store.setResult(result);
    setConfigOpen(false); // Auto-collapse configuration to give table maximum focus
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

  // Structured Configuration Deck (2 Columns)
  const renderConfigCards = () => (
    <div className="grid gap-5 lg:grid-cols-2">
      {/* Card 1: Data Source & Entries */}
      <Card className="border bg-card shadow-xs">
        <CardHeader className="border-b px-5 py-3.5 flex flex-row items-center justify-between">
          <CardTitle className="text-sm font-semibold">1. 数据来源与输入条目</CardTitle>
          <Badge variant="outline" className="font-mono text-[11px]">{store.mode}</Badge>
        </CardHeader>
        <CardContent className="p-5 flex flex-col gap-4">
          <FieldGroup className="gap-3">
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
                    <SelectItem value="module">模块 (Module)</SelectItem>
                    <SelectItem value="chip">芯片 (Chip)</SelectItem>
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
                  rows={2}
                  className="min-h-[60px] max-h-[140px] resize-y overflow-y-auto font-mono text-xs [field-sizing:fixed]"
                  placeholder={'Z:/Ldtd/\nD:/Ldtd/'}
                  value={store.chipDefaultRootsInput}
                  onChange={(event) => store.setChipDefaultRootsInput(event.target.value)}
                />
              )}
            </Field>

            <ConfigSection
              title="条目清单 (Entry IDs)"
              meta={
                <div className="flex items-center gap-2">
                  <Badge variant="secondary">{isEntryCountPending ? '计算中' : `${entryCount} 条`}</Badge>
                  {store.entriesInput && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="xs"
                      onClick={() => store.setEntriesInput('')}
                      className="h-5 px-1.5 text-[11px] text-muted-foreground hover:text-destructive"
                    >
                      清空
                    </Button>
                  )}
                </div>
              }
            >
              <Textarea
                id="entries-input"
                rows={6}
                aria-label="条目列表"
                className="min-h-[140px] max-h-[220px] resize-y overflow-y-auto font-mono text-xs leading-relaxed [field-sizing:fixed]"
                placeholder="/abs/path/to/shellA&#10;/abs/path/to/shellB"
                value={store.entriesInput}
                onChange={(event) => store.setEntriesInput(event.target.value)}
              />
            </ConfigSection>
          </FieldGroup>
        </CardContent>
      </Card>

      {/* Card 2: Tests & Conditions */}
      <Card className="border bg-card shadow-xs">
        <CardHeader className="border-b px-5 py-3.5 flex flex-row items-center justify-between">
          <CardTitle className="text-sm font-semibold">2. 测试项目与参数过滤</CardTitle>
          <Badge variant="secondary" className="text-[11px]">{selectedMeasurementCount} 项已选</Badge>
        </CardHeader>
        <CardContent className="p-5 flex flex-col gap-4">
          <ConfigSection
            title="测试文件类型"
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
              <p className="mt-1 flex items-center gap-1.5 text-xs text-destructive">
                <AlertCircle className="size-3.5" />
                至少选择一个测试文件
              </p>
            )}
          </ConfigSection>

          <div className="h-px bg-border/60" />

          <FieldGroup className="gap-3.5">
            <Field>
              <FieldLabel htmlFor="current-input">电流点设置</FieldLabel>
              <Input
                id="current-input"
                placeholder='例如：最大电流 / a / 12~19'
                value={store.currentInput}
                onChange={(event) => store.setCurrentInput(event.target.value)}
              />
            </Field>

            {store.mode === 'module' && (
              <FieldSet className="gap-2.5">
                <div className="flex items-center justify-between gap-3">
                  <FieldLegend variant="label" className="mb-0">站别 / 测试类别</FieldLegend>
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
                      className="h-7 w-28 text-xs"
                    />
                    <Button type="submit" size="sm" variant="secondary" className="h-7 px-2 text-xs">添加</Button>
                  </form>
                </div>
                <div className="grid gap-2 sm:grid-cols-2 max-h-[150px] overflow-y-auto pr-1">
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
          </FieldGroup>
        </CardContent>
      </Card>
    </div>
  );

  return (
    <div className="data-fetch-workbench flex flex-col gap-5 w-full">
      {/* 1. Page Header (Standard shadcn-admin) */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground">数据提取</h1>
          <p className="text-xs text-muted-foreground mt-0.5">
            配置数据来源与测试项目，并发解析芯片/模块测试数据
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {store.result && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setConfigOpen(!configOpen)}
              className="h-8 gap-1.5 text-xs shadow-xs"
            >
              <SlidersHorizontal className="size-3.5" />
              {configOpen ? '收起配置' : '调整配置'}
            </Button>
          )}
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={busy}
            onClick={runEnvironmentDiagnostics}
            className="h-8 gap-1.5 text-xs shadow-xs"
          >
            <ShieldCheck className="size-3.5" />
            读取环境检测
          </Button>
          {store.result && store.result.records.length > 0 && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => downloadDataFetchAsXlsx(store.result!.records, 'Data_Fetch_Output.xlsx')}
              className="h-8 gap-1.5 text-xs shadow-xs"
            >
              <Download className="size-3.5" />
              导出 Excel
            </Button>
          )}
          {store.result && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => store.setResult(null)}
              className="h-8 gap-1 text-xs text-muted-foreground hover:text-destructive"
              title="清空结果重新配置"
            >
              <Trash2 className="size-3.5" />
              重置
            </Button>
          )}
        </div>
      </div>

      {/* 2. Results Mode vs. Initial Config Mode */}
      {store.result ? (
        <>
          {/* Top KPI Metrics Row (shadcn-admin KPI Stat Cards) */}
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <MetricCard
              label="总提取记录"
              value={`${resultStats.total} 条`}
              icon={<Hash className="size-4" />}
            />
            <MetricCard
              label="有效独立条目"
              value={`${resultStats.entries} 个`}
              icon={<Layers className="size-4" />}
            />
            <MetricCard
              label="解析成功"
              value={`${resultStats.success} 条`}
              color="success"
              icon={<CheckCircle2 className="size-4" />}
            />
            <Card
              onClick={() => (resultStats.failures > 0 || resultStats.infos.length > 0) && setShowErrors(true)}
              className={cn(
                "rounded-lg border bg-card text-card-foreground shadow-xs transition-colors select-none",
                (resultStats.failures > 0 || resultStats.infos.length > 0) && "cursor-pointer hover:bg-muted/50 border-destructive/40"
              )}
            >
              <CardContent className="p-4">
                <div className="flex items-center justify-between space-y-0 pb-1.5">
                  <span className="text-xs font-medium text-muted-foreground">异常与提示</span>
                  <AlertCircle className={cn("size-4 shrink-0", resultStats.failures > 0 ? "text-destructive" : "text-muted-foreground")} />
                </div>
                <div className={cn("text-2xl font-bold tracking-tight", resultStats.failures > 0 ? "text-destructive" : "text-foreground")}>
                  {resultStats.failures > 0 ? `${resultStats.failures} 项错误` : resultStats.infos.length > 0 ? `${resultStats.infos.length} 项提示` : '0 项异常'}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Expandable Configuration Deck */}
          {configOpen && (
            <div className="flex flex-col gap-3 rounded-lg border bg-muted/10 p-4 shadow-xs">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  修改提取参数
                </span>
                <Button
                  type="button"
                  size="sm"
                  disabled={!canRun}
                  onClick={runDataFetch}
                  className="h-7 text-xs px-3 shadow-xs"
                >
                  {busy ? <Loader2 className="mr-1.5 size-3.5 animate-spin" /> : <Play className="mr-1.5 size-3.5" />}
                  重新提取
                </Button>
              </div>
              {renderConfigCards()}
            </div>
          )}

          {/* Full-width Data Table (Takes 100% of workspace width) */}
          <Card className="border bg-card shadow-xs">
            <CardHeader className="flex flex-row items-center justify-between border-b px-5 py-3">
              <div className="flex items-center gap-2">
                <ListChecks className="size-4 text-primary" />
                <CardTitle className="text-sm font-semibold">实测解析明细</CardTitle>
                <Badge variant="secondary" className="text-[11px] font-mono">{store.result.records.length} 行</Badge>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              <DataFetchTable rows={store.result.records} />
            </CardContent>
          </Card>
        </>
      ) : (
        /* Initial Setup Mode: Balanced 2-Column Deck + Action Row */
        <div className="flex flex-col gap-5">
          {renderConfigCards()}

          <Card className="border bg-card shadow-xs">
            <CardContent className="flex flex-col sm:flex-row items-center justify-between gap-4 p-4">
              <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                <span>就绪状态：</span>
                <Badge variant={entryCount > 0 ? 'secondary' : 'outline'}>
                  {entryCount > 0 ? `${entryCount} 条待扫描` : '请输入条目'}
                </Badge>
                <Badge variant={selectedMeasurementCount > 0 ? 'secondary' : 'outline'}>
                  {selectedMeasurementCount > 0 ? `${selectedMeasurementCount} 项测试文件` : '请选测试文件'}
                </Badge>
                <Badge variant="outline">
                  {store.mode === 'module' ? `模块模式 (${selectedTestCount} 站别)` : '芯片模式'}
                </Badge>
              </div>
              <Button
                type="button"
                size="lg"
                className="h-9 px-6 font-semibold w-full sm:w-auto shrink-0 shadow-xs"
                disabled={!canRun}
                onClick={runDataFetch}
              >
                {busy ? <Loader2 className="mr-2 size-4 animate-spin" /> : <Play className="mr-2 size-4" />}
                开始提取数据
              </Button>
            </CardContent>
          </Card>
        </div>
      )}

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
