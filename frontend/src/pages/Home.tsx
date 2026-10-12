import Dashboard from './Dashboard'
import TodayPage from '../dashboard/TodayPage'
import { useDesign } from '../theme/useDesign'

// "/" -- the depot design opens on Today (work queue first); classic keeps
// its Dashboard of figures.
export default function Home() {
  const { design } = useDesign()
  return design === 'depot' ? <TodayPage /> : <Dashboard />
}
