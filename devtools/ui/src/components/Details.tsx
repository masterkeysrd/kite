import { useState } from 'preact/hooks';
import { BoxModel } from './BoxModel';
import { Tabs } from './Tabs';
import { PropertyTable } from './PropertyTable';

interface DetailsProps {
  node: any;
  onJumpToElement?: (domUniqueId: string) => void;
}

function cleanAndFormatPrimitive(val: any): string {
  if (val === null || val === undefined) return '—';
  
  // Handle Go's Option[T] wrapper
  if (typeof val === 'object' && 'Val' in val && 'Set' in val) {
    if (!val.Set) return '—';
    return cleanAndFormatPrimitive(val.Val);
  }
  
  // Handle RGBA colors
  if (typeof val === 'object' && 'R' in val && 'G' in val && 'B' in val) {
    const alpha = val.A !== undefined ? (val.A / 255).toFixed(2) : '1';
    return `rgba(${val.R}, ${val.G}, ${val.B}, ${parseFloat(alpha)})`;
  }
  
  // Handle Edges (margin/padding)
  if (typeof val === 'object' && val.Top !== undefined && val.Right !== undefined && val.Bottom !== undefined && val.Left !== undefined) {
    return `${val.Top} ${val.Right} ${val.Bottom} ${val.Left}`;
  }
  if (typeof val === 'object' && val.top !== undefined && val.right !== undefined && val.bottom !== undefined && val.left !== undefined) {
    return `${val.top} ${val.right} ${val.bottom} ${val.left}`;
  }

  return String(val);
}

interface CollapsibleValueProps {
  value: any;
  label?: string;
  depth?: number;
}

export function CollapsibleValue({ value, label, depth = 0 }: CollapsibleValueProps) {
  let unwrappedValue = value;
  if (typeof value === 'object' && value !== null && 'Val' in value && 'Set' in value) {
    if (!value.Set) {
      unwrappedValue = null;
    } else {
      unwrappedValue = value.Val;
    }
  }

  const [expanded, setExpanded] = useState(depth < 1);

  const isObject = unwrappedValue !== null && typeof unwrappedValue === 'object';
  
  const isColor = isObject && 'R' in unwrappedValue && 'G' in unwrappedValue && 'B' in unwrappedValue;
  const isEdges = isObject && (
    (unwrappedValue.Top !== undefined && unwrappedValue.Right !== undefined) ||
    (unwrappedValue.top !== undefined && unwrappedValue.right !== undefined)
  );

  const isArray = Array.isArray(unwrappedValue);
  const isEmpty = isObject && !isColor && !isEdges && (isArray ? unwrappedValue.length === 0 : Object.keys(unwrappedValue).length === 0);

  if (unwrappedValue === null || unwrappedValue === undefined) {
    return (
      <div class="json-row" style={{ paddingLeft: `${depth * 12}px` }}>
        {label && <span class="json-key">{label}: </span>}
        <span class="json-val json-val-null">—</span>
      </div>
    );
  }

  if (!isObject || isColor || isEdges) {
    const formatted = cleanAndFormatPrimitive(unwrappedValue);
    let valClass = `json-val json-val-${typeof unwrappedValue}`;
    if (isColor) valClass = 'json-val json-val-color';
    return (
      <div class="json-row" style={{ paddingLeft: `${depth * 12}px` }}>
        {label && <span class="json-key">{label}: </span>}
        <span class={valClass}>{typeof unwrappedValue === 'string' && !isColor && !isEdges ? `"${formatted}"` : formatted}</span>
      </div>
    );
  }

  if (isEmpty) {
    return (
      <div class="json-row" style={{ paddingLeft: `${depth * 12}px` }}>
        {label && <span class="json-key">{label}: </span>}
        <span class="json-bracket">{isArray ? '[]' : '{}'}</span>
      </div>
    );
  }

  const entries: [string, any][] = isArray ? unwrappedValue.map((v: any, i: number) => [String(i), v]) : Object.entries(unwrappedValue);
  const filteredEntries = entries.filter(([_, v]: [string, any]) => {
    if (typeof v === 'object' && v !== null && 'Val' in v && 'Set' in v && !v.Set) {
      return false;
    }
    return true;
  });

  const summaryText = isArray ? `Array(${unwrappedValue.length})` : `Object {${filteredEntries.slice(0, 3).map(([k]: [string, any]) => k).join(', ')}${filteredEntries.length > 3 ? '...' : ''}}`;

  return (
    <div class="json-node" style={{ paddingLeft: `${depth * 12}px` }}>
      <div class="json-summary" onClick={() => setExpanded(!expanded)} style={{ cursor: 'pointer', display: 'flex', alignItems: 'center' }}>
        <span class="toggle" style={{ 
          fontSize: '8px', 
          marginRight: '4px', 
          transform: expanded ? 'rotate(90deg)' : 'none', 
          display: 'inline-block', 
          transition: 'transform 0.1s',
          color: 'var(--text-muted)'
        }}>
          ▶
        </span>
        {label && <span class="json-key" style={{ marginRight: '4px' }}>{label}: </span>}
        <span class="json-type-summary" style={{ color: 'var(--text-muted)', fontSize: '10px' }}>{summaryText}</span>
      </div>
      {expanded && (
        <div class="json-children" style={{ borderLeft: '1px dashed var(--border-color)', marginLeft: '4px', marginTop: '2px' }}>
          {filteredEntries.map(([key, val]: [string, any]) => (
            <CollapsibleValue key={key} label={key} value={val} depth={1} />
          ))}
        </div>
      )}
    </div>
  );
}

