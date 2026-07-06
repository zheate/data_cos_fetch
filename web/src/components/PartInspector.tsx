import type { CosRow } from '../helpers/types';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { WAREHOUSE_AVAILABLE, WAREHOUSE_NEED_CONFIRM } from '../helpers/utils';
import { Package, GitCommit, Layers, User, MapPin, Tag, Cpu, ChevronRight, CornerDownRight } from 'lucide-react';
import { cn } from '@/lib/utils';

export function PartInspector({ part }: { part: CosRow | null }) {
  if (!part) {
    return (
      <div className="flex h-full items-center justify-center text-center p-6 border border-dashed rounded bg-muted/5">
        <p className="text-xs text-muted-foreground">请在左侧列表中选择一个器件查看物料 BOM 属性及生命周期</p>
      </div>
    );
  }

  // Calculate lifecycle state
  let stage: 'draft' | 'review' | 'released' = 'draft';
  if (part.isolation === '是' || (part.warehouse && WAREHOUSE_NEED_CONFIRM.includes(part.warehouse))) {
    stage = 'review';
  } else if (part.warehouse && WAREHOUSE_AVAILABLE.includes(part.warehouse)) {
    stage = 'released';
  }

  const isIsolated = part.isolation === '是';

  return (
    <Card className="h-full border bg-card shadow-sm rounded text-xs select-none">
      <CardHeader className="border-b bg-muted/20 py-2.5 px-4 flex flex-row items-center justify-between space-y-0">
        <CardTitle className="text-xs font-bold text-foreground flex items-center gap-1.5">
          <Cpu className="h-3.5 w-3.5 text-primary" />
          物料检查器
        </CardTitle>
        <Badge
          variant={stage === 'released' || stage === 'review' ? 'outline' : 'secondary'}
          className={cn(
            "rounded-sm font-semibold scale-90",
            stage === 'released' && "border-success/40 bg-success/10 text-success",
            stage === 'review' && "border-warning/40 bg-warning/10 text-warning"
          )}
        >
          {stage === 'released' ? 'Released' : stage === 'review' ? 'In Review' : 'Draft'}
        </Badge>
      </CardHeader>
      <CardContent className="p-4 flex flex-col gap-4 overflow-y-auto max-h-[700px]">
        {/* Lifecycle Flow */}
        <div>
          <h4 className="font-semibold text-muted-foreground mb-2 text-[10px] uppercase tracking-wider">生命周期阶段</h4>
          <div className="grid grid-cols-3 gap-1 text-center relative mt-1.5">
            <div className={`p-1.5 border rounded-sm flex flex-col items-center gap-0.5 ${stage === 'draft' ? 'border-primary/50 bg-primary/5 text-primary font-bold' : 'bg-muted/40 text-muted-foreground/60'}`}>
              <span className="text-[10px]">Draft</span>
              <span className="text-[9px] font-normal">入库草稿</span>
            </div>
            <div className={`p-1.5 border rounded-sm flex flex-col items-center gap-0.5 ${stage === 'review' ? 'border-warning/50 bg-warning/5 text-warning font-bold' : stage === 'released' ? 'bg-muted/30 text-foreground/80' : 'bg-muted/40 text-muted-foreground/60'}`}>
              <span className="text-[10px]">In Review</span>
              <span className="text-[9px] font-normal">审查隔离</span>
            </div>
            <div className={`p-1.5 border rounded-sm flex flex-col items-center gap-0.5 ${stage === 'released' ? 'border-success/40 bg-success/10 text-success font-bold' : 'bg-muted/40 text-muted-foreground/60'}`}>
              <span className="text-[10px]">Released</span>
              <span className="text-[9px] font-normal">发布可用</span>
            </div>
          </div>
        </div>

        <Separator />

        {/* Part properties grid */}
        <div>
          <h4 className="font-semibold text-muted-foreground mb-2 text-[10px] uppercase tracking-wider">物料属性</h4>
          <div className="grid grid-cols-2 gap-x-4 gap-y-2.5 rounded border p-3 bg-muted/10">
            <div className="flex flex-col gap-0.5">
              <span className="text-[10px] text-muted-foreground flex items-center gap-1"><Tag className="size-3" /> 器件号</span>
              <span className="font-mono text-foreground font-semibold truncate">{part.device_id}</span>
            </div>
            <div className="flex flex-col gap-0.5">
              <span className="text-[10px] text-muted-foreground flex items-center gap-1"><Package className="size-3" /> 批次号</span>
              <span className="font-mono text-foreground truncate">{part.item_num ?? '-'}</span>
            </div>
            <div className="flex flex-col gap-0.5">
              <span className="text-[10px] text-muted-foreground flex items-center gap-1"><Layers className="size-3" /> 盒号</span>
              <span className="font-mono text-foreground truncate">{part.box_num ?? '-'}</span>
            </div>
            <div className="flex flex-col gap-0.5">
              <span className="text-[10px] text-muted-foreground flex items-center gap-1"><MapPin className="size-3" /> 仓库</span>
              <span className="text-foreground truncate">{part.warehouse ?? '-'}</span>
            </div>
            <div className="flex flex-col gap-0.5">
              <span className="text-[10px] text-muted-foreground flex items-center gap-1"><User className="size-3" /> 货主</span>
              <span className="text-foreground truncate">{part.owner ?? '-'}</span>
            </div>
            <div className="flex flex-col gap-0.5">
              <span className="text-[10px] text-muted-foreground flex items-center gap-1"><GitCommit className="size-3" /> 隔离</span>
              <span className={`font-semibold ${isIsolated ? 'text-destructive' : 'text-success'}`}>{isIsolated ? '已隔离' : '正常'}</span>
            </div>
          </div>
        </div>

        <Separator />

        {/* Wavelength parameters */}
        <div>
          <h4 className="font-semibold text-muted-foreground mb-2 text-[10px] uppercase tracking-wider">技术指标</h4>
          <div className="grid grid-cols-2 gap-2 text-right">
            <div className="flex items-center justify-between p-2 rounded border bg-card">
              <span className="text-muted-foreground">2A 波长</span>
              <span className="font-mono font-semibold text-foreground">{part.two_a_wavelength_nm ? `${part.two_a_wavelength_nm.toFixed(3)} nm` : '-'}</span>
            </div>
            <div className="flex items-center justify-between p-2 rounded border bg-card">
              <span className="text-muted-foreground">中心波长</span>
              <span className="font-mono font-semibold text-foreground">{part.center_wavelength_nm ? `${part.center_wavelength_nm.toFixed(3)} nm` : '-'}</span>
            </div>
            <div className="flex items-center justify-between p-2 rounded border bg-card">
              <span className="text-muted-foreground">冷波长</span>
              <span className="font-mono font-semibold text-foreground">{part.cold_wavelength_nm ? `${part.cold_wavelength_nm.toFixed(3)} nm` : '-'}</span>
            </div>
            <div className="flex items-center justify-between p-2 rounded border bg-card">
              <span className="text-muted-foreground">峰值波长</span>
              <span className="font-mono font-semibold text-foreground">{part.peak_wavelength_nm ? `${part.peak_wavelength_nm.toFixed(3)} nm` : '-'}</span>
            </div>
          </div>
        </div>

        <Separator />

        {/* BOM Hierarchy Tree */}
        <div>
          <h4 className="font-semibold text-muted-foreground mb-2 text-[10px] uppercase tracking-wider">装配 BOM 树</h4>
          <div className="font-mono text-[11px] leading-relaxed p-3 border rounded bg-muted/5 flex flex-col gap-1">
            <div className="flex items-center gap-1 text-muted-foreground">
              <ChevronRight className="size-3 shrink-0" />
              <span>Assembly: WAFER_{part.item_num?.substring(0, 4) || 'LOT'} (Wafer Parent)</span>
            </div>
            <div className="flex items-center gap-1 pl-3 text-muted-foreground">
              <ChevronRight className="size-3 shrink-0" />
              <span>Sub-Lot: {part.item_num || 'ITEM_LOT'}</span>
            </div>
            <div className="flex items-center gap-1 pl-6 text-muted-foreground">
              <ChevronRight className="size-3 shrink-0" />
              <span>Container: BOX_{part.box_num || '001'}</span>
            </div>
            <div className="flex items-center gap-1 pl-9 text-primary font-semibold">
              <CornerDownRight className="size-3 text-primary shrink-0" />
              <span>Part: {part.device_id} (Active P/N)</span>
            </div>
            <div className="flex flex-col gap-0.5 pl-12 border-l ml-[45px] py-1 border-primary/30 text-muted-foreground text-[10px]">
              <span>* Wavelength Mode: {part.center_wavelength_nm ? 'Center-dominant' : 'Raw'}</span>
              <span>* State: {stage.toUpperCase()}</span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
