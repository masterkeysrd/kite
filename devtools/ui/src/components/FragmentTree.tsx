import { useState } from 'preact/hooks';

interface FragmentTreeProps {
  fragment: any;
  overlayFragments?: any[];
}

export function FragmentTree({ fragment, overlayFragments }: FragmentTreeProps) {
  if (!fragment) return <div class="empty-msg">No fragments available</div>;

  return (
    <div class="tree-content">
      <div class="section-title">Main Viewport Fragments</div>
      <FragmentNode node={fragment} depth={0} />
      {overlayFragments && overlayFragments.length > 0 && (
        <div class="overlays-section">
          <div class="section-title">Overlay Fragments</div>
          {overlayFragments.map((f: any, i: number) => (
            <FragmentNode key={i} node={f} depth={0} />
          ))}
        </div>
      )}
    </div>
  );
}

function FragmentNode({ node, depth }: { node: any; depth: number }) {
  if (!node) return null;
  const [expanded, setExpanded] = useState(depth < 2);
  const hasChildren = node.children && node.children.length > 0;

  return (
    <div class="node-wrapper">
      <div class="node-header fragment-node">
        <span 
          class={`toggle ${hasChildren ? '' : 'hidden'}`} 
          onClick={(e) => { e.stopPropagation(); setExpanded(!expanded); }}
        >
          {hasChildren ? (expanded ? '▼' : '▶') : ''}
        </span>
        <span class="fragment-name">Fragment({node.name})</span>
        <span class="fragment-info">
          offset={node.offset?.X ?? 0},{node.offset?.Y ?? 0} size={Math.round(node.size?.Width ?? 0)}x{Math.round(node.size?.Height ?? 0)}
        </span>
      </div>
      {expanded && hasChildren && (
        <div class="node-children">
          {node.children.map((child: any, i: number) => (
            <FragmentNode key={i} node={child} depth={depth + 1} />
          ))}
        </div>
      )}
      {expanded && node.clusters && (
        <div class="node-clusters">
          {node.clusters.map((c: any, i: number) => (
            <div key={i} class="cluster">
              <span class="cluster-text">"{c.text}"</span>
              <span class="cluster-info">width={c.width} break={c.breakClass}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
