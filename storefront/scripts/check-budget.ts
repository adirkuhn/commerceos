// Fails the build when the shopper-facing bundle breaks its budget.
// Run after `vite build` (see package.json "build").
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { checkBudget, initialScripts } from './budget.ts'

const client = join(import.meta.dirname, '..', 'dist', 'client')
const index = readFileSync(join(client, 'index.html'), 'utf8')
const files = initialScripts(index).map(src => {
  const path = join(client, src.replace(/^\/store\//, ''))
  return { path: src, content: readFileSync(path) }
})
const report = checkBudget(files)
for (const f of report.files) console.log(`  ${f.path}  ${(f.gzip / 1024).toFixed(1)} KB gz`)
console.log(`initial JS: ${(report.totalGzip / 1024).toFixed(1)} KB gz`)
if (report.problems.length) {
  for (const p of report.problems) console.error(`budget: ${p}`)
  process.exit(1)
}
