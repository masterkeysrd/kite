interface PropertyTableProps {
  data: Record<string, any>;
  title?: string;
  highlightValues?: boolean;
}

export function PropertyTable({ data, title, highlightValues }: PropertyTableProps) {
  if (!data || Object.keys(data).length === 0) {
    return (
      <div class="property-table-section">
        {title && <div class="section-header">{title}</div>}
        <div class="empty-msg">No properties to display</div>
      </div>
    );
  }

  const formatValue = (val: any): string => {
    if (val === null || val === undefined) return '—';
    
    // Handle Go's Option[T] wrapper
    if (typeof val === 'object' && 'Val' in val && 'Set' in val) {
      if (!val.Set) return '—';
      return formatValue(val.Val);
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
    
    if (typeof val === 'object') {
      if (Array.isArray(val)) {
        if (val.length === 0) return '[]';
        return val.map(item => formatValue(item)).join(' | ');
      }
      const entries = Object.entries(val)
        .map(([k, v]) => {
          const formatted = formatValue(v);
          if (formatted === '—') return null;
          return `${k}: ${formatted}`;
        })
        .filter(x => x !== null);
        
      if (entries.length === 0) return '—';
      return entries.join(', ');
    }
    
    return String(val);
  };

  const isDefaultValue = (val: any) => {
    const s = String(val);
    const defaults = ['0', 'none', 'auto', 'inherit', 'initial', 'transparent', 'false', '0 0 0 0'];
    return defaults.includes(s.toLowerCase());
  };

  return (
    <div class="property-table-section">
      {title && <div class="section-header">{title}</div>}
      <table class="props-table">
        <tbody>
          {Object.entries(data).sort().map(([key, value]) => {
            const formatted = formatValue(value);
            const isMuted = highlightValues && isDefaultValue(formatted);
            return (
              <tr key={key} class={isMuted ? 'prop-row-muted' : ''}>
                <td class="prop-name">{key}</td>
                <td class={`prop-value ${highlightValues && !isMuted ? 'highlight' : ''}`}>{formatted}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
