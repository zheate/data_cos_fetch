import type { CosRow } from '../helpers/types';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { Package, Layers, MapPin, User, Cpu } from 'lucide-react';

export function PartInspector({ part }: { part: CosRow | null }) {
  if (!part) {
    return (
      <div className="flex h-full min-h-[320px] items-center justify-center text-center p-6 border border-dashed rounded-lg bg-muted/5">
        <p className="text-xs text-muted-foreground">在列表中选择器件以查看详细参数</p>
      </div>
    );
  }

  const isIsolated = part.isolation === '是';

  return (
    <Card className="h-full border bg-card shadow-sm rounded text-xs select-none">
      <CardHeader className="border-b bg-muted/20 py-2.5 px-4 flex flex-row items-center justify-between space-y-0">
        <CardTitle className="text-xs font-semibold text-foreground flex items-center gap-1.5">
          <Cpu className="h-3.5 w-3.5 text-primary" />
          器件属性详情
        </CardTitle>
        <Badge
          variant={isIsolated ? 'destructive' : 'secondary'}
          className="rounded-xs font-medium text-[10px]"
        >
          {isIsolated ? '已隔离' : '正常'}
        </Badge>
      </CardHeader>

      <CardContent className="p-4 flex flex-col gap-4 overflow-y-auto max-h-[700px]">
        {/* Device ID Header */}
        <div className="rounded-md border bg-muted/15 p-2.5">
          <span className="text-[10px] uppercase tracking-wider text-muted-foreground">器件编号</span>
          <p className="font-mono text-sm font-bold text-foreground mt-0.5 break-all select-all">
            {part.device_id}
          </p>
        </div>

        {/* Part properties grid */}
        <div>
          <h4 className="font-semibold text-muted-foreground mb-2 text-[10px] uppercase tracking-wider">物料信息</h4>
          <div className="grid grid-cols-2 gap-x-3 gap-y-2.5 rounded-md border p-3 bg-muted/5 text-xs">
            <div className="flex flex-col gap-0.5">
              <span className="text-[10px] text-muted-foreground flex items-center gap-1">
                <Package className="size-3" /> 批次 (ItemNum)
              </span>
              <span className="font-mono text-foreground font-medium truncate">{part.item_num ?? '-'}</span>
            </div>
            <div className="flex flex-col gap-0.5">
              <span className="text-[10px] text-muted-foreground flex items-center gap-1">
                <Layers className="size-3" /> 盒号
              </span>
              <span className="font-mono text-foreground font-medium truncate">{part.box_num ?? '-'}</span>
            </div>
            <div className="flex flex-col gap-0.5">
              <span className="text-[10px] text-muted-foreground flex items-center gap-1">
                <MapPin className="size-3" /> 仓库
              </span>
              <span className="text-foreground font-medium truncate">{part.warehouse ?? '-'}</span>
            </div>
            <div className="flex flex-col gap-0.5">
              <span className="text-[10px] text-muted-foreground flex items-center gap-1">
                <User className="size-3" /> 货主
              </span>
              <span className="text-foreground font-medium truncate">{part.owner ?? '-'}</span>
            </div>
          </div>
        </div>

        <Separator />

        {/* Wavelength parameters */}
        <div>
          <h4 className="font-semibold text-muted-foreground mb-2 text-[10px] uppercase tracking-wider">实测波长指标</h4>
          <div className="grid grid-cols-2 gap-2 text-xs">
            <div className="flex flex-col gap-0.5 p-2 rounded-md border bg-card">
              <span className="text-[10px] text-muted-foreground">2A 波长</span>
              <span className="font-mono font-semibold text-foreground">
                {part.two_a_wavelength_nm ? `${part.two_a_wavelength_nm.toFixed(3)} nm` : '-'}
              </span>
            </div>
            <div className="flex flex-col gap-0.5 p-2 rounded-md border bg-card">
              <span className="text-[10px] text-muted-foreground">中心波长</span>
              <span className="font-mono font-semibold text-foreground">
                {part.center_wavelength_nm ? `${part.center_wavelength_nm.toFixed(3)} nm` : '-'}
              </span>
            </div>
            <div className="flex flex-col gap-0.5 p-2 rounded-md border bg-card">
              <span className="text-[10px] text-muted-foreground">冷波长</span>
              <span className="font-mono font-semibold text-foreground">
                {part.cold_wavelength_nm ? `${part.cold_wavelength_nm.toFixed(3)} nm` : '-'}
              </span>
            </div>
            <div className="flex flex-col gap-0.5 p-2 rounded-md border bg-card">
              <span className="text-[10px] text-muted-foreground">峰值波长</span>
              <span className="font-mono font-semibold text-foreground">
                {part.peak_wavelength_nm ? `${part.peak_wavelength_nm.toFixed(3)} nm` : '-'}
              </span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
