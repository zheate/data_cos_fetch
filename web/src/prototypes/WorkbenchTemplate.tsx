import { useEffect, useMemo, useState } from 'react';
import { Activity, ArrowDownToLine, ArrowLeft, ArrowRight, Check, Database, FileSpreadsheet, Filter, PanelLeftClose, PanelLeftOpen, Play, Search, SlidersHorizontal, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import './workbench-template.css';
import './linear-light-template.css';

// Throwaway UI template. All records are synthetic and state is memory-only.
// Design method: unslop by Damir Sokolovsky (@heydamir), CC BY 4.0.
const records = Array.from({ length: 48 }, (_, i) => ({
  id: `COS-26${String(110401 + i).padStart(6, '0')}`,
  box: `BX-${String(1 + Math.floor(i / 12)).padStart(3, '0')}`,
  wavelength: 970.3 + ((i * 17) % 61) / 10,
  power: 8.1 + ((i * 7) % 24) / 10,
  voltage: 1.65 + ((i * 3) % 12) / 100,
  group: i < 40 ? `G-${String(1 + Math.floor(i / 8)).padStart(2, '0')}` : '未入组',
}));
type PreviewState = 'ready' | 'empty' | 'error';
const categories = ['耦合测试', 'Pre测试', '低温储存后测试', 'Post测试', '封盖测试', '高温测试'];
const steps = ['原始数据源', '基础波长筛选', '二次筛选与选盒', '参数配置与成组', '电性能提取'];

export default function WorkbenchTemplate() {
  const [mode, setMode] = useState<'cos' | 'fetch'>('fetch');
  const [step, setStep] = useState(3);
  const [configOpen, setConfigOpen] = useState(true);
  const [tab, setTab] = useState('分组明细');
  const [query, setQuery] = useState('');
  const [preview, setPreview] = useState<PreviewState>('ready');
  const [running, setRunning] = useState(false);
  const [selected, setSelected] = useState<string | null>(null);
  const [message, setMessage] = useState('');
  const [minimum, setMinimum] = useState('970');
  const [maximum, setMaximum] = useState('977');
  const [size, setSize] = useState('8');
  const [path, setPath] = useState('COS_20261104.xlsx');
  const [entries, setEntries] = useState('Module-20261104-01\nModule-20261104-02');
  const [page, setPage] = useState(1);
  const [stations, setStations] = useState(categories);
  const [checks, setChecks] = useState(['LVI', 'lambd']);
  const [appliedRange, setAppliedRange] = useState([970, 977]);
  useEffect(() => {
    if (!running) return;
    const timer = window.setTimeout(() => {
      setRunning(false);
      setAppliedRange([Number(minimum), Number(maximum)]);
      setPreview('ready');
      setMessage('演示运行结束 · 仅更新样板数据');
    }, 1200);
    return () => window.clearTimeout(timer);
  }, [running, minimum, maximum]);
  const visible = useMemo(() => preview === 'ready' ? records.filter(r =>
    r.wavelength >= appliedRange[0] && r.wavelength <= appliedRange[1] &&
    (tab !== '剩余散件' || r.group === '未入组') &&
    `${r.id} ${r.box} ${r.group} Module-20261104-${records.indexOf(r) < 24 ? '01' : '02'} ${categories[Math.floor(records.indexOf(r) / 4) % 6]}`.toLowerCase().includes(query.toLowerCase())) : [], [appliedRange, preview, query, tab]);
  const rows = visible.slice((page - 1) * 12, page * 12);
  const part = records.find(r => r.id === selected);
  const bins = Array.from({ length: 14 }, (_, i) => records.filter(r => r.wavelength >= 970 + i * .5 && r.wavelength < 970.5 + i * .5).length);
  const invalid = !path.trim() || !Number.isFinite(Number(minimum)) || !Number.isFinite(Number(maximum)) || Number(minimum) >= Number(maximum) || Number(size) < 1 || (mode === 'fetch' && !entries.trim());
  function changeMode(next: 'cos' | 'fetch') { setMode(next); setTab('分组明细'); setSelected(null); setPage(1); }
  function run() { setRunning(true); setMessage(''); setPage(1); }
  function exportDemo() {
    const content = mode === 'fetch' ? '\ufeff条目编号,站别,电流(A),功率(W),电压(V),波长(nm)\n' + visible.map(r => { const i = records.indexOf(r); return [`Module-20261104-${i < 24 ? '01' : '02'}`, categories[Math.floor(i / 4) % 6], (12 + i % 4).toFixed(1), r.power.toFixed(2), r.voltage.toFixed(2), r.wavelength.toFixed(2)].join(','); }).join('\n') : '\ufeff器件编号,盒号,波长(nm),功率(W),电压(V),组号\n' + visible.map(r => [r.id, r.box, r.wavelength.toFixed(2), r.power.toFixed(2), r.voltage.toFixed(2), r.group].join(',')).join('\n');
    const url = URL.createObjectURL(new Blob([content], { type: 'text/csv;charset=utf-8' }));
    const a = document.createElement('a'); a.href = url; a.download = mode === 'fetch' ? '数据提取_样板示例数据.csv' : 'COS_样板示例数据.csv'; a.click(); URL.revokeObjectURL(url);
    setMessage('导出演示 CSV · 内容为样板数据');
  }
  return <div className={`wb-template ${mode === 'fetch' ? 'wb-fetch-page' : ''}`}>
    <header className="wb-topbar">
      <a className="wb-brand" href="?template=ui"><Activity /><span>Data CoS Suite</span></a>
      <nav aria-label="主要导航" className="wb-nav">
        <button aria-current={mode === 'fetch' ? 'page' : undefined} onClick={() => changeMode('fetch')}><Database />数据提取</button>
        <button aria-current={mode === 'cos' ? 'page' : undefined} onClick={() => changeMode('cos')}><Filter />COS 筛选</button>
      </nav>
      <div className="wb-topmeta"><span className="wb-prototype-label">样板数据</span><a href="/" title="返回正式界面"><ArrowLeft />正式界面</a></div>
    </header>
    <div className="wb-titlebar">
      <div>{mode === 'fetch' && !configOpen && <Button variant="ghost" size="icon" aria-label="展开配置" title="展开配置" onClick={() => setConfigOpen(true)}><PanelLeftOpen /></Button>}{mode === 'cos' && <div className="wb-eyebrow">工程数据工作台<span>/</span>波长筛选与成组</div>}<h1>{mode === 'cos' ? 'COS 筛选' : '数据提取'}</h1></div>
      <div className="wb-title-actions"><label className="wb-state-label">预览状态<select aria-label="预览状态" value={preview} onChange={e => {setPreview(e.target.value as PreviewState);setPage(1);setSelected(null);}}><option value="ready">结果示例</option><option value="empty">首次使用</option><option value="error">读取失败</option></select></label><Button variant="outline" onClick={exportDemo} disabled={!visible.length || running}><ArrowDownToLine />导出示例</Button></div>
    </div>
    <div className={`wb-workspace ${configOpen ? '' : 'wb-config-hidden'}`}>
      {configOpen && <aside className="wb-config" aria-label="任务配置">
        <div className="wb-panel-title"><span><SlidersHorizontal />任务配置</span><Button variant="ghost" size="icon" title="收起配置" aria-label="收起配置" onClick={() => setConfigOpen(false)}><PanelLeftClose /></Button></div>
        {mode === 'cos' ? <nav className="wb-steps" aria-label="筛选流程">{steps.map((name, i) => <button key={name} aria-current={step === i ? 'step' : undefined} onClick={() => setStep(i)}><span className="wb-step-index">{i < step ? <Check /> : String(i + 1).padStart(2, '0')}</span><span>{name}</span>{step === i && <span className="wb-current-step">配置中</span>}</button>)}</nav> : null}
        {mode === 'fetch' ? <div className="wb-fetch-config-grid">
          <section className="wb-fetch-source" aria-labelledby="fetch-source-title">
            <h2 id="fetch-source-title" className="sr-only">数据来源</h2>
            <div className="wb-labeled-control"><label htmlFor="template-extraction-mode">数据模式</label><Select defaultValue="module"><SelectTrigger id="template-extraction-mode" aria-label="数据模式"><SelectValue /></SelectTrigger><SelectContent className="wb-template-select-menu" position="popper"><SelectItem value="module">模块（Module）</SelectItem><SelectItem value="chip">芯片（Chip）</SelectItem></SelectContent></Select></div>
            <label>模块根目录<Input value={path === 'COS_20261104.xlsx' ? 'Z:/Ldtd/fcp/' : path} onChange={e => setPath(e.target.value)} /></label>
          </section>
          <section className="wb-fetch-entries" aria-labelledby="fetch-entries-title">
            <h2 id="fetch-entries-title">条目清单</h2>
            <label className="wb-entry-label"><Textarea aria-label="条目清单" rows={4} value={entries} onChange={e => setEntries(e.target.value)} /></label>
          </section>
          <section className="wb-fetch-parameters" aria-labelledby="fetch-parameters-title">
            <h2 id="fetch-parameters-title" className="sr-only">提取参数</h2>
            <div className="wb-fetch-test-row">
              <fieldset><legend>测试文件类型</legend><div className="wb-checks">{['LVI', 'Rth', 'lambd'].map(name => <label key={name}><Checkbox checked={checks.includes(name)} onCheckedChange={v => setChecks(v ? [...checks, name] : checks.filter(c => c !== name))} />{name}</label>)}</div></fieldset>
              <label>电流点 / A<Input defaultValue="12~15" placeholder="例如：12~19" /></label>
            </div>
            <fieldset><legend>站别 / 测试类别</legend><div className="wb-stations">{categories.map(name => <label key={name}><Checkbox checked={stations.includes(name)} onCheckedChange={v => setStations(v ? [...stations, name] : stations.filter(c => c !== name))} />{name}</label>)}</div></fieldset>
          </section>
          {invalid && <p className="wb-validation" role="alert">请检查路径与条目</p>}
        </div> : <div className="wb-config-body">
          <div className="wb-config-heading"><h2>{steps[step]}</h2><p>设置当前步骤的参数</p></div>
          {step === 0 && <label>数据路径<Input value={path} onChange={e => setPath(e.target.value)} /></label>}
          {step === 1 && <label>波长字段<select><option>冷波长</option><option>中心波长</option><option>2A波长</option><option>峰值波长</option></select></label>}
          {(step === 1 || step === 3) && <><div className="wb-field-pair"><label>波长下限 / nm<Input type="number" step="0.1" value={minimum} onChange={e => setMinimum(e.target.value)} /></label><label>波长上限 / nm<Input type="number" step="0.1" value={maximum} onChange={e => setMaximum(e.target.value)} /></label></div><label>每组数量<Input type="number" min="1" value={size} onChange={e => setSize(e.target.value)} /></label></>}
          {step === 2 && <><label>ItemNum<select><option>全部 ItemNum</option><option>演示批次 A</option></select></label><label>自动选盒数量<Input type="number" min="1" defaultValue="4" /></label><div className="wb-hint">可以按盒号复核候选器件</div></>}
          {step === 3 && <><label>成组算法<select><option>最优成组</option><option>贪心成组</option><option>平顶成组</option><option>黄蒙算法</option></select></label><label>组内最大极差 / nm<Input type="number" min="0" step="0.1" defaultValue="2.0" /></label><div className="wb-hint">样板仅演示布局与交互。运行演示只应用波长区间。</div></>}
          {step === 4 && <><fieldset><legend>测试文件类型</legend><div className="wb-checks">{['LVI', 'Rth', 'lambd'].map(name => <label key={name}><Checkbox checked={checks.includes(name)} onCheckedChange={v => setChecks(v ? [...checks, name] : checks.filter(c => c !== name))} />{name}</label>)}</div></fieldset><label>电流点 / A<Input defaultValue="12~15" placeholder="例如：12~19" /></label></>}
          {invalid && <p className="wb-validation" role="alert">请检查路径、条目与数值范围</p>}
        </div>}
        <div className="wb-config-footer">{mode === 'cos' && <div><span className="wb-dot" />{running ? '演示运行中' : '仅在样板内运行'}</div>}<Button disabled={invalid} onClick={() => running ? (setRunning(false), setMessage('演示已取消')) : run()}>{running ? <X /> : <Play />}{running ? '取消运行' : '运行演示'}</Button></div>
      </aside>}
      <main className="wb-results">
        <div className="wb-context">{!configOpen && mode === 'cos' && <Button variant="ghost" size="icon" aria-label="展开配置" title="展开配置" onClick={() => setConfigOpen(true)}><PanelLeftOpen /></Button>}<FileSpreadsheet /><span>{mode === 'cos' ? path : '模块测试数据 / 2026-11-04'}</span><span className="wb-context-meta">示例批次 · 本地预览</span></div>
        {mode === 'cos' && <section className="wb-metrics" aria-label="结果摘要">{([['示例器件', preview === 'ready' ? '48' : '—', '条'], ['生成组数', preview === 'ready' ? '5' : '—', '组'], ['入组器件', preview === 'ready' ? '40' : '—', '条'], ['入组率', preview === 'ready' ? '83.3' : '—', '%']]).map(([name, value, unit]) => <div key={name}><span>{name}</span><p>{value}<small>{unit}</small></p></div>)}</section>}
        {mode === 'cos' && preview === 'ready' && <section className="wb-distribution" aria-label="示例波长分布"><div className="wb-chart-heading"><h2>冷波长分布</h2><span><i />全部示例器件 · 48 条</span><span className="wb-chart-range">970–977 nm</span></div><div className="wb-chart"><div className="wb-chart-scale"><span>6 条</span><span>3</span><span>0</span></div><div className="wb-bars">{bins.map((count, i) => <div key={i} className={i >= 4 && i <= 9 ? 'wb-bar-accent' : ''} style={{height: `${count / 6 * 100}%`}} title={`${(970 + i * .5).toFixed(1)}–${(970.5 + i * .5).toFixed(1)} nm：${count} 条`} />)}</div></div><div className="wb-axis"><span>970</span><span>972</span><span>974</span><span>977 nm</span></div></section>}
        {mode === 'fetch' && <div className="wb-extraction-context"><div><span className="wb-dot" />模块解析示例</div><span>LVI / lambd</span><span>12–15 A</span><span>完整单位随表头显示</span></div>}
        <section className="wb-table-panel" aria-label="数据结果">
          <div className="wb-results-bar"><div className="wb-tabs" role="tablist" aria-label="结果视图">{(mode === 'cos' ? ['分组明细', '剩余散件', '电性能'] : ['解析明细', '异常记录']).map(name => <button role="tab" onKeyDown={e => { if (e.key === 'ArrowRight' || e.key === 'ArrowLeft') { const buttons = Array.from(e.currentTarget.parentElement!.querySelectorAll<HTMLButtonElement>('button')); const next = buttons[(buttons.indexOf(e.currentTarget) + (e.key === 'ArrowRight' ? 1 : buttons.length - 1)) % buttons.length]; next.focus(); next.click(); e.preventDefault(); } }} aria-selected={tab === name || (name === '解析明细' && tab === '分组明细')} key={name} onClick={() => {setTab(name);setQuery('');setPage(1);setSelected(null);}}>{name}{name === '解析明细' && <span className="wb-tab-count">{visible.length}</span>}{name === '剩余散件' && <span>8</span>}</button>)}</div>
          <div className="wb-table-tools">{mode === 'cos' && <span>{tab === '电性能' ? '电性能实测示例' : tab === '异常记录' ? '示例中无异常记录' : '器件明细'}<small>{tab === '异常记录' ? 0 : visible.length} 条</small></span>}<label className="wb-search"><Search /><Input aria-label={mode === 'fetch' ? '搜索条目或站别' : '搜索器件、盒号或组号'} placeholder={mode === 'fetch' ? '搜索条目 / 站别' : '搜索器件 / 盒号 / 组号'} value={query} onChange={e => {setQuery(e.target.value);setPage(1);}} /></label></div></div>
          {running ? <div className="wb-empty" role="status"><Activity /><h3>正在运行演示</h3><p>可以随时取消，配置保留在当前页面</p><Button variant="outline" onClick={() => {setRunning(false);setMessage('演示已取消');}}>取消运行</Button></div> : preview !== 'ready' || !visible.length || tab === '异常记录' ? <div className="wb-empty" role={preview === 'error' ? 'alert' : 'status'}><Database /><h3>{preview === 'error' ? '批次文件读取失败' : preview === 'empty' ? (mode === 'fetch' ? '从一组测试条目开始' : '从一份批次数据开始') : tab === '异常记录' ? '没有异常记录' : '没有匹配的器件'}</h3><p>{preview === 'error' ? '演示错误：文件路径不可访问。请检查数据路径。' : preview === 'empty' ? '在任务配置中选择数据源，再运行演示' : '当前结果为样板示例数据'}</p><Button variant="outline" onClick={() => {if (query) {setQuery('');setPage(1);} else {setConfigOpen(true);setStep(0);}}}>{query ? '清除搜索' : '查看数据配置'}</Button></div> : <><div className="wb-table-scroll">{mode === 'fetch' ? <Table><TableHeader><TableRow><TableHead>条目编号</TableHead><TableHead>站别</TableHead><TableHead>电流 <span>A</span></TableHead><TableHead>功率 <span>W</span></TableHead><TableHead>电压 <span>V</span></TableHead><TableHead>波长 <span>nm</span></TableHead></TableRow></TableHeader><TableBody>{rows.map(r => { const i = records.indexOf(r); return <TableRow key={r.id}><TableCell>Module-20261104-{i < 24 ? '01' : '02'}</TableCell><TableCell>{categories[Math.floor(i / 4) % 6]}</TableCell><TableCell>{(12 + i % 4).toFixed(1)}</TableCell><TableCell>{r.power.toFixed(2)}</TableCell><TableCell>{r.voltage.toFixed(2)}</TableCell><TableCell>{r.wavelength.toFixed(2)}</TableCell></TableRow>; })}</TableBody></Table> : <table><thead><tr><th>器件编号</th><th>盒号</th><th>冷波长 <span>nm</span></th>{tab === '电性能' ? <><th>功率 <span>W</span></th><th>电压 <span>V</span></th></> : <th>组号</th>}<th className="wb-action-cell">详情</th></tr></thead><tbody>{rows.map(r => <tr key={r.id} className={selected === r.id ? 'wb-selected' : ''}><td>{r.id}</td><td>{r.box}</td><td>{r.wavelength.toFixed(2)}</td>{tab === '电性能' ? <><td>{r.power.toFixed(2)}</td><td>{r.voltage.toFixed(2)}</td></> : <td><span className={r.group === '未入组' ? 'wb-remaining' : 'wb-group'}>{r.group}</span></td>}<td className="wb-action-cell"><Button variant="ghost" size="icon" aria-label={`查看 ${r.id}`} title="查看器件详情" onClick={() => setSelected(r.id)}><ArrowRight /></Button></td></tr>)}</tbody></table>}</div><div className="wb-pager"><span>第 {(page - 1) * 12 + 1}–{Math.min(page * 12, visible.length)} 条，共 {visible.length} 条</span><div><Button variant="ghost" size="icon" aria-label="上一页" disabled={page === 1} onClick={() => setPage(page - 1)}><ArrowLeft /></Button><span>{page} / {Math.max(1, Math.ceil(visible.length / 12))}</span><Button variant="ghost" size="icon" aria-label="下一页" disabled={page * 12 >= visible.length} onClick={() => setPage(page + 1)}><ArrowRight /></Button></div></div></>}
        </section>
        {(message || mode === 'cos') && <div className="wb-status" role="status">{message || '样板数据仅用于检查布局，不连接数据后端'}</div>}
      </main>
      {part && <aside className="wb-inspector" aria-label="器件详情"><div className="wb-panel-title"><span>器件详情</span><Button variant="ghost" size="icon" title="关闭详情" aria-label="关闭详情" onClick={() => setSelected(null)}><X /></Button></div><div className="wb-inspector-body"><span className="wb-eyebrow">示例器件</span><h2>{part.id}</h2><dl>{[['盒号', part.box], ['组号', part.group], ['冷波长', `${part.wavelength.toFixed(2)} nm`], ['功率', `${part.power.toFixed(2)} W`], ['电压', `${part.voltage.toFixed(2)} V`]].map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{value}</dd></div>)}</dl><p className="wb-hint">正式接入时，此处复用器件检查器。</p></div></aside>}
    </div>
    {mode === 'cos' && <footer className="wb-credit">布局样板 / 2026.11 <span>Design method: unslop by Damir Sokolovsky (@heydamir), CC BY 4.0</span></footer>}
  </div>;
}
