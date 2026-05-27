import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from 'recharts';
import type { HistogramBin } from '../helpers/types';
import { toFixed } from '../helpers/utils';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

type WavelengthHistogramProps = {
  title: string;
  unit: string;
  bins: HistogramBin[];
  valueCount: number;
};

export function WavelengthHistogram({ title, unit, bins, valueCount }: WavelengthHistogramProps) {
  if (valueCount === 0 || bins.length === 0) {
    return (
      <Card className="shadow-none">
        <CardHeader>
          <CardTitle className="text-sm">{title}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">暂无数据</p>
        </CardContent>
      </Card>
    );
  }

  const data = bins.map((bin) => ({
    range: `${toFixed(bin.start, 2)}~${toFixed(bin.end, 2)}`,
    count: bin.count,
  }));

  return (
    <Card className="shadow-none">
      <CardHeader>
        <CardTitle className="text-sm">{title}</CardTitle>
        <CardDescription>
          样本数：{valueCount} · 单位：{unit}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={220}>
          <BarChart data={data} margin={{ top: 4, right: 8, bottom: 4, left: 0 }}>
            <CartesianGrid strokeDasharray="3 3" opacity={0.15} />
            <XAxis
              dataKey="range"
              tick={{ fontSize: 10 }}
              interval="preserveStartEnd"
              angle={-30}
              textAnchor="end"
              height={50}
            />
            <YAxis tick={{ fontSize: 11 }} width={36} />
            <Tooltip
              contentStyle={{
                borderRadius: 8,
                fontSize: 12,
                backgroundColor: 'var(--card)',
                border: '1px solid var(--border)',
              }}
            />
            <Bar dataKey="count" radius={[4, 4, 0, 0]}>
              {data.map((_, index) => (
                <Cell
                  key={index}
                  fill={['var(--chart-1)', 'var(--chart-2)', 'var(--chart-3)', 'var(--chart-4)'][index % 4]}
                />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}
