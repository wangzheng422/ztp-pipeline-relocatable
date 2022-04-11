import { IResource, Metadata } from '../common';

export interface Route extends IResource {
  apiVersion: 'route.openshift.io/v1';
  kind: 'Route';
  metadata: Metadata;
  spec?: {
    host?: string;
  };
}
