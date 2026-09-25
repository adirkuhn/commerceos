import type { CreateSKUPayload, SKU, SKUPricingMode, SKUStatus, SKUUnitType, UpdateSKUPayload } from './types'
import { formatMinorUnitsAsInput, parseIntOrUndefined, parseOptionalPriceToMinorUnits, parsePriceToMinorUnits } from './productFormat'
import { dimensionInputsFromSKU, dimensionsPayload, type DimensionInputs } from './skuPayloads'

// The product editor's flat single-SKU form (only relevant when the product
// isn't in variants mode), as one value instead of ~20 separate useStates.

export interface FlatSkuForm extends DimensionInputs {
  skuCode: string
  barcode: string
  price: string
  salePrice: string
  saleEndDate: string
  currency: string
  skuStatus: SKUStatus
  isVirtual: boolean
  // Case-pack relationship (see pms.SKU.CaseOfSKUID) -- only editable once
  // the SKU already exists, since it links to another already-existing SKU.
  caseOfSKUID: string | null
  caseOfSKULabel: string
  unitsPerCase: string
  // Weight pool (see pms.SKU.UnitType/PortionedFromSKUID) -- a SKU is
  // either a weight pool itself (unit_type kilo), or portioned from one,
  // never both.
  unitType: SKUUnitType
  portionedFromSKUID: string | null
  portionedFromLabel: string
  // Weight-embedded (butcher/deli scale) barcode support (see
  // pms.SKU.ScaleCode/PricingMode/PricePerKiloMinorUnits) -- scaleCode '' means
  // this SKU isn't sold that way; pricingMode/pricePerKilo only matter when set.
  scaleCode: string
  pricingMode: SKUPricingMode
  pricePerKilo: string
  // null means "use the tenant's default tax class" (see TaxClass.is_default)
  // -- '' in the <select> represents that same choice.
  taxClassId: string | null
}

export type SetFlatSkuField = <K extends keyof FlatSkuForm>(key: K, value: FlatSkuForm[K]) => void
export type PatchFlatSku = (patch: Partial<FlatSkuForm> | ((prev: FlatSkuForm) => Partial<FlatSkuForm>)) => void

export const EMPTY_FLAT_SKU_FORM: FlatSkuForm = {
  skuCode: '', barcode: '', price: '', salePrice: '', saleEndDate: '', currency: 'USD', skuStatus: 'active',
  isVirtual: false, weightGrams: '', lengthMM: '', widthMM: '', heightMM: '',
  caseOfSKUID: null, caseOfSKULabel: '', unitsPerCase: '',
  unitType: 'unit', portionedFromSKUID: null, portionedFromLabel: '',
  scaleCode: '', pricingMode: 'fixed', pricePerKilo: '', taxClassId: null,
}

// Every field loaded from an existing SKU. The two *Label fields aren't
// part of the SKU (they're only known after picking in this session), so
// they're left untouched.
export function flatSkuFormFromSKU(sku: SKU): Omit<FlatSkuForm, 'caseOfSKULabel' | 'portionedFromLabel'> {
  return {
    skuCode: sku.sku_code,
    barcode: sku.barcode ?? '',
    price: formatMinorUnitsAsInput(sku.price_minor_units),
    salePrice: sku.sale_price_minor_units != null ? formatMinorUnitsAsInput(sku.sale_price_minor_units) : '',
    saleEndDate: sku.sale_end_date ?? '',
    currency: sku.currency_code,
    skuStatus: sku.status,
    isVirtual: sku.is_virtual,
    ...dimensionInputsFromSKU(sku),
    caseOfSKUID: sku.case_of_sku_id ?? null,
    unitsPerCase: sku.units_per_case != null ? String(sku.units_per_case) : '',
    unitType: sku.unit_type ?? 'unit',
    portionedFromSKUID: sku.portioned_from_sku_id ?? null,
    scaleCode: sku.scale_code ?? '',
    pricingMode: sku.pricing_mode ?? 'fixed',
    pricePerKilo: sku.price_per_kilo_minor_units != null ? formatMinorUnitsAsInput(sku.price_per_kilo_minor_units) : '',
    taxClassId: sku.tax_class_id ?? null,
  }
}

// The fields every flat-form SKU write sends. Adding a SKU to an existing
// variant sends exactly this.
export function flatSkuBasePayload(f: FlatSkuForm): CreateSKUPayload {
  return {
    sku_code: f.skuCode,
    barcode: f.barcode || undefined,
    price_minor_units: parsePriceToMinorUnits(f.price),
    sale_price_minor_units: parseOptionalPriceToMinorUnits(f.salePrice),
    sale_end_date: f.saleEndDate || null,
    currency_code: f.currency,
    is_virtual: f.isVirtual,
    ...dimensionsPayload(f.isVirtual, f),
    scale_code: f.scaleCode || null,
    pricing_mode: f.pricingMode,
    price_per_kilo_minor_units: f.pricingMode === 'per_weight' ? parseOptionalPriceToMinorUnits(f.pricePerKilo) : null,
    tax_class_id: f.taxClassId,
    status: f.skuStatus,
  }
}

// The very first SKU of a brand-new product -- also carries the weight-pool
// link, which is settable before the first save.
export function flatSkuCreatePayload(f: FlatSkuForm): CreateSKUPayload {
  return {
    ...flatSkuBasePayload(f),
    unit_type: f.unitType,
    portioned_from_sku_id: f.portionedFromSKUID,
  }
}

// Updating the product's one existing SKU: everything the form edits, plus
// tracks_expiry passed through (UpdateSKU is a full replace).
export function flatSkuUpdatePayload(f: FlatSkuForm, current: SKU): UpdateSKUPayload {
  return {
    ...flatSkuBasePayload(f),
    tracks_expiry: current.tracks_expiry,
    case_of_sku_id: f.caseOfSKUID,
    units_per_case: f.caseOfSKUID ? parseIntOrUndefined(f.unitsPerCase) ?? null : null,
    unit_type: f.unitType,
    portioned_from_sku_id: f.portionedFromSKUID,
  }
}
