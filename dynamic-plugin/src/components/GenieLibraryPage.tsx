import * as React from 'react';
import { useEffect, useState } from 'react';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { DashboardMCPClient } from '../services/dashboardClient';
import { GenieLayout } from './shared';

export default function GenieLibraryPage() {
  const [dashboards, setDashboards] = useState<DashboardListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const client = new DashboardMCPClient(
      `${window.location.origin}/api/proxy/plugin/genie-plugin/dashboard-mcp/`,
    );
    client
      .listDashboards()
      .then(({ layouts }) => {
        setDashboards(layouts || []);
        setLoading(false);
      })
      .catch((e) => {
        setError(e?.message || 'Failed to load dashboards');
        setLoading(false);
      });
  }, []);

  return (
    <GenieLayout title="Library">
      <div style={{ padding: '20px' }}>
        {!loading && !error && (
          <Table aria-label="Dashboards table" variant="compact">
            <Thead>
              <Tr>
                <Th>Name</Th>
                <Th>Description</Th>
                <Th>Active</Th>
                <Th>Created</Th>
                <Th>Updated</Th>
                <Th>Layout ID</Th>
              </Tr>
            </Thead>
            <Tbody>
              {dashboards.map((d) => (
                <Tr key={d.id}>
                  <Td modifier="nowrap">
                    <a href={`/genie/widgets?dashboardId=${d.layoutId}`}>{d.name || d.layoutId}</a>
                  </Td>
                  <Td modifier="nowrap">{d.description || '-'}</Td>
                  <Td>{d.isActive ? '✓' : ''}</Td>
                  <Td>{d.createdAt ? new Date(d.createdAt).toLocaleString() : '-'}</Td>
                  <Td>{d.updatedAt ? new Date(d.updatedAt).toLocaleString() : '-'}</Td>
                  <Td>
                    <code style={{ fontSize: '0.85em' }}>{d.layoutId}</code>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </div>
    </GenieLayout>
  );
}

export type DashboardListItem = {
  id: string;
  layoutId: string;
  name: string;
  description: string;
  isActive?: boolean;
  createdAt?: string;
  updatedAt?: string;
};
