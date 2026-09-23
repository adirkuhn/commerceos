import { useQueryEnum } from '../../routing/useQueryState'
import { REPORTS_PERIODS, DEFAULT_REPORTS_PERIOD, type ReportsPeriod } from '../period'

// The report period picker's value, kept in ?period= so a refresh or a
// shared link shows the same range.
export function useReportPeriod(): [ReportsPeriod, (period: ReportsPeriod) => void] {
  return useQueryEnum<ReportsPeriod>('period', REPORTS_PERIODS, DEFAULT_REPORTS_PERIOD)
}
