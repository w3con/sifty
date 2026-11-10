// DOCX worker: reads one .docx file and returns its visible text through the Go-core worker
// protocol. Mammoth is used only for raw text extraction; converted HTML is never rendered.
import process from 'node:process';
import mammoth from 'mammoth';

let input = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (chunk) => { input += chunk; });
process.stdin.on('end', async () => {
  try {
    const req = JSON.parse(input);
    if (req.method !== 'extractDocx') throw new Error(`unsupported method: ${req.method}`);
    const path = req.params && req.params.path;
    if (typeof path !== 'string' || path === '') throw new Error('path is required');
    const result = await mammoth.extractRawText({ path });
    process.stdout.write(JSON.stringify({ ok: true, result: { text: result.value } }));
  } catch (e) {
    process.stdout.write(JSON.stringify({ ok: false, error: String(e && e.message ? e.message : e) }));
  }
});
