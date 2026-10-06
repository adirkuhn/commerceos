import type { TFunction } from 'i18next'
import type { OrderUrgency } from './quantity'
import type { SeasonEventKey } from './types'

// Literal keys (not t(`urgency.${u}`)) so i18n:extract keeps them.
export const urgencyLabels = (t: TFunction<'purchasePlanning'>): Record<Exclude<OrderUrgency, 'none'>, string> => ({
  overdue: t('urgency.overdue'),
  today: t('urgency.today'),
  soon: t('urgency.soon'),
  later: t('urgency.later'),
})

export const seasonEventLabels = (t: TFunction<'purchasePlanning'>): Record<SeasonEventKey, string> => ({
  christmas: t('season.event.christmas'),
  sinterklaas: t('season.event.sinterklaas'),
  black_friday: t('season.event.blackFriday'),
  easter: t('season.event.easter'),
  carnival: t('season.event.carnival'),
  valentines: t('season.event.valentines'),
  mothers_day: t('season.event.mothersDay'),
  fathers_day_br: t('season.event.fathersDayBR'),
  festa_junina: t('season.event.festaJunina'),
  dia_dos_namorados: t('season.event.diaDosNamorados'),
  fathers_day_nl: t('season.event.fathersDayNL'),
  kings_day: t('season.event.kingsDay'),
  halloween: t('season.event.halloween'),
  dia_das_criancas: t('season.event.diaDasCriancas'),
  sint_maarten: t('season.event.sintMaarten'),
  new_years_eve: t('season.event.newYearsEve'),
})
