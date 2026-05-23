import { useState } from 'preact/hooks';
import { BoxModel } from './BoxModel';
import { Tabs } from './Tabs';
import { PropertyTable } from './PropertyTable';

interface DetailsProps {
  node: any;
}

export function Details({ node }: DetailsProps) {
  const [activeTab, setActiveTab] = useState('computed');

  if (!node) return <div class="details-empty">Select a node to see details</div>;

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
