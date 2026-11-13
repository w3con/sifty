#!/usr/bin/env node
import path from 'node:path';
import os from 'node:os';
import { fileURLToPath } from 'node:url';
import { restoreArchive } from './transfer-lib.mjs';

try {
  const [major, minor] = process.versions.node.split('.').map(Number);
  if (major < 22 || major === 22 && minor < 13) throw Error('Install Node >=22.13 and Git first (brew install node poppler; xcode-select --install if Git is missing)');
  const args = process.argv.slice(2);
  let target = path.join(os.homedir(), 'Dev/fcto'), verifyOnly = false, install = false;
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--target' && args[i + 1]) target = path.resolve(args[++i]);
    else if (args[i] === '--verify-only') verifyOnly = true;
    else if (args[i] === '--install') install = true;
    else if (args[i] === '--no-install') install = false;
    else throw Error('Usage: node restore.mjs [--target /new/path] [--no-install|--install] [--verify-only]');
  }
  restoreArchive(path.dirname(fileURLToPath(import.meta.url)), target, { verifyOnly, install });
  console.log(verifyOnly ? 'Archive verified; no checkout installed.' : `Verified checkout restored to ${target}. Assessment remains Off. No services started. See TRANSFER.md.`);
} catch (e) { console.error(`Restore refused: ${e.message}`); process.exitCode = 1; }
