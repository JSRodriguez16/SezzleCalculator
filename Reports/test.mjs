import { spawn } from 'node:child_process';
import { mkdir, rm, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const backendDir = path.join(root, 'Backend');
const reportsDir = path.join(root, 'reports');
const coverageTxtPath = path.join(reportsDir, 'coverage.txt');
const tempProfilePath = path.join(reportsDir, 'coverage.tmp.out');
const goBin = process.env.GO_BIN || 'go';

function run(command, args, cwd, quiet = false) {
  return new Promise((resolve) => {
    const stdout = [];
    const stderr = [];
    let started = false;
    const child = spawn(command, args, {
      cwd,
      env: process.env,
      windowsHide: true,
      shell: false,
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    child.on('spawn', () => { started = true; });
    child.stdout.on('data', (chunk) => {
      stdout.push(chunk);
      if (!quiet) process.stdout.write(chunk);
    });
    child.stderr.on('data', (chunk) => {
      stderr.push(chunk);
      process.stderr.write(chunk);
    });
    child.on('error', (error) => {
      console.error(`No se pudo ejecutar ${command}: ${error.message}`);
      resolve({ code: 1, started, stdout: '', stderr: error.message });
    });
    child.on('close', (code) => resolve({
      code: code ?? 1,
      started,
      stdout: Buffer.concat(stdout).toString('utf8'),
      stderr: Buffer.concat(stderr).toString('utf8'),
    }));
  });
}

async function main() {
  await mkdir(reportsDir, { recursive: true });
  await rm(coverageTxtPath, { force: true });
  await rm(tempProfilePath, { force: true });

  console.log('Ejecutando pruebas unitarias del backend...');
  const testResult = await run(goBin, ['test', '-v', '-count=1', '-covermode=atomic', `-coverprofile=${tempProfilePath}`, './...'], backendDir, false);

  if (testResult.code !== 0) {
    console.error('\nLas pruebas unitarias fallaron.');
    await rm(tempProfilePath, { force: true });
    process.exit(1);
  }

  console.log('\nGenerando reporte de cobertura en TXT...');
  const coverResult = await run(goBin, ['tool', 'cover', `-func=${tempProfilePath}`], backendDir, true);
  await rm(tempProfilePath, { force: true });

  if (coverResult.code !== 0) {
    console.error('\nNo se pudo calcular la cobertura con go tool cover.');
    process.exit(1);
  }

  await writeFile(coverageTxtPath, coverResult.stdout, 'utf8');
  console.log(coverResult.stdout);
  console.log(`Reporte de cobertura guardado en: ${coverageTxtPath}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
