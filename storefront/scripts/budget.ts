import { gzipSync } from 'node:zlib'

// Performance budget for what a shopper downloads before the page is
// interactive: the runtime entry and the modules it preloads (from the
// built index.html). The editor (GrapesJS) must never be among them.

export const INITIAL_JS_BUDGET_GZIP = 120 * 1024

export interface BudgetFile {
  path: string
  content: Buffer | string
}

export interface BudgetReport {
  totalGzip: number
  files: { path: string; gzip: number }[]
  problems: string[]
}

/** The JS files index.html loads up front (entry script + modulepreloads). */
export function initialScripts(indexHTML: string): string[] {
  const refs = [
    ...indexHTML.matchAll(/<script[^>]*type="module"[^>]*src="([^"]+)"/g),
    ...indexHTML.matchAll(/<link[^>]*rel="modulepreload"[^>]*href="([^"]+)"/g),
  ].map(m => m[1])
  return [...new Set(refs)]
}

export function checkBudget(files: BudgetFile[], budget = INITIAL_JS_BUDGET_GZIP): BudgetReport {
  const sized = files.map(f => ({ path: f.path, gzip: gzipSync(f.content).length }))
  const totalGzip = sized.reduce((sum, f) => sum + f.gzip, 0)
  const problems: string[] = []
  if (totalGzip > budget) problems.push(`initial JS is ${kb(totalGzip)} gzipped, over the ${kb(budget)} budget`)
  for (const f of files) {
    if (/grapesjs/i.test(String(f.content))) problems.push(`${f.path} contains the editor (GrapesJS)`)
  }
  return { totalGzip, files: sized, problems }
}

const kb = (bytes: number) => `${(bytes / 1024).toFixed(1)} KB`
