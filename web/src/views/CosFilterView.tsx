import { lazy, Suspense, useEffect, useMemo, useState } from 'react';
import {
  AlertCircle,
  CheckCircle2,
  Cpu,
  Download,
  FolderOpen,
  Hash,
  Layers,
  PanelLeftClose,
  PanelLeftOpen,
  Percent,
  Trash2,
  X,
  Zap,
} from 'lucide-react';
import { useCosFilterStore } from '../stores/cos-filter-store';
import {
  downloadDataFetchAsCsv,
  downloadGroupingAsCsv,
  downloadGroupingAsXlsx,
  downloadRowsAsXlsx,
} from '../helpers/csv';
import {
  getWavelengthValue,
  standardDeviation,
  WAREHOUSE_AVAILABLE,
  WAREHOUSE_NEED_CONFIRM,
} from '../helpers/utils';
import { MetricCard } from '../components/MetricCard';
import { CosTable } from '../components/CosTable';
import { GroupSummaryTable } from '../components/GroupSummaryTable';
import { DataFetchTable } from '../components/DataFetchTable';
import { CosFilterSidebar } from '../components/CosFilterSidebar';
import { PartInspector } from '../components/PartInspector';
import type { CosGroupResultTab, GroupSummaryRow, CosRow } from '../helpers/types';
import { useShallow } from 'zustand/react/shallow';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Skeleton } from '@/components/ui/skeleton';
import { EmptyState } from '@/components/EmptyState';

const CosTrendPanel = lazy(() =>
  import('../components/CosTrendPanel').then((module) => ({ default: module.CosTrendPanel })),
);

const GROUP_RESULT_TABS: readonly CosGroupResultTab[] = ['groups', 'remaining', 'power', 'trend', 'export'];

const isGroupResultTab = (value: string | null): value is CosGroupResultTab =>
  value !== null && GROUP_RESULT_TABS.includes(value as CosGroupResultTab);

