import { useState, useEffect } from 'preact/hooks';
import { Tree } from './components/Tree';
import { FragmentTree } from './components/FragmentTree';
import { Details } from './components/Details';
import { ComponentsTree } from './components/ComponentsTree';
import { findNodeByIdInPayload, findVDOMNodeById } from './utils';
import { 
  ProfilerSidebar, 
  ProfilerFlamechart, 
  parseTraceSpans 
} from './components/Profiler';
import type { Span } from './components/Profiler';
import './app.css';

interface Payload {
  dom: any;
  overlays?: any[];
  fragments?: any;
  overlayFragments?: any[];
  extensions?: {
    kitex?: any[];
  };
}

export function App() {
  const [payload, setPayload] = useState<Payload | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [view, setView] = useState<'dom' | 'fragments' | 'profiler' | 'components'>('dom');
  const [sidebarWidth, setSidebarWidth] = useState(450);
  const [isResizing, setIsResizing] = useState(false);

  // Profiler State
  const [isProfiling, setIsProfiling] = useState(false);
  const [traceEvents, setTraceEvents] = useState<any[] | null>(null);
  const [selectedSpan, setSelectedSpan] = useState<Span | null>(null);
  const [hoveredJobId, setHoveredJobId] = useState<string | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  // Zoom and Timeframe Filter States
  const [zoom, setZoom] = useState(1);
  const [filterStart, setFilterStart] = useState(0);
  const [filterEnd, setFilterEnd] = useState(0);

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

  let selectedNode = null;
  if (selectedId && payload) {
    if (view === 'components') {
      selectedNode = findVDOMNodeById(payload.extensions?.kitex, selectedId);
    } else {
      selectedNode = findNodeByIdInPayload(payload, selectedId);
    }
  }

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

  // Reconstruct spans from trace events
  const spans = traceEvents ? parseTraceSpans(traceEvents) : [];
  const minTs = spans.length > 0 ? Math.min(...spans.map(s => s.start)) : 0;
  const maxTs = spans.length > 0 ? Math.max(...spans.map(s => s.end)) : 0;
  const totalDuration = maxTs - minTs;

  // Sync zoom and range filters when traceEvents is updated
  useEffect(() => {
    if (traceEvents && spans.length > 0) {
      const durationMs = totalDuration / 1000;
      setFilterStart(0);
      setFilterEnd(durationMs);
      setZoom(1);
    }
  }, [traceEvents]);

  // Group spans by thread ID
  const spansByTid: { [tid: number]: Span[] } = {};
  for (const span of spans) {
    if (!spansByTid[span.tid]) {
      spansByTid[span.tid] = [];
    }
    spansByTid[span.tid].push(span);
  }

  const sortedTids = Object.keys(spansByTid)
    .map(Number)
    .sort((a, b) => {
      if (a === 1) return -1;
      if (b === 1) return 1;
      return a - b;
    });

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
              {payload?.extensions?.kitex && (
                <button 
                  class={view === 'components' ? 'active' : ''} 
                  onClick={() => setView('components')}
                >
                  Components
                </button>
              )}
              <button 
                class={view === 'fragments' ? 'active' : ''} 
                onClick={() => setView('fragments')}
              >
                Fragments
              </button>
              <button 
                class={view === 'profiler' ? 'active' : ''} 
                onClick={() => setView('profiler')}
              >
                Profiler
              </button>
            </div>
            {view !== 'profiler' && (
              <button class="icon-btn" onClick={downloadDump} title="Download Dump">
                ⬇️
              </button>
            )}
          </div>
        </header>
        <main id="tree">
          {view === 'profiler' ? (
            <ProfilerSidebar
              isProfiling={isProfiling}
              setIsProfiling={setIsProfiling}
              traceEvents={traceEvents}
              setTraceEvents={setTraceEvents}
              selectedSpan={selectedSpan}
              setSelectedSpan={setSelectedSpan}
              errorMsg={errorMsg}
              setErrorMsg={setErrorMsg}
              spans={spans}
              minTs={minTs}
              maxTs={maxTs}
              zoom={zoom}
              setZoom={setZoom}
              filterStart={filterStart}
              setFilterStart={setFilterStart}
              filterEnd={filterEnd}
              setFilterEnd={setFilterEnd}
              clearProfile={() => {
                setTraceEvents(null);
                setSelectedSpan(null);
                setHoveredJobId(null);
                setErrorMsg(null);
                setZoom(1);
                setFilterStart(0);
                setFilterEnd(0);
              }}
            />
          ) : payload ? (
            view === 'dom' ? (
              <Tree 
                node={payload.dom} 
                overlays={payload.overlays} 
                selectedId={selectedId} 
                onSelect={setSelectedId} 
              />
            ) : view === 'components' ? (
              <ComponentsTree 
                roots={payload.extensions?.kitex || []}
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
        {view === 'profiler' ? (
          <ProfilerFlamechart
            spans={spans}
            minTs={minTs}
            spansByTid={spansByTid}
            sortedTids={sortedTids}
            hoveredJobId={hoveredJobId}
            setHoveredJobId={setHoveredJobId}
            selectedSpan={selectedSpan}
            setSelectedSpan={setSelectedSpan}
            isProfiling={isProfiling}
            zoom={zoom}
            setZoom={setZoom}
            filterStart={filterStart}
            setFilterStart={setFilterStart}
            filterEnd={filterEnd}
            setFilterEnd={setFilterEnd}
          />
        ) : (
          <Details 
            node={selectedNode} 
            onJumpToElement={(domUniqueId) => {
              setView('dom');
              setSelectedId(domUniqueId);
            }} 
          />
        )}
      </main>
    </div>
  );
}
