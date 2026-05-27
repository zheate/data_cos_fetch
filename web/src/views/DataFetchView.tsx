import { useDeferredValue, useMemo } from 'react';
import { DatabaseBackup, Download, Files, ListFilter, Loader2, Play, Rows3, Sparkles } from 'lucide-react';
import { useAppStore } from '../stores/app-store';
import { useDataFetchStore } from '../stores/data-fetch-store';
import { request } from '../helpers/api';
import { parseLines, parseCurrentPoints, DEFAULT_TEST_CATEGORIES } from '../helpers/utils';
import { downloadDataFetchAsCsv } from '../helpers/csv';
import { MetricCard } from '../components/MetricCard';
import { DataFetchTable } from '../components/DataFetchTable';
import type { DataFetchResponse, ExtractionMode } from '../helpers/types';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
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
import { Separator } from '@/components/ui/separator';
import { Textarea } from '@/components/ui/textarea';
import { EmptyState } from '@/components/EmptyState';

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
    <Field orientation="horizontal" className="rounded-lg border bg-background px-3 py-2.5">
      <Checkbox id={id} checked={checked} onCheckedChange={(value) => onCheckedChange(value === true)} />
      <FieldContent>
        <FieldLabel htmlFor={id}>{label}</FieldLabel>
      </FieldContent>
    </Field>
  );
}

