import React from 'react';

import { Page } from '../Page';
import { ContentThreeRows } from '../ContentThreeRows';

export const SubnetPage: React.FC = () => {
  return (
    <Page>
      <ContentThreeRows
        top={<div>Top</div>}
        middle={<div>middle</div>}
        bottom={<div>bottom</div>}
      />
    </Page>
  );
};
