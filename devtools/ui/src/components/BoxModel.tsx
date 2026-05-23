interface BoxModelProps {
  computed: any;
  layout?: any;
}

export function BoxModel({ computed, layout }: BoxModelProps) {
  if (!computed && !layout) return <div class="box-model-placeholder">No box model data</div>;

  const margin = computed?.margin || { top: 0, right: 0, bottom: 0, left: 0 };
  const padding = computed?.padding || { top: 0, right: 0, bottom: 0, left: 0 };
  const borderEdges = computed?.border?.edges || { top: false, right: false, bottom: false, left: false };
  
  const getBorderWidth = (side: string) => borderEdges[side] ? 1 : 0;

  return (
    <div class="box-model-wrapper">
      <div class="box-model">
        <div class="bm-layer bm-margin">
          <span class="bm-label">margin</span>
          <div class="bm-values">
            <span class="bm-v-top">{margin.top}</span>
            <div class="bm-middle">
              <span class="bm-v-left">{margin.left}</span>
              <div class="bm-layer bm-border">
                <span class="bm-label">border</span>
                <div class="bm-values">
                  <span class="bm-v-top">{getBorderWidth('top')}</span>
                  <div class="bm-middle">
                    <span class="bm-v-left">{getBorderWidth('left')}</span>
                    <div class="bm-layer bm-padding">
                      <span class="bm-label">padding</span>
                      <div class="bm-values">
                        <span class="bm-v-top">{padding.top}</span>
                        <div class="bm-middle">
                          <span class="bm-v-left">{padding.left}</span>
                          <div class="bm-layer bm-content">
                             {layout?.size ? `${Math.round(layout.size.Width ?? 0)} × ${Math.round(layout.size.Height ?? 0)}` : 'content'}
                          </div>
                          <span class="bm-v-right">{padding.right}</span>
                        </div>
                        <span class="bm-v-bottom">{padding.bottom}</span>
                      </div>
                    </div>
                    <span class="bm-v-right">{getBorderWidth('right')}</span>
                  </div>
                  <span class="bm-v-bottom">{getBorderWidth('bottom')}</span>
                </div>
              </div>
              <span class="bm-v-right">{margin.right}</span>
            </div>
            <span class="bm-v-bottom">{margin.bottom}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
