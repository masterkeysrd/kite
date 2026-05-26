export function nodeUniqueId(node: any): string {
  if (!node) return '';
  const kind = node.kind || 'unknown';
  const name = node.name || 'unnamed';
  const id = node.id || '';
  const x = node.rect?.Origin?.X ?? 0;
  const y = node.rect?.Origin?.Y ?? 0;
  return `${kind}:${name}:${id}:${x},${y}`;
}

export function findNodeByIdInPayload(payload: any, id: string): any {
  if (!payload) return null;
  let found = findNodeById(payload.dom, id);
  if (found) return found;
  if (payload.overlays) {
    for (const overlay of payload.overlays) {
      found = findNodeById(overlay, id);
      if (found) return found;
    }
  }
  return null;
}

function findNodeById(node: any, id: string): any {
  if (nodeUniqueId(node) === id) return node;
  if (node.children) {
    for (const child of node.children) {
      const found = findNodeById(child, id);
      if (found) return found;
    }
  }
  return null;
}

export function findVDOMNodeById(roots: any[] | undefined, id: string): any {
  if (!roots) return null;
  for (const root of roots) {
    const found = findVDOMNode(root, id);
    if (found) return found;
  }
  return null;
}

function findVDOMNode(node: any, id: string): any {
  if (node.uniqueId === id) return node;
  if (node.children) {
    for (const child of node.children) {
      const found = findVDOMNode(child, id);
      if (found) return found;
    }
  }
  return null;
}

