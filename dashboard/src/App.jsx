import React, { useState } from 'react';
import { Activity, HardDrive, Package, Crown } from 'lucide-react';
import Overview from './components/Overview';
import DeviceDetail from './components/DeviceDetail';
import AssetInventory from './components/AssetInventory';
import Applications from './components/Applications';

export default function App() {
  const [activeTab, setActiveTab] = useState('overview');
  const [selectedServerId, setSelectedServerId] = useState(null);

  const handleSelectDevice = (serverId) => {
    setSelectedServerId(serverId);
    setActiveTab('detail');
  };

  const handleBackToOverview = () => {
    setSelectedServerId(null);
    setActiveTab('overview');
  };

  const handleRenamed = (newServerId) => {
    setSelectedServerId(newServerId);
  };

  const tabs = [
    { id: 'overview', label: 'Daily Ledger', icon: Activity, desc: 'Telemetry overview' },
    { id: 'assets', label: 'Estate Inventory', icon: HardDrive, desc: 'Hardware registers' },
    { id: 'apps', label: 'Provisions', icon: Package, desc: 'Installed applications' },
  ];

  return (
    <div className="min-h-screen bg-parchment text-ink flex flex-col font-body antialiased selection:bg-gold-leaf selection:text-ink">
      {/* Header - The Estate Seal */}
      <header className="sticky top-0 z-50 bg-parchment border-b border-ink-muted/30 px-6 py-4">
        <div className="max-w-7xl mx-auto">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-ink rounded-sm border border-ink shadow-wax flex-shrink-0">
                <Crown className="w-7 h-7 text-gold-leaf" strokeWidth={1.5} />
              </div>
              <div>
                <h1 className="font-display text-2xl md:text-3xl font-semibold text-ink tracking-tight leading-tight m-0">
                  SENESCHAL
                </h1>
                <p className="font-mono-ui text-[11px] text-ink-muted uppercase tracking-widest mt-0.5 m-0">
                  Estate Steward &middot; Infrastructure Register
                </p>
              </div>
            </div>

            {/* Navigation Tabs - Ledger Style */}
            <nav className="flex flex-wrap items-center gap-1 bg-parchment-dark border border-ink-muted/30 p-1.5">
              {tabs.map((tab) => {
                const Icon = tab.icon;
                const isActive = activeTab === tab.id || (activeTab === 'detail' && tab.id === 'overview');
                return (
                  <button
                    key={tab.id}
                    onClick={() => { setSelectedServerId(null); setActiveTab(tab.id); }}
                    className={`flex items-center gap-2 px-4 py-2.5 text-[11px] font-mono-ui uppercase tracking-wider transition-all duration-200 focus-ring rounded-sm ${
                      isActive
                        ? 'bg-ink text-parchment shadow-wax'
                        : 'text-ink-muted hover:text-ink hover:bg-ink/5'
                    }`}
                  >
                    <Icon className="w-3.5 h-3.5" strokeWidth={2} />
                    <span>{tab.label}</span>
                  </button>
                );
              })}
            </nav>
          </div>

          {/* Divider with gold accent */}
          <hr className="divider-gold" />
        </div>
      </header>

      {/* Main Content - The Ledger */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-6 py-8">
        {activeTab === 'overview' && <Overview onSelectDevice={handleSelectDevice} />}
        {activeTab === 'detail' && selectedServerId && <DeviceDetail serverId={selectedServerId} onBack={handleBackToOverview} onRenamed={handleRenamed} />}
        {activeTab === 'assets' && <AssetInventory />}
        {activeTab === 'apps' && <Applications />}
      </main>

      {/* Footer - The Seal */}
      <footer className="border-t border-ink-muted/30 bg-parchment-dark py-6 px-6">
        <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-4 text-center md:text-left">
          <p className="font-mono-ui text-[11px] text-ink-muted uppercase tracking-wider m-0">
            SENESCHAL Estate Steward &copy; 2026 PT Tata Optima Property
          </p>
          <p className="font-mono-ui text-[11px] text-ink-muted m-0">
            IT Programme Coordinator &middot; Bagues Putra Tawaqqal
          </p>
        </div>
      </footer>
    </div>
  );
}