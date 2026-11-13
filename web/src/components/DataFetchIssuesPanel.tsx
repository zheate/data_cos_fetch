import { AlertCircle, CheckCircle2, Info } from 'lucide-react';

const ERROR_CLASSIFICATIONS = [
  {
    label: '访问权限',
    matches: ['permission denied', 'access denied', '拒绝访问', '权限不足', 'os error 5'],
    guidance: '后端进程没有读取权限。请检查加密软件白名单及共享目录权限。',
  },
  {
    label: '文件解密或格式',
    matches: ['failed to open workbook', 'excel parser', 'content probe', 'zip', 'ole', 'calamine'],
    guidance: '文件可被定位，但无法按 Excel 内容解析。请确认后端能够读取解密后的文件内容。',
  },
  {
    label: '路径或盘符',
    matches: ['not found', 'cannot find', '找不到', 'failed to read directory', 'os error 2', 'os error 3'],
    guidance: '后端看不到该路径。请检查映射盘、UNC 路径以及后端运行权限。',
  },
] as const;

function classifyExtractionError(message: string) {
  const normalized = message.toLowerCase();
  return ERROR_CLASSIFICATIONS.find((item) => item.matches.some((token) => normalized.includes(token))) ?? {
    label: '数据读取',
    guidance: '保留完整错误信息，并结合运行日志进一步定位。',
  };
}

export function DataFetchIssuesPanel({ errors, infos }: { errors: string[]; infos: string[] }) {
  if (errors.length === 0 && infos.length === 0) {
    return (
      <div className="flex min-h-72 flex-col items-center justify-center px-6 py-12 text-center" role="status">
        <span className="mb-4 flex size-11 items-center justify-center rounded-full bg-success/10 text-success">
          <CheckCircle2 className="size-5" />
        </span>
        <h3 className="text-sm font-semibold">没有异常或查找提示</h3>
        <p className="mt-1.5 max-w-sm text-xs leading-5 text-muted-foreground">本次提取未报告错误。若记录数量不符合预期，可检查左侧条目与站别筛选。</p>
      </div>
    );
  }

  return (
    <div className="max-h-[620px] space-y-3 overflow-y-auto p-4 sm:p-5">
      <div className="flex flex-col gap-1 border-b pb-4 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
        <p>按错误类型查看原因与处理建议，原始信息已保留供排查。</p>
        <p className="shrink-0">{errors.length} 项错误 · {infos.length} 项提示</p>
      </div>
      {errors.map((error, index) => {
        const classification = classifyExtractionError(error);
        return (
          <article key={`error-${index}`} className="rounded-lg border border-destructive/20 bg-destructive/[0.035] p-4">
            <h3 className="flex items-center gap-2 text-xs font-semibold text-destructive">
              <AlertCircle className="size-4" />{classification.label}
              <span className="font-normal text-muted-foreground">错误 {index + 1}</span>
            </h3>
            <p className="mt-3 whitespace-pre-wrap break-words font-mono text-xs leading-5 text-foreground">{error}</p>
            <p className="mt-3 border-t border-destructive/10 pt-3 text-xs leading-5 text-muted-foreground">{classification.guidance}</p>
          </article>
        );
      })}
      {infos.map((info, index) => (
        <article key={`info-${index}`} className="rounded-lg border border-warning/25 bg-warning/[0.045] p-4">
          <h3 className="flex items-center gap-2 text-xs font-semibold text-amber-700 dark:text-amber-400">
            <Info className="size-4" />查找提示
            <span className="font-normal text-muted-foreground">提示 {index + 1}</span>
          </h3>
          <p className="mt-3 whitespace-pre-wrap break-words font-mono text-xs leading-5 text-foreground">{info}</p>
        </article>
      ))}
    </div>
  );
}
