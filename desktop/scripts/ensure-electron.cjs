const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');

const distPath = path.resolve(__dirname, '..', 'node_modules', 'electron', 'dist');
const installScript = path.resolve(__dirname, '..', 'node_modules', 'electron', 'install.js');

if (fs.existsSync(distPath)) {
  console.log('[ensure-electron] electron binary is ready');
  process.exit(0);
}

if (!fs.existsSync(installScript)) {
  console.error('[ensure-electron] electron install script not found, run: npm install');
  process.exit(1);
}

console.log('[ensure-electron] electron binary missing, downloading...');
execFileSync(process.execPath, [installScript], { stdio: 'inherit' });
console.log('[ensure-electron] electron binary installed');
