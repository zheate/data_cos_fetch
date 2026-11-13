import { AlertCircle, AlertTriangle, CheckCircle2, FolderOpen, ShieldCheck, X } from 'lucide-react';
import { Dialog } from 'radix-ui';
import type {
  ExtractionDiagnosticStatus,
  ExtractionDiagnosticsResponse,
} from '../helpers/types';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

function diagnosticStatusMeta(status: ExtractionDiagnosticStatus) {
  if (status === 'pass') {
    return { label: '通过', className: 'text-success', icon: CheckCircle2 };
  }
  if (status === 'warning') {
    return { label: '警告', className: 'text-amber-700 dark:text-amber-400', icon: AlertTriangle };
  }
  return { label: '失败', className: 'text-destructive', icon: AlertCircle };
}

type EnvironmentDiagnosticsDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  diagnostics: ExtractionDiagnosticsResponse | null;
  logPath: string;
  onOpenLogs: () => void;
};

export function EnvironmentDiagnosticsDialog({
  open,
  onOpenChange,
  diagnostics,
  logPath,
  onOpenLogs,
}: EnvironmentDiagnosticsDialogProps) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/45 backdrop-blur-[2px] data-[state=closed]:animate-out data-[state=open]:animate-in data-[state=closed]:fade-out data-[state=open]:fade-in" />
        <Dialog.Content className="fixed left-1/2 top-1/2 z-50 flex max-h-[88dvh] w-[calc(100%-2rem)] max-w-3xl -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-xl border bg-background shadow-2xl focus:outline-none data-[state=closed]:animate-out data-[state=open]:animate-in data-[state=closed]:fade-out data-[state=open]:fade-in">
          <div className="flex items-start justify-between gap-4 border-b px-5 py-4">
            <div>
              <Dialog.Title className="flex items-center gap-2 text-base font-semibold">
                <ShieldCheck className="size-5 text-primary" />读取环境检测
              </Dialog.Title>
              <Dialog.Description className="mt-1 text-xs text-muted-foreground">
                {diagnostics
                  ? `后端进程 ${diagnostics.process_id} · ${diagnostics.failed} 项失败 · ${diagnostics.warnings} 项警告`
                  : '检查数据目录、读取权限和运行环境'}
              </Dialog.Description>
            </div>
            <Dialog.Close asChild>
              <Button type="button" variant="ghost" size="icon-sm" aria-label="关闭环境检测"><X /></Button>
            </Dialog.Close>
          </div>

          <div className="min-h-0 space-y-3 overflow-y-auto p-5">
            {diagnostics?.checks.map((check, index) => {
              const meta = diagnosticStatusMeta(check.status);
              const StatusIcon = meta.icon;
              return (
                <article key={`${check.code}-${check.path ?? index}`} className="rounded-lg border bg-card p-4">
                  <div className="flex items-center justify-between gap-3">
                    <h3 className={`flex items-center gap-2 text-sm font-semibold ${meta.className}`}>
                      <StatusIcon className="size-4" />{check.label}
                    </h3>
                    <Badge variant="outline" className={meta.className}>{meta.label}</Badge>
                  </div>
                  {check.path && <p className="mt-2 break-all font-mono text-xs leading-5 text-muted-foreground">{check.path}</p>}
                  <p className="mt-2 text-xs leading-5 text-foreground">{check.detail}</p>
                  {check.os_error_code !== null && (
                    <p className="mt-1 font-mono text-[11px] text-muted-foreground">系统错误码：{check.os_error_code}</p>
                  )}
                </article>
              );
            })}

            {diagnostics && (
              <dl className="space-y-3 rounded-lg bg-muted/35 p-4 text-xs">
                <div>
                  <dt className="font-semibold">供管理员识别的后端程序</dt>
                  <dd className="mt-1.5 break-all font-mono leading-5 text-muted-foreground">{diagnostics.executable_path}</dd>
                </div>
                <div>
                  <dt className="font-semibold">缓存目录</dt>
                  <dd className="mt-1.5 break-all font-mono leading-5 text-muted-foreground">{diagnostics.cache_directory}</dd>
                </div>
                {logPath && (
                  <div>
                    <dt className="font-semibold">运行日志</dt>
                    <dd className="mt-1.5 break-all font-mono leading-5 text-muted-foreground">{logPath}</dd>
                  </div>
                )}
              </dl>
            )}
          </div>

          <div className="flex shrink-0 justify-end gap-2 border-t px-5 py-4">
            {window.desktopRuntime?.openLogsFolder && (
              <Button type="button" variant="outline" onClick={onOpenLogs}>
                <FolderOpen />打开日志目录
              </Button>
            )}
            <Dialog.Close asChild><Button type="button">完成</Button></Dialog.Close>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
