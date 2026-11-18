import React, { useEffect, useRef, useState } from 'react';
import { Activity, HardDrive, Package, Crown, BookOpen, Wrench, ShieldAlert, Menu, X, ChevronRight } from 'lucide-react';
import Overview from './components/Overview';
import DeviceDetail from './components/DeviceDetail';
import AssetInventory from './components/AssetInventory';
import Applications from './components/Applications';
import UpgradeAdvisor from './components/UpgradeAdvisor';
import PolicyAudit from './components/PolicyAudit';
import ApiReference from './components/ApiReference';

const tabs = [
  { id: 'overview', label: 'Ringkasan', icon: Activity, desc: 'Kondisi infrastruktur' },
  { id: 'assets', label: 'Inventaris', icon: HardDrive, desc: 'Perangkat & kepemilikan' },
  { id: 'upgrades', label: 'Rekomendasi', icon: Wrench, desc: 'Kebutuhan upgrade' },
  { id: 'apps', label: 'Aplikasi', icon: Package, desc: 'Software terpasang' },
  { id: 'policy', label: 'Kebijakan & Audit', icon: ShieldAlert, desc: 'Lisensi & kepatuhan' },
  { id: 'api', label: 'Referensi API', icon: BookOpen, desc: 'Dokumentasi integrasi' },
];

export default function App() {
  const [activeTab, setActiveTab] = useState('overview');
  const [selectedServerId, setSelectedServerId] = useState(null);
  const [origin, setOrigin] = useState('overview');
  const [menuOpen, setMenuOpen] = useState(false);
  const returnScroll = useRef(0);
  const returnFocus = useRef(null);
  const sidebar = useRef(null);
  const menuButton = useRef(null);
  const currentTab = tabs.find(tab => tab.id === (activeTab === 'detail' ? origin : activeTab));

  useEffect(() => {
    if (!menuOpen) return;
    const previousOverflow = document.body.style.overflow;
    const trigger = menuButton.current;
    document.body.style.overflow = 'hidden';
    const focusFrame = requestAnimationFrame(() => sidebar.current?.querySelector('button').focus());
    const closeOnEscape = event => { if (event.key === 'Escape') setMenuOpen(false); };
    const closeOnDesktop = () => { if (window.innerWidth >= 1024) setMenuOpen(false); };
    window.addEventListener('resize', closeOnDesktop);
    document.addEventListener('keydown', closeOnEscape);
    return () => {
      document.body.style.overflow = previousOverflow;
      cancelAnimationFrame(focusFrame);
      window.removeEventListener('resize', closeOnDesktop);
      document.removeEventListener('keydown', closeOnEscape);
      trigger?.focus();
    };
  }, [menuOpen]);

  const navigate = (id) => {
    setActiveTab(id);
    setSelectedServerId(null);
    setMenuOpen(false);
    window.scrollTo(0, 0);
  };

  const handleSelectDevice = (id) => {
    returnScroll.current = window.scrollY;
    returnFocus.current = document.activeElement;
    setOrigin(activeTab);
    setSelectedServerId(id);
    setActiveTab('detail');
    window.scrollTo(0, 0);
  };

  const handleBack = () => {
    setSelectedServerId(null);
    setActiveTab(origin);
    requestAnimationFrame(() => {
      window.scrollTo(0, returnScroll.current);
      returnFocus.current?.focus({ preventScroll: true });
    });
  };

  const trapMenuFocus = (event) => {
    if (!menuOpen) return;
    if (event.key === 'Escape') setMenuOpen(false);
    if (event.key !== 'Tab') return;
    const buttons = [...sidebar.current.querySelectorAll('button')];
    const first = buttons[0];
    const last = buttons.at(-1);
    if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
    if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
  };

  return (
    <div className="app-shell">
      {menuOpen && <div className="sidebar-scrim" onClick={() => setMenuOpen(false)} aria-hidden="true" />}
      <aside id="main-navigation" ref={sidebar} className={`sidebar ${menuOpen ? 'is-open' : ''}`} onKeyDown={trapMenuFocus} role={menuOpen ? 'dialog' : undefined} aria-modal={menuOpen || undefined} aria-label="Navigasi utama">
        <div className="sidebar-brand">
          <span className="brand-icon"><Crown size={24} strokeWidth={1.6} /></span>
          <div><div className="brand-name">SENESCHAL</div><div className="brand-subtitle">Infrastructure monitor</div></div>
          <button className="sidebar-close" onClick={() => setMenuOpen(false)} aria-label="Tutup navigasi"><X size={20} /></button>
        </div>
        <p className="sidebar-caption">Workspace</p>
        <nav className="sidebar-nav" aria-label="Menu utama">
          {tabs.map(({ id, label, icon: Icon, desc }) => (
            <button key={id} className={`nav-item ${currentTab.id === id ? 'is-active' : ''}`} onClick={() => navigate(id)} aria-current={currentTab.id === id ? 'page' : undefined}>
              <Icon size={20} strokeWidth={1.7} /><span className="nav-copy"><span>{label}</span><span className="nav-description">{desc}</span></span>
            </button>
          ))}
        </nav>
        <div className="sidebar-footer"><span className="brand-icon"><ShieldAlert size={19} /></span><div>PT Tata Optima Property<br /><span>IT Operations</span></div></div>
      </aside>
      <div className="app-workspace" inert={menuOpen ? true : undefined}>
        <header className="app-topbar">
          <button ref={menuButton} className="mobile-menu button-icon" onClick={() => setMenuOpen(true)} aria-expanded={menuOpen} aria-controls="main-navigation" aria-label="Buka navigasi"><Menu size={22} /></button>
          <nav className="breadcrumb" aria-label="Breadcrumb">
            <span>Workspace</span><ChevronRight size={14} />
            {activeTab === 'detail' ? <><button onClick={handleBack}>{currentTab.label}</button><ChevronRight size={14} /><strong>{selectedServerId}</strong></> : <strong>{currentTab.label}</strong>}
          </nav>
          <span className="topbar-status"><Activity size={16} /> Infrastructure monitoring</span>
        </header>
        <main id="main-content" className="app-main">
          <div hidden={activeTab !== 'overview'}><Overview onSelectDevice={handleSelectDevice} active={activeTab === 'overview'} /></div>
          {activeTab === 'detail' && selectedServerId && <DeviceDetail serverId={selectedServerId} onBack={handleBack} onRenamed={setSelectedServerId} />}
          {activeTab === 'assets' && <AssetInventory />}
          <div hidden={activeTab !== 'upgrades'}>{(activeTab === 'upgrades' || (activeTab === 'detail' && origin === 'upgrades')) && <UpgradeAdvisor onSelectDevice={handleSelectDevice} />}</div>
          {activeTab === 'apps' && <Applications />}
          <div hidden={activeTab !== 'policy'}>{(activeTab === 'policy' || (activeTab === 'detail' && origin === 'policy')) && <PolicyAudit onSelectDevice={handleSelectDevice} />}</div>
          {activeTab === 'api' && <ApiReference />}
        </main>
        <footer className="app-footer"><span>Seneschal · Infrastructure & asset management</span><span>© 2026 PT Tata Optima Property</span></footer>
      </div>
    </div>
  );
}
