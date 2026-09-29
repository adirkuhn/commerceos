// Translation file helpers (see docs/i18n.md):
//   node scripts/i18n.ts check        report missing / blank / wrong-placeholder texts
//   node scripts/i18n.ts add <locale>  start a new language from the English files
//   node scripts/i18n.ts types        regenerate src/i18n/resources.gen.d.ts (typed keys)
import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { compareCatalog, type Catalog } from '../src/i18n/catalog.ts'
import { typesSource } from '../src/i18n/typesSource.ts'

const i18nDir = join(import.meta.dirname, '..', 'src', 'i18n')
const root = join(i18nDir, 'locales')
const read = (locale: string, file: string) => JSON.parse(readFileSync(join(root, locale, file), 'utf8')) as Catalog
const namespaceFiles = () => readdirSync(join(root, 'en')).filter(f => f.endsWith('.json')).sort()

function blankCopy(c: Catalog): Catalog {
  return Object.fromEntries(Object.entries(c).map(([k, v]) => [k, typeof v === 'string' ? '' : blankCopy(v)]))
}

function check(): number {
  let problems = 0
  for (const locale of readdirSync(root).filter(l => l !== 'en').sort()) {
    for (const file of namespaceFiles()) {
      const target = existsSync(join(root, locale, file)) ? read(locale, file) : {}
      const issues = compareCatalog(read('en', file), target)
      const count = Object.values(issues).reduce((n, list) => n + list.length, 0)
      problems += count
      console.log(`${count ? '✗' : '✓'} ${locale}/${file}${count ? ` (${count})` : ''}`)
      for (const [kind, keys] of Object.entries(issues)) for (const key of keys) console.log(`    ${kind}: ${key}`)
    }
  }
  return problems
}

function add(locale: string | undefined) {
  if (!locale || !/^[a-z]{2}(-[A-Z]{2})?$/.test(locale)) throw new Error('usage: node scripts/i18n.ts add <locale>, e.g. de or es-MX')
  mkdirSync(join(root, locale), { recursive: true })
  for (const file of namespaceFiles()) {
    const path = join(root, locale, file)
    if (existsSync(path)) continue
    writeFileSync(path, JSON.stringify(blankCopy(read('en', file)), null, 2) + '\n')
    console.log(`created ${locale}/${file} (blank texts fall back to English)`)
  }
  console.log(`\nNext: translate the files, run "npm run i18n:check", then register "${locale}" as described in docs/i18n.md.`)
}

function types() {
  writeFileSync(join(i18nDir, 'resources.gen.d.ts'), typesSource(namespaceFiles()))
  console.log('wrote src/i18n/resources.gen.d.ts')
}

const [cmd, arg] = process.argv.slice(2)
if (cmd === 'check') process.exit(check() ? 1 : 0)
else if (cmd === 'add') add(arg)
else if (cmd === 'types') types()
else console.log('usage: node scripts/i18n.ts check | add <locale> | types')
