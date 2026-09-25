import type { CreateSKUPayload, SKU, SKUPricingMode, SKUStatus, SKUUnitType, UpdateSKUPayload, VariantWithSKUs } from './types'
import { formatMinorUnitsAsInput, parseOptionalPriceToMinorUnits, parsePriceToMinorUnits } from './productFormat'
import { dimensionInputsFromSKU, dimensionsPayload, skuPassThroughPayload, type DimensionInputs } from './skuPayloads'

// Form values for VariantRow's two inline SKU panels ("+ Add SKU" and
// "Edit variant"), and the payloads they send.

export interface VariantSkuDraft extends DimensionInputs {
  skuCode: string
  barcode: string
  price: string
  salePrice: string
  saleEndDate: string
  currency: string
  status: SKUStatus
  isVirtual: boolean
  // Weight pool (see pms.SKU.UnitType/PortionedFromSKUID) -- a SKU is
  // either a weight pool itself (unit_type kilo), or portioned from one,
  // never both.
  unitType: SKUUnitType
  portionedFromSKUID: string | null
  portionedFromLabel: string
  // See the product editor's flat form for the weight-embedded
  // (butcher/deli scale) barcode -- same convention here.
  scaleCode: string
  pricingMode: SKUPricingMode
  pricePerKilo: string
}

export function emptyVariantSkuDraft(currency: string): VariantSkuDraft {
  return {
    skuCode: '', barcode: '', price: '', salePrice: '', saleEndDate: '', currency, status: 'active',
    isVirtual: false, weightGrams: '', lengthMM: '', widthMM: '', heightMM: '',
    unitType: 'unit', portionedFromSKUID: null, portionedFromLabel: '',
    scaleCode: '', pricingMode: 'fixed', pricePerKilo: '',
  }
}

// Adding a SKU to a variant. No tax_class_id: a variant SKU starts on the
// tenant default class.
export function variantSkuAddPayload(d: VariantSkuDraft): CreateSKUPayload {
  return {
    sku_code: d.skuCode,
    barcode: d.barcode || undefined,
    price_minor_units: parsePriceToMinorUnits(d.price),
    sale_price_minor_units: parseOptionalPriceToMinorUnits(d.salePrice),
    sale_end_date: d.saleEndDate || null,
    currency_code: d.currency,
    is_virtual: d.isVirtual,
    ...dimensionsPayload(d.isVirtual, d),
    unit_type: d.unitType,
    portioned_from_sku_id: d.portionedFromSKUID,
    scale_code: d.scaleCode || null,
    pricing_mode: d.pricingMode,
    price_per_kilo_minor_units: d.pricingMode === 'per_weight' ? parseOptionalPriceToMinorUnits(d.pricePerKilo) : null,
    status: d.status,
  }
}

// "Edit variant" covers the variant name and the SKU's own fields --
// everything except barcode and the weight-pool link, which have their own
// dedicated inline controls.
export interface VariantEditForm extends DimensionInputs {
  name: string
  skuCode: string
  price: string
  salePrice: string
  saleEndDate: string
  currency: string
  status: SKUStatus
  isVirtual: boolean
}

export function variantEditFormFrom(variant: VariantWithSKUs, sku: SKU | undefined): VariantEditForm {
  if (!sku) {
    return {
      name: variant.name, skuCode: '', price: '', salePrice: '', saleEndDate: '', currency: 'USD', status: 'active',
      isVirtual: false, weightGrams: '', lengthMM: '', widthMM: '', heightMM: '',
    }
  }
  return {
    name: variant.name,
    skuCode: sku.sku_code,
    price: formatMinorUnitsAsInput(sku.price_minor_units),
    salePrice: sku.sale_price_minor_units != null ? formatMinorUnitsAsInput(sku.sale_price_minor_units) : '',
    saleEndDate: sku.sale_end_date ?? '',
    currency: sku.currency_code,
    status: sku.status,
    isVirtual: sku.is_virtual,
    ...dimensionInputsFromSKU(sku),
  }
}

export function variantEditSkuPayload(f: VariantEditForm, sku: SKU): UpdateSKUPayload {
  return {
    ...skuPassThroughPayload(sku),
    sku_code: f.skuCode,
    price_minor_units: parsePriceToMinorUnits(f.price),
    sale_price_minor_units: parseOptionalPriceToMinorUnits(f.salePrice),
    sale_end_date: f.saleEndDate || null,
    currency_code: f.currency,
    ...dimensionsPayload(f.isVirtual, f),
    is_virtual: f.isVirtual,
    status: f.status,
  }
}
