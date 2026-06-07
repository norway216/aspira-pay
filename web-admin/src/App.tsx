import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import Transactions from './pages/Transactions'
import Users from './pages/Users'
import Ledger from './pages/Ledger'
import Audit from './pages/Audit'

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<Dashboard />} />
        <Route path="/transactions" element={<Transactions />} />
        <Route path="/users" element={<Users />} />
        <Route path="/ledger" element={<Ledger />} />
        <Route path="/audit" element={<Audit />} />
      </Route>
    </Routes>
  )
}