export function Details({ node, onJumpToElement }: DetailsProps) {
  const [activeTab, setActiveTab] = useState('computed');
  const [activeVDOMTab, setActiveVDOMTab] = useState('props');

  if (!node) return <div class="details-empty">Select a node to see details</div>;

  const isVDOM = node.uniqueId && node.uniqueId.startsWith('vdom-');

  if (isVDOM) {
    const getFileBasename = (path: string) => {
      if (!path) return '';
      const parts = path.split('/');
      return parts[parts.length - 1];
    };

    const vdomTabs = [
      {
        id: 'props',
        label: 'Props',
        content: (
          <div class="tab-panel">
            <div class="props-list" style={{ display: 'flex', flexDirection: 'column', gap: '4px', padding: '4px 0' }}>
              {node.props && Object.keys(node.props).length > 0 ? (
                Object.entries(node.props).map(([key, val]: [string, any]) => (
                  <CollapsibleValue key={key} label={key} value={val} />
                ))
              ) : (
                <div class="empty-msg">No props available</div>
              )}
            </div>
          </div>
        )
      },
      {
        id: 'hooks',
        label: 'Hooks',
        content: (
          <div class="tab-panel">
            <div class="hooks-list" style={{ display: 'flex', flexDirection: 'column', gap: '4px', padding: '4px 0' }}>
              {node.state && node.state.length > 0 ? (
                node.state.map((stateVal: any, idx: number) => {
                  const label = node.state.length === 1 ? 'State' : `State [${idx}]`;
                  return (
                    <CollapsibleValue key={idx} label={label} value={stateVal} />
                  );
                })
              ) : (
                <div class="empty-msg">No hooks state available</div>
              )}
            </div>
          </div>
        )
      },
      {
        id: 'source',
        label: 'Source',
        content: (
          <div class="tab-panel">
            <div class="source-group">
              <div class="source-title">Declaration Site</div>
              {node.declFile ? (
                <div class="source-value" title={node.declFile}>
                  <span class="file-icon">📄</span> {getFileBasename(node.declFile)}:{node.declLine}
                  <div class="full-path">{node.declFile}</div>
                </div>
              ) : (
                <div class="empty-msg">No declaration site found</div>
              )}
            </div>

            <div class="source-group" style={{ marginTop: '16px' }}>
              <div class="source-title">Instantiation Site</div>
              {node.instFile ? (
                <div class="source-value" title={node.instFile}>
                  <span class="file-icon">⚡</span> {getFileBasename(node.instFile)}:{node.instLine}
                  <div class="full-path">{node.instFile}</div>
                </div>
              ) : (
                <div class="empty-msg">No instantiation site found (EnableDevMode must be true)</div>
              )}
            </div>
          </div>
        )
      },
      {
        id: 'raw',
        label: 'Raw',
        content: (
          <div class="tab-panel">
            <pre class="raw-data">{JSON.stringify(node, null, 2)}</pre>
          </div>
        )
      }
    ];

    return (
      <div class="details-container">
        <div class="details-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span class="component-name">&lt;{node.name}&gt;</span>
            {node.key && <span class="badge badge-key">key={node.key}</span>}
          </div>
          {node.domUniqueId && onJumpToElement && (
            <button 
              class="btn-jump" 
              onClick={() => onJumpToElement(node.domUniqueId!)}
              title="Jump to associated physical DOM Element"
            >
              🔍 Jump to Element
            </button>
          )}
        </div>
        <Tabs tabs={vdomTabs} activeTab={activeVDOMTab} onChange={setActiveVDOMTab} />
      </div>
    );
  }

  const tabs = [
    {
      id: 'computed',
      label: 'Computed',
      content: (
        <div class="tab-panel">
          <BoxModel computed={node.computed} layout={node.rect} />
          <PropertyTable data={node.computed || {}} title="Computed Properties" highlightValues />
        </div>
      )
    },
    {
      id: 'styles',
      label: 'Styles',
      content: (
        <div class="tab-panel">
          <PropertyTable data={node.raw || {}} title="Element Styles" highlightValues />
          <PropertyTable data={node.default || {}} title="Default Styles" highlightValues />
          <PropertyTable data={node.intrinsic || {}} title="Intrinsic Styles" highlightValues />
        </div>
      )
    },
    {
      id: 'fragments',
      label: 'Fragments',
      content: (
        <div class="tab-panel">
           {node.fragment ? (
             <pre class="raw-data">{JSON.stringify(node.fragment, null, 2)}</pre>
           ) : (
             <div class="empty-msg">No fragment data available for this node</div>
           )}
        </div>
      )
    },
    {
      id: 'raw',
      label: 'Raw',
      content: (
        <div class="tab-panel">
          <pre class="raw-data">{JSON.stringify(node, null, 2)}</pre>
        </div>
      )
    }
  ];

  return (
    <div class="details-container">
      <div class="details-header">
        <span class="tag-name">{node.name}</span>
        {node.id && <span class="badge badge-id">#{node.id}</span>}
      </div>
      <Tabs tabs={tabs} activeTab={activeTab} onChange={setActiveTab} />
    </div>
  );
}


