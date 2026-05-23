import { useState, useEffect } from 'preact/hooks';
import { Tree } from './components/Tree';
import { FragmentTree } from './components/FragmentTree';
import { Details } from './components/Details';
import { findNodeByIdInPayload } from './utils';
import './app.css';

interface Payload {
  dom: any;
  overlays?: any[];
  fragments?: any;
  overlayFragments?: any[];
}

export function App() {
  const [payload, setPayload] = useState<Payload | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [view, setView] = useState<'dom' | 'fragments'>('dom');
  const [sidebarWidth, setSidebarWidth] = useState(450);
  const [isResizing, setIsResizing] = useState(false);

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return;
      const newWidth = Math.max(200, Math.min(e.clientX, window.innerWidth * 0.7));
      setSidebarWidth(newWidth);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
    };

    if (isResizing) {
      window.addEventListener('mousemove', handleMouseMove);
      window.addEventListener('mouseup', handleMouseUp);
    }

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isResizing]);

  useEffect(() => {
    const evtSource = new EventSource("/stream");
    evtSource.onmessage = (event) => {
      const data = JSON.parse(event.data);
      setPayload(data);
    };
    return () => evtSource.close();
  }, []);

  const selectedNode = selectedId && payload ? findNodeByIdInPayload(payload, selectedId) : null;

  const downloadDump = () => {
    if (!payload) return;
    const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `kite-dump-${new Date().toISOString()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div class={`inspector-container ${isResizing ? 'resizing' : ''}`}>
      <aside class="sidebar" style={{ width: `${sidebarWidth}px` }}>
        <header class="sidebar-header">
          <div class="header-top">
            <div class="view-toggle">
              <button 
                class={view === 'dom' ? 'active' : ''} 
                onClick={() => setView('dom')}
              >
                DOM
              </button>
              <button 
                class={view === 'fragments' ? 'active' : ''} 
                onClick={() => setView('fragments')}
              >
                Fragments
              </button>
            </div>
            <button class="icon-btn" onClick={downloadDump} title="Download Dump">
              ⬇️
            </button>
          </div>
        </header>
        <main id="tree">
          {payload ? (
            view === 'dom' ? (
              <Tree 
                node={payload.dom} 
                overlays={payload.overlays} 
                selectedId={selectedId} 
                onSelect={setSelectedId} 
              />
            ) : (
              <FragmentTree 
                fragment={payload.fragments} 
                overlayFragments={payload.overlayFragments} 
              />
            )
          ) : (
            <div class="loading">Connecting to engine...</div>
          )}
        </main>
      </aside>
      <div 
        class="resizer" 
        onMouseDown={() => setIsResizing(true)}
      />
      <main id="details">
        <Details node={selectedNode} />
      </main>
    </div>
  );
}
