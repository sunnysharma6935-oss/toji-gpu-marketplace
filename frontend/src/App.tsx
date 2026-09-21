import { BrowserRouter, Routes, Route } from 'react-router-dom'

import { Home } from '@/pages/Home'
import { HowItWorks } from '@/pages/HowItWorks'
import { Marketplace } from '@/pages/customer/Marketplace'
import { GPUDetail } from '@/pages/customer/GPUDetail'
import { Login } from '@/pages/auth/Login'
import { Signup } from '@/pages/auth/Signup'

import { DashboardHome } from '@/pages/customer/DashboardHome'
import { MyRentals } from '@/pages/customer/MyRentals'
import { RentalDetail } from '@/pages/customer/RentalDetail'
import { Billing } from '@/pages/customer/Billing'
import { Account } from '@/pages/customer/Account'

import { HostLanding } from '@/pages/host/HostLanding'
import { HostOnboarding } from '@/pages/host/HostOnboarding'
import {
  HostDashboardOverview,
  HostMachines,
  HostListings,
  HostEarnings,
  HostAccount,
} from '@/pages/host/HostDashboardPages'

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Marketing */}
        <Route path="/" element={<Home />} />
        <Route path="/how-it-works" element={<HowItWorks />} />
        <Route path="/host" element={<HostLanding />} />

        {/* Marketplace */}
        <Route path="/marketplace" element={<Marketplace />} />
        <Route path="/marketplace/:listingId" element={<GPUDetail />} />

        {/* Auth */}
        <Route path="/login" element={<Login />} />
        <Route path="/signup" element={<Signup />} />

        {/* Customer dashboard */}
        <Route path="/dashboard" element={<DashboardHome />} />
        <Route path="/dashboard/rentals" element={<MyRentals />} />
        <Route path="/dashboard/rentals/:rentalId" element={<RentalDetail />} />
        <Route path="/dashboard/billing" element={<Billing />} />
        <Route path="/dashboard/account" element={<Account />} />

        {/* Host */}
        <Route path="/host/onboarding" element={<HostOnboarding />} />
        <Route path="/host/dashboard" element={<HostDashboardOverview />} />
        <Route path="/host/machines" element={<HostMachines />} />
        <Route path="/host/listings" element={<HostListings />} />
        <Route path="/host/earnings" element={<HostEarnings />} />
        <Route path="/host/account" element={<HostAccount />} />
      </Routes>
    </BrowserRouter>
  )
}
