package channels

import (
	"strconv"
	"strings"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

// bolUnitID translates a free-text unit string (whatever the merchant typed
// into the Bol tab's Content Unit / Content per Dispenser Unit fields) into
// Bol's unece unit code for AttributeValue.unitId. Confirmed against the
// "Capacity"/"Capacity per dispenser" attributes' units lists in
// productdatamodel.s-bol.com/v10/datamodel_v10_en.json. Unrecognized input
// returns "" (no unitId sent) rather than guessing or erroring -- the value
// itself still gets submitted, same as before this fix, just without a
// unit Bol can act on; better than blocking the whole content save over a
// free-text field Bol never validated to begin with.
func bolUnitID(unit string) string {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "ml", "milliliter", "milliliters", "milliliter(s)":
		return "unece.unit.MLT"
	case "l", "liter", "liters", "litre", "litres":
		return "unece.unit.LTR"
	case "cl", "centiliter", "centiliters":
		return "unece.unit.CLT"
	case "g", "gram", "grams":
		return "unece.unit.GRM"
	case "kg", "kilogram", "kilograms":
		return "unece.unit.KGM"
	case "mg", "milligram", "milligrams":
		return "unece.unit.MGM"
	case "stuk", "stuks", "piece", "pieces":
		return "Pieces"
	default:
		return ""
	}
}

// bolProductContentAttributes flattens BolProductContent into the id/value
// map (and, for number+unit attributes, the companion unit map)
// SubmitProductContent sends. nil content (nothing filled in yet) returns
// nil/nil, not an error -- SubmitProductContent still sends EAN/Name/
// Description on their own.
//
// Keys were originally the Dutch UI label text (see git blame), which is
// NOT the real attribute id -- Bol's id is a separate, English, data-model
// key (e.g. "Manufacturer Name", not "Fabrikant Naam"). A wrong id isn't
// rejected: the POST still 202s and Bol silently drops that attribute, so
// this shipped for a while with content-save "succeeding" while fields
// never actually appeared on bol.com (caught 2026-08-25 in prod:
// Manufacturer Name/Address/Email never updated, no error anywhere). Every
// key below is now confirmed against Bol's real, published data model
// (productdatamodel.s-bol.com/v10/datamodel_v10_en.json -- id field --
// cross-referenced against the _nl.json file's name field to translate
// each Dutch label; values are submitted in Dutch since language is "nl").
//
// Two categories need more than a renamed key:
//   - Number+unit attributes (Capacity, Capacity per dispenser,
//     Width/Height/Length, Weight, Number Pieces In Package) -- Bol wants
//     value and unitId as sibling fields on one AttributeValue, not two
//     attributes. ContentAmount+ContentUnit merge into one "Capacity"
//     entry (unit via bolUnitID); same for the dispenser pair. Width/
//     Height/Length/Weight keep their own value but gain a fixed unitId
//     (the DB fields are always mm/g, no user-entered unit to translate).
//     Items-per-package gets a fixed "Pieces" unitId (no unit field exists
//     for it in the domain type, and "how many items" is inherently a
//     piece count).
//   - LOV-constrained attributes (Skin Type, Shelf life sealed/after
//     opening, Country of Origin, Hazard Statements, Substance,
//     Precautionary Statement) -- Bol only accepts values from a fixed
//     list. The id fix alone doesn't guarantee a match: see
//     ProductBolTab.tsx, which now renders these as selects populated from
//     the same real LOV data instead of free text, so a saved value is
//     guaranteed to be one Bol recognizes.
//
// Two fields stay free text deliberately: Language Instructions/Packaging
// only have LOV entries scoped per GPC category (e.g. "Language
// Instructions@30003886"), not a global list, so there's no single static
// dropdown that's correct for every category -- picking one product's
// category-scoped list and presenting it as universal would be its own
// silent-mismatch bug. Package Insert (a document URL) has no LOV at all.
func bolProductContentAttributes(c *channelsDomain.BolProductContent) (attrs map[string]string, units map[string]string) {
	if c == nil {
		return nil, nil
	}
	attrs = map[string]string{}
	units = map[string]string{}
	setStr := func(key string, v *string) {
		if v != nil && *v != "" {
			attrs[key] = *v
		}
	}
	setBool := func(key string, v *bool) {
		if v != nil {
			attrs[key] = strconv.FormatBool(*v)
		}
	}
	setIntWithUnit := func(key string, v *int32, unitID string) {
		if v != nil {
			attrs[key] = strconv.FormatInt(int64(*v), 10)
			units[key] = unitID
		}
	}
	setAmountWithUnit := func(key string, amount, unit *string) {
		if amount == nil || *amount == "" {
			return
		}
		attrs[key] = *amount
		if unit != nil {
			if id := bolUnitID(*unit); id != "" {
				units[key] = id
			}
		}
	}

	setStr("Manufacturer Email", c.ManufacturerContact)
	setStr("Manufacturer Address", c.ManufacturerAddress)
	setStr("Manufacturer Name", c.ManufacturerName)
	setStr("Shelf life sealed product", c.ShelfLifeClosed)
	setStr("Shelf life period after opening", c.ShelfLifeAfterOpening)
	setStr("Ingredients", c.Ingredients)
	setAmountWithUnit("Capacity", c.ContentAmount, c.ContentUnit)
	setStr("Country of Origin", c.CountryOfOrigin)
	setStr("Substance", c.Substance)
	setStr("Language Instructions", c.InstructionLanguage)
	setStr("Language Packaging", c.PackagingLanguage)

	setIntWithUnit("Number Pieces In Package", c.ItemsPerPackage, "Pieces")
	setStr("Package Insert", c.PackageInsertURL)
	setBool("Gift", c.IsGift)
	setBool("Dermatological Tested", c.DermatologicallyTested)
	setStr("Hazard Statements", c.HazardStatements)
	setStr("Skin Type", c.SkinType)
	setBool("Hypo Allergenic", c.Hypoallergenic)
	setBool("Including Zincoxide", c.ContainsZincOxide)
	setAmountWithUnit("Capacity per dispenser", c.ContentPerDispenserAmount, c.ContentPerDispenserUnit)
	setStr("Brand", c.Brand)
	setBool("Fragrance Free", c.FragranceFree)
	setBool("Ph Neutral", c.PHNeutral)
	setBool("Cooling Effect", c.CoolingEffect)
	setIntWithUnit("Width", c.PackageWidthMM, "unece.unit.MMT")
	setIntWithUnit("Height", c.PackageHeightMM, "unece.unit.MMT")
	setIntWithUnit("Length", c.PackageLengthMM, "unece.unit.MMT")
	setIntWithUnit("Weight", c.PackageWeightG, "unece.unit.GRM")
	setStr("Precautionary Statement", c.PrecautionStatements)
	setStr("Precautionary Measures", c.Precautions)
	setStr("Warning Label", c.Warnings)
	// c.VideoURL is deliberately not sent: "Video" is not a real attribute
	// id (searched the full data model, no match, nor any generic
	// product-video attribute) -- it was silently dropped same as
	// everything else in this map before the fix. Bol's Asset (images)
	// mechanism is the most likely real fit for video, but no label value
	// for it is confirmed (spec only documents "FRONT" for images), so
	// this isn't guessed at rather than shipping another unverified guess.

	if len(attrs) == 0 {
		return nil, nil
	}
	return attrs, units
}
