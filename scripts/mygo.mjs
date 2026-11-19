import { spawn, spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { copyFileSync, mkdirSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const native = join(root, 'native');
const command = process.argv[2] || 'start';
const exe = process.platform === 'win32' ? '.exe' : '';
const guiBuildFlags = process.platform === 'win32' ? ['-ldflags=-H=windowsgui'] : [];
const toolchain = join(root, '.tmp', 'toolchains');
const uiOutput = process.env.DATA_COS_UI_OUTPUT
  ? resolve(process.env.DATA_COS_UI_OUTPUT)
  : join(root, 'output', 'playwright', 'uix-refactor');

function run(binary, args, options = {}) {
  const result = spawnSync(binary, args, { cwd: native, stdio: 'inherit', ...options });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`${binary} exited with ${result.status}`);
}

async function goExecutable() {
  const candidates = [process.env.DATA_COS_GO, 'go', join(toolchain, 'go', 'bin', `go${exe}`)].filter(Boolean);
  for (const candidate of candidates) {
    const result = spawnSync(candidate, ['version'], { encoding: 'utf8' });
    if (result.status === 0) return candidate;
  }
  if (process.platform !== 'darwin' || process.arch !== 'arm64') {
    throw new Error('Install Go 1.27.4 or newer, or set DATA_COS_GO to its executable.');
  }
  console.log('Downloading Go 1.27.4 to the project .tmp directory');
  mkdirSync(toolchain, { recursive: true });
  const response = await fetch('https://go.dev/dl/go1.27.4.darwin-arm64.tar.gz');
  if (!response.ok) throw new Error(`Go download failed: HTTP ${response.status}`);
  const archive = Buffer.from(await response.arrayBuffer());
  const expected = '91a44a2b8a716600ad94a4f143e156116620db8828086a6e3602291be9bd0bb4';
  if (createHash('sha256').update(archive).digest('hex') !== expected) throw new Error('Go archive checksum mismatch');
  const archivePath = join(toolchain, 'go1.27.4.tar.gz');
  writeFileSync(archivePath, archive);
  run('tar', ['-xzf', archivePath, '-C', toolchain]);
  return join(toolchain, 'go', 'bin', 'go');
}

async function main() {
  if (!['bootstrap', 'start', 'build', 'test', 'e2e'].includes(command)) throw new Error(`Unknown command: ${command}`);
  const go = await goExecutable();
  const goEnv = { ...process.env, PATH: `${dirname(go)}${process.platform === 'win32' ? ';' : ':'}${process.env.PATH}` };
  if (command === 'bootstrap') {
    run(go, ['mod', 'download'], { env: goEnv });
    run('cargo', ['fetch', '--manifest-path', join(root, 'rust', 'Cargo.toml')]);
    return;
  }
  const release = command === 'build';
  run('cargo', ['build', '--manifest-path', join(root, 'rust', 'Cargo.toml'), '-p', 'data-cos-data-adapter', '--bin', 'data-cos-extract', ...(release ? ['--release'] : [])]);
  const engine = join(root, 'rust', 'target', release ? 'release' : 'debug', `data-cos-extract${exe}`);
  if (command === 'test') {
    run(go, ['test', '-race', './...'], { env: { ...goEnv, DATA_COS_TEST_ENGINE: engine } });
    run(go, ['vet', './...'], { env: goEnv });
    return;
  }
  if (command === 'e2e') {
    const fixtures = join(root, '.tmp', 'mygo-fixtures');
    run(go, ['test', '-run', 'TestRealEngineFixtures|TestNativeViewAtDesktopSizes|TestWorkbenchDarkThemeAndValidation', '-count=1', './...'], { env: { ...goEnv, DATA_COS_TEST_ENGINE: engine, DATA_COS_FIXTURES_DIR: fixtures, DATA_COS_SCREENSHOT_DIR: uiOutput } });
    const binary = join(native, 'build', `data-cos-mygo-e2e${exe}`);
    mkdirSync(dirname(binary), { recursive: true });
    run(go, ['build', ...guiBuildFlags, '-tags', 'e2e', '-o', binary, '.'], { env: goEnv });
    run('python3', [join(root, 'scripts', 'check_mygo.py'), '--binary', binary, '--engine', engine], {env:{...goEnv,DATA_COS_UI_OUTPUT:uiOutput}});
    return;
  }
  if (release) {
    const goArch = { x64: 'amd64', arm64: 'arm64' }[process.arch];
    if (!goArch) throw new Error(`Unsupported build architecture: ${process.arch}`);
    // Platform resources prevent a helper from one host entering another build.
    const resourceDir = join(native, 'resources', `${process.platform === 'win32' ? 'windows' : process.platform}-${goArch}`);
    mkdirSync(resourceDir, { recursive: true });
    copyFileSync(engine, join(resourceDir, `data-cos-extract${exe}`));
    run(go, ['tool', 'mygo', 'build', '-skip-dmg', '-skip-notarize'], { env: goEnv });
    return;
  }
  const binary = join(native, 'build', `data-cos-mygo${exe}`);
  mkdirSync(dirname(binary), { recursive: true });
  run(go, ['build', ...guiBuildFlags, '-o', binary, '.'], { env: goEnv });
  const child = spawn(binary, [], { cwd: native, stdio: 'inherit', env: { ...goEnv, DATA_COS_EXTRACT_BIN: engine } });
  for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP']) process.once(signal, () => child.kill(signal));
  child.once('error', (error) => { console.error(error.message); process.exitCode = 1; });
  child.once('exit', (code, signal) => { process.exitCode = signal ? 0 : (code ?? 1); });
}

main().catch((error) => { console.error(error.message); process.exitCode = 1; });