export function DataFetchView() {
  const { apiBase, token, busy, withTask } = useAppStore();
  const store = useDataFetchStore();
  const deferredEntriesInput = useDeferredValue(store.entriesInput);

  const entryCount = useMemo(() => parseLines(deferredEntriesInput).length, [deferredEntriesInput]);
  const isEntryCountPending = deferredEntriesInput !== store.entriesInput;
  const selectedTestCount = store.mode === 'module' ? store.selectedTests.length : 0;

  const runDataFetch = async () => {
    await withTask(async () => {
      const entries = parseLines(store.entriesInput);
      if (entries.length === 0) {
        throw new Error('请至少输入一个条目。');
      }
      if (store.selectedMeasurements.length === 0) {
        throw new Error('请至少选择一个测试文件。');
      }

      const payload = {
        mode: store.mode,
        entries,
        test_categories: store.mode === 'module' ? store.selectedTests : undefined,
        measurements: store.selectedMeasurements,
        current_points: parseCurrentPoints(store.currentInput),
      };

      const result = await request<DataFetchResponse>(apiBase, token, '/api/v1/data-fetch/extract', payload);
      store.setResult(result);
      return result;
    }, {
      loading: '正在提取数据...',
      success: (result) => `数据提取完成：${result.total} 条合并记录`
    });
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[340px_minmax(0,1fr)] xl:grid-cols-[380px_minmax(0,1fr)]">
      <Card className="shadow-sm border-border/60 bg-card/80 backdrop-blur-sm h-fit">
        <CardHeader className="border-b border-border/50 bg-muted/30">
          <CardTitle>数据提取配置</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-6 pt-6">
          <section className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Rows3 className="text-muted-foreground" />
                <div>
                  <p className="text-sm font-medium">输入来源</p>
                </div>
              </div>
              <Badge variant="secondary">{isEntryCountPending ? '计算中' : `${entryCount} 条目`}</Badge>
            </div>

            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="data-fetch-mode">提取模式</FieldLabel>
                <Select
                  value={store.mode}
                  onValueChange={(value) => {
                    if (value === 'module' || value === 'chip') {
                      store.setMode(value as ExtractionMode);
                    }
                  }}
                >
                  <SelectTrigger id="data-fetch-mode" className="w-full">
                    <SelectValue placeholder="选择提取模式" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="module">模块级 (Module)</SelectItem>
                      <SelectItem value="chip">芯片级 (Chip)</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>

              <Field>
                <FieldLabel htmlFor="entries-input">条目列表</FieldLabel>
                <Textarea
                  id="entries-input"
                  rows={7}
                  placeholder="/abs/path/to/shellA&#10;/abs/path/to/shellB"
                  value={store.entriesInput}
                  onChange={(event) => store.setEntriesInput(event.target.value)}
                />
              </Field>
            </FieldGroup>
          </section>

          <Separator />

          <section className="flex flex-col gap-4">
            <div className="flex items-center gap-2">
              <ListFilter className="text-muted-foreground" />
              <div>
                <p className="text-sm font-medium">范围选择</p>
              </div>
            </div>

            <FieldGroup>
              {store.mode === 'module' && (() => {
                const allCategories = [...DEFAULT_TEST_CATEGORIES, ...store.customTests];
                return (
                  <FieldSet>
                    <div className="flex items-center justify-between">
                      <FieldLegend variant="label">测试站别</FieldLegend>
                      <form
                        onSubmit={(e) => {
                          e.preventDefault();
                          const input = new FormData(e.currentTarget).get('newCategory') as string;
                          if (input && input.trim()) {
                            store.addCustomTest(input.trim());
                            e.currentTarget.reset();
                          }
                        }}
                        className="flex items-center gap-2"
                      >
                        <Input 
                          name="newCategory" 
                          placeholder="输入新增站别名称"
                          className="h-7 w-36 text-xs" 
                        />
                        <Button type="submit" size="sm" variant="secondary" className="h-7 px-2 text-xs">添加到列表</Button>
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
                                className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground opacity-0 hover:bg-muted hover:text-destructive group-hover:opacity-100"
                                onClick={() => store.removeCustomTest(category)}
                                title="移除站别"
                              >
                                &times;
                              </button>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  </FieldSet>
                );
              })()}



              <Field>
                <FieldLabel htmlFor="current-input">电流点参数</FieldLabel>
                <Input
                  id="current-input"
                  placeholder='无输入为最大电流点，"a" 代表所有，支持区间如 12~19'
                  value={store.currentInput}
                  onChange={(event) => store.setCurrentInput(event.target.value)}
                />
              </Field>
            </FieldGroup>
          </section>

          <Separator />

          <section className="flex flex-col gap-4">
            <div className="flex items-center gap-2">
              <Sparkles className="text-muted-foreground" />
              <div>
                <p className="text-sm font-medium">执行动作</p>
              </div>
            </div>

            <Alert>
              <Files />
              <AlertTitle>当前输出范围</AlertTitle>
              <AlertDescription>
                已录入 {entryCount} 条条目{store.mode === 'module' ? `，站别 ${selectedTestCount} 个` : ''}。
              </AlertDescription>
            </Alert>

            <Button type="button" className="w-full" disabled={busy} onClick={runDataFetch}>
              {busy ? <Loader2 className="animate-spin" data-icon="inline-start" /> : <Play data-icon="inline-start" />}
              开始提取数据
            </Button>
          </section>
        </CardContent>
      </Card>

      <div className="flex flex-col gap-4">
        <div className="grid gap-4 md:grid-cols-2">
          <MetricCard label="条目数量" value={String(entryCount)} color="primary" icon={<Rows3 />} />
          <MetricCard label="测试站别" value={store.mode === 'module' ? String(selectedTestCount) : '-'} icon={<ListFilter />} />
        </div>

        <Card className="min-h-[680px] shadow-sm border-border/60 bg-card/80 backdrop-blur-sm">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 border-b border-border/50 bg-muted/30 pb-4">
            <CardTitle>提取结果</CardTitle>
            {store.result && store.result.records.length > 0 && (
              <Button type="button" variant="outline" size="sm" onClick={() => downloadDataFetchAsCsv(store.result!.records, 'Data_Fetch_Output.csv')}>
                <Download className="mr-2 h-4 w-4" />
                导出数据 CSV
              </Button>
            )}
          </CardHeader>
          <CardContent className="flex flex-col gap-4 pt-6">
            {!store.result ? (
              <EmptyState
                icon={<DatabaseBackup className="size-6" />}
                title="等待数据提取"
                className="min-h-[560px] justify-center"
              />
            ) : (
              <>
                <div className="grid gap-4 md:grid-cols-2">
                  <MetricCard label="记录总数" value={String(store.result.total)} color="primary" icon={<DatabaseBackup />} />
                  <MetricCard label="提示信息" value={String(store.result.infos.length)} icon={<Sparkles />} />
                </div>

                {store.result.infos.length > 0 && (
                  <Alert>
                    <Sparkles />
                    <AlertTitle>附加提示</AlertTitle>
                    <AlertDescription>
                      <div className="flex flex-col gap-1">
                        {store.result.infos.map((info) => (
                          <p key={info}>{info}</p>
                        ))}
                      </div>
                    </AlertDescription>
                  </Alert>
                )}

                <DataFetchTable rows={store.result.records} />
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
