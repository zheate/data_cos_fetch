import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion';
import { useCosFilterStore } from '../stores/cos-filter-store';
import {
  DataSourcePanel,
  Step1Config,
  Step2Config,
  Step3Config,
} from './CosFilterSidebarPanels';

function StepBadge({ text, tone = 'secondary' }: { text: string; tone?: 'secondary' | 'outline' }) {
  return <Badge variant={tone}>{text}</Badge>;
}

export function CosFilterSidebar() {
  const state = useCosFilterStore();

  return (
    <Card className="h-fit shadow-sm border-border/60 bg-card/80 backdrop-blur-sm">
      <CardHeader className="border-b">
        <CardTitle>COS 物料筛选</CardTitle>
      </CardHeader>

      <CardContent className="pt-4">
        <Accordion
          type="single"
          collapsible
          value={String(state.activeStep)}
          onValueChange={(value) => state.setActiveStep(value ? Number(value) : 0)}
          className="flex flex-col gap-3"
        >
          <AccordionItem value="0" className="rounded-xl border px-4">
            <AccordionTrigger className="py-4 hover:no-underline">
              <div className="flex flex-1 items-start justify-between gap-3">
                <div className="text-left">
                  <p className="font-medium">原始数据源</p>
                </div>
                <StepBadge text={state.loadedCosCount > 0 ? `${state.loadedCosCount} 条` : `${state.batchFiles.length} 文件`} />
              </div>
            </AccordionTrigger>
            <AccordionContent>
              <DataSourcePanel />
            </AccordionContent>
          </AccordionItem>

          <AccordionItem value="1" className="rounded-xl border px-4">
            <AccordionTrigger className="py-4 hover:no-underline">
              <div className="flex flex-1 items-start justify-between gap-3">
                <div className="text-left">
                  <p className="font-medium">1. 基础波长筛选</p>
                </div>
                <StepBadge
                  text={state.step1Rows.length > 0 ? `${state.step1Rows.length} 条` : state.loadedCosCount > 0 ? '待执行' : '等待加载'}
                  tone={state.step1Rows.length > 0 ? 'secondary' : 'outline'}
                />
              </div>
            </AccordionTrigger>
            <AccordionContent>
              <Step1Config />
            </AccordionContent>
          </AccordionItem>

          <AccordionItem value="2" className="rounded-xl border px-4">
            <AccordionTrigger className="py-4 hover:no-underline">
              <div className="flex flex-1 items-start justify-between gap-3">
                <div className="text-left">
                  <p className="font-medium">2. 二次筛选与选盒</p>
                </div>
                <StepBadge
                  text={state.step2Rows.length > 0 ? `${state.step2Rows.length} 条` : state.step1Rows.length > 0 ? '待执行' : '等待上一步'}
                  tone={state.step2Rows.length > 0 ? 'secondary' : 'outline'}
                />
              </div>
            </AccordionTrigger>
            <AccordionContent>
              <Step2Config />
            </AccordionContent>
          </AccordionItem>

          <AccordionItem value="3" className="rounded-xl border px-4">
            <AccordionTrigger className="py-4 hover:no-underline">
              <div className="flex flex-1 items-start justify-between gap-3">
                <div className="text-left">
                  <p className="font-medium">3. 参数配置与成组</p>
                </div>
                <StepBadge
                  text={state.groupResult ? `${state.groupResult.group_count} 组` : state.step2Rows.length > 0 ? '待执行' : '等待上一步'}
                  tone={state.groupResult ? 'secondary' : 'outline'}
                />
              </div>
            </AccordionTrigger>
            <AccordionContent>
              <Step3Config />
            </AccordionContent>
          </AccordionItem>
        </Accordion>
      </CardContent>
    </Card>
  );
}
