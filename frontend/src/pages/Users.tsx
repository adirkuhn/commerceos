import { useState } from 'react'
import { Plus, Search, Shield, MoreHorizontal, CheckCircle, XCircle } from 'lucide-react'
import clsx from 'clsx'

type Role = 'Admin' | 'Manager' | 'Warehouse' | 'Sales' | 'Finance' | 'Viewer'

interface User {
  id: string
  name: string
  email: string
  role: Role
  active: boolean
  lastLogin: string
  avatar: string
}

const USERS: User[] = [
  { id: '1', name: 'Alex Kuhn', email: 'alex@omnistock.io', role: 'Admin', active: true, lastLogin: '2m ago', avatar: 'AK' },
  { id: '2', name: 'Maria Santos', email: 'maria@omnistock.io', role: 'Manager', active: true, lastLogin: '1h ago', avatar: 'MS' },
  { id: '3', name: 'Tom de Vries', email: 'tom@omnistock.io', role: 'Warehouse', active: true, lastLogin: '30m ago', avatar: 'TV' },
  { id: '4', name: 'Emma Weber', email: 'emma@omnistock.io', role: 'Sales', active: true, lastLogin: '3h ago', avatar: 'EW' },
  { id: '5', name: 'Carlos Ruiz', email: 'carlos@omnistock.io', role: 'Finance', active: false, lastLogin: '2 days ago', avatar: 'CR' },
  { id: '6', name: 'Lisa Chen', email: 'lisa@omnistock.io', role: 'Viewer', active: true, lastLogin: '5h ago', avatar: 'LC' },
]

const ROLE_COLORS: Record<Role, string> = {
  Admin: 'bg-red-500/15 text-red-400',
  Manager: 'bg-blue-500/15 text-blue-400',
  Warehouse: 'bg-amber-500/15 text-amber-400',
  Sales: 'bg-green-500/15 text-green-400',
  Finance: 'bg-purple-500/15 text-purple-400',
  Viewer: 'bg-gray-500/15 text-gray-400',
}

const PERMISSIONS: { label: string; roles: Role[] }[] = [
  { label: 'View Products', roles: ['Admin', 'Manager', 'Warehouse', 'Sales', 'Finance', 'Viewer'] },
  { label: 'Edit Products', roles: ['Admin', 'Manager'] },
  { label: 'Manage Inventory', roles: ['Admin', 'Manager', 'Warehouse'] },
  { label: 'View Orders', roles: ['Admin', 'Manager', 'Warehouse', 'Sales', 'Finance', 'Viewer'] },
  { label: 'Process Orders', roles: ['Admin', 'Manager', 'Warehouse', 'Sales'] },
  { label: 'Manage Integrations', roles: ['Admin', 'Manager'] },
  { label: 'Access Shipping', roles: ['Admin', 'Manager', 'Warehouse'] },
  { label: 'View Reports', roles: ['Admin', 'Manager', 'Finance'] },
  { label: 'Manage Users', roles: ['Admin'] },
  { label: 'Access Settings', roles: ['Admin'] },
]

const ALL_ROLES: Role[] = ['Admin', 'Manager', 'Warehouse', 'Sales', 'Finance', 'Viewer']

export default function Users() {
  const [search, setSearch] = useState('')
  const [showPermMatrix, setShowPermMatrix] = useState(false)

  const filtered = USERS.filter(u =>
    u.name.toLowerCase().includes(search.toLowerCase()) ||
    u.email.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search users..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <button
          onClick={() => setShowPermMatrix(!showPermMatrix)}
          className={clsx(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-lg border text-xs transition-colors',
            showPermMatrix
              ? 'bg-blue-600/15 border-blue-500/30 text-blue-400'
              : 'dark:bg-[#1F2937] bg-white dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-600',
          )}
        >
          <Shield size={13} /> Permissions Matrix
        </button>
        <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 ml-auto font-medium">
          <Plus size={13} /> Invite User
        </button>
      </div>

      <div className="flex-1 overflow-hidden flex gap-0">
        {/* User table */}
        <div className={clsx('flex flex-col overflow-hidden', showPermMatrix ? 'flex-1' : 'w-full')}>
          <div className="flex-1 overflow-auto">
            <table className="w-full text-xs">
              <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
                <tr>
                  <th className="px-4 py-2.5 w-8">
                    <input type="checkbox" className="rounded accent-blue-600 w-3 h-3" />
                  </th>
                  {['User', 'Role', 'Status', 'Last Login', ''].map(h => (
                    <th key={h} className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
                {filtered.map(user => (
                  <tr key={user.id} className="hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors group">
                    <td className="px-4 py-3">
                      <input type="checkbox" className="rounded accent-blue-600 w-3 h-3" />
                    </td>
                    <td className="px-3 py-3">
                      <div className="flex items-center gap-2.5">
                        <div className="w-7 h-7 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-[10px] font-bold text-white shrink-0">
                          {user.avatar}
                        </div>
                        <div>
                          <p className="font-medium dark:text-gray-100 text-gray-900">{user.name}</p>
                          <p className="text-[10px] dark:text-gray-500 text-gray-400">{user.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-3 py-3">
                      <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium', ROLE_COLORS[user.role])}>
                        {user.role}
                      </span>
                    </td>
                    <td className="px-3 py-3">
                      <div className="flex items-center gap-1.5">
                        {user.active
                          ? <CheckCircle size={12} className="text-green-400" />
                          : <XCircle size={12} className="text-gray-400" />
                        }
                        <span className={clsx('text-[10px] font-medium', user.active ? 'text-green-400' : 'dark:text-gray-500 text-gray-400')}>
                          {user.active ? 'Active' : 'Inactive'}
                        </span>
                      </div>
                    </td>
                    <td className="px-3 py-3 dark:text-gray-500 text-gray-400">{user.lastLogin}</td>
                    <td className="px-3 py-3">
                      <button className="opacity-0 group-hover:opacity-100 w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200">
                        <MoreHorizontal size={14} className="dark:text-gray-400 text-gray-500" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Permissions matrix */}
        {showPermMatrix && (
          <div className="w-[480px] border-l dark:border-[#2D3748] border-gray-200 flex flex-col overflow-hidden">
            <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
              <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Permissions Matrix</h3>
              <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">RBAC — role-based access control</p>
            </div>
            <div className="flex-1 overflow-auto">
              <table className="w-full text-xs">
                <thead className="sticky top-0 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
                  <tr>
                    <th className="px-3 py-2 text-left font-medium dark:text-gray-500 text-gray-400 text-[10px] uppercase">Permission</th>
                    {ALL_ROLES.map(r => (
                      <th key={r} className="px-2 py-2 text-center">
                        <span className={clsx('px-1.5 py-0.5 rounded text-[9px] font-medium', ROLE_COLORS[r])}>
                          {r}
                        </span>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
                  {PERMISSIONS.map(perm => (
                    <tr key={perm.label} className="hover:dark:bg-[#1F2937]/60 hover:bg-gray-50">
                      <td className="px-3 py-2.5 dark:text-gray-300 text-gray-700">{perm.label}</td>
                      {ALL_ROLES.map(role => (
                        <td key={role} className="px-2 py-2.5 text-center">
                          {perm.roles.includes(role)
                            ? <CheckCircle size={13} className="text-green-400 mx-auto" />
                            : <div className="w-3 h-0.5 dark:bg-[#2D3748] bg-gray-200 mx-auto rounded" />
                          }
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
