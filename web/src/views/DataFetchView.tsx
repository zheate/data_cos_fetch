import { useDeferredValue, useMemo, useState } from 'react';
import {
  Database,
  Download,
  Loader2,
  PanelLeftOpen,
  Search,
  ShieldCheck,
  Trash2,
} from 'lucide-react';
import { toast } from 'sonner';
import { useAppStore } from '../stores/app-store';
import { useDataFetchStore } from '../stores/data-fetch-store';
import { request } from '../helpers/api';
import { parseLines, parseCurrentPoints } from '../helpers/utils';
import { downloadDataFetchAsXlsx } from '../helpers/csv';
import { DataFetchConfigPanel } from '../components/DataFetchConfigPanel';
import { DataFetchIssuesPanel } from '../components/DataFetchIssuesPanel';
import { DataFetchTable } from '../components/DataFetchTable';
import { EnvironmentDiagnosticsDialog } from '../components/EnvironmentDiagnosticsDialog';
import type {
  DataFetchExtractPayload,
  DataFetchResponse,
  ExtractionDiagnosticsResponse,
} from '../helpers/types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';

type ResultTab = 'records' | 'issues';

export function DataFetchView() {
  const { apiBase, token, busy, withTask, logPath } = useAppStore();
  const store = useDataFetchStore();
  const deferredEntriesInput = useDeferredValue(store.entriesInput);
  const entryCount = useMemo(() => parseLines(deferredEntriesInput).length, [deferredEntriesInput]);
  const canRun = !busy && entryCount > 0 && store.selectedMeasurements.length > 0;
  const [configOpen, setConfigOpen] = useState(true);
  const [activeResultTab, setActiveResultTab] = useState<ResultTab>('records');
  const [searchTerm, setSearchTerm] = useState('');
  const [showDiagnostics, setShowDiagnostics] = useState(false);
  const [diagnostics, setDiagnostics] = useState<ExtractionDiagnosticsResponse | null>(null);

  const resultStats = useMemo(() => {
    const result = store.result;
    if (!result) return { errors: [], infos: [], issues: 0 };
    return {
      errors: result.errors || [],
      infos: result.infos || [],
      issues: (result.errors?.length || 0) + (result.infos?.length || 0),
    };
  }, [store.result]);

  const runDataFetch = async () => {
    const toastId = toast.loading('正在提取测试数据');
    const result = await withTask(async () => {
      const entries = parseLines(store.entriesInput);
      if (entries.length === 0) throw new Error('请先输入至少一个条目。');
      if (store.selectedMeasurements.length === 0) throw new Error('请至少选择一项测试文件。');

      const moduleDefaultRoot = store.moduleDefaultRoot.trim();
      const chipDefaultRoots = parseLines(
        store.chipDefaultRootsInput?.trim() ? store.chipDefaultRootsInput : 'Z:/Ldtd/\nZ:/Ldtd/Ldtd/',
      );
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
    setSearchTerm('');
    const hasErrors = result.errors.length > 0;
    const hasInfos = result.infos.length > 0;
    setActiveResultTab(hasErrors || (result.total === 0 && hasInfos) ? 'issues' : 'records');

    if (result.total === 0 && hasErrors) {
      toast.error(`提取失败：0 条记录，${result.errors.length} 条错误`, { id: toastId });
    } else if (hasErrors) {
      toast.warning(`部分完成：${result.total} 条记录，${result.errors.length} 条错误`, { id: toastId });
    } else if (result.total === 0) {
      toast.warning('未提取到记录，请检查配置或运行环境检测。', { id: toastId });
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

  const openConfig = (focusEntries = false) => {
    setConfigOpen(true);
    if (focusEntries) {
      window.requestAnimationFrame(() => document.getElementById('entries-input')?.focus());
    }
  };

  const clearResult = () => {
    store.setResult(null);
    setActiveResultTab('records');
    setSearchTerm('');
    setConfigOpen(true);
  };

  return (
    <div className="data-fetch-workbench flex w-full flex-col">
      <header className="flex min-h-10 flex-col gap-3 pb-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 items-center gap-2">
          {!configOpen && (
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              onClick={() => openConfig()}
              aria-label="展开任务配置"
              title="展开任务配置"
            >
              <PanelLeftOpen />
            </Button>
          )}
          <h1 className="text-[17px] font-medium leading-6 tracking-tight text-foreground">数据提取</h1>
          {busy && (
            <span className="ml-1 flex items-center gap-1.5 text-[11px] text-muted-foreground" role="status">
              <Loader2 className="size-3 animate-spin" />正在提取
            </span>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button type="button" variant="outline" size="sm" disabled={busy} onClick={runEnvironmentDiagnostics} className="bg-card">
            {busy ? <Loader2 className="animate-spin" /> : <ShieldCheck />}读取环境检测
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="bg-card disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100"
            disabled={!store.result || store.result.records.length === 0 || busy}
            onClick={() => store.result && downloadDataFetchAsXlsx(store.result.records, 'Data_Fetch_Output.xlsx')}
          >
            <Download />导出 Excel
          </Button>
          {store.result && (
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              onClick={clearResult}
              disabled={busy}
              aria-label="清空提取结果"
              title="清空提取结果"
              className="text-muted-foreground hover:text-destructive"
            >
              <Trash2 />
            </Button>
          )}
        </div>
      </header>

      <div className={cn('grid items-start', configOpen ? 'md:grid-cols-[260px_minmax(0,1fr)] lg:grid-cols-[280px_minmax(0,1fr)]' : 'grid-cols-1')}>
        {configOpen && (
          <div className="pr-0 md:pr-4 lg:pr-5">
            <DataFetchConfigPanel
              busy={busy}
              canRun={canRun}
              entryCount={entryCount}
              hasResult={Boolean(store.result)}
              onRun={runDataFetch}
              onCollapse={() => setConfigOpen(false)}
            />
          </div>
        )}

        <section
          className={cn(
            'min-w-0 overflow-hidden rounded-md border bg-card',
            configOpen && 'mt-5 md:mt-0',
          )}
          aria-label="提取结果工作区"
        >
          <Tabs value={activeResultTab} onValueChange={(value) => setActiveResultTab(value as ResultTab)}>
            <div className="flex min-h-11 flex-wrap items-center justify-between gap-2 border-b px-4">
              <TabsList variant="line" className="h-11 gap-5 rounded-none" aria-label="结果视图">
                <TabsTrigger value="records" className="h-11 gap-1.5 rounded-none px-0 text-xs after:bottom-0">
                  解析明细
                  <span className="text-[11px] font-normal tabular-nums text-muted-foreground">{store.result?.records.length ?? 0}</span>
                </TabsTrigger>
                <TabsTrigger value="issues" className="h-11 gap-1.5 rounded-none px-0 text-xs after:bottom-0">
                  异常记录
                  <span className={cn('text-[11px] font-normal tabular-nums', resultStats.issues > 0 ? 'text-destructive' : 'text-muted-foreground')}>
                    {resultStats.issues}
                  </span>
                </TabsTrigger>
              </TabsList>

              <div className="ml-auto flex items-center gap-2 py-1.5">
                {activeResultTab === 'records' && (
                  <div className="relative w-full sm:w-[220px]">
                    <Search className="absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                    <Input
                      type="search"
                      aria-label="搜索条目或站别"
                      placeholder="搜索条目 / 站别"
                      className="h-7 border-border bg-muted/30 pl-8 text-xs"
                      value={searchTerm}
                      onChange={(event) => setSearchTerm(event.target.value)}
                      disabled={!store.result}
                    />
                  </div>
                )}
              </div>
            </div>

            {busy && store.result && (
              <div className="flex items-center gap-2 border-b bg-primary/5 px-4 py-2 text-[11px] text-primary" role="status">
                <Loader2 className="size-3 animate-spin" />正在按当前配置更新，旧结果暂时保留。
              </div>
            )}

            <TabsContent value="records" className="m-0">
              {store.result ? (
                <DataFetchTable
                  rows={store.result.records}
                  searchTerm={searchTerm}
                  onSearchChange={setSearchTerm}
                  pageSize={12}
                />
              ) : busy ? (
                <div className="flex min-h-[400px] flex-col items-center justify-center px-6 py-12 text-center" role="status">
                  <Loader2 className="mb-4 size-5 animate-spin text-primary" />
                  <p className="text-xs text-muted-foreground">正在扫描并解析测试文件</p>
                </div>
              ) : (
                <div className="flex min-h-[400px] flex-col items-center justify-center px-6 py-12 text-center">
                  <span className="mb-4 flex size-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
                    <Database className="size-4" />
                  </span>
                  <h3 className="text-sm font-medium text-foreground">等待提取</h3>
                  <p className="mt-1.5 max-w-sm text-xs leading-5 text-muted-foreground">填写左侧任务配置并运行提取，解析记录会显示在这里。</p>
                  {!configOpen && (
                    <Button type="button" variant="outline" size="sm" className="mt-4" onClick={() => openConfig(true)}>
                      展开任务配置
                    </Button>
                  )}
                </div>
              )}
            </TabsContent>

            <TabsContent value="issues" className="m-0">
              {store.result ? (
                <DataFetchIssuesPanel errors={resultStats.errors} infos={resultStats.infos} />
              ) : (
                <div className="flex min-h-[400px] items-center justify-center px-6 text-center text-xs text-muted-foreground">
                  提取完成后，错误与查找提示会显示在这里。
                </div>
              )}
            </TabsContent>
          </Tabs>
        </section>
      </div>

      <EnvironmentDiagnosticsDialog
        open={showDiagnostics}
        onOpenChange={setShowDiagnostics}
        diagnostics={diagnostics}
        logPath={logPath}
        onOpenLogs={openLogsFolder}
      />
    </div>
  );
}
