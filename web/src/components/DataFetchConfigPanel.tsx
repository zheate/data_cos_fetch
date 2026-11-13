import { useMemo, useState } from 'react';
import {
  AlertCircle,
  Loader2,
  PanelLeftClose,
  Play,
  Plus,
  SlidersHorizontal,
  X,
} from 'lucide-react';
import { useDataFetchStore } from '../stores/data-fetch-store';
import { DEFAULT_TEST_CATEGORIES, MEASUREMENTS } from '../helpers/utils';
import type { ExtractionMode } from '../helpers/types';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
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

const MEASUREMENT_META: Record<string, string> = {
  LVI: '电性能',
  Rth: '热阻',
  lambd: '波长',
};

type DataFetchConfigPanelProps = {
  busy: boolean;
  canRun: boolean;
  entryCount: number;
  hasResult: boolean;
  onRun: () => void;
  onCollapse: () => void;
};

export function DataFetchConfigPanel({
  busy,
  canRun,
  entryCount,
  hasResult,
  onRun,
  onCollapse,
}: DataFetchConfigPanelProps) {
  const store = useDataFetchStore();
  const [addingCategory, setAddingCategory] = useState(false);
  const allCategories = useMemo(
    () => [...DEFAULT_TEST_CATEGORIES, ...store.customTests],
    [store.customTests],
  );
  const allTestsSelected = store.selectedTests.length === allCategories.length;
  const missingItems = [
    entryCount === 0 ? '条目' : null,
    store.selectedMeasurements.length === 0 ? '测试文件' : null,
  ].filter(Boolean);

  return (
    <aside className="data-fetch-config flex min-w-0 flex-col" aria-label="提取任务配置">
      <div className="flex h-9 shrink-0 items-center justify-between">
        <div className="flex items-center gap-2 text-xs font-medium text-foreground">
          <SlidersHorizontal className="size-3.5" />
          <h2>任务配置</h2>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          onClick={onCollapse}
          aria-label="收起任务配置"
          title="收起任务配置"
        >
          <PanelLeftClose />
        </Button>
      </div>

      <fieldset disabled={busy} className="min-h-0 flex-1 space-y-4 overflow-y-auto pt-2 pr-1">
        <div className="space-y-1.5">
          <label htmlFor="data-fetch-mode" className="text-xs text-muted-foreground">数据模式</label>
          <Select
            value={store.mode}
            onValueChange={(value) => {
              if (value === 'module' || value === 'chip') store.setMode(value as ExtractionMode);
            }}
          >
            <SelectTrigger id="data-fetch-mode" className="w-full bg-card">
              <SelectValue placeholder="选择模式" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="module">模块（Module）</SelectItem>
                <SelectItem value="chip">芯片（Chip）</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-1.5">
          <label htmlFor="default-root-input" className="text-xs text-muted-foreground">
            {store.mode === 'module' ? '模块根目录' : '芯片根目录'}
          </label>
          {store.mode === 'module' ? (
            <Input
              id="default-root-input"
              className="bg-card font-mono text-xs"
              value={store.moduleDefaultRoot}
              onChange={(event) => store.setModuleDefaultRoot(event.target.value)}
            />
          ) : (
            <Textarea
              id="default-root-input"
              rows={2}
              className="min-h-[58px] resize-y bg-card font-mono text-xs leading-5 [field-sizing:fixed]"
              placeholder={'Z:/Ldtd/\nD:/Ldtd/'}
              aria-describedby="chip-root-hint"
              value={store.chipDefaultRootsInput}
              onChange={(event) => store.setChipDefaultRootsInput(event.target.value)}
            />
          )}
          {store.mode === 'chip' && (
            <p id="chip-root-hint" className="text-[10px] leading-4 text-muted-foreground">每行一个目录，按顺序查找。</p>
          )}
        </div>

        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <label htmlFor="entries-input" className="text-xs text-muted-foreground">条目清单</label>
            <div className="flex items-center gap-1 text-[10px] text-muted-foreground">
              <span className="tabular-nums">{entryCount > 0 ? `${entryCount} 项` : '未输入'}</span>
              {store.entriesInput && (
                <button
                  type="button"
                  onClick={() => store.setEntriesInput('')}
                  className="ml-1 rounded px-1 py-0.5 hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
                >
                  清空
                </button>
              )}
            </div>
          </div>
          <Textarea
            id="entries-input"
            rows={3}
            className="min-h-[74px] resize-y bg-card font-mono text-xs leading-6 [field-sizing:fixed]"
            placeholder={'每行一个条目或路径\nModule-20261104-01'}
            value={store.entriesInput}
            onChange={(event) => store.setEntriesInput(event.target.value)}
          />
        </div>

        <fieldset className="space-y-2" aria-labelledby="measurements-label">
          <legend id="measurements-label" className="text-xs text-muted-foreground">测试文件类型</legend>
          <div className="flex flex-wrap items-center gap-x-5 gap-y-2 pt-0.5">
            {MEASUREMENTS.map((measurement) => (
              <label key={measurement} className="flex cursor-pointer items-center gap-1.5 text-xs text-foreground">
                <Checkbox
                  checked={store.selectedMeasurements.includes(measurement)}
                  onCheckedChange={(value) => store.toggleMeasurement(measurement, value === true)}
                  aria-label={`${measurement}，${MEASUREMENT_META[measurement]}`}
                />
                <span>{measurement}</span>
              </label>
            ))}
          </div>
        </fieldset>

        <div className="space-y-1.5">
          <label htmlFor="current-input" className="text-xs text-muted-foreground">电流点 / A</label>
          <Input
            id="current-input"
            className="bg-card font-mono text-xs"
            placeholder="12~15"
            aria-describedby="current-input-hint"
            value={store.currentInput}
            onChange={(event) => store.setCurrentInput(event.target.value)}
          />
          <p id="current-input-hint" className="text-[10px] leading-4 text-muted-foreground">
            支持范围与逗号组合。留空取最大电流，a 表示全部。
          </p>
        </div>

        {store.mode === 'module' && (
          <fieldset className="space-y-2" aria-labelledby="categories-label">
            <div className="flex items-center justify-between">
              <legend id="categories-label" className="text-xs text-muted-foreground">站别 / 测试类别</legend>
              <button
                type="button"
                onClick={() => store.setSelectedTests(allTestsSelected ? [] : allCategories)}
                className="rounded px-1 py-0.5 text-[10px] text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
              >
                {allTestsSelected ? '全不选' : '全选'}
              </button>
            </div>
            <div className="grid grid-cols-2 gap-x-2 gap-y-1">
              {allCategories.map((category) => {
                const checked = store.selectedTests.includes(category);
                const isCustom = store.customTests.includes(category);
                return (
                  <div key={category} className="group flex min-w-0 items-center gap-1.5">
                    <label className="flex min-w-0 flex-1 cursor-pointer items-center gap-1.5 py-1 text-xs text-foreground">
                      <Checkbox
                        checked={checked}
                        onCheckedChange={(value) => store.toggleTest(category, value === true)}
                        aria-label={category}
                      />
                      <span className="truncate" title={category}>{category}</span>
                    </label>
                    {isCustom && (
                      <button
                        type="button"
                        onClick={() => store.removeCustomTest(category)}
                        className="flex size-4 shrink-0 items-center justify-center rounded text-muted-foreground opacity-0 hover:text-destructive focus-visible:opacity-100 group-hover:opacity-100"
                        aria-label={`移除 ${category}`}
                      >
                        <X className="size-2.5" />
                      </button>
                    )}
                  </div>
                );
              })}
            </div>

            {addingCategory ? (
              <form
                className="flex gap-1.5 pt-1"
                onSubmit={(event) => {
                  event.preventDefault();
                  const form = event.currentTarget;
                  const input = new FormData(form).get('newCategory');
                  if (typeof input === 'string' && input.trim()) {
                    store.addCustomTest(input.trim());
                    form.reset();
                    setAddingCategory(false);
                  }
                }}
              >
                <Input name="newCategory" autoFocus aria-label="新增站别" placeholder="站别名称" className="h-7 min-w-0 bg-card text-xs" />
                <Button type="submit" size="sm" variant="secondary" className="shrink-0">添加</Button>
                <Button type="button" size="icon-sm" variant="ghost" onClick={() => setAddingCategory(false)} aria-label="取消新增站别">
                  <X />
                </Button>
              </form>
            ) : (
              <Button
                type="button"
                variant="ghost"
                size="xs"
                className="-ml-2 mt-0.5 text-muted-foreground"
                onClick={() => setAddingCategory(true)}
              >
                <Plus />添加站别
              </Button>
            )}
          </fieldset>
        )}
      </fieldset>

      <div className="shrink-0 space-y-2 pt-4">
        {!canRun && !busy && (
          <p className="flex items-center gap-1.5 text-[10px] leading-4 text-warning" aria-live="polite">
            <AlertCircle className="size-3" />请填写{missingItems.join('、')}后开始提取
          </p>
        )}
        <Button
          type="button"
          className="h-8 w-full disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100"
          disabled={!canRun}
          onClick={onRun}
        >
          {busy ? <Loader2 className="animate-spin" /> : <Play />}
          {busy ? '正在提取' : hasResult ? '重新提取' : '运行提取'}
        </Button>
      </div>
    </aside>
  );
}
