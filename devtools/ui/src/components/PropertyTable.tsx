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
    if (typeof val === 'object') {
      if (val.top !== undefined && val.right !== undefined) {
        return `${val.top} ${val.right} ${val.bottom} ${val.left}`;
      }
      return JSON.stringify(val);
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
