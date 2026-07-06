import { lazy, Suspense, useEffect, useMemo, useState } from 'react';
import {
  AlertCircle,
  CheckCircle2,
  Download,
  FolderOpen,
  Hash,
  Layers,
  Percent,
  Trash2,
} from 'lucide-react';
import { useCosFilterStore } from '../stores/cos-filter-store';
import { downloadGroupingAsCsv, downloadRowsAsCsv } from '../helpers/csv';
import {
  getWavelengthValue,
  standardDeviation,
  WAREHOUSE_AVAILABLE,
  WAREHOUSE_NEED_CONFIRM,
} from '../helpers/utils';
import { MetricCard } from '../components/MetricCard';
import { CosTable } from '../components/CosTable';
import { GroupSummaryTable } from '../components/GroupSummaryTable';
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

const GROUP_RESULT_TABS: readonly CosGroupResultTab[] = ['groups', 'remaining', 'trend', 'export'];

const isGroupResultTab = (value: string | null): value is CosGroupResultTab =>
  value !== null && GROUP_RESULT_TABS.includes(value as CosGroupResultTab);

export function CosFilterView() {
  const [activePart, setActivePart] = useState<CosRow | null>(null);
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
    <div className="grid gap-6 lg:grid-cols-[320px_minmax(0,1fr)]">
      <CosFilterSidebar />

      <div className="flex flex-col gap-4">
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <MetricCard label="已加载批次量" value={String(loadedCosCount)} color="primary" icon={<FolderOpen />} />
          <MetricCard label="基础筛选通过" value={String(step1Rows.length)} icon={<CheckCircle2 />} />
          <MetricCard label="深度筛选通过" value={String(step2Rows.length)} icon={<Layers />} />
          <MetricCard
            label="最终总入组率"
            value={groupResult ? `${groupingRate.toFixed(1)}%` : '-'}
            color={groupResult ? 'success' : 'default'}
            icon={<Percent />}
          />
        </div>

        {!loadedCosCount ? (
          <EmptyState
            icon={<FolderOpen className="size-6" />}
            title="等待数据源加载"
            className="min-h-[720px] justify-center"
          />
        ) : (
          <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px] items-start">
            <Card className="min-h-[720px] border bg-card shadow-sm">
              <CardHeader className="border-b bg-muted/20 py-3.5">
                <CardTitle className="text-sm font-bold text-foreground">筛选结果工作区</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-6 pt-6">
                {groupResult ? (
                  <>
                    <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                      <div className="flex flex-col gap-2">
                        <div className="flex flex-wrap items-center gap-2">
                          <h3 className="text-sm font-bold tracking-tight text-foreground">自动组合结果</h3>
                          <Badge variant="secondary" className="rounded-md">{groupResult.group_count} 组</Badge>
                          <Badge variant="outline" className="rounded-md">{groupResult.remaining_count} 条未入组</Badge>
                        </div>
                      </div>
                    </div>

                    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                      <MetricCard label="成功生成组数" value={String(groupResult.group_count)} color="primary" icon={<Layers />} />
                      <MetricCard label="包含散件总数" value={String(groupedChipCount)} icon={<Hash />} />
                      <MetricCard label="去重项" value={String(groupingDedupRemoved)} icon={<Trash2 />} />
                      <MetricCard
                        label="未入组散件"
                        value={String(groupResult.remaining_count)}
                        color={groupResult.remaining_count > 0 ? 'warning' : 'default'}
                        icon={<AlertCircle />}
                      />
                    </div>

                    <Tabs value={groupResultTab} onValueChange={(value) => isGroupResultTab(value) ? setGroupResultTab(value) : null}>
                      <TabsList variant="line" className="w-full justify-start border-b rounded-none px-0 bg-transparent h-9 gap-6">
                        <TabsTrigger value="groups">分组明细</TabsTrigger>
                        <TabsTrigger value="remaining">剩余散件</TabsTrigger>
                        <TabsTrigger value="trend">图表概览</TabsTrigger>
                        <TabsTrigger value="export">导出工具</TabsTrigger>
                      </TabsList>

                      <TabsContent value="groups" className="pt-5">
                        <div className="grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)]">
                          <div className="flex flex-col gap-3">
                            <div className="flex items-center justify-between gap-3">
                              <div>
                                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">分组摘要</p>
                              </div>
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
                              <div>
                                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">当前选中组</p>
                              </div>
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
                                  onRowClick={setActivePart}
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

                      <TabsContent value="remaining" className="pt-5">
                        {(groupResult.remaining ?? []).length === 0 ? (
                          <Alert>
                            <CheckCircle2 />
                            <AlertTitle>所有候选都已入组</AlertTitle>
                            <AlertDescription>当前结果里没有剩余散件，说明可用候选已经全部参与成组。</AlertDescription>
                          </Alert>
                        ) : (
                          <div className="flex flex-col gap-4">
                            <div className="flex items-center justify-between gap-3">
                              <div>
                                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">未入组散件</p>
                              </div>
                              <Badge variant="outline">{(groupResult.remaining ?? []).length} 条</Badge>
                            </div>
                            <CosTable
                              rows={groupResult.remaining ?? []}
                              label="Remaining COS rows"
                              activeRow={displayActivePart}
                              onRowClick={setActivePart}
                            />
                          </div>
                        )}
                      </TabsContent>

                      <TabsContent value="trend" className="pt-5">
                        <Suspense
                          fallback={<Skeleton className="min-h-[320px] w-full rounded" />}
                        >
                          <CosTrendPanel groupResult={groupResult} step1Params={step1Params} />
                        </Suspense>
                      </TabsContent>

                      <TabsContent value="export" className="pt-5">
                        <Card className="border bg-card shadow-sm">
                          <CardHeader className="border-b bg-muted/20 py-3 pb-3">
                            <CardTitle className="text-sm font-bold">导出结果集</CardTitle>
                          </CardHeader>
                          <CardContent className="flex flex-col gap-4 pt-4">
                            <div className="flex flex-col gap-3 sm:flex-row">
                              <Button type="button" onClick={() => downloadGroupingAsCsv(groupResult, 'COS_Grouping_Output.csv')}>
                                <Download data-icon="inline-start" />
                                导出完整分组
                              </Button>
                              <Button
                                type="button"
                                variant="outline"
                                disabled={(groupResult.remaining ?? []).length === 0}
                                onClick={() => downloadRowsAsCsv(groupResult.remaining ?? [], 'COS_Remaining.csv')}
                              >
                                <Download data-icon="inline-start" />
                                导出剩余散件
                              </Button>
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
                      <Button type="button" variant="outline" size="sm" onClick={() => downloadRowsAsCsv(step2Rows, 'COS_Step2_Output.csv')} className="rounded-lg">
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
                      </TabsList>

                      <TabsContent value="available" className="pt-5">
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
                            onRowClick={setActivePart}
                          />
                        )}
                      </TabsContent>

                      <TabsContent value="review" className="pt-5">
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
                            onRowClick={setActivePart}
                          />
                        )}
                      </TabsContent>
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
                      onRowClick={setActivePart}
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

            <div className="sticky top-16 z-20">
              <PartInspector part={displayActivePart} />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
