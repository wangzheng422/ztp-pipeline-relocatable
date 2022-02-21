import React from 'react';
import { ProgressStepper, ProgressStep } from '@patternfly/react-core';

import { useWizardProgressContext } from './WizardProgressContext';

import './WizardProgress.css';

export const WizardProgress: React.FC = () => {
  const { steps } = useWizardProgressContext();

  return (
    <ProgressStepper isCenterAligned className="wizard-progress">
      <ProgressStep
        // description=""
        id="apiaddr"
        titleId="apiaddr-title"
        aria-label="API step"
        {...steps.apiaddr}
      >
        API
      </ProgressStep>
      <ProgressStep
        id="ingressip"
        titleId="ingressip-title"
        aria-label="Ingress step"
        {...steps.ingressip}
      >
        Ingress
      </ProgressStep>
      <ProgressStep id="domain" titleId="domain-title" aria-label="Domain step" {...steps.domain}>
        Domain
      </ProgressStep>
      <ProgressStep id="sshkey" titleId="sshkey-title" aria-label="Ssh key step" {...steps.sshkey}>
        SSH key
      </ProgressStep>
    </ProgressStepper>
  );
};
