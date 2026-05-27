const fs = require('node:fs');
const path = require('node:path');
const { spawnSync } = require('node:child_process');

const arch = process.argv[2];
const electronArchByArch = {
  arm64: '--arm64',
  x64: '--x64',
};

const builderArch = electronArchByArch[arch];

if (!builderArch) {
  console.error(`[dist-win] unsupported arch: ${arch}`);
  process.exit(1);
}

const desktopRoot = path.resolve(__dirname, '..');
const packageJson = JSON.parse(fs.readFileSync(path.join(desktopRoot, 'package.json'), 'utf8'));
const outputDir = path.join(desktopRoot, packageJson.build.directories.output);
const setupBaseName = `${packageJson.build.productName} Setup ${packageJson.version}`;
const zipBaseName = `${packageJson.build.productName}-${packageJson.version}-win`;
const npmCmd = process.platform === 'win32' ? 'npm.cmd' : 'npm';
const npxCmd = process.platform === 'win32' ? 'npx.cmd' : 'npx';

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: desktopRoot,
    stdio: 'inherit',
    shell: process.platform === 'win32',
    ...options,
  });

  if (result.error) {
    console.error(`[dist-win] failed to start ${command}: ${result.error.message}`);
    process.exit(1);
  }

  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

function renameIfExists(from, to) {
  if (!fs.existsSync(from)) {
    return;
  }

  if (fs.existsSync(to)) {
    fs.rmSync(to, { force: true });
  }

  fs.renameSync(from, to);
  console.log(`[dist-win] renamed ${path.basename(from)} -> ${path.basename(to)}`);
}

run(npmCmd, ['run', 'build:renderer']);
run(npmCmd, ['run', arch === 'x64' ? 'build:api:win:x64' : 'build:api:win']);
run(
  npxCmd,
  ['electron-builder@26.0.12', '--win', builderArch, '--config.win.signAndEditExecutable=false'],
  {
    env: {
      ...process.env,
      CSC_IDENTITY_AUTO_DISCOVERY: 'false',
    },
  },
);

renameIfExists(
  path.join(outputDir, `${setupBaseName}.exe`),
  path.join(outputDir, `${setupBaseName} ${arch}.exe`),
);
renameIfExists(
  path.join(outputDir, `${setupBaseName}.exe.blockmap`),
  path.join(outputDir, `${setupBaseName} ${arch}.exe.blockmap`),
);
renameIfExists(
  path.join(outputDir, `${zipBaseName}.zip`),
  path.join(outputDir, `${packageJson.build.productName}-${packageJson.version}-${arch}-win.zip`),
);
