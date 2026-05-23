import { useState } from 'preact/hooks';
import { nodeUniqueId } from '../utils';

interface TreeProps {
  node: any;
  overlays?: any[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}

export function Tree({ node, overlays, selectedId, onSelect }: TreeProps) {
  return (
    <div class="tree-content">
      <TreeNode node={node} selectedId={selectedId} onSelect={onSelect} depth={0} />
      {overlays && overlays.length > 0 && (
        <div class="overlays-section">
          <div class="section-title">Overlays</div>
          {overlays.map((overlay: any) => (
            <TreeNode key={nodeUniqueId(overlay)} node={overlay} selectedId={selectedId} onSelect={onSelect} depth={0} />
          ))}
        </div>
      )}
    </div>
  );
}

interface TreeNodeProps {
  node: any;
  selectedId: string | null;
  onSelect: (id: string) => void;
  depth: number;
}

function TreeNode({ node, selectedId, onSelect, depth }: TreeNodeProps) {
  const [expanded, setExpanded] = useState(depth < 2);
  const id = nodeUniqueId(node);
  const isSelected = selectedId === id;

  const hasChildren = node.children && node.children.length > 0;

  return (
    <div class="node-wrapper">
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
        <span class="tag-name">{node.name}</span>
        {node.id && <span class="badge badge-id">#{node.id}</span>}
        {node.class && <span class="badge badge-class">.{node.class}</span>}
        {node.text && <span class="text-preview">"{node.text}"</span>}
      </div>
      {expanded && hasChildren && (
        <div class="node-children">
          {node.children.map((child: any) => (
            <TreeNode 
              key={nodeUniqueId(child)} 
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
