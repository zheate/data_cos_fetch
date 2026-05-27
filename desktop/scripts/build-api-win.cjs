const fs = require('node:fs');
const path = require('node:path');
const { spawnSync } = require('node:child_process');

const arch = process.argv[2];
const targetByArch = {
  arm64: 'aarch64-pc-windows-msvc',
  x64: 'x86_64-pc-windows-msvc',
};

const target = targetByArch[arch];

if (!target) {
  console.error(`[build-api-win] unsupported arch: ${arch}`);
  process.exit(1);
}

const desktopRoot = path.resolve(__dirname, '..');
const rustRoot = path.resolve(desktopRoot, '..', 'rust');
const manifestPath = path.join(rustRoot, 'Cargo.toml');
const releaseDir = path.join(rustRoot, 'target', 'release');
const releaseExe = path.join(releaseDir, 'data-cos-api.exe');
const isNativeWindowsBuild =
  process.platform === 'win32' &&
  ((arch === 'x64' && process.arch === 'x64') || (arch === 'arm64' && process.arch === 'arm64'));

const cargoArgs = isNativeWindowsBuild
  ? ['build', '--manifest-path', manifestPath, '-p', 'data-cos-api', '--release']
  : ['xwin', 'build', '--manifest-path', manifestPath, '-p', 'data-cos-api', '--target', target, '--release'];

const result = spawnSync('cargo', cargoArgs, {
  cwd: rustRoot,
  stdio: 'inherit',
  shell: process.platform === 'win32',
});

if (result.status !== 0) {
  process.exit(result.status ?? 1);
}

const sourceExe = isNativeWindowsBuild
  ? releaseExe
  : path.join(rustRoot, 'target', target, 'release', 'data-cos-api.exe');

if (sourceExe === releaseExe) {
  console.log(`[build-api-win] native build ready at ${releaseExe}`);
  process.exit(0);
}

fs.mkdirSync(releaseDir, { recursive: true });
fs.copyFileSync(sourceExe, releaseExe);

console.log(`[build-api-win] copied ${sourceExe} -> ${releaseExe}`);
