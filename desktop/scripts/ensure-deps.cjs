// Preflight for `npm run dev`: node dependencies are installed by the one-off
// `npm run bootstrap` step instead of on every launch, so fail loudly and with
// the exact command to run when they are missing.
const fs = require('node:fs');
const path = require('node:path');

const desktopRoot = path.resolve(__dirname, '..');
const webRoot = path.resolve(desktopRoot, '..', 'web');

const requirements = [
  { label: 'concurrently', target: path.join(desktopRoot, 'node_modules', 'concurrently') },
  { label: 'wait-on', target: path.join(desktopRoot, 'node_modules', 'wait-on') },
  { label: 'cross-env', target: path.join(desktopRoot, 'node_modules', 'cross-env') },
  { label: 'electron', target: path.join(desktopRoot, 'node_modules', 'electron') },
  { label: 'vite (web)', target: path.join(webRoot, 'node_modules', 'vite') },
];

const missing = requirements.filter((item) => !fs.existsSync(item.target));

if (missing.length === 0) {
  process.exit(0);
}

console.error('');
console.error(`[ensure-deps] 缺少依赖: ${missing.map((item) => item.label).join(', ')}`);
console.error('[ensure-deps] 依赖只在 bootstrap 时安装，请先执行一次：');
console.error('[ensure-deps] Dependencies are installed by bootstrap only, run it once first:');
console.error('');
console.error('    npm run bootstrap        # 在仓库根目录，或：cd desktop && npm run bootstrap');
console.error('');
console.error('[ensure-deps] 该命令会安装 desktop/web 依赖、准备 electron 二进制并预热 cargo 依赖缓存。');
console.error('[ensure-deps] It installs desktop/web deps, prepares the electron binary and warms the cargo cache.');
console.error('');
process.exit(1);
