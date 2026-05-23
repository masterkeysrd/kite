interface Tab {
  id: string;
  label: string;
  content: import('preact').ComponentChildren;
}

interface TabsProps {
  tabs: Tab[];
  activeTab: string;
  onChange: (id: string) => void;
}

export function Tabs({ tabs, activeTab, onChange }: TabsProps) {
  return (
    <div class="tabs-container">
      <div class="tabs-header">
        {tabs.map((tab) => (
          <div
            key={tab.id}
            class={`tab ${activeTab === tab.id ? 'active' : ''}`}
            onClick={() => onChange(tab.id)}
          >
            {tab.label}
          </div>
        ))}
      </div>
      <div class="tab-content">
        {tabs.find((t) => t.id === activeTab)?.content}
      </div>
    </div>
  );
}
