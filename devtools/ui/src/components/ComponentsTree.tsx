import { useState } from 'preact/hooks';

interface ComponentsTreeProps {
  roots: any[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}

export function ComponentsTree({ roots, selectedId, onSelect }: ComponentsTreeProps) {
  if (!roots || roots.length === 0) {
    return <div class="empty-msg">No components found</div>;
  }

  return (
    <div class="tree-content">
      {roots.map((root: any) => (
        <ComponentsTreeNode 
          key={root.uniqueId} 
          node={root} 
          selectedId={selectedId} 
          onSelect={onSelect} 
          depth={0} 
        />
      ))}
    </div>
  );
}

interface TreeNodeProps {
  node: any;
  selectedId: string | null;
  onSelect: (id: string) => void;
  depth: number;
}

function ComponentsTreeNode({ node, selectedId, onSelect, depth }: TreeNodeProps) {
  const [expanded, setExpanded] = useState(depth < 3);
  const id = node.uniqueId;
  const isSelected = selectedId === id;
  const hasChildren = node.children && node.children.length > 0;

  const displayName = node.name || 'Component';

  return (
    <div class="node-wrapper" style={{ marginLeft: depth > 0 ? '16px' : '0' }}>
      <div 
        class={`node-header ${isSelected ? 'selected' : ''}`} 
        onClick={() => onSelect(id)}
      >
        <span 
          class={`toggle ${hasChildren ? '' : 'hidden'}`} 
          onClick={(e) => { e.stopPropagation(); setExpanded(!expanded); }}
        >
          {hasChildren ? (expanded ? '▼' : '▶') : ''}
        </span>
        <span class="component-tag-name">&lt;{displayName}&gt;</span>
        {node.key && <span class="badge badge-key">key={node.key}</span>}
        {node.domId && <span class="badge badge-id">#{node.domId}</span>}
      </div>
      {expanded && hasChildren && (
        <div class="node-children">
          {node.children.map((child: any) => (
            <ComponentsTreeNode 
              key={child.uniqueId} 
              node={child} 
              selectedId={selectedId} 
              onSelect={onSelect} 
              depth={depth + 1} 
            />
          ))}
        </div>
      )}
    </div>
  );
}