export function CosFilterView() {
  const [activePart, setActivePart] = useState<CosRow | null>(null);
  const [inspectorOpen, setInspectorOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const {
    loadedCosCount,
    step1Rows,
    step2Rows,
    groupResult,
    step1Params,
    selectedGroupIndex,
    setSelectedGroupIndex,
    groupResultTab,
    setGroupResultTab,
    groupingDedupRemoved,
    step4Result,
    setActiveStep,
  } = useCosFilterStore(useShallow((state) => ({
    loadedCosCount: state.loadedCosCount,
    step1Rows: state.step1Rows,
    step2Rows: state.step2Rows,
    groupResult: state.groupResult,
    step1Params: state.step1Params,
    selectedGroupIndex: state.selectedGroupIndex,
    setSelectedGroupIndex: state.setSelectedGroupIndex,
    groupResultTab: state.groupResultTab,
    setGroupResultTab: state.setGroupResultTab,
    groupingDedupRemoved: state.groupingDedupRemoved,
    step4Result: state.step4Result,
    setActiveStep: state.setActiveStep,
  })));

  const step2AvailableRows = useMemo(
    () => step2Rows.filter((row) => WAREHOUSE_AVAILABLE.includes(row.warehouse ?? '')),
    [step2Rows],
  );
  const step2NeedConfirmRows = useMemo(
    () => step2Rows.filter((row) => WAREHOUSE_NEED_CONFIRM.includes(row.warehouse ?? '')),
    [step2Rows],
  );

  const groupedChipCount = useMemo(
    () => (groupResult ? (groupResult.groups || []).reduce((acc, group) => acc + (group?.length || 0), 0) : 0),
    [groupResult],
  );

  const groupingRate = useMemo(() => {
    if (!groupResult) return 0;
    const base = groupedChipCount + groupResult.remaining_count;
    return base <= 0 ? 0 : (groupedChipCount / base) * 100;
  }, [groupResult, groupedChipCount]);

  const groupSummaryRows = useMemo<GroupSummaryRow[]>(() => {
    if (!groupResult || !step1Params) return [];
    return (groupResult.groups || []).map((group, index) => {
      const values = (group || [])
        .map((row) => getWavelengthValue(row, step1Params.wavelength_field))
        .filter((value): value is number => value !== null);
      const min = values.length > 0 ? values.reduce((left, right) => Math.min(left, right)) : null;
      const max = values.length > 0 ? values.reduce((left, right) => Math.max(left, right)) : null;
      const avg = values.length > 0 ? values.reduce((left, right) => left + right, 0) / values.length : null;
      const diff = min !== null && max !== null ? max - min : null;
      const cv = avg !== null && avg !== 0 && values.length > 1 ? (standardDeviation(values) / avg) * 100 : null;
      return {
        groupId: `Group_${String(index + 1).padStart(3, '0')}`,
        count: group.length,
        avg,
        min,
        max,
        diff,
        cv,
      };
    });
  }, [groupResult, step1Params]);

  const safeSelectedGroupIndex = useMemo(() => {
    if (groupSummaryRows.length === 0) return 0;
    return Math.max(0, Math.min(selectedGroupIndex, groupSummaryRows.length - 1));
  }, [groupSummaryRows.length, selectedGroupIndex]);

  const step4Stats = useMemo(() => {
    if (!step4Result || step4Result.records.length === 0) return null;
    const records = step4Result.records;
    const powers = records.map((r) => r.power_w).filter((v): v is number => v !== null && v !== undefined);
    const voltages = records.map((r) => r.voltage_v).filter((v): v is number => v !== null && v !== undefined);
    const efficiencies = records.map((r) => r.efficiency_pct).filter((v): v is number => v !== null && v !== undefined);

    const avgPower = powers.length > 0 ? powers.reduce((a, b) => a + b, 0) / powers.length : null;
    const avgVoltage = voltages.length > 0 ? voltages.reduce((a, b) => a + b, 0) / voltages.length : null;
    const avgEfficiency = efficiencies.length > 0 ? efficiencies.reduce((a, b) => a + b, 0) / efficiencies.length : null;

    return {
      count: records.length,
      avgPower,
      avgVoltage,
      avgEfficiency,
    };
  }, [step4Result]);

  const selectedGroupSummary = useMemo(
    () => (groupSummaryRows.length > 0 ? groupSummaryRows[safeSelectedGroupIndex] : null),
    [groupSummaryRows, safeSelectedGroupIndex],
  );

  const selectedGroupRows = useMemo(() => {
    const groups = groupResult?.groups ?? [];
    if (groups.length === 0) return [];
    const index = Math.max(0, Math.min(safeSelectedGroupIndex, groups.length - 1));
    return groups[index] ?? [];
  }, [groupResult, safeSelectedGroupIndex]);

  useEffect(() => {
    setSelectedGroupIndex(0);
  }, [groupResult, setSelectedGroupIndex]);

  const activeCandidateRows = useMemo(() => {
    if (groupResult) {
      if (groupResultTab === 'groups') return selectedGroupRows;
      if (groupResultTab === 'remaining') return groupResult.remaining ?? [];
      return [];
    }
    if (step2Rows.length > 0) return step2Rows;
    if (step1Rows.length > 0) return step1Rows;
    return [];
  }, [groupResult, groupResultTab, selectedGroupRows, step1Rows, step2Rows]);

  const displayActivePart = useMemo(() => {
    if (activePart && activeCandidateRows.some((row) => row === activePart || row.device_id === activePart.device_id)) {
      return activePart;
    }
    return activeCandidateRows[0] ?? null;
  }, [activeCandidateRows, activePart]);

  return (
    <div className="cos-filter-workbench flex flex-col gap-5 w-full">
      {/* 1. Page Header (Standard shadcn-admin) */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground">COS 筛选</h1>
          <p className="text-xs text-muted-foreground mt-0.5">
            波长聚类、离散度过滤与自动组盘分析
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            className="h-8 gap-1.5 text-xs shadow-xs"
          >
            {sidebarCollapsed ? <PanelLeftOpen className="size-3.5" /> : <PanelLeftClose className="size-3.5" />}
            {sidebarCollapsed ? '展开筛选配置' : '收起筛选配置'}
          </Button>

          {displayActivePart && (
            <Button
              type="button"
              variant={inspectorOpen ? 'secondary' : 'outline'}
              size="sm"
              onClick={() => setInspectorOpen(!inspectorOpen)}
              className="h-8 gap-1.5 text-xs shadow-xs"
            >
              <Cpu className="size-3.5 text-primary" />
              {inspectorOpen ? '收起器件详情' : `器件: ${displayActivePart.device_id.slice(-8)}`}
            </Button>
          )}

          {groupResult && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => downloadGroupingAsCsv(groupResult, 'COS_Grouping_Output.csv')}
              className="h-8 gap-1.5 text-xs shadow-xs"
            >
              <Download className="size-3.5" />
              导出分组 CSV
            </Button>
          )}
        </div>
      </div>

      {/* 2. Main Workspace Layout */}
      <div className="flex flex-col lg:flex-row gap-6 items-start w-full">
        {!sidebarCollapsed && (
          <aside className="w-full lg:w-80 shrink-0">
            <CosFilterSidebar />
          </aside>
        )}

        <div className="flex flex-1 flex-col gap-4 min-w-0 w-full">
          {/* Top KPI Metrics Row */}
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {groupResult ? (
              <>
                <MetricCard label="生成组数" value={`${groupResult.group_count} 组`} color="primary" icon={<Layers />} />
                <MetricCard
                  label="入组散件"
                  value={`${groupedChipCount} 条 (${groupingRate.toFixed(1)}%)`}
                  color="success"
                  icon={<CheckCircle2 />}
                />
                <MetricCard
                  label="未入组散件"
                  value={`${groupResult.remaining_count} 条`}
                  color={groupResult.remaining_count > 0 ? 'warning' : 'default'}
                  icon={<AlertCircle />}
                />
                <MetricCard label="去重过滤量" value={`${groupingDedupRemoved} 条`} icon={<Trash2 />} />
              </>
            ) : (
              <>
                <MetricCard label="已加载批次量" value={String(loadedCosCount)} color="primary" icon={<FolderOpen />} />
                <MetricCard label="基础筛选通过" value={String(step1Rows.length)} icon={<CheckCircle2 />} />
                <MetricCard label="深度筛选通过" value={String(step2Rows.length)} icon={<Layers />} />
                <MetricCard
                  label="最终总入组率"
                  value="-"
                  icon={<Percent />}
                />
              </>
            )}
          </div>

          {/* Results Workspace Card (Full-width) */}
          {!loadedCosCount ? (
            <EmptyState
              icon={<FolderOpen className="size-6" />}
              title="等待数据源加载"
              description="请在左侧筛选设置中选择原始测试数据并加载。"
              className="min-h-[560px] justify-center"
            />
          ) : (
            <Card className="min-h-[720px] border bg-card shadow-xs">
              <CardHeader className="border-b bg-card/80 py-3 px-5 flex flex-row items-center justify-between">
                <CardTitle className="text-sm font-semibold text-foreground">筛选结果工作区</CardTitle>
                {groupResult && (
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">{groupResult.group_count} 组</Badge>
                    {groupResult.remaining_count > 0 && (
                      <Badge variant="outline" className="text-warning border-warning/40">{groupResult.remaining_count} 条未入组</Badge>
                    )}
                  </div>
                )}
              </CardHeader>
              <CardContent className="flex flex-col gap-4 p-5">
                {groupResult ? (
                  <>
                    <Tabs value={groupResultTab} onValueChange={(value) => isGroupResultTab(value) ? setGroupResultTab(value) : null}>
                      <TabsList variant="line" className="w-full justify-start border-b rounded-none px-0 bg-transparent h-9 gap-6">
                        <TabsTrigger value="groups">分组明细</TabsTrigger>
                        <TabsTrigger value="remaining">剩余散件</TabsTrigger>
                        <TabsTrigger value="power" className="gap-1.5">
                          电性能 (功率)
                          {step4Result && (
                            <Badge variant="secondary" className="px-1.5 py-0 text-[10px] font-mono leading-none">
                              {step4Result.records.length}
                            </Badge>
                          )}
                        </TabsTrigger>
                        <TabsTrigger value="trend">图表概览</TabsTrigger>
                        <TabsTrigger value="export">导出工具</TabsTrigger>
                      </TabsList>

                      <TabsContent value="groups" className="pt-4">
                        <div className="grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)]">
                          <div className="flex flex-col gap-3">
                            <div className="flex items-center justify-between gap-3">
                              <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">分组摘要列表</p>
                              <Badge variant="outline">{groupSummaryRows.length} 组</Badge>
                            </div>
                            <GroupSummaryTable
                              rows={groupSummaryRows}
                              selectedGroupIndex={safeSelectedGroupIndex}
                              onSelectionChange={setSelectedGroupIndex}
                            />
                          </div>

                          <div className="flex flex-col gap-3">
                            <div className="flex items-center justify-between gap-3">
                              <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">当前选中组</p>
                              {selectedGroupSummary ? <Badge variant="secondary">{selectedGroupSummary.groupId}</Badge> : null}
                            </div>

                            {selectedGroupSummary ? (
                              <>
                                <div className="grid gap-2 sm:grid-cols-3">
                                  <MetricCard label="组内项数" value={String(selectedGroupSummary.count)} icon={<Hash />} />
                                  <MetricCard label="平均波长" value={selectedGroupSummary.avg === null ? '-' : selectedGroupSummary.avg.toFixed(3)} icon={<Percent />} />
                                  <MetricCard label="极差值" value={selectedGroupSummary.diff === null ? '-' : selectedGroupSummary.diff.toFixed(3)} icon={<AlertCircle />} />
                                </div>
                                <CosTable
                                  rows={selectedGroupRows}
                                  label={`${selectedGroupSummary.groupId} 详情`}
                                  activeRow={displayActivePart}
                                  onRowClick={(row) => {
                                    setActivePart(row);
                                    setInspectorOpen(true);
                                  }}
                                />
                              </>
                            ) : (
                              <EmptyState
                                icon={<Layers />}
                                title="暂无分组详情"
                                description="生成分组结果后，左侧表格会自动选中第一组。"
                              />
                            )}
                          </div>
                        </div>
                      </TabsContent>

                      <TabsContent value="remaining" className="pt-4">
                        {(groupResult.remaining ?? []).length === 0 ? (
                          <Alert>
                            <CheckCircle2 />
                            <AlertTitle>所有候选都已入组</AlertTitle>
                            <AlertDescription>当前结果里没有剩余散件，说明可用候选已经全部参与成组。</AlertDescription>
                          </Alert>
                        ) : (
                          <div className="flex flex-col gap-4">
                            <div className="flex items-center justify-between gap-3">
                              <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">未入组散件</p>
                              <Badge variant="outline">{(groupResult.remaining ?? []).length} 条</Badge>
                            </div>
                            <CosTable
                              rows={groupResult.remaining ?? []}
                              label="Remaining COS rows"
                              activeRow={displayActivePart}
                              onRowClick={(row) => {
                                setActivePart(row);
                                setInspectorOpen(true);
                              }}
                            />
                          </div>
                        )}
                      </TabsContent>

                      <TabsContent value="power" className="pt-4 flex flex-col gap-4">
                        {step4Result && step4Result.records.length > 0 ? (
                          <>
                            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                              <div className="flex items-center gap-2">
                                <Zap className="size-4 text-primary" />
                                <h3 className="text-sm font-semibold text-foreground">COS 电性能实测分析</h3>
                                <Badge variant="secondary">{step4Stats?.count ?? 0} 条实测记录</Badge>
                              </div>
                              <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() => downloadDataFetchAsCsv(step4Result.records, 'COS_Electrical_Power_Output.csv')}
                                className="h-8 gap-1.5 text-xs shadow-xs"
                              >
                                <Download className="size-3.5" />
                                导出电性能 CSV
                              </Button>
                            </div>

                            {step4Stats && (
                              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                                <MetricCard
                                  label="提取器件数"
                                  value={`${step4Stats.count} 颗`}
                                  color="primary"
                                  icon={<Cpu />}
                                />
                                <MetricCard
                                  label="平均光功率"
                                  value={step4Stats.avgPower !== null ? `${step4Stats.avgPower.toFixed(3)} W` : '-'}
                                  color="success"
                                  icon={<Zap />}
                                />
                                <MetricCard
                                  label="平均工作电压"
                                  value={step4Stats.avgVoltage !== null ? `${step4Stats.avgVoltage.toFixed(3)} V` : '-'}
                                  icon={<Hash />}
                                />
                                <MetricCard
                                  label="平均光电效率"
                                  value={step4Stats.avgEfficiency !== null ? `${step4Stats.avgEfficiency.toFixed(1)}%` : '-'}
                                  icon={<Percent />}
                                />
                              </div>
                            )}

                            {step4Result.errors && step4Result.errors.length > 0 && (
                              <Alert variant="destructive">
                                <AlertCircle className="size-4" />
                                <AlertTitle>提取异常 ({step4Result.errors.length} 项)</AlertTitle>
                                <AlertDescription className="text-xs max-h-24 overflow-y-auto space-y-1">
                                  {step4Result.errors.map((err, idx) => (
                                    <div key={idx}>{err}</div>
                                  ))}
                                </AlertDescription>
                              </Alert>
                            )}

                            <div className="rounded-lg border bg-card shadow-xs overflow-hidden">
                              <DataFetchTable rows={step4Result.records} />
                            </div>
                          </>
                        ) : (
                          <EmptyState
                            icon={<Zap className="size-6 text-muted-foreground" />}
                            title="尚未提取电性能 (功率) 数据"
                            description="您可以在左侧「4. 电性能提取」中选择提取目标（全部成组/当前组/候选条目），配置测试路径与电流，一键提取真实功率与 LVI 数据。"
                            className="py-16"
                          >
                            <Button
                              type="button"
                              size="sm"
                              onClick={() => setActiveStep(4)}
                              className="shadow-xs text-xs h-8"
                            >
                              前往配置并提取
                            </Button>
                          </EmptyState>
                        )}
                      </TabsContent>

                      <TabsContent value="trend" className="pt-4">
                        <Suspense
                          fallback={<Skeleton className="min-h-[320px] w-full rounded" />}
                        >
                          <CosTrendPanel groupResult={groupResult} step1Params={step1Params} />
                        </Suspense>
                      </TabsContent>

                      <TabsContent value="export" className="pt-4">
                        <Card className="border bg-card shadow-xs">
                          <CardHeader className="border-b bg-muted/20 py-3 pb-3">
                            <CardTitle className="text-sm font-bold">导出结果集</CardTitle>
                          </CardHeader>
                          <CardContent className="flex flex-col gap-4 pt-4">
                            <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap">
                              <Button type="button" onClick={() => downloadGroupingAsXlsx(groupResult, 'COS_Grouping_Output.xlsx')}>
                                <Download data-icon="inline-start" />
                                导出完整分组
                              </Button>
                              <Button
                                type="button"
                                variant="outline"
                                disabled={(groupResult.remaining ?? []).length === 0}
                                onClick={() => downloadRowsAsXlsx(groupResult.remaining ?? [], 'COS_Remaining.xlsx')}
                              >
                                <Download data-icon="inline-start" />
                                导出剩余散件
                              </Button>
                              {step4Result && (
                                <Button
                                  type="button"
                                  variant="outline"
                                  onClick={() => downloadDataFetchAsCsv(step4Result.records, 'COS_Electrical_Power_Output.csv')}
                                >
                                  <Download data-icon="inline-start" />
                                  导出电性能数据 ({step4Result.records.length} 条)
                                </Button>
                              )}
                            </div>
                          </CardContent>
                        </Card>
                      </TabsContent>
                    </Tabs>
                  </>
                ) : step2Rows.length > 0 && step1Params ? (
                  <>
                    <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                      <div className="flex flex-col gap-2">
                        <div className="flex flex-wrap items-center gap-2">
                          <h3 className="text-sm font-bold tracking-tight text-foreground">深度条件筛选结果</h3>
                          <Badge variant="secondary" className="rounded-md">{step2Rows.length} 条保留</Badge>
                        </div>
                      </div>
                      <Button type="button" variant="outline" size="sm" onClick={() => downloadRowsAsXlsx(step2Rows, 'COS_Step2_Output.xlsx')} className="rounded-lg">
                        <Download className="mr-1.5 h-4 w-4" />
                        导出筛选快照
                      </Button>
                    </div>

                    <div className="grid gap-4 md:grid-cols-2">
                      <MetricCard label="良品仓可直接使用" value={String(step2AvailableRows.length)} color="success" icon={<CheckCircle2 />} />
                      <MetricCard label="待人工确认" value={String(step2NeedConfirmRows.length)} color="warning" icon={<AlertCircle />} />
                    </div>

                    <Tabs defaultValue="available">
                      <TabsList variant="line" className="w-full justify-start border-b rounded-none px-0 bg-transparent h-9 gap-6">
                        <TabsTrigger value="available">可直接使用</TabsTrigger>
                        <TabsTrigger value="review">待确认</TabsTrigger>
                        {step4Result && (
                          <TabsTrigger value="power" className="gap-1.5">
                            电性能 (功率)
                            <Badge variant="secondary" className="px-1.5 py-0 text-[10px] font-mono leading-none">
                              {step4Result.records.length}
                            </Badge>
                          </TabsTrigger>
                        )}
                      </TabsList>

                      <TabsContent value="available" className="pt-4">
                        {step2AvailableRows.length === 0 ? (
                          <EmptyState
                            icon={<CheckCircle2 />}
                            title="当前没有可直接使用的记录"
                            description="可以切到“待确认”查看需要人工确认的仓位记录。"
                          />
                        ) : (
                          <CosTable
                            rows={step2AvailableRows}
                            label="Available COS rows"
                            activeRow={displayActivePart}
                            onRowClick={(row) => {
                              setActivePart(row);
                              setInspectorOpen(true);
                            }}
                          />
                        )}
                      </TabsContent>

                      <TabsContent value="review" className="pt-4">
                        {step2NeedConfirmRows.length === 0 ? (
                          <EmptyState
                            icon={<AlertCircle />}
                            title="当前没有待确认记录"
                            description="说明这一轮筛选后，没有命中需要人工复核的仓位。"
                          />
                        ) : (
                          <CosTable
                            rows={step2NeedConfirmRows}
                            label="Review COS rows"
                            activeRow={displayActivePart}
                            onRowClick={(row) => {
                              setActivePart(row);
                              setInspectorOpen(true);
                            }}
                          />
                        )}
                      </TabsContent>

                      {step4Result && (
                        <TabsContent value="power" className="pt-4 flex flex-col gap-4">
                          <div className="rounded-lg border bg-card shadow-xs overflow-hidden">
                            <DataFetchTable rows={step4Result.records} />
                          </div>
                        </TabsContent>
                      )}
                    </Tabs>
                  </>
                ) : step1Rows.length > 0 ? (
                  <>
                    <div className="flex flex-col gap-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <h3 className="text-sm font-bold tracking-tight text-foreground">基础波长筛选结果</h3>
                        <Badge variant="secondary">{step1Rows.length} 条保留</Badge>
                      </div>
                    </div>

                    <CosTable
                      rows={step1Rows.slice(0, 100)}
                      label="Step 1 preview rows"
                      activeRow={displayActivePart}
                      onRowClick={(row) => {
                        setActivePart(row);
                        setInspectorOpen(true);
                      }}
                    />
                  </>
                ) : (
                  <>
                    <div className="flex flex-col gap-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <h3 className="text-sm font-bold tracking-tight text-foreground">批次加载概况</h3>
                        <Badge variant="secondary">已加载 {loadedCosCount} 条</Badge>
                      </div>
                    </div>
                  </>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* 3. Slide-over Part Inspector Drawer (Overlay, doesn't compress table width) */}
      {inspectorOpen && displayActivePart && (
        <div className="fixed inset-y-0 right-0 z-50 flex w-[380px] max-w-[calc(100vw-2rem)] flex-col border-l bg-background shadow-2xl animate-in slide-in-from-right duration-200">
          <div className="flex h-12 items-center justify-between border-b px-4 bg-muted/20">
            <span className="text-xs font-semibold text-foreground flex items-center gap-2">
              <Cpu className="size-4 text-primary" />
              器件属性检查器
            </span>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              onClick={() => setInspectorOpen(false)}
              className="size-7 rounded hover:bg-muted"
              aria-label="关闭详情面板"
            >
              <X className="size-4" />
            </Button>
          </div>
          <div className="flex-1 overflow-y-auto p-4">
            <PartInspector part={displayActivePart} />
          </div>
        </div>
      )}
    </div>
  );
}
