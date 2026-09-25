import type { SKU, UpdateSKUPayload } from './types'
import { parseIntOrUndefined } from './productFormat'

// UpdateSKU is a full replace on the server -- every field not being edited
// must be sent back unchanged, or it silently resets (tracks_expiry, the
// scale-barcode fields, the case/weight-pool links...). This is that
// "everything as loaded" payload; callers spread their edits on top.
//
// tax_class_id is deliberately NOT included: the variant-row inline editors
// have never sent it, and changing that is a behaviour change, not a
// refactor. Callers that do send it add it explicitly.
export function skuPassThroughPayload(sku: SKU): UpdateSKUPayload {
  return {
    sku_code: sku.sku_code,
    barcode: sku.barcode ?? undefined,
    price_minor_units: sku.price_minor_units,
    sale_price_minor_units: sku.sale_price_minor_units,
    sale_end_date: sku.sale_end_date,
    currency_code: sku.currency_code,
    weight_grams: sku.weight_grams,
    length_mm: sku.length_mm,
    width_mm: sku.width_mm,
    height_mm: sku.height_mm,
    is_virtual: sku.is_virtual,
    tracks_expiry: sku.tracks_expiry,
    case_of_sku_id: sku.case_of_sku_id,
    units_per_case: sku.units_per_case,
    unit_type: sku.unit_type,
    portioned_from_sku_id: sku.portioned_from_sku_id,
    scale_code: sku.scale_code,
    pricing_mode: sku.pricing_mode,
    price_per_kilo_minor_units: sku.price_per_kilo_minor_units,
    status: sku.status,
  }
}

export interface DimensionInputs {
  weightGrams: string
  lengthMM: string
  widthMM: string
  heightMM: string
}

export interface DimensionsPayload {
  weight_grams: number | undefined
  length_mm: number | undefined
  width_mm: number | undefined
  height_mm: number | undefined
}

// A virtual SKU never ships, so it can never carry weight/dimensions
// (enforced server-side too — see ErrVirtualSKUCannotHaveDimensions).
export function dimensionsPayload(isVirtual: boolean, d: DimensionInputs): DimensionsPayload {
  if (isVirtual) return { weight_grams: undefined, length_mm: undefined, width_mm: undefined, height_mm: undefined }
  return {
    weight_grams: parseIntOrUndefined(d.weightGrams),
    length_mm: parseIntOrUndefined(d.lengthMM),
    width_mm: parseIntOrUndefined(d.widthMM),
    height_mm: parseIntOrUndefined(d.heightMM),
  }
}

export function dimensionInputsFromSKU(sku: SKU): DimensionInputs {
  return {
    weightGrams: sku.weight_grams != null ? String(sku.weight_grams) : '',
    lengthMM: sku.length_mm != null ? String(sku.length_mm) : '',
    widthMM: sku.width_mm != null ? String(sku.width_mm) : '',
    heightMM: sku.height_mm != null ? String(sku.height_mm) : '',
  }
}
