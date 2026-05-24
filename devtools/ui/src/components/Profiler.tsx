import { useState, useRef, useEffect } from 'preact/hooks';

export interface TraceEvent {
  name: string;
  ph: 'B' | 'E';
  ts: number; // microseconds
  pid: number;
  tid: number;
}

export interface Span {
  name: string;
  tid: number;
  start: number; // microseconds
  end: number;   // microseconds
  duration: number; // microseconds
  depth: number;
  jobId?: string;
  jobType?: string;
}

export interface ProfilerSidebarProps {
  isProfiling: boolean;
  setIsProfiling: (v: boolean) => void;
  traceEvents: TraceEvent[] | null;
  setTraceEvents: (v: TraceEvent[] | null) => void;
  selectedSpan: Span | null;
  setSelectedSpan: (s: Span | null) => void;
  errorMsg: string | null;
  setErrorMsg: (s: string | null) => void;
  spans: Span[];
  minTs: number;
  maxTs: number;
  clearProfile: () => void;

  // Zoom and Filter Range Props
  zoom: number;
  setZoom: (z: number) => void;
  filterStart: number;
  setFilterStart: (v: number) => void;
  filterEnd: number;
  setFilterEnd: (v: number) => void;
}

export function ProfilerSidebar({
  isProfiling,
  setIsProfiling,
  traceEvents,
  setTraceEvents,
  selectedSpan,
  setSelectedSpan,
  errorMsg,
  setErrorMsg,
  spans,
  minTs,
  maxTs,
  clearProfile,
  zoom,
  setZoom,
  filterStart,
  setFilterStart,
  filterEnd,
  setFilterEnd,
}: ProfilerSidebarProps) {
  const startProfiling = async () => {
    try {
      setErrorMsg(null);
      const res = await fetch('/debug/trace/start', { method: 'POST' });
      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
      setIsProfiling(true);
      setTraceEvents(null);
      setSelectedSpan(null);
    } catch (err: any) {
      setErrorMsg(`Failed to start profiling: ${err.message}`);
    }
  };

  const stopProfiling = async () => {
    try {
      setErrorMsg(null);
      const res = await fetch('/debug/trace/stop', { method: 'POST' });
      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
      const data = await res.json();
      setTraceEvents(data);
      setIsProfiling(false);
    } catch (err: any) {
      setErrorMsg(`Failed to stop profiling: ${err.message}`);
      setIsProfiling(false);
    }
  };

  const downloadTrace = () => {
    if (!traceEvents) return;
    const blob = new Blob([JSON.stringify(traceEvents, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `kite-trace-${new Date().toISOString()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  // Helper to find initiating Frame for a Job ID
  const findInitiatingFrame = (jobId: string): Span | null => {
    const submitSpan = spans.find(s => s.tid === 1 && s.name.startsWith('JobSubmit:') && s.jobId === jobId);
    if (!submitSpan) return null;
    return spans.find(s => s.tid === 1 && s.name === 'Frame' && s.start <= submitSpan.start && s.end >= submitSpan.end) || null;
  };

  // Helper to find JobRun span for a Job ID
  const findJobRunSpan = (jobId: string): Span | null => {
    return spans.find(s => s.tid > 1 && s.name.startsWith('JobRun:') && s.jobId === jobId) || null;
  };

  // Helper to find all jobs submitted during a Frame
  const findJobsForFrame = (frame: Span): Span[] => {
    return spans.filter(s => s.tid === 1 && s.name.startsWith('JobSubmit:') && s.start >= frame.start && s.end <= frame.end);
  };

  const totalDurationMs = (maxTs - minTs) / 1000;

  return (
    <div class="profiler-sidebar">
      <div class="sidebar-section">
        <h3>Profiler Controls</h3>
        <div class="control-buttons">
          {!isProfiling ? (
            <button class="btn btn-start" onClick={startProfiling}>
              🔴 Start Profiling
            </button>
          ) : (
            <button class="btn btn-stop" onClick={stopProfiling}>
              ⏹️ Stop & Dump
            </button>
          )}
          {traceEvents && (
            <>
              <button class="btn btn-download" onClick={downloadTrace}>
                ⬇️ Download Trace
              </button>
              <button class="btn btn-clear" onClick={clearProfile}>
                🗑️ Clear
              </button>
            </>
          )}
        </div>
        {isProfiling && (
          <div class="profiling-status">
            <span class="pulse-dot"></span> Profiling active. Interact with terminal to trace frames...
          </div>
        )}
        {errorMsg && <div class="error-msg">{errorMsg}</div>}
      </div>

      {traceEvents && spans.length > 0 && (
        <div class="sidebar-section">
          <h3>Zoom & Timeframe</h3>
          <div class="zoom-controls">
            <div class="control-row-flex">
              <span class="control-label">Zoom: {zoom.toFixed(1)}x</span>
              <input 
                type="range" 
                min="1" 
                max="20" 
                step="0.5"
                value={zoom} 
                onInput={(e) => setZoom(parseFloat((e.target as HTMLInputElement).value))}
                class="range-slider"
              />
            </div>
            <div class="zoom-buttons">
              <button class="btn btn-zoom" onClick={() => setZoom(Math.max(1, zoom - 1))}>➖ Out</button>
              <button class="btn btn-zoom" onClick={() => setZoom(Math.min(20, zoom + 1))}>➕ In</button>
            </div>
          </div>

          <div class="range-controls">
            <div class="control-row-flex">
              <span class="control-label">Start Time: {filterStart.toFixed(1)} ms</span>
              <input 
                type="range" 
                min="0" 
                max={totalDurationMs} 
                step={Math.max(0.1, totalDurationMs / 100)}
                value={filterStart} 
                onInput={(e) => {
                  const val = parseFloat((e.target as HTMLInputElement).value);
                  setFilterStart(Math.min(val, filterEnd - 0.1));
                }}
                class="range-slider"
              />
            </div>
            <div class="control-row-flex">
              <span class="control-label">End Time: {filterEnd.toFixed(1)} ms</span>
              <input 
                type="range" 
                min="0" 
                max={totalDurationMs} 
                step={Math.max(0.1, totalDurationMs / 100)}
                value={filterEnd} 
                onInput={(e) => {
                  const val = parseFloat((e.target as HTMLInputElement).value);
                  setFilterEnd(Math.max(val, filterStart + 0.1));
                }}
                class="range-slider"
              />
            </div>
            <button class="btn btn-reset-range" onClick={() => {
              setFilterStart(0);
              setFilterEnd(totalDurationMs);
            }}>
              🔄 Reset Timeframe
            </button>
          </div>
        </div>
      )}

      {selectedSpan ? (
        <div class="sidebar-section detail-card">
          <h3>Event Details</h3>
          <div class="detail-row">
            <span class="detail-label">Name:</span>
            <span class="detail-value event-name-val">{selectedSpan.name}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Thread:</span>
            <span class="detail-value">
              {selectedSpan.tid === 1 ? 'Main Thread (TID 1)' : `Worker Thread (TID ${selectedSpan.tid})`}
            </span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Duration:</span>
            <span class="detail-value">{(selectedSpan.duration / 1000).toFixed(3)} ms</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Start Time:</span>
            <span class="detail-value">{((selectedSpan.start - minTs) / 1000).toFixed(3)} ms</span>
          </div>

          {/* Job Relations */}
          {selectedSpan.jobId && (
            <div class="job-relations">
              <h4>Job Relations</h4>
              <div class="detail-row">
                <span class="detail-label">Job ID:</span>
                <span class="detail-value">{selectedSpan.jobId}</span>
              </div>
              {selectedSpan.name.startsWith('JobRun:') ? (
                (() => {
                  const frame = findInitiatingFrame(selectedSpan.jobId!);
                  return frame ? (
                    <div class="relation-link" onClick={() => setSelectedSpan(frame)}>
                      🔗 Jump to Initiating Frame ({((frame.start - minTs) / 1000).toFixed(1)}ms)
                    </div>
                  ) : (
                    <div class="relation-none">No initiating frame found</div>
                  );
                })()
              ) : (
                (() => {
                  const runSpan = findJobRunSpan(selectedSpan.jobId!);
                  return runSpan ? (
                    <div class="relation-link" onClick={() => setSelectedSpan(runSpan)}>
                      🔗 Jump to Worker Run (TID {runSpan.tid}, {(runSpan.duration / 1000).toFixed(1)}ms)
                    </div>
                  ) : (
                    <div class="relation-none">Job has not run or finished yet</div>
                  );
                })()
              )}
            </div>
          )}

          {/* Frame Jobs */}
          {selectedSpan.name === 'Frame' && (() => {
            const jobs = findJobsForFrame(selectedSpan);
            return jobs.length > 0 ? (
              <div class="frame-jobs">
                <h4>Submitted Background Jobs</h4>
                <ul class="jobs-list">
                  {jobs.map(job => (
                    <li 
                      key={job.jobId} 
                      class="job-item-link"
                      onClick={() => {
                        const runSpan = findJobRunSpan(job.jobId!);
                        if (runSpan) setSelectedSpan(runSpan);
                      }}
                    >
                      ⚡ {job.jobType || 'Job'} ({job.jobId})
                    </li>
                  ))}
                </ul>
              </div>
            ) : null;
          })()}
        </div>
      ) : (
        <div class="sidebar-section select-placeholder">
          Click on a phase block or background job in the flamechart to inspect details.
        </div>
      )}
    </div>
  );
}

export interface ProfilerFlamechartProps {
  spans: Span[];
  minTs: number;
  spansByTid: { [tid: number]: Span[] };
  sortedTids: number[];
  hoveredJobId: string | null;
  setHoveredJobId: (id: string | null) => void;
  selectedSpan: Span | null;
  setSelectedSpan: (s: Span | null) => void;
  isProfiling: boolean;

  // Zoom and Timeframe Filter Props
  zoom: number;
  setZoom: (z: number) => void;
  filterStart: number;
  setFilterStart: (v: number) => void;
  filterEnd: number;
  setFilterEnd: (v: number) => void;
}

interface CallTreeNode {
  name: string;
  selfMs: number;
  totalMs: number;
  count: number;
  children: CallTreeNode[];
}

export function ProfilerFlamechart({
  spans,
  minTs,
  spansByTid,
  sortedTids,
  hoveredJobId,
  setHoveredJobId,
  selectedSpan,
  setSelectedSpan,
  isProfiling,
  zoom,
  setZoom,
  filterStart,
  setFilterStart,
  filterEnd,
  setFilterEnd,
}: ProfilerFlamechartProps) {
  // Local Tooltip State
  const [tooltip, setTooltip] = useState<{ x: number; y: number; span: Span } | null>(null);

  // Bottom Panel State
  const [bottomTab, setBottomTab] = useState<'summary' | 'bottom-up' | 'call-tree' | 'event-log'>('summary');
  const [searchQuery, setSearchQuery] = useState('');
  const [expandedPaths, setExpandedPaths] = useState<Record<string, boolean>>({});
  const [sortCol, setSortCol] = useState<'name' | 'selfMs' | 'totalMs' | 'count'>('selfMs');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');

  // Timeframe selection drag state
  const zoomWrapperRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const [dragRange, setDragRange] = useState<{ startPct: number; endPct: number } | null>(null);
  const isDraggingRef = useRef<boolean>(false);
  const dragStartXPctRef = useRef<number>(0);

  if (spans.length === 0) {
    return (
      <div class="empty-flamechart">
        {isProfiling ? (
          <div class="status-msg text-pulse">Recording profiling data...</div>
        ) : (
          <div class="status-msg">No trace data. Click "Start Profiling" to begin capture.</div>
        )}
      </div>
    );
  }

  // Calculate visible range parameters
  const activeDurationMs = filterEnd - filterStart;
  const rangeStartUs = minTs + filterStart * 1000;
  const rangeEndUs = minTs + filterEnd * 1000;
  const rangeDurationUs = activeDurationMs * 1000 || 1;

  // Helper to convert mouse event clientX to percentage of the zoom wrapper
  const getPctFromEvent = (e: MouseEvent): number | null => {
    if (!zoomWrapperRef.current) return null;
    const rect = zoomWrapperRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left;
    return Math.max(0, Math.min(100, (x / rect.width) * 100));
  };

  // Mouse wheel zoom handling inside scroll container
  useEffect(() => {
    const scrollContainer = scrollContainerRef.current;
    if (!scrollContainer) return;

    const handleWheel = (e: WheelEvent) => {
      if (e.ctrlKey || e.metaKey) {
        e.preventDefault();
        
        const rect = scrollContainer.getBoundingClientRect();
        const mouseX = e.clientX - rect.left;
        const scrollLeft = scrollContainer.scrollLeft;
        
        // Horizontal position in the zoom-wrapper (scaled coordinates)
        const contentX = scrollLeft + mouseX;
        
        const zoomDelta = e.deltaY < 0 ? 1.15 : 0.85;
        const nextZoom = Math.min(20, Math.max(1, zoom * zoomDelta));
        
        if (nextZoom !== zoom) {
          const ratio = nextZoom / zoom;
          const targetScrollLeft = contentX * ratio - mouseX;
          
          setZoom(nextZoom);
          requestAnimationFrame(() => {
            if (scrollContainerRef.current) {
              scrollContainerRef.current.scrollLeft = targetScrollLeft;
            }
          });
        }
      }
    };

    scrollContainer.addEventListener('wheel', handleWheel, { passive: false });
    return () => {
      scrollContainer.removeEventListener('wheel', handleWheel);
    };
  }, [zoom, setZoom]);

  // Drag select timeframe handling
  useEffect(() => {
    if (!dragRange) return;

    const handleGlobalMouseMove = (e: MouseEvent) => {
      if (!isDraggingRef.current) return;
      const pct = getPctFromEvent(e);
      if (pct === null) return;
      setDragRange(prev => prev ? { ...prev, endPct: pct } : null);
    };

    const handleGlobalMouseUp = (e: MouseEvent) => {
      if (!isDraggingRef.current) return;
      isDraggingRef.current = false;
      
      const pct = getPctFromEvent(e);
      if (pct !== null && dragRange) {
        const startPct = dragStartXPctRef.current;
        const endPct = pct;
        
        if (Math.abs(endPct - startPct) > 0.5) {
          const t1 = filterStart + (startPct / 100) * activeDurationMs;
          const t2 = filterStart + (endPct / 100) * activeDurationMs;
          const newStart = Math.min(t1, t2);
          const newEnd = Math.max(t1, t2);
          
          if (newEnd - newStart > 0.1) {
            setFilterStart(newStart);
            setFilterEnd(newEnd);
          }
        }
      }
      setDragRange(null);
    };

    window.addEventListener('mousemove', handleGlobalMouseMove);
    window.addEventListener('mouseup', handleGlobalMouseUp);
    return () => {
      window.removeEventListener('mousemove', handleGlobalMouseMove);
      window.removeEventListener('mouseup', handleGlobalMouseUp);
    };
  }, [dragRange, filterStart, filterEnd, activeDurationMs]);

  const handleRulerMouseDown = (e: MouseEvent) => {
    const pct = getPctFromEvent(e);
    if (pct === null) return;
    isDraggingRef.current = true;
    dragStartXPctRef.current = pct;
    setDragRange({ startPct: pct, endPct: pct });
    e.preventDefault();
  };

  const handleRulerDoubleClick = () => {
    setFilterStart(0);
    const totalDurationMs = spans.length > 0 ? (Math.max(...spans.map(s => s.end)) - Math.min(...spans.map(s => s.start))) / 1000 : 0;
    setFilterEnd(totalDurationMs);
    setZoom(1);
  };

  const handleMouseMove = (e: MouseEvent, span: Span) => {
    setTooltip({
      x: e.clientX + 12,
      y: e.clientY + 12,
      span,
    });
  };

  const handleMouseLeave = () => {
    setTooltip(null);
  };

  const formatDuration = (us: number) => {
    if (us < 1000) {
      return `${us.toFixed(0)} μs`;
    }
    return `${(us / 1000).toFixed(3)} ms`;
  };

  // Helper to compute overlap of a span with the selected range
  const getSpanRangeOverlap = (s: Span) => {
    const sStart = Math.max(s.start, rangeStartUs);
    const sEnd = Math.min(s.end, rangeEndUs);
    if (sEnd <= sStart) return { start: 0, end: 0, duration: 0 };
    return { start: sStart, end: sEnd, duration: sEnd - sStart };
  };

  // Compute self and overlap durations for all overlapping spans in range
  const analyzedSpans = spans.map((s, idx) => {
    const overlap = getSpanRangeOverlap(s);
    return {
      span: s,
      idx,
      overlapStart: overlap.start,
      overlapEnd: overlap.end,
      overlapDuration: overlap.duration,
      selfDuration: overlap.duration,
    };
  }).filter(item => item.overlapDuration > 0);

  // Subtract direct child durations to calculate self time for each span in range
  for (const parent of analyzedSpans) {
    let childSum = 0;
    for (const child of analyzedSpans) {
      if (
        child.span.tid === parent.span.tid &&
        child.span.start >= parent.span.start &&
        child.span.end <= parent.span.end &&
        child.span.depth === parent.span.depth + 1
      ) {
        childSum += child.overlapDuration;
      }
    }
    parent.selfDuration = Math.max(0, parent.overlapDuration - childSum);
  }

  // 1. Calculate Main-Thread Summary Categories
  let renderingUs = 0;
  let layoutUs = 0;
  let paintingUs = 0;
  let systemUs = 0;
  let totalActiveUs = 0;

  for (const item of analyzedSpans) {
    if (item.span.tid === 1) {
      const name = item.span.name;
      const selfDur = item.selfDuration;
      totalActiveUs += selfDur;

      if (name.startsWith('Phase:Sync') || name.startsWith('Phase:Tasks') || name.startsWith('Phase:Style')) {
        renderingUs += selfDur;
      } else if (name.startsWith('Phase:Layout') || name.includes('Layout')) {
        layoutUs += selfDur;
      } else if (name.startsWith('Phase:Paint') || name.includes('Paint')) {
        paintingUs += selfDur;
      } else {
        systemUs += selfDur;
      }
    }
  }

  const idleUs = Math.max(0, rangeDurationUs - totalActiveUs);

  const renderingMs = renderingUs / 1000;
  const layoutMs = layoutUs / 1000;
  const paintingMs = paintingUs / 1000;
  const systemMs = systemUs / 1000;
  const idleMs = idleUs / 1000;

  const renderingPct = (renderingUs / rangeDurationUs) * 100;
  const layoutPct = (layoutUs / rangeDurationUs) * 100;
  const paintingPct = (paintingUs / rangeDurationUs) * 100;
  const systemPct = (systemUs / rangeDurationUs) * 100;
  const idlePct = (idleUs / rangeDurationUs) * 100;

  // 2. Compute Bottom-Up statistics
  const bottomUpMap = new Map<string, { name: string; selfUs: number; totalUs: number; count: number }>();
  for (const item of analyzedSpans) {
    const name = item.span.name;
    const existing = bottomUpMap.get(name) || { name, selfUs: 0, totalUs: 0, count: 0 };
    existing.selfUs += item.selfDuration;
    existing.totalUs += item.overlapDuration;
    existing.count += 1;
    bottomUpMap.set(name, existing);
  }

  const bottomUpData = Array.from(bottomUpMap.values()).map(item => ({
    name: item.name,
    selfMs: item.selfUs / 1000,
    totalMs: item.totalUs / 1000,
    selfPct: (item.selfUs / rangeDurationUs) * 100,
    totalPct: (item.totalUs / rangeDurationUs) * 100,
    count: item.count,
  }));

  const sortedBottomUp = [...bottomUpData].sort((a, b) => {
    let valA = a[sortCol];
    let valB = b[sortCol];
    if (typeof valA === 'string') {
      return sortDir === 'asc' ? valA.localeCompare(valB as string) : (valB as string).localeCompare(valA);
    }
    return sortDir === 'asc' ? (valA as number) - (valB as number) : (valB as number) - (valA as number);
  });

  // 3. Construct and aggregate Call Tree
  const buildCallTree = (): CallTreeNode[] => {
    const mainThreadItems = analyzedSpans.filter(item => item.span.tid === 1);
    
    const roots = mainThreadItems.filter(item => {
      return !mainThreadItems.some(parent => 
        parent !== item &&
        parent.span.start <= item.span.start &&
        parent.span.end >= item.span.end &&
        parent.span.depth < item.span.depth
      );
    });

    const buildNode = (item: typeof analyzedSpans[0]): CallTreeNode => {
      const childrenItems = mainThreadItems.filter(child => 
        child.span.start >= item.span.start &&
        child.span.end <= item.span.end &&
        child.span.depth === item.span.depth + 1
      );

      return {
        name: item.span.name,
        selfMs: item.selfDuration / 1000,
        totalMs: item.overlapDuration / 1000,
        count: 1,
        children: childrenItems.map(buildNode),
      };
    };

    const directRoots = roots.map(buildNode);

    const aggregateNodes = (nodes: CallTreeNode[]): CallTreeNode[] => {
      const map = new Map<string, CallTreeNode>();
      for (const node of nodes) {
        const existing = map.get(node.name);
        if (existing) {
          existing.selfMs += node.selfMs;
          existing.totalMs += node.totalMs;
          existing.count += node.count;
          existing.children.push(...node.children);
        } else {
          map.set(node.name, { ...node, children: [...node.children] });
        }
      }

      return Array.from(map.values()).map(node => {
        node.children = aggregateNodes(node.children);
        node.children.sort((a, b) => b.totalMs - a.totalMs);
        return node;
      });
    };

    const aggregated = aggregateNodes(directRoots);
    aggregated.sort((a, b) => b.totalMs - a.totalMs);
    return aggregated;
  };

  const callTree = buildCallTree();

  const renderCallTreeRows = (nodes: CallTreeNode[], parentPath: string = '', depth: number = 0): any[] => {
    const rows: any[] = [];
    for (const node of nodes) {
      const currentPath = parentPath ? `${parentPath}/${node.name}` : node.name;
      const isExpanded = !!expandedPaths[currentPath];
      const hasChildren = node.children.length > 0;
      
      const toggleExpand = (e: MouseEvent) => {
        setExpandedPaths(prev => ({ ...prev, [currentPath]: !prev[currentPath] }));
        e.stopPropagation();
      };

      rows.push(
        <tr key={currentPath}>
          <td style={{ paddingLeft: `${depth * 16 + 8}px`, fontFamily: 'var(--font-mono)' }}>
            {hasChildren && (
              <span class="tree-toggle-arrow" onClick={toggleExpand} style={{ marginRight: '6px', cursor: 'pointer', display: 'inline-block', width: '10px' }}>
                {isExpanded ? '▼' : '▶'}
              </span>
            )}
            {!hasChildren && <span style={{ marginRight: '6px', display: 'inline-block', width: '10px' }} />}
            {node.name}
          </td>
          <td class="num-cell">{node.selfMs.toFixed(3)} ms</td>
          <td class="num-cell">{((node.selfMs / (activeDurationMs || 1)) * 100).toFixed(1)}%</td>
          <td class="num-cell">{node.totalMs.toFixed(3)} ms</td>
          <td class="num-cell">{((node.totalMs / (activeDurationMs || 1)) * 100).toFixed(1)}%</td>
          <td class="num-cell">{node.count}</td>
        </tr>
      );

      if (hasChildren && isExpanded) {
        rows.push(...renderCallTreeRows(node.children, currentPath, depth + 1));
      }
    }
    return rows;
  };

  // 4. Filter Event Log
  const eventLogData = analyzedSpans.map(item => ({
    timeMs: (item.span.start - minTs) / 1000,
    durationMs: item.overlapDuration / 1000,
    selfMs: item.selfDuration / 1000,
    tid: item.span.tid,
    name: item.span.name,
    jobId: item.span.jobId,
  })).sort((a, b) => a.timeMs - b.timeMs);

  const filteredEventLog = eventLogData.filter(evt => 
    evt.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
    (evt.jobId && evt.jobId.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  return (
    <div class="flamechart-main">
      <div 
        ref={scrollContainerRef}
        class="flamechart-scroll"
      >
        {/* Outer container with width scaled by zoom multiplier */}
        <div 
          ref={zoomWrapperRef}
          class="flamechart-zoom-wrapper" 
          style={{ width: `${100 * zoom}%` }}
        >
          {dragRange && (
            <div 
              class="flamechart-selection-overlay" 
              style={{
                left: `${Math.min(dragRange.startPct, dragRange.endPct)}%`,
                width: `${Math.abs(dragRange.endPct - dragRange.startPct)}%`
              }}
            />
          )}

          <div 
            class="flamechart-ruler"
            onMouseDown={handleRulerMouseDown}
            onDblClick={handleRulerDoubleClick}
            title="Drag to zoom into timeframe. Double-click to reset."
          >
            {renderRulerMarkers(filterStart, filterEnd)}
          </div>

          <div class="flamechart-tracks">
            {sortedTids.map(tid => {
              const threadSpans = spansByTid[tid] || [];
              const maxDepth = Math.max(...threadSpans.map(s => s.depth), 0);
              const trackHeight = (maxDepth + 1) * 24;

              return (
                <div key={tid} class="thread-track-row" style={{ height: `${trackHeight + 35}px` }}>
                  <div class="thread-label">
                    {tid === 1 ? 'Main Thread (TID 1)' : `Worker Thread (TID ${tid})`}
                  </div>
                  <div class="thread-track-content" style={{ height: `${trackHeight}px` }}>
                    {threadSpans.map((span, idx) => {
                      const spanStartMs = (span.start - minTs) / 1000;
                      const spanEndMs = (span.end - minTs) / 1000;

                      // Filter out if completely outside the timeframe range
                      if (spanEndMs < filterStart || spanStartMs > filterEnd) {
                        return null;
                      }

                      // Crop positions dynamically within timeframe bounds
                      const visibleStartMs = Math.max(spanStartMs, filterStart);
                      const visibleEndMs = Math.min(spanEndMs, filterEnd);

                      const leftPct = ((visibleStartMs - filterStart) / activeDurationMs) * 100;
                      const widthPct = span.duration === 0 ? 0 : ((visibleEndMs - visibleStartMs) / activeDurationMs) * 100;
                      
                      let displayClass = 'span-block';
                      if (span.name.startsWith('Phase:Sync')) displayClass += ' phase-sync';
                      else if (span.name.startsWith('Phase:Tasks')) displayClass += ' phase-tasks';
                      else if (span.name.startsWith('Phase:Style')) displayClass += ' phase-style';
                      else if (span.name.startsWith('Phase:Layout')) displayClass += ' phase-layout';
                      else if (span.name.startsWith('Phase:Paint')) displayClass += ' phase-paint';
                      else if (span.name === 'Frame') displayClass += ' phase-frame';
                      else if (span.name.startsWith('JobSubmit:')) displayClass += ' job-submit-marker';
                      else if (span.name.startsWith('JobRun:')) displayClass += ' job-run-block';
                      else if (span.name.includes('Layout')) displayClass += ' sub-layout';
                      else if (span.name.includes('Paint')) displayClass += ' sub-paint';

                      const isHighlighted = hoveredJobId && span.jobId === hoveredJobId;
                      const isFaded = hoveredJobId && span.jobId !== hoveredJobId;
                      
                      if (isHighlighted) displayClass += ' highlight-job-active';
                      if (isFaded) displayClass += ' fade-out';
                      if (selectedSpan === span) displayClass += ' selected-span';

                      return (
                        <div
                          key={idx}
                          class={displayClass}
                          style={{
                            left: `${leftPct}%`,
                            width: span.duration === 0 ? 'auto' : `${widthPct}%`,
                            top: `${span.depth * 24}px`,
                          }}
                          onMouseEnter={() => span.jobId && setHoveredJobId(span.jobId)}
                          onMouseMove={(e) => handleMouseMove(e, span)}
                          onMouseLeave={() => {
                            handleMouseLeave();
                            if (span.jobId) setHoveredJobId(null);
                          }}
                          onClick={() => setSelectedSpan(span)}
                        >
                          <span class="span-label">
                            {span.name.startsWith('JobSubmit:') ? '◆ Submit' : span.name}
                          </span>
                        </div>
                      );
                    })}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* Floating Interactive Tooltip */}
      {tooltip && (
        <div 
          class="flamechart-tooltip" 
          style={{ left: `${tooltip.x}px`, top: `${tooltip.y}px` }}
        >
          <div class="tooltip-name">{tooltip.span.name}</div>
          <div class="tooltip-row">
            <span class="tooltip-label">Duration:</span>
            <span class="tooltip-value">{formatDuration(tooltip.span.duration)}</span>
          </div>
          <div class="tooltip-row">
            <span class="tooltip-label">Thread:</span>
            <span class="tooltip-value">
              {tooltip.span.tid === 1 ? 'Main Thread (TID 1)' : `Worker Thread (TID ${tooltip.span.tid})`}
            </span>
          </div>
          {tooltip.span.jobId && (
            <div class="tooltip-row">
              <span class="tooltip-label">Job ID:</span>
              <span class="tooltip-value">{tooltip.span.jobId}</span>
            </div>
          )}
        </div>
      )}

      {/* Bottom Panel */}
      <div class="profiler-bottom-panel">
        <div class="bottom-panel-header">
          <div class="bottom-panel-tabs">
            <button 
              class={`panel-tab ${bottomTab === 'summary' ? 'active' : ''}`}
              onClick={() => setBottomTab('summary')}
            >
              📊 Summary
            </button>
            <button 
              class={`panel-tab ${bottomTab === 'bottom-up' ? 'active' : ''}`}
              onClick={() => setBottomTab('bottom-up')}
            >
              ⏳ Bottom-Up
            </button>
            <button 
              class={`panel-tab ${bottomTab === 'call-tree' ? 'active' : ''}`}
              onClick={() => setBottomTab('call-tree')}
            >
              🌳 Call Tree
            </button>
            <button 
              class={`panel-tab ${bottomTab === 'event-log' ? 'active' : ''}`}
              onClick={() => setBottomTab('event-log')}
            >
              📋 Event Log
            </button>
          </div>
          {bottomTab === 'event-log' && (
            <div class="panel-search">
              <input 
                type="text" 
                placeholder="Filter events..." 
                value={searchQuery}
                onInput={(e) => setSearchQuery((e.target as HTMLInputElement).value)}
                class="search-input"
              />
            </div>
          )}
        </div>

        <div class="bottom-panel-content">
          {bottomTab === 'summary' && (
            <div class="summary-panel">
              <div class="summary-chart-container">
                <div class="summary-stacked-bar">
                  <div class="bar-segment bar-rendering" style={{ width: `${renderingPct}%` }} title={`Rendering: ${renderingMs.toFixed(2)} ms (${renderingPct.toFixed(1)}%)`} />
                  <div class="bar-segment bar-layout" style={{ width: `${layoutPct}%` }} title={`Layout: ${layoutMs.toFixed(2)} ms (${layoutPct.toFixed(1)}%)`} />
                  <div class="bar-segment bar-painting" style={{ width: `${paintingPct}%` }} title={`Painting: ${paintingMs.toFixed(2)} ms (${paintingPct.toFixed(1)}%)`} />
                  <div class="bar-segment bar-system" style={{ width: `${systemPct}%` }} title={`System: ${systemMs.toFixed(2)} ms (${systemPct.toFixed(1)}%)`} />
                  <div class="bar-segment bar-idle" style={{ width: `${idlePct}%` }} title={`Idle: ${idleMs.toFixed(2)} ms (${idlePct.toFixed(1)}%)`} />
                </div>
                <div class="summary-chart-label">
                  Selected Range: {activeDurationMs.toFixed(1)} ms
                </div>
              </div>
              <div class="summary-legend">
                <div class="legend-item">
                  <span class="legend-color bar-rendering"></span>
                  <span class="legend-label">Rendering:</span>
                  <span class="legend-value">{renderingMs.toFixed(3)} ms ({renderingPct.toFixed(1)}%)</span>
                </div>
                <div class="legend-item">
                  <span class="legend-color bar-layout"></span>
                  <span class="legend-label">Layout:</span>
                  <span class="legend-value">{layoutMs.toFixed(3)} ms ({layoutPct.toFixed(1)}%)</span>
                </div>
                <div class="legend-item">
                  <span class="legend-color bar-painting"></span>
                  <span class="legend-label">Painting:</span>
                  <span class="legend-value">{paintingMs.toFixed(3)} ms ({paintingPct.toFixed(1)}%)</span>
                </div>
                <div class="legend-item">
                  <span class="legend-color bar-system"></span>
                  <span class="legend-label">System:</span>
                  <span class="legend-value">{systemMs.toFixed(3)} ms ({systemPct.toFixed(1)}%)</span>
                </div>
                <div class="legend-item">
                  <span class="legend-color bar-idle"></span>
                  <span class="legend-label">Idle:</span>
                  <span class="legend-value">{idleMs.toFixed(3)} ms ({idlePct.toFixed(1)}%)</span>
                </div>
              </div>
            </div>
          )}

          {bottomTab === 'bottom-up' && (
            <div class="panel-table-container">
              <table class="panel-table">
                <thead>
                  <tr>
                    <th onClick={() => { setSortCol('name'); setSortDir(sortCol === 'name' && sortDir === 'desc' ? 'asc' : 'desc'); }}>Event Name</th>
                    <th class="num-cell" onClick={() => { setSortCol('selfMs'); setSortDir(sortCol === 'selfMs' && sortDir === 'desc' ? 'asc' : 'desc'); }}>Self Time</th>
                    <th class="num-cell">%</th>
                    <th class="num-cell" onClick={() => { setSortCol('totalMs'); setSortDir(sortCol === 'totalMs' && sortDir === 'desc' ? 'asc' : 'desc'); }}>Total Time</th>
                    <th class="num-cell">%</th>
                    <th class="num-cell" onClick={() => { setSortCol('count'); setSortDir(sortCol === 'count' && sortDir === 'desc' ? 'asc' : 'desc'); }}>Count</th>
                  </tr>
                </thead>
                <tbody>
                  {sortedBottomUp.map(node => (
                    <tr key={node.name}>
                      <td class="event-name-cell">{node.name}</td>
                      <td class="num-cell">{node.selfMs.toFixed(3)} ms</td>
                      <td class="num-cell">{node.selfPct.toFixed(1)}%</td>
                      <td class="num-cell">{node.totalMs.toFixed(3)} ms</td>
                      <td class="num-cell">{node.totalPct.toFixed(1)}%</td>
                      <td class="num-cell">{node.count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {bottomTab === 'call-tree' && (
            <div class="panel-table-container">
              <table class="panel-table">
                <thead>
                  <tr>
                    <th>Call Stacks (Aggregated)</th>
                    <th class="num-cell">Self Time</th>
                    <th class="num-cell">%</th>
                    <th class="num-cell">Total Time</th>
                    <th class="num-cell">%</th>
                    <th class="num-cell">Count</th>
                  </tr>
                </thead>
                <tbody>
                  {callTree.length === 0 ? (
                    <tr>
                      <td colSpan={6} class="empty-cell">No main thread events in range</td>
                    </tr>
                  ) : (
                    renderCallTreeRows(callTree)
                  )}
                </tbody>
              </table>
            </div>
          )}

          {bottomTab === 'event-log' && (
            <div class="panel-table-container">
              <table class="panel-table">
                <thead>
                  <tr>
                    <th class="num-cell">Start Time</th>
                    <th class="num-cell">Duration</th>
                    <th class="num-cell">Self Time</th>
                    <th>Thread</th>
                    <th>Event Name</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredEventLog.length === 0 ? (
                    <tr>
                      <td colSpan={5} class="empty-cell">No matching events</td>
                    </tr>
                  ) : (
                    filteredEventLog.map((evt, idx) => (
                      <tr key={idx}>
                        <td class="num-cell">{evt.timeMs.toFixed(3)} ms</td>
                        <td class="num-cell">{evt.durationMs.toFixed(3)} ms</td>
                        <td class="num-cell">{evt.selfMs.toFixed(3)} ms</td>
                        <td>{evt.tid === 1 ? 'Main Thread' : `Worker ${evt.tid}`}</td>
                        <td class="event-name-cell">
                          {evt.name}
                          {evt.jobId && <span class="job-id-pill">{evt.jobId}</span>}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export function parseTraceSpans(events: TraceEvent[]): Span[] {
  const spans: Span[] = [];
  const tids = Array.from(new Set(events.map(e => e.tid)));

  for (const tid of tids) {
    const threadEvents = events.filter(e => e.tid === tid);
    threadEvents.sort((a, b) => {
      if (a.ts !== b.ts) return a.ts - b.ts;
      if (a.ph !== b.ph) return a.ph === 'B' ? -1 : 1;
      return 0;
    });

    const stack: Span[] = [];
    for (const ev of threadEvents) {
      if (ev.ph === 'B') {
        let jobId: string | undefined;
        let jobType: string | undefined;
        if (ev.name.startsWith('JobSubmit:') || ev.name.startsWith('JobRun:')) {
          const parts = ev.name.split(':');
          if (parts.length >= 3) {
            jobType = parts[1];
            jobId = parts[2];
          }
        }
        const span: Span = {
          name: ev.name,
          tid: ev.tid,
          start: ev.ts,
          end: ev.ts,
          duration: 0,
          depth: stack.length,
          jobId,
          jobType,
        };
        stack.push(span);
        spans.push(span);
      } else if (ev.ph === 'E') {
        let matchIdx = -1;
        for (let i = stack.length - 1; i >= 0; i--) {
          if (stack[i].name === ev.name) {
            matchIdx = i;
            break;
          }
        }
        if (matchIdx !== -1) {
          const span = stack[matchIdx];
          span.end = ev.ts;
          span.duration = ev.ts - span.start;
          stack.splice(matchIdx);
        }
      }
    }
  }

  return spans.sort((a, b) => a.start - b.start);
}

function renderRulerMarkers(filterStart: number, filterEnd: number) {
  const durationMs = filterEnd - filterStart;
  const numMarkers = 8;
  const step = durationMs / numMarkers;
  const markers = [];

  for (let i = 0; i <= numMarkers; i++) {
    const val = filterStart + i * step;
    const leftPct = (i / numMarkers) * 100;
    markers.push(
      <div key={i} class="ruler-marker" style={{ left: `${leftPct}%` }}>
        <span class="marker-label">{val.toFixed(1)} ms</span>
      </div>
    );
  }

  return markers;
}
